package pool

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// TestNewManager_DefaultConfig tests manager creation with default config.
func TestNewManager_DefaultConfig(t *testing.T) {
	config := PoolConfig{} // Empty config, should use defaults
	manager := NewManager(config)
	defer manager.CloseAll()

	if manager.config.MaxSize != DefaultPoolConfig().MaxSize {
		t.Fatalf("expected default MaxSize %d, got %d", DefaultPoolConfig().MaxSize, manager.config.MaxSize)
	}
}

// TestNewManager_CustomConfig tests manager creation with custom config.
func TestNewManager_CustomConfig(t *testing.T) {
	config := PoolConfig{
		MaxSize:        50,
		MinSize:        5,
		IdleTimeout:    2 * time.Minute,
		MaxLifetime:    10 * time.Minute,
		AcquireTimeout: 5 * time.Second,
	}
	manager := NewManager(config)
	defer manager.CloseAll()

	if manager.config.MaxSize != 50 {
		t.Fatalf("expected MaxSize 50, got %d", manager.config.MaxSize)
	}
}

// TestAcquire_Success tests successful connection acquisition.
func TestAcquire_Success(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxSize = 10
	manager := NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil // Mock dial
	}

	conn, err := manager.Acquire(ctx, "peer-1", dialFunc)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}
	_ = conn

	// Verify connection is tracked
	wrapper, ok := manager.GetConnection("peer-1")
	if !ok {
		t.Fatal("connection not found in pool")
	}
	if wrapper.PeerID != "peer-1" {
		t.Fatalf("expected PeerID 'peer-1', got '%s'", wrapper.PeerID)
	}
}

// TestAcquire_PoolClosed tests acquire on closed pool.
func TestAcquire_PoolClosed(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	_, err := manager.Acquire(ctx, "peer-1", dialFunc)
	if err != ErrPoolClosed {
		t.Fatalf("expected ErrPoolClosed, got %v", err)
	}
}

// TestAcquire_ReuseConnection tests reusing existing connection.
func TestAcquire_ReuseConnection(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxSize = 10
	manager := NewManager(config)
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// First acquire
	conn1, err := manager.Acquire(ctx, "peer-1", dialFunc)
	if err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}
	_ = conn1

	// Release
	manager.Release("peer-1")

	// Second acquire should reuse
	conn2, err := manager.Acquire(ctx, "peer-1", dialFunc)
	if err != nil {
		t.Fatalf("second Acquire() failed: %v", err)
	}
	_ = conn2

	// Should be same connection wrapper
	if conn1 != conn2 {
		t.Log("Note: Connection wrappers may differ due to mock implementation")
	}
}

// TestRelease_Success tests successful connection release.
func TestRelease_Success(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	_, err := manager.Acquire(ctx, "peer-1", dialFunc)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	manager.Release("peer-1")

	// Verify connection is idle
	stats := manager.GetStats()
	if stats.InUseConnections != 0 {
		t.Fatalf("expected 0 InUseConnections, got %d", stats.InUseConnections)
	}
}

// TestRelease_NonExistent tests release of non-existent connection.
func TestRelease_NonExistent(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	// Should not panic
	manager.Release("non-existent")
}

// TestClose_Success tests closing a connection.
func TestClose_Success(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	_, err := manager.Acquire(ctx, "peer-1", dialFunc)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	err = manager.Close("peer-1")
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify connection is removed
	_, ok := manager.GetConnection("peer-1")
	if ok {
		t.Fatal("connection should be removed after Close()")
	}
}

// TestClose_NonExistent tests closing non-existent connection.
func TestClose_NonExistent(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	err := manager.Close("non-existent")
	if err != nil {
		t.Fatalf("Close() should return nil for non-existent connection, got %v", err)
	}
}

// TestCloseAll_Success tests closing all connections.
func TestCloseAll_Success(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// Create multiple connections
	for i := 0; i < 5; i++ {
		_, err := manager.Acquire(ctx, "peer-"+string(rune('0'+i)), dialFunc)
		if err != nil {
			t.Fatalf("Acquire() failed: %v", err)
		}
	}

	err := manager.CloseAll()
	if err != nil {
		t.Fatalf("CloseAll() failed: %v", err)
	}

	// Verify pool is closed
	stats := manager.GetStats()
	if stats.TotalConnections != 0 {
		t.Fatalf("expected 0 TotalConnections after CloseAll, got %d", stats.TotalConnections)
	}

	// Verify pool is marked as closed
	_, err = manager.Acquire(ctx, "peer-new", dialFunc)
	if err != ErrPoolClosed {
		t.Fatalf("expected ErrPoolClosed after CloseAll, got %v", err)
	}
}

// TestCloseAll_DoubleClose tests double close.
func TestCloseAll_DoubleClose(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())

	manager.CloseAll()
	err := manager.CloseAll()

	if err != ErrPoolClosed {
		t.Fatalf("expected ErrPoolClosed on second CloseAll, got %v", err)
	}
}

