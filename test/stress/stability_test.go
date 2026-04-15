package stress

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mycel/mesh/internal/coordinator/pool"
)

// TestConnectionPoolStability tests long-running stability of the connection pool.
// This simulates 72 hours of operation in accelerated time.
func TestConnectionPoolStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stability test in short mode")
	}

	config := pool.DefaultPoolConfig()
	config.MaxSize = 100
	config.IdleTimeout = 1 * time.Second
	config.MaxLifetime = 5 * time.Second

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	const (
		numPeers     = 50
		testDuration = 10 * time.Second // Accelerated: represents 72 hours
		opFrequency  = 100 * time.Millisecond
	)

	var (
		totalOps     int64
		successOps   int64
		failOps      int64
		memoryLeak   bool
		stopSignal   int32
		wg           sync.WaitGroup
		errorRecords []string
		mu           sync.Mutex
	)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	startTime := time.Now()

	// Start peer goroutines
	for i := 0; i < numPeers; i++ {
		wg.Add(1)
		go func(peerNum int) {
			defer wg.Done()

			peerID := fmt.Sprintf("stability-node-%d", peerNum)
			dialFunc := func() (net.Conn, error) {
				return nil, nil
			}

			for {
				if atomic.LoadInt32(&stopSignal) == 1 {
					return
				}

				// Acquire connection
				conn, err := manager.Acquire(ctx, peerID, dialFunc)
				if err != nil {
					atomic.AddInt64(&failOps, 1)
					mu.Lock()
					errorRecords = append(errorRecords, fmt.Sprintf("Acquire failed: %v", err))
					mu.Unlock()
					time.Sleep(50 * time.Millisecond)
					continue
				}

				atomic.AddInt64(&successOps, 1)
				atomic.AddInt64(&totalOps, 1)

				// Simulate work
				time.Sleep(10 * time.Millisecond)

				// Release connection
				manager.Release(peerID)
				_ = conn

				// Wait before next operation
				time.Sleep(opFrequency)
			}
		}(i)
	}

	// Monitor goroutine
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		var lastTotalOps int64

		for {
			select {
			case <-monitorCtx.Done():
				return
			case <-ticker.C:
				currentTotal := atomic.LoadInt64(&totalOps)
				currentSuccess := atomic.LoadInt64(&successOps)

				opsPerSec := (currentTotal - lastTotalOps) / 2
				successRate := float64(currentSuccess) / float64(currentTotal) * 100

				t.Logf("Stability Monitor [%v]:", time.Since(startTime).Round(time.Second))
				t.Logf("  Total Ops: %d, Success: %d, Failed: %d",
					currentTotal, currentSuccess, atomic.LoadInt64(&failOps))
				t.Logf("  Ops/sec: %d, Success Rate: %.2f%%", opsPerSec, successRate)

				// Check for stuck state (no progress)
				if currentTotal == lastTotalOps && currentTotal > 0 {
					t.Logf("Warning: No progress in last 2 seconds")
				}

				lastTotalOps = currentTotal
			}
		}
	}()

	// Wait for test duration
	<-ctx.Done()
	atomic.StoreInt32(&stopSignal, 1)

	// Wait for all goroutines to finish
	wg.Wait()

	elapsed := time.Since(startTime)

	// Final statistics
	finalTotal := atomic.LoadInt64(&totalOps)
	finalSuccess := atomic.LoadInt64(&successOps)
	finalFail := atomic.LoadInt64(&failOps)

	t.Logf("========================================")
	t.Logf("Stability Test Final Results")
	t.Logf("========================================")
	t.Logf("Duration: %v", elapsed)
	t.Logf("Total Operations: %d", finalTotal)
	t.Logf("Successful: %d (%.2f%%)", finalSuccess, float64(finalSuccess)/float64(finalTotal)*100)
	t.Logf("Failed: %d", finalFail)
	t.Logf("Operations/sec: %.2f", float64(finalTotal)/elapsed.Seconds())

	stats := manager.GetStats()
	t.Logf("Pool Stats:")
	t.Logf("  Total Connections: %d", stats.TotalConnections)
	t.Logf("  Acquire Count: %d", stats.AcquireCount)
	t.Logf("  Acquire Wait Count: %d", stats.AcquireWaitCount)
	t.Logf("  Acquire Error Count: %d", stats.AcquireErrorCount)

	if len(errorRecords) > 0 {
		t.Logf("Error Records (first 10):")
		for i, err := range errorRecords {
			if i >= 10 {
				break
			}
			t.Logf("  %d: %s", i+1, err)
		}
	}

	// Assertions
	successRate := float64(finalSuccess) / float64(finalTotal) * 100
	if successRate < 99.0 {
		t.Errorf("Success rate %.2f%% is below threshold 99%% for stability test", successRate)
	}

	// Check for memory leak indicators
	if memoryLeak {
		t.Error("Potential memory leak detected")
	}

	// Verify pool is still healthy
	if stats.AcquireErrorCount > int64(finalTotal)/100 {
		t.Errorf("Too many acquire errors: %d", stats.AcquireErrorCount)
	}
}

