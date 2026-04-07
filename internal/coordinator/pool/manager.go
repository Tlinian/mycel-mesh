// Package pool provides connection pool management for Coordinator.
package pool

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrPoolClosed   = errors.New("connection pool is closed")
	ErrPoolExhausted = errors.New("connection pool exhausted")
	ErrInvalidAddr  = errors.New("invalid address")
)

// ConnWrapper wraps a net.Conn with metadata.
type ConnWrapper struct {
	Conn      net.Conn
	CreatedAt time.Time
	LastUsed  time.Time
	InUse     bool
	PeerID    string
	mu        sync.Mutex
}

// MarkInUse marks the connection as in use.
func (c *ConnWrapper) MarkInUse(peerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.InUse = true
	c.LastUsed = time.Now()
	c.PeerID = peerID
}

// MarkIdle marks the connection as idle.
func (c *ConnWrapper) MarkIdle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.InUse = false
	c.LastUsed = time.Now()
}

// IsHealthy checks if the connection is still healthy.
func (c *ConnWrapper) IsHealthy(idleTimeout time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn == nil {
		return false
	}

	// Check if connection has been idle too long
	if time.Since(c.LastUsed) > idleTimeout {
		return false
	}

	// Check if underlying connection is still valid
	return true
}

// Close closes the wrapped connection.
func (c *ConnWrapper) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// PoolConfig holds configuration for the connection pool.
type PoolConfig struct {
	// MaxSize is the maximum number of connections in the pool.
	MaxSize int
	// MinSize is the minimum number of connections to maintain.
	MinSize int
	// IdleTimeout is the maximum idle time before a connection is closed.
	IdleTimeout time.Duration
	// MaxLifetime is the maximum lifetime of a connection.
	MaxLifetime time.Duration
	// AcquireTimeout is the timeout for acquiring a connection from the pool.
	AcquireTimeout time.Duration
}

// DefaultPoolConfig returns a default pool configuration.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxSize:        100, // Support 100 concurrent nodes
		MinSize:        10,
		IdleTimeout:    5 * time.Minute,
		MaxLifetime:    30 * time.Minute,
		AcquireTimeout: 10 * time.Second,
	}
}

// Manager manages a pool of connections for Coordinator.
type Manager struct {
	config      PoolConfig
	connections map[string]*ConnWrapper // peerID -> connection
	idleChan    chan *ConnWrapper
	mu          sync.RWMutex
	closed      bool
	ctx         context.Context
	cancel      context.CancelFunc

	// Statistics
	statsMu     sync.Mutex
	stats       PoolStats
}

// PoolStats holds statistics about the connection pool.
type PoolStats struct {
	TotalConnections   int
	IdleConnections    int
	InUseConnections   int
	AcquireCount       int64
	AcquireWaitCount   int64
	AcquireErrorCount  int64
}

