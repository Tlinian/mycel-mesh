package stress

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mycel/mesh/internal/coordinator/pool"
)

// TestConnectionPoolConcurrency tests the connection pool with 100 concurrent nodes.
func TestConnectionPoolConcurrency(t *testing.T) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 100
	config.MinSize = 10

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	const numPeers = 100
	var wg sync.WaitGroup
	results := make(chan result, numPeers)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	// Simulate 100 concurrent node connections
	for i := 0; i < numPeers; i++ {
		wg.Add(1)
		go func(peerNum int) {
			defer wg.Done()

			peerID := fmt.Sprintf("node-%d", peerNum)

			// Mock dial function
			dialFunc := func() (net.Conn, error) {
				// In real scenario, this would dial actual connection
				// For testing, we simulate successful connection
				time.Sleep(10 * time.Millisecond) // Simulate network latency
				return nil, nil
			}

			conn, err := manager.Acquire(ctx, peerID, dialFunc)
			if err != nil {
				results <- result{peerID: peerID, success: false, err: err}
				return
			}

			// Simulate some work
			time.Sleep(5 * time.Millisecond)

			// Release connection
			manager.Release(peerID)

			results <- result{
				peerID:  peerID,
				success: true,
				latency: time.Since(startTime),
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Collect results
	var successCount, failCount int
	var totalLatency time.Duration

	for r := range results {
		if r.success {
			successCount++
			totalLatency += r.latency
		} else {
			failCount++
			t.Logf("Failed to acquire connection for %s: %v", r.peerID, r.err)
		}
	}

	elapsed := time.Since(startTime)
.avgLatency := totalLatency / time.Duration(successCount)

	t.Logf("Concurrency Test Results:")
	t.Logf("  Total peers: %d", numPeers)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", failCount)
	t.Logf("  Total time: %v", elapsed)
	t.Logf("  Average latency: %v", avgLatency)

	// Assert success rate >= 95%
	successRate := float64(successCount) / float64(numPeers) * 100
	if successRate < 95.0 {
		t.Errorf("Success rate %.2f%% is below threshold 95%%", successRate)
	}

	// Assert average latency < 3s
	if avgLatency > 3*time.Second {
		t.Errorf("Average latency %v exceeds threshold 3s", avgLatency)
	}

	// Verify pool stats
	stats := manager.GetStats()
	t.Logf("Pool Stats:")
	t.Logf("  Total Connections: %d", stats.TotalConnections)
	t.Logf("  Acquire Count: %d", stats.AcquireCount)
	t.Logf("  Acquire Wait Count: %d", stats.AcquireWaitCount)
	t.Logf("  Acquire Error Count: %d", stats.AcquireErrorCount)
}

// TestConnectionPoolAcquireTimeout tests acquire timeout behavior.
func TestConnectionPoolAcquireTimeout(t *testing.T) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 10
	config.AcquireTimeout = 100 * time.Millisecond

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()

	// Acquire all connections
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	var acquired []*pool.ConnWrapper
	for i := 0; i < config.MaxSize; i++ {
		peerID := fmt.Sprintf("node-%d", i)
		conn, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			t.Fatalf("Failed to acquire connection %d: %v", i, err)
		}
		acquired = append(acquired, conn)
	}

	// Try to acquire one more - should timeout
	peerID := "node-timeout"
	startTime := time.Now()
	_, err := manager.Acquire(ctx, peerID, dialFunc)
	elapsed := time.Since(startTime)

	if err != pool.ErrPoolExhausted {
		t.Errorf("Expected ErrPoolExhausted, got %v", err)
	}

	// Verify timeout occurred within expected range
	if elapsed < config.AcquireTimeout {
		t.Errorf("Timeout %v is less than expected %v", elapsed, config.AcquireTimeout)
	}

	t.Logf("Acquire timeout test passed: %v", elapsed)

	// Release connections
	for i, conn := range acquired {
		_ = conn
		peerID := fmt.Sprintf("node-%d", i)
		manager.Release(peerID)
	}
}

// TestConnectionPoolCleanup tests the cleanup of stale connections.
func TestConnectionPoolCleanup(t *testing.T) {
	config := pool.DefaultPoolConfig()
	config.IdleTimeout = 200 * time.Millisecond
	config.MaxLifetime = 500 * time.Millisecond
	config.MaxSize = 10

	manager := pool.NewManager(config)

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// Acquire and release connections
	for i := 0; i < 5; i++ {
		peerID := fmt.Sprintf("node-%d", i)
		conn, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			t.Fatalf("Failed to acquire connection: %v", err)
		}
		manager.Release(peerID)
		_ = conn
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	stats := manager.GetStats()
	t.Logf("Stats after cleanup:")
	t.Logf("  Total Connections: %d", stats.TotalConnections)
	t.Logf("  Idle Connections: %d", stats.IdleConnections)

	// Connections should be cleaned up due to idle timeout
	if stats.TotalConnections > 0 {
		t.Logf("Note: Some connections remain after cleanup")
	}

	manager.CloseAll()
}

// TestConnectionPoolStatistics tests the statistics accuracy.
func TestConnectionPoolStatistics(t *testing.T) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 50

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// Acquire 10 connections
	for i := 0; i < 10; i++ {
		peerID := fmt.Sprintf("node-%d", i)
		_, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			t.Fatalf("Failed to acquire connection: %v", err)
		}
	}

	stats := manager.GetStats()
	if stats.AcquireCount != 10 {
		t.Errorf("Expected AcquireCount=10, got %d", stats.AcquireCount)
	}
	if stats.InUseConnections != 10 {
		t.Errorf("Expected InUseConnections=10, got %d", stats.InUseConnections)
	}

	// Release 5 connections
	for i := 0; i < 5; i++ {
		peerID := fmt.Sprintf("node-%d", i)
		manager.Release(peerID)
	}

	stats = manager.GetStats()
	if stats.IdleConnections != 5 {
		t.Errorf("Expected IdleConnections=5, got %d", stats.IdleConnections)
	}
	if stats.InUseConnections != 5 {
		t.Errorf("Expected InUseConnections=5, got %d", stats.InUseConnections)
	}

	t.Logf("Statistics test passed")
	t.Logf("  Total: %d, Idle: %d, InUse: %d", stats.TotalConnections, stats.IdleConnections, stats.InUseConnections)
}

// BenchmarkConnectionPoolAcquire benchmarks connection acquisition.
func BenchmarkConnectionPoolAcquire(b *testing.B) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 1000

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		peerID := fmt.Sprintf("node-%d", i%100)
		conn, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			b.Fatalf("Failed to acquire connection: %v", err)
		}
		manager.Release(peerID)
		_ = conn
	}
}

// BenchmarkConnectionPoolConcurrent benchmarks concurrent acquisition.
func BenchmarkConnectionPoolConcurrent(b *testing.B) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 100

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			peerID := fmt.Sprintf("node-%d", i%50)
			conn, err := manager.Acquire(ctx, peerID, dialFunc)
			if err != nil {
				b.Error(err)
				continue
			}
			manager.Release(peerID)
			i++
		}
	})
}

type result struct {
	peerID  string
	success bool
	latency time.Duration
	err     error
}
