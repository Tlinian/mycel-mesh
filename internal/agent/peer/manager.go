// Package peer provides peer management for the agent.
package peer

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/mycel/mesh/api/pb"
	"github.com/mycel/mesh/internal/cli/client"
)

// Manager manages peer synchronization with the coordinator.
type Manager struct {
	client     *client.Client
	nodeID     string
	peers      map[string]*Peer // nodeID -> Peer
	mu         sync.RWMutex
	lastSync   time.Time
	onPeerAdd  func(peer *Peer)
	onPeerRemove func(nodeID string)
}

// Peer represents a network peer.
type Peer struct {
	NodeID    string
	Name      string
	IP        string
	PublicKey string
	NATInfo   *NATInfo
	Status    string
	LastSeen  time.Time
}

// NATInfo contains NAT detection results.
type NATInfo struct {
	NATType    string
	PublicIP   string
	PublicPort int
	CanPunch   bool
	LocalIPs   []string
}

// NewManager creates a new peer manager.
func NewManager(grpcClient *client.Client, nodeID string) *Manager {
	return &Manager{
		client: grpcClient,
		nodeID: nodeID,
		peers:  make(map[string]*Peer),
	}
}

// SetOnPeerAdd sets the callback for when a peer is added.
func (m *Manager) SetOnPeerAdd(callback func(peer *Peer)) {
	m.onPeerAdd = callback
}

// SetOnPeerRemove sets the callback for when a peer is removed.
func (m *Manager) SetOnPeerRemove(callback func(nodeID string)) {
	m.onPeerRemove = callback
}

// SyncPeers synchronizes peers with the coordinator.
// Returns new peers, removed peers, and error.
func (m *Manager) SyncPeers(ctx context.Context, natInfo *pb.NATInfo) ([]*Peer, []string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Send heartbeat to coordinator
	resp, err := m.client.Heartbeat(ctx, m.nodeID, natInfo)
	if err != nil {
		return nil, nil, err
	}

	if resp.Error != "" {
		return nil, nil, fmt.Errorf("heartbeat error: %s", resp.Error)
	}

	var newPeers []*Peer
	var removedPeers []string

	// Process peer updates
	for _, peerInfo := range resp.PeerUpdates {
		if peerInfo.NodeId == m.nodeID {
			continue // Skip self
		}

		existing, exists := m.peers[peerInfo.NodeId]

		peer := &Peer{
			NodeID:    peerInfo.NodeId,
			Name:      peerInfo.Name,
			IP:        peerInfo.Ip,
			PublicKey: peerInfo.PublicKey,
			Status:    peerInfo.Status,
			LastSeen:  time.Now(),
		}

		if peerInfo.NatInfo != nil {
			peer.NATInfo = &NATInfo{
				NATType:    peerInfo.NatInfo.NatType,
				PublicIP:   peerInfo.NatInfo.PublicIp,
				PublicPort: int(peerInfo.NatInfo.PublicPort),
				CanPunch:   peerInfo.NatInfo.CanPunch,
				LocalIPs:   peerInfo.NatInfo.LocalIps,
			}
		}

		m.peers[peerInfo.NodeId] = peer

		if !exists || existing.PublicKey != peer.PublicKey {
			newPeers = append(newPeers, peer)
			if m.onPeerAdd != nil {
				m.onPeerAdd(peer)
			}
		}
	}

	// Process offline nodes
	for _, nodeID := range resp.OfflineNodes {
		if _, exists := m.peers[nodeID]; exists {
			delete(m.peers, nodeID)
			removedPeers = append(removedPeers, nodeID)
			if m.onPeerRemove != nil {
				m.onPeerRemove(nodeID)
			}
		}
	}

	m.lastSync = time.Now()
	return newPeers, removedPeers, nil
}

// GetPeers returns all known peers.
func (m *Manager) GetPeers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		peers = append(peers, p)
	}
	return peers
}

// GetPeer returns a specific peer by node ID.
func (m *Manager) GetPeer(nodeID string) *Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.peers[nodeID]
}

// GetOnlinePeers returns all online peers.
func (m *Manager) GetOnlinePeers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]*Peer, 0)
	for _, p := range m.peers {
		if p.Status == "online" {
			peers = append(peers, p)
		}
	}
	return peers
}

// GetPeerEndpoint returns the best endpoint for a peer.
// Uses public IP if available, otherwise tries local IPs.
func (p *Peer) GetEndpoint() *net.UDPAddr {
	if p.NATInfo == nil {
		return nil
	}

	// Prefer public endpoint
	if p.NATInfo.PublicIP != "" && p.NATInfo.PublicPort > 0 {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", p.NATInfo.PublicIP, p.NATInfo.PublicPort))
		if err == nil {
			return addr
		}
	}

	// Try local IPs as fallback
	for _, localIP := range p.NATInfo.LocalIPs {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:51820", localIP))
		if err == nil {
			return addr
		}
	}

	return nil
}

// StartSyncLoop starts a background sync loop.
func (m *Manager) StartSyncLoop(ctx context.Context, interval time.Duration, natInfoProvider func() *pb.NATInfo) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				natInfo := natInfoProvider()
				newPeers, removedPeers, err := m.SyncPeers(ctx, natInfo)
				if err != nil {
					log.Printf("Peer sync failed: %v", err)
					continue
				}

				if len(newPeers) > 0 {
					log.Printf("Added %d new peers", len(newPeers))
				}
				if len(removedPeers) > 0 {
					log.Printf("Removed %d peers", len(removedPeers))
				}
			}
		}
	}()
}

// LastSync returns the last sync time.
func (m *Manager) LastSync() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSync
}

// PeerCount returns the number of known peers.
func (m *Manager) PeerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peers)
}