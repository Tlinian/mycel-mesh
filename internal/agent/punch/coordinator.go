// Package punch provides NAT hole punching coordination for the agent.
package punch

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/mycel/mesh/internal/agent/peer"
	"github.com/mycel/mesh/internal/agent/tun"
	"github.com/mycel/mesh/internal/coordinator/nat"
	"github.com/mycel/mesh/internal/pkg/stun"
)

// Coordinator manages hole punching attempts with peers.
type Coordinator struct {
	puncher    *nat.HolePuncher
	tunManager *tun.Manager
	localNAT   *stun.NATInfo
	mu         sync.Mutex
	results    map[string]*nat.PunchResult // peerID -> result
	onSuccess  func(peerID string, endpoint *net.UDPAddr)
}

// NewCoordinator creates a new punch coordinator.
func NewCoordinator(tunManager *tun.Manager) *Coordinator {
	return &Coordinator{
		results: make(map[string]*nat.PunchResult),
	}
}

// Start initializes the punch coordinator.
func (c *Coordinator) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create hole puncher with default config
	config := nat.DefaultPunchConfig()
	config.ListenPort = 51820 // Use WireGuard port

	c.puncher = nat.NewHolePuncher(config)
	if err := c.puncher.Start(); err != nil {
		return err
	}

	c.localNAT = c.puncher.GetNATInfo()
	log.Printf("Punch coordinator started")
	log.Printf("Local NAT type: %s", c.localNAT.Type)
	if c.localNAT.PublicAddr != nil {
		log.Printf("Public address: %s", c.localNAT.PublicAddr.String())
	}

	return nil
}

// Stop stops the punch coordinator.
func (c *Coordinator) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.puncher != nil {
		return c.puncher.Stop()
	}
	return nil
}

// SetOnSuccess sets the callback for successful punch.
func (c *Coordinator) SetOnSuccess(callback func(peerID string, endpoint *net.UDPAddr)) {
	c.onSuccess = callback
}

// TryPunchPeer attempts to punch a hole with a specific peer.
func (c *Coordinator) TryPunchPeer(ctx context.Context, p *peer.Peer) (*nat.PunchResult, error) {
	c.mu.Lock()
	if c.puncher == nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("punch coordinator not started")
	}
	c.mu.Unlock()

	// Check if punch is possible
	remoteNAT := stun.NATType(p.NATInfo.NATType)
	if !nat.CanPunch(c.localNAT.Type, remoteNAT) {
		log.Printf("Cannot punch with %s: NAT types incompatible (local=%s, remote=%s)",
			p.Name, c.localNAT.Type, remoteNAT)
		return nil, nil
	}

	// Get peer addresses
	var publicAddr *net.UDPAddr
	var privateAddr *net.UDPAddr

	if p.NATInfo.PublicIP != "" && p.NATInfo.PublicPort > 0 {
		publicAddr, _ = net.ResolveUDPAddr("udp",
			fmt.Sprintf("%s:%d", p.NATInfo.PublicIP, p.NATInfo.PublicPort))
	}

	// Try local IPs as private addresses
	for _, localIP := range p.NATInfo.LocalIPs {
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(localIP, "51820"))
		if err == nil {
			privateAddr = addr
			break
		}
	}

	if publicAddr == nil && privateAddr == nil {
		log.Printf("No valid address for peer %s", p.Name)
		return nil, nil
	}

	log.Printf("Attempting punch with %s (public=%s, private=%s)",
		p.Name, publicAddr, privateAddr)

	// Perform punch
	result, err := c.puncher.Punch(ctx, p.NodeID, publicAddr, privateAddr)
	if err != nil {
		log.Printf("Punch failed with %s: %v", p.Name, err)
		return nil, err
	}

	if result.Success {
		log.Printf("Punch succeeded with %s: endpoint=%s, latency=%dms",
			p.Name, result.RemoteAddr.String(), result.Latency.Milliseconds())

		c.mu.Lock()
		c.results[p.NodeID] = result
		c.mu.Unlock()

		// Update WireGuard endpoint
		if c.onSuccess != nil {
			c.onSuccess(p.NodeID, result.RemoteAddr)
		}
	}

	return result, nil
}

// TryPunchPeers attempts to punch with multiple peers concurrently.
func (c *Coordinator) TryPunchPeers(ctx context.Context, peers []*peer.Peer) map[string]*nat.PunchResult {
	results := make(map[string]*nat.PunchResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range peers {
		wg.Add(1)
		go func(peer *peer.Peer) {
			defer wg.Done()

			// Create sub-context with timeout
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.TryPunchPeer(pctx, peer)
			if err == nil && result != nil && result.Success {
				mu.Lock()
				results[peer.NodeID] = result
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()
	return results
}

// GetLocalNATInfo returns the local NAT information.
func (c *Coordinator) GetLocalNATInfo() *stun.NATInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.localNAT
}

// GetPunchResult returns the punch result for a peer.
func (c *Coordinator) GetPunchResult(peerID string) *nat.PunchResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.results[peerID]
}

// StartPunchLoop starts a background loop to punch with new peers.
func (c *Coordinator) StartPunchLoop(ctx context.Context, peerManager interface{}, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get online peers
				// pm := peerManager.(*peer.Manager)
				// peers := pm.GetOnlinePeers()
				// c.TryPunchPeers(ctx, peers)
			}
		}
	}()
}

// IsP2PCapable checks if the local node can do P2P.
func (c *Coordinator) IsP2PCapable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.localNAT == nil {
		return false
	}
	return c.localNAT.CanP2P
}

// GetPublicEndpoint returns the public endpoint for this node.
func (c *Coordinator) GetPublicEndpoint() *net.UDPAddr {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.localNAT == nil || c.localNAT.PublicAddr == nil {
		return nil
	}

	return c.localNAT.PublicAddr
}