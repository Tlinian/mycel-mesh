package nat

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mycel/mesh/internal/pkg/stun"
)

// ConnectionState represents the state of a P2P connection.
type ConnectionState int

const (
	ConnectionStateUnknown ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnected
	ConnectionStateDisconnected
	ConnectionStateFailed
)

func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateUnknown:
		return "unknown"
	case ConnectionStateConnecting:
		return "connecting"
	case ConnectionStateConnected:
		return "connected"
	case ConnectionStateDisconnected:
		return "disconnected"
	case ConnectionStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// P2PConnection represents a P2P connection with a remote peer.
type P2PConnection struct {
	PeerID        string
	State         ConnectionState
	PublicAddr    *net.UDPAddr
	PrivateAddr   *net.UDPAddr
	LocalAddr     *net.UDPAddr
	Conn          *net.UDPConn
	CreatedAt     time.Time
	LastActiveAt  time.Time
	BytesReceived uint64
	BytesSent     uint64
	Latency       time.Duration
	mu            sync.RWMutex
}

// UpdateState updates the connection state.
func (c *P2PConnection) UpdateState(state ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = state
	if state == ConnectionStateConnected {
		c.LastActiveAt = time.Now()
	}
}

// IsActive returns true if the connection is active.
func (c *P2PConnection) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == ConnectionStateConnected
}

// ConnectionManager manages multiple P2P connections.
type ConnectionManager struct {
	puncher     *HolePuncher
	connections map[string]*P2PConnection
	mu          sync.RWMutex
	eventChan   chan ConnectionEvent
	ctx         context.Context
	cancel      context.CancelFunc
}

// ConnectionEvent represents a connection event.
type ConnectionEvent struct {
	Type       string
	PeerID     string
	Connection *P2PConnection
	Error      error
}

const (
	EventConnected    = "connected"
	EventDisconnected = "disconnected"
	EventFailed       = "failed"
)

// NewConnectionManager creates a new connection manager.
func NewConnectionManager(puncher *HolePuncher) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConnectionManager{
		puncher:     puncher,
		connections: make(map[string]*P2PConnection),
		eventChan:   make(chan ConnectionEvent, 100),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the connection manager.
func (m *ConnectionManager) Start() error {
	if m.puncher == nil {
		return fmt.Errorf("puncher is nil")
	}
	return m.puncher.Start()
}

// Stop stops the connection manager and closes all connections.
func (m *ConnectionManager) Stop() error {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connections {
		if conn.Conn != nil {
			conn.Conn.Close()
		}
	}
	m.connections = make(map[string]*P2PConnection)

	if m.puncher != nil {
		return m.puncher.Stop()
	}
	return nil
}

// Connect attempts to establish a P2P connection with a peer.
func (m *ConnectionManager) Connect(ctx context.Context, peerID string, publicAddr, privateAddr *net.UDPAddr) (*P2PConnection, error) {
	m.mu.Lock()
	if existing, ok := m.connections[peerID]; ok {
		if existing.IsActive() {
			m.mu.Unlock()
			return existing, nil
		}
	}

	conn := &P2PConnection{
		PeerID:      peerID,
		State:       ConnectionStateConnecting,
		PublicAddr:  publicAddr,
		PrivateAddr: privateAddr,
		CreatedAt:   time.Now(),
	}
	m.connections[peerID] = conn
	m.mu.Unlock()

	// Perform hole punching
	result, err := m.puncher.Punch(ctx, peerID, publicAddr, privateAddr)
	if err != nil {
		conn.UpdateState(ConnectionStateFailed)
		m.emitEvent(ConnectionEvent{
			Type:       EventFailed,
			PeerID:     peerID,
			Connection: conn,
			Error:      err,
		})
		return conn, err
	}

	if result.Success {
		conn.UpdateState(ConnectionStateConnected)
		conn.LocalAddr = result.LocalAddr
		conn.Latency = result.Latency
		conn.LastActiveAt = time.Now()

		m.emitEvent(ConnectionEvent{
			Type:       EventConnected,
			PeerID:     peerID,
			Connection: conn,
		})

		return conn, nil
	}

	conn.UpdateState(ConnectionStateFailed)
	m.emitEvent(ConnectionEvent{
		Type:       EventFailed,
		PeerID:     peerID,
		Connection: conn,
	})
	return conn, fmt.Errorf("connection failed")
}

// Disconnect closes a P2P connection.
func (m *ConnectionManager) Disconnect(peerID string) error {
	m.mu.Lock()
	conn, ok := m.connections[peerID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("connection not found: %s", peerID)
	}
	delete(m.connections, peerID)
	m.mu.Unlock()

	if conn.Conn != nil {
		conn.Conn.Close()
	}
	conn.UpdateState(ConnectionStateDisconnected)

	m.emitEvent(ConnectionEvent{
		Type:       EventDisconnected,
		PeerID:     peerID,
		Connection: conn,
	})

	return nil
}

// GetConnection returns a P2P connection by peer ID.
func (m *ConnectionManager) GetConnection(peerID string) (*P2PConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[peerID]
	return conn, ok
}

// ListConnections returns all P2P connections.
func (m *ConnectionManager) ListConnections() []*P2PConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connections := make([]*P2PConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetConnectionStats returns statistics about connections.
func (m *ConnectionManager) GetConnectionStats() ConnectionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := ConnectionStats{}
	for _, conn := range m.connections {
		switch conn.State {
		case ConnectionStateConnected:
			stats.Connected++
		case ConnectionStateConnecting:
			stats.Connecting++
		case ConnectionStateFailed:
			stats.Failed++
		case ConnectionStateDisconnected:
			stats.Disconnected++
		}
	}
	stats.Total = len(m.connections)
	return stats
}

// ConnectionStats holds statistics about connections.
type ConnectionStats struct {
	Total       int
	Connected   int
	Connecting  int
	Failed      int
	Disconnected int
}

// Events returns the event channel for receiving connection events.
func (m *ConnectionManager) Events() <-chan ConnectionEvent {
	return m.eventChan
}

func (m *ConnectionManager) emitEvent(event ConnectionEvent) {
	select {
	case m.eventChan <- event:
	default:
		// Channel full, drop event
	}
}

// GetPublicIP is a convenience function to get the local public IP.
func (m *ConnectionManager) GetPublicIP() (string, error) {
	if m.puncher == nil {
		return "", fmt.Errorf("puncher is nil")
	}
	info := m.puncher.GetNATInfo()
	if info == nil || info.PublicAddr == nil {
		return "", fmt.Errorf("NAT info not available")
	}
	return info.PublicAddr.IP.String(), nil
}

// GetNATType returns the local NAT type.
func (m *ConnectionManager) GetNATType() stun.NATType {
	if m.puncher == nil {
		return stun.NATUnknown
	}
	info := m.puncher.GetNATInfo()
	if info == nil {
		return stun.NATUnknown
	}
	return info.Type
}

// SimpleConnect is a convenience function for one-shot P2P connection.
func SimpleConnect(ctx context.Context, peerID string, publicAddr, privateAddr *net.UDPAddr) (*P2PConnection, error) {
	puncher := NewHolePuncher(DefaultPunchConfig())
	manager := NewConnectionManager(puncher)

	if err := manager.Start(); err != nil {
		return nil, err
	}

	conn, err := manager.Connect(ctx, peerID, publicAddr, privateAddr)
	if err != nil {
		manager.Stop()
		return nil, err
	}

	// Keep manager running for the connection
	go func() {
		<-ctx.Done()
		manager.Stop()
	}()

	return conn, nil
}