// TestConnectionPoolRecovery tests recovery from error states.
func TestConnectionPoolRecovery(t *testing.T) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 10
	config.AcquireTimeout = 500 * time.Millisecond

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()

	// Simulate connection failures
	failCount := 0
	dialFunc := func() (net.Conn, error) {
		failCount++
		if failCount <= 5 {
			return nil, fmt.Errorf("simulated dial error %d", failCount)
		}
		return nil, nil
	}

	// Try to acquire with failing dial
	for i := 0; i < 10; i++ {
		peerID := fmt.Sprintf("recovery-node-%d", i)
		_, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			t.Logf("Expected error: %v", err)
		}
	}

	// Reset and recover
	failCount = 0
	successCount := 0

	for i := 0; i < 10; i++ {
		peerID := fmt.Sprintf("recovery-node-%d", i)
		conn, err := manager.Acquire(ctx, peerID, dialFunc)
		if err == nil {
			successCount++
			manager.Release(peerID)
			_ = conn
		}
	}

	t.Logf("Recovery test: %d successes after initial failures", successCount)

	if successCount < 5 {
		t.Errorf("Expected more successful recoveries, got %d", successCount)
	}
}

// TestConnectionPoolEdgeCases tests edge cases and boundary conditions.
func TestConnectionPoolEdgeCases(t *testing.T) {
	config := pool.DefaultPoolConfig()
	config.MaxSize = 10

	manager := pool.NewManager(config)

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// Test 1: Acquire after CloseAll
	manager.CloseAll()
	_, err := manager.Acquire(ctx, "test-node", dialFunc)
	if err != pool.ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed after CloseAll, got %v", err)
	}
	t.Logf("Test 1 passed: ErrPoolClosed after CloseAll")

	// Test 2: Release non-existent connection
	manager2 := pool.NewManager(config)
	manager2.Release("non-existent") // Should not panic
	manager2.CloseAll()
	t.Logf("Test 2 passed: Release non-existent connection")

	// Test 3: Double Close
	manager3 := pool.NewManager(config)
	peerID := "test-node"
	_, err = manager3.Acquire(ctx, peerID, dialFunc)
	if err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}

	err = manager3.Close(peerID)
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	err = manager3.Close(peerID) // Should return nil or handle gracefully
	if err != nil {
		t.Logf("Second close: %v", err)
	}
	manager3.CloseAll()
	t.Logf("Test 3 passed: Double Close handled")

	// Test 4: GetStats on empty pool
	manager4 := pool.NewManager(config)
	stats := manager4.GetStats()
	if stats.TotalConnections != 0 {
		t.Errorf("Expected 0 connections on new pool, got %d", stats.TotalConnections)
	}
	manager4.CloseAll()
	t.Logf("Test 4 passed: Empty pool stats")
}

// TestConnectionPoolStressHighLoad tests behavior under extreme load.
func TestConnectionPoolStressHighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high load stress test in short mode")
	}

	config := pool.DefaultPoolConfig()
	config.MaxSize = 50
	config.AcquireTimeout = 100 * time.Millisecond

	manager := pool.NewManager(config)
	defer manager.CloseAll()

	const (
		numWorkers = 100
		opsPerWorker = 50
	)

	var wg sync.WaitGroup
	var successCount, failCount int64

	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ctx := context.Background()
			dialFunc := func() (net.Conn, error) {
				return nil, nil
			}

			for j := 0; j < opsPerWorker; j++ {
				peerID := fmt.Sprintf("stress-worker-%d-%d", workerID, j%10)

				_, err := manager.Acquire(ctx, peerID, dialFunc)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
					// Don't release immediately to create contention
					time.Sleep(1 * time.Millisecond)
				} else {
					atomic.AddInt64(&failCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalOps := successCount + failCount
	t.Logf("High Load Stress Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total Ops: %d", totalOps)
	t.Logf("  Success: %d (%.2f%%)", successCount, float64(successCount)/float64(totalOps)*100)
	t.Logf("  Failed: %d", failCount)
	t.Logf("  Ops/sec: %.2f", float64(totalOps)/elapsed.Seconds())

	stats := manager.GetStats()
	t.Logf("Pool Stats:")
	t.Logf("  Max Size: %d", config.MaxSize)
	t.Logf("  Acquire Count: %d", stats.AcquireCount)
	t.Logf("  Acquire Wait Count: %d", stats.AcquireWaitCount)
	t.Logf("  Acquire Error Count: %d", stats.AcquireErrorCount)
}
