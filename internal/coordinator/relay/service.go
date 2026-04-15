// Package relay provides relay service for nodes that cannot establish direct P2P connections.
package relay

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Service provides relay functionality for symmetric NAT nodes.
type Service struct {
	port       int
	conn       *net.UDPConn
	connections map[string]*RelayConnection // connectionID -> connection
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// RelayConnection represents an active relay connection between two nodes.
type RelayConnection struct {
	ID           string
	NodeA        *RelayNode
	NodeB        *RelayNode
	CreatedAt    time.Time
	LastActivity time.Time
	BytesSent    uint64
	BytesRecv    uint64
}

// RelayNode represents one endpoint of a relay connection.
type RelayNode struct {
	NodeID    string
	PublicKey string
	Endpoint  *net.UDPAddr
}

// Config holds relay service configuration.
type Config struct {
	Port int // UDP port to listen on (default: 51821)
}

// DefaultConfig returns default relay configuration.
func DefaultConfig() Config {
	return Config{
		Port: 51821,
	}
}

// NewService creates a new relay service.
func NewService(config Config) *Service {
	if config.Port == 0 {
		config.Port = 51821
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		port:       config.Port,
		connections: make(map[string]*RelayConnection),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the relay service.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := &net.UDPAddr{IP: net.IPv4zero, Port: s.port}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}

	s.conn = conn
	log.Printf("Relay service listening on :%d", s.port)

	// Start packet relay loop
	go s.relayLoop()

	return nil
}

// Stop stops the relay service.
func (s *Service) Stop() error {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	s.connections = make(map[string]*RelayConnection)
	return nil
}

// StartRelay creates a relay connection between two nodes.
func (s *Service) StartRelay(nodeA, nodeB *RelayNode) (*RelayConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate connection ID
	connID := fmt.Sprintf("%s-%s", nodeA.NodeID, nodeB.NodeID)

	// Check if connection already exists
	if _, exists := s.connections[connID]; exists {
		return s.connections[connID], nil
	}

	conn := &RelayConnection{
		ID:        connID,
		NodeA:     nodeA,
		NodeB:     nodeB,
		CreatedAt: time.Now(),
	}

	s.connections[connID] = conn
	log.Printf("Relay connection started: %s <-> %s", nodeA.NodeID, nodeB.NodeID)

	return conn, nil
}

// StopRelay stops a relay connection.
func (s *Service) StopRelay(connID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.connections[connID]; !exists {
		return fmt.Errorf("connection not found")
	}

	delete(s.connections, connID)
	log.Printf("Relay connection stopped: %s", connID)

	return nil
}

// relayLoop handles packet relay between connected nodes.
func (s *Service) relayLoop() {
	buf := make([]byte, 65535)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if s.ctx.Err() != nil {
				return // Context cancelled, normal shutdown
			}
			log.Printf("Relay read error: %v", err)
			continue
		}

		// Find connection for this endpoint
		s.mu.RLock()
		var targetConn *RelayConnection
		var targetEndpoint *net.UDPAddr

		for _, conn := range s.connections {
			if conn.NodeA.Endpoint != nil && conn.NodeA.Endpoint.String() == addr.String() {
				targetConn = conn
				targetEndpoint = conn.NodeB.Endpoint
				conn.BytesRecv += uint64(n)
				break
			}
			if conn.NodeB.Endpoint != nil && conn.NodeB.Endpoint.String() == addr.String() {
				targetConn = conn
				targetEndpoint = conn.NodeA.Endpoint
				conn.BytesRecv += uint64(n)
				break
			}
		}
		s.mu.RUnlock()

		if targetConn == nil || targetEndpoint == nil {
			// Unknown endpoint, ignore packet
			continue
		}

		// Relay packet to target
		if _, err := s.conn.WriteToUDP(buf[:n], targetEndpoint); err != nil {
			log.Printf("Relay write error: %v", err)
			continue
		}

		targetConn.LastActivity = time.Now()
		targetConn.BytesSent += uint64(n)
	}
}

// GetConnection returns a relay connection by ID.
func (s *Service) GetConnection(connID string) *RelayConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections[connID]
}

// ListConnections returns all active relay connections.
func (s *Service) ListConnections() []*RelayConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conns := make([]*RelayConnection, 0, len(s.connections))
	for _, conn := range s.connections {
		conns = append(conns, conn)
	}
	return conns
}

// GetStats returns relay service statistics.
func (s *Service) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalBytesSent, totalBytesRecv uint64
	for _, conn := range s.connections {
		totalBytesSent += conn.BytesSent
		totalBytesRecv += conn.BytesRecv
	}

	return Stats{
		ActiveConnections: len(s.connections),
		TotalBytesSent:    totalBytesSent,
		TotalBytesRecv:    totalBytesRecv,
	}
}

// Stats holds relay service statistics.
type Stats struct {
	ActiveConnections int
	TotalBytesSent    uint64
	TotalBytesRecv    uint64
}

// UpdateNodeEndpoint updates a node's endpoint in an existing connection.
func (s *Service) UpdateNodeEndpoint(connID string, nodeID string, newEndpoint *net.UDPAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, exists := s.connections[connID]
	if !exists {
		return fmt.Errorf("connection not found")
	}

	if conn.NodeA.NodeID == nodeID {
		conn.NodeA.Endpoint = newEndpoint
	} else if conn.NodeB.NodeID == nodeID {
		conn.NodeB.Endpoint = newEndpoint
	} else {
		return fmt.Errorf("node not in connection")
	}

	return nil
}

// CleanupStaleConnections removes connections with no activity for a duration.
func (s *Service) CleanupStaleConnections(timeout time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, conn := range s.connections {
		if time.Since(conn.LastActivity) > timeout {
			delete(s.connections, id)
			count++
			log.Printf("Cleaned up stale relay connection: %s", id)
		}
	}

	return count
}

// StartCleanupLoop starts periodic cleanup of stale connections.
func (s *Service) StartCleanupLoop(interval, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.CleanupStaleConnections(timeout)
			}
		}
	}()
}