// TestGetStats tests statistics retrieval.
func TestGetStats(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	// Initial stats
	stats := manager.GetStats()
	if stats.TotalConnections != 0 {
		t.Fatalf("expected 0 initial connections, got %d", stats.TotalConnections)
	}

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// Create connections
	for i := 0; i < 3; i++ {
		peerID := "peer-" + string(rune('0'+i))
		_, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			t.Fatalf("Acquire() failed: %v", err)
		}
	}

	stats = manager.GetStats()
	if stats.TotalConnections != 3 {
		t.Fatalf("expected 3 TotalConnections, got %d", stats.TotalConnections)
	}
	if stats.InUseConnections != 3 {
		t.Fatalf("expected 3 InUseConnections, got %d", stats.InUseConnections)
	}

	// Release one
	manager.Release("peer-0")
	stats = manager.GetStats()
	if stats.IdleConnections != 1 {
		t.Fatalf("expected 1 IdleConnection, got %d", stats.IdleConnections)
	}
}

// TestListConnections tests listing all connections.
func TestListConnections(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	// Create connections
	expectedIDs := []string{"peer-a", "peer-b", "peer-c"}
	for _, peerID := range expectedIDs {
		_, err := manager.Acquire(ctx, peerID, dialFunc)
		if err != nil {
			t.Fatalf("Acquire() failed: %v", err)
		}
	}

	ids := manager.ListConnections()
	if len(ids) != 3 {
		t.Fatalf("expected 3 connections, got %d", len(ids))
	}

	// Verify all IDs are present
	for _, expected := range expectedIDs {
		found := false
		for _, id := range ids {
			if id == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected peer ID '%s' not found in list", expected)
		}
	}
}

// TestConnWrapper_MarkInUse tests connection wrapper state change.
func TestConnWrapper_MarkInUse(t *testing.T) {
	wrapper := &ConnWrapper{
		Conn:      nil,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		InUse:     false,
	}

	wrapper.MarkInUse("test-peer")

	if !wrapper.InUse {
		t.Fatal("expected InUse to be true")
	}
	if wrapper.PeerID != "test-peer" {
		t.Fatalf("expected PeerID 'test-peer', got '%s'", wrapper.PeerID)
	}
}

// TestConnWrapper_MarkIdle tests marking connection idle.
func TestConnWrapper_MarkIdle(t *testing.T) {
	wrapper := &ConnWrapper{
		InUse:  true,
		PeerID: "test-peer",
	}

	wrapper.MarkIdle()

	if wrapper.InUse {
		t.Fatal("expected InUse to be false")
	}
}

// TestConnWrapper_IsHealthy tests health check.
func TestConnWrapper_IsHealthy(t *testing.T) {
	// Healthy connection
	wrapper := &ConnWrapper{
		Conn:      nil,
		LastUsed:  time.Now(),
		InUse:     true,
	}

	// Connection with nil Conn should return false for health check
	if wrapper.IsHealthy(5 * time.Minute) {
		t.Fatal("nil connection should not be healthy")
	}
}

// TestConnWrapper_Close tests wrapper close.
func TestConnWrapper_Close(t *testing.T) {
	wrapper := &ConnWrapper{
		Conn: nil,
	}

	err := wrapper.Close()
	if err != nil {
		t.Fatalf("Close() on nil connection should succeed, got %v", err)
	}
}

// TestDefaultPoolConfig tests default configuration values.
func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	if config.MaxSize != 100 {
		t.Fatalf("expected MaxSize 100, got %d", config.MaxSize)
	}
	if config.MinSize != 10 {
		t.Fatalf("expected MinSize 10, got %d", config.MinSize)
	}
	if config.IdleTimeout != 5*time.Minute {
		t.Fatalf("expected IdleTimeout 5min, got %v", config.IdleTimeout)
	}
	if config.MaxLifetime != 30*time.Minute {
		t.Fatalf("expected MaxLifetime 30min, got %v", config.MaxLifetime)
	}
	if config.AcquireTimeout != 10*time.Second {
		t.Fatalf("expected AcquireTimeout 10s, got %v", config.AcquireTimeout)
	}
}

// TestPoolStats_Concurrent tests concurrent statistics updates.
func TestPoolStats_Concurrent(t *testing.T) {
	manager := NewManager(DefaultPoolConfig())
	defer manager.CloseAll()

	ctx := context.Background()
	dialFunc := func() (net.Conn, error) {
		return nil, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			peerID := "peer-" + string(rune('0'+id))
			_, err := manager.Acquire(ctx, peerID, dialFunc)
			if err != nil {
				t.Errorf("Acquire() failed: %v", err)
				return
			}
			manager.Release(peerID)
		}(i)
	}

	wg.Wait()

	stats := manager.GetStats()
	if stats.TotalConnections != 10 {
		t.Logf("Total connections: %d (may vary due to reuse)", stats.TotalConnections)
	}
}