// NewManager creates a new connection pool manager.
func NewManager(config PoolConfig) *Manager {
	if config.MaxSize <= 0 {
		config.MaxSize = DefaultPoolConfig().MaxSize
	}
	if config.MinSize <= 0 {
		config.MinSize = DefaultPoolConfig().MinSize
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = DefaultPoolConfig().IdleTimeout
	}
	if config.MaxLifetime <= 0 {
		config.MaxLifetime = DefaultPoolConfig().MaxLifetime
	}
	if config.AcquireTimeout <= 0 {
		config.AcquireTimeout = DefaultPoolConfig().AcquireTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config:      config,
		connections: make(map[string]*ConnWrapper),
		idleChan:    make(chan *ConnWrapper, config.MaxSize),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start background cleanup goroutine
	go manager.cleanupLoop()

	return manager
}

// Acquire acquires a connection for the given peerID.
// If no connection is available and pool is not at max capacity, creates a new one.
func (m *Manager) Acquire(ctx context.Context, peerID string, dialFunc func() (net.Conn, error)) (*ConnWrapper, error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	m.mu.RUnlock()

	// Try to get an existing idle connection
	m.mu.Lock()
	if conn, ok := m.connections[peerID]; ok && conn.IsHealthy(m.config.IdleTimeout) {
		conn.MarkInUse(peerID)
		m.statsMu.Lock()
		m.stats.AcquireCount++
		m.stats.InUseConnections++
		m.stats.IdleConnections--
		m.statsMu.Unlock()
		m.mu.Unlock()
		return conn, nil
	}
	m.mu.Unlock()

	// Check if we can create a new connection
	m.mu.RLock()
	currentSize := len(m.connections)
	m.mu.RUnlock()

	if currentSize >= m.config.MaxSize {
		// Pool exhausted, wait for an available connection
		m.statsMu.Lock()
		m.stats.AcquireWaitCount++
		m.statsMu.Unlock()

		select {
		case conn := <-m.idleChan:
			if conn != nil && conn.IsHealthy(m.config.IdleTimeout) {
				conn.MarkInUse(peerID)
				m.statsMu.Lock()
				m.stats.AcquireCount++
				m.stats.InUseConnections++
				m.stats.IdleConnections--
				m.statsMu.Unlock()
				return conn, nil
			}
		case <-time.After(m.config.AcquireTimeout):
			m.statsMu.Lock()
			m.stats.AcquireErrorCount++
			m.statsMu.Unlock()
			return nil, ErrPoolExhausted
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-m.ctx.Done():
			return nil, ErrPoolClosed
		}
	}

	// Create a new connection
	newConn, err := dialFunc()
	if err != nil {
		m.statsMu.Lock()
		m.stats.AcquireErrorCount++
		m.statsMu.Unlock()
		return nil, err
	}

	wrapper := &ConnWrapper{
		Conn:      newConn,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		InUse:     true,
		PeerID:    peerID,
	}

	m.mu.Lock()
	m.connections[peerID] = wrapper
	m.mu.Unlock()

	m.statsMu.Lock()
	m.stats.AcquireCount++
	m.stats.TotalConnections++
	m.stats.InUseConnections++
	m.statsMu.Unlock()

	return wrapper, nil
}

// Release releases a connection back to the pool.
func (m *Manager) Release(peerID string) {
	m.mu.RLock()
	conn, ok := m.connections[peerID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	conn.MarkIdle()

	m.statsMu.Lock()
	m.stats.InUseConnections--
	m.stats.IdleConnections++
	m.statsMu.Unlock()
}

// Close closes a connection and removes it from the pool.
func (m *Manager) Close(peerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[peerID]
	if !ok {
		return nil
	}

	delete(m.connections, peerID)
	m.statsMu.Lock()
	m.stats.TotalConnections--
	if conn.InUse {
		m.stats.InUseConnections--
	} else {
		m.stats.IdleConnections--
	}
	m.statsMu.Unlock()

	return conn.Close()
}

// CloseAll closes all connections in the pool.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrPoolClosed
	}
	m.closed = true
	m.cancel()

	var lastErr error
	for peerID, conn := range m.connections {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
		delete(m.connections, peerID)
	}

	close(m.idleChan)

	m.statsMu.Lock()
	m.stats.TotalConnections = 0
	m.stats.IdleConnections = 0
	m.stats.InUseConnections = 0
	m.statsMu.Unlock()

	return lastErr
}

// GetStats returns current pool statistics.
func (m *Manager) GetStats() PoolStats {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	// Update connection counts
	m.mu.RLock()
	m.stats.TotalConnections = len(m.connections)
	inUse := 0
	for _, conn := range m.connections {
		conn.mu.Lock()
		if conn.InUse {
			inUse++
		}
		conn.mu.Unlock()
	}
	m.stats.InUseConnections = inUse
	m.stats.IdleConnections = m.stats.TotalConnections - inUse
	m.mu.RUnlock()

	return m.stats
}

// GetConnection returns the connection for a peerID without marking it in use.
func (m *Manager) GetConnection(peerID string) (*ConnWrapper, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[peerID]
	return conn, ok
}

// ListConnections returns all peer IDs with active connections.
func (m *Manager) ListConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.connections))
	for peerID := range m.connections {
		ids = append(ids, peerID)
	}
	return ids
}

// connectionCount returns the current number of connections.
func (m *Manager) connectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// cleanupLoop periodically cleans up stale connections.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup removes connections that have exceeded their lifetime or idle timeout.
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for peerID, conn := range m.connections {
		shouldRemove := false

		// Check max lifetime
		if m.config.MaxLifetime > 0 && now.Sub(conn.CreatedAt) > m.config.MaxLifetime {
			shouldRemove = true
		}

		// Check idle timeout (only for idle connections)
		if !conn.InUse && m.config.IdleTimeout > 0 && now.Sub(conn.LastUsed) > m.config.IdleTimeout {
			shouldRemove = true
		}

		// Check if underlying connection is still valid
		if !conn.IsHealthy(m.config.IdleTimeout) {
			shouldRemove = true
		}

		if shouldRemove {
			conn.Close()
			delete(m.connections, peerID)

			m.statsMu.Lock()
			m.stats.TotalConnections--
			if conn.InUse {
				m.stats.InUseConnections--
			} else {
				m.stats.IdleConnections--
			}
			m.statsMu.Unlock()
		}
	}
}
