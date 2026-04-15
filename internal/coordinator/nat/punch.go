// Package nat provides NAT traversal functionality including UDP hole punching.
package nat

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mycel/mesh/internal/pkg/stun"
)

// PunchConfig holds configuration for hole punching.
type PunchConfig struct {
	// STUNServers is the list of STUN servers to use.
	STUNServers []string
	// ListenPort is the local port to bind for UDP listening.
	ListenPort int
	// DialTimeout is the timeout for dialing connections.
	DialTimeout time.Duration
	// PunchInterval is the interval between punch attempts.
	PunchInterval time.Duration
	// MaxPunchAttempts is the maximum number of punch attempts.
	MaxPunchAttempts int
}

// DefaultPunchConfig returns a default punch configuration.
func DefaultPunchConfig() PunchConfig {
	return PunchConfig{
		STUNServers:      stun.DefaultSTUNServers,
		ListenPort:       0, // Use any available port
		DialTimeout:      2 * time.Second,
		PunchInterval:    100 * time.Millisecond,
		MaxPunchAttempts: 30,
	}
}

// PunchResult contains the result of a hole punching attempt.
type PunchResult struct {
	// Success indicates if the punch was successful.
	Success bool
	// RemoteAddr is the remote peer's address.
	RemoteAddr *net.UDPAddr
	// LocalAddr is the local address used.
	LocalAddr *net.UDPAddr
	// Attempts is the number of attempts made.
	Attempts int
	// Latency is the measured round-trip latency.
	Latency time.Duration
}

// HolePuncher performs UDP hole punching for P2P connections.
type HolePuncher struct {
	config      PunchConfig
	conn        *net.UDPConn
	localInfo   *stun.NATInfo
	mu          sync.Mutex
	punching    map[string]bool
	punchResult map[string]*PunchResult
}

// NewHolePuncher creates a new hole puncher with the given configuration.
func NewHolePuncher(config PunchConfig) *HolePuncher {
	if len(config.STUNServers) == 0 {
		config.STUNServers = stun.DefaultSTUNServers
	}
	return &HolePuncher{
		config:      config,
		punching:    make(map[string]bool),
		punchResult: make(map[string]*PunchResult),
	}
}

// Start initializes the hole puncher by binding a UDP port and detecting NAT type.
func (h *HolePuncher) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Bind UDP port
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: h.config.ListenPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}
	h.conn = conn

	// Detect NAT type
	stunClient := stun.NewClient(stun.ClientConfig{
		STUNServers: h.config.STUNServers,
	})
	detector := stun.NewNATDetector(stunClient)
	h.localInfo, err = detector.DetectNATType()
	if err != nil {
		return fmt.Errorf("detect NAT type: %w", err)
	}

	return nil
}

// Stop closes the UDP connection and cleans up resources.
func (h *HolePuncher) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conn != nil {
		if err := h.conn.Close(); err != nil {
			return err
		}
		h.conn = nil
	}
	return nil
}

// GetLocalAddr returns the local UDP address.
func (h *HolePuncher) GetLocalAddr() *net.UDPAddr {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conn == nil {
		return nil
	}
	return h.conn.LocalAddr().(*net.UDPAddr)
}

// GetNATInfo returns the detected NAT information.
func (h *HolePuncher) GetNATInfo() *stun.NATInfo {
	return h.localInfo
}

// Punch attempts to establish a P2P connection with the remote peer.
// The peerInfo contains the remote peer's public and private addresses.
func (h *HolePuncher) Punch(ctx context.Context, peerID string, peerPublicAddr, peerPrivateAddr *net.UDPAddr) (*PunchResult, error) {
	h.mu.Lock()
	if h.conn == nil {
		h.mu.Unlock()
		return nil, fmt.Errorf("hole puncher not started")
	}
	if h.punching[peerID] {
		h.mu.Unlock()
		return nil, fmt.Errorf("already punching for peer %s", peerID)
	}
	h.punching[peerID] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.punching, peerID)
		h.mu.Unlock()
	}()

	// Prepare punch packet
	punchData := h.createPunchPacket(peerID)

	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt < h.config.MaxPunchAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Send punch packet to public address
		if _, err := h.conn.WriteToUDP(punchData, peerPublicAddr); err != nil {
			lastErr = err
			time.Sleep(h.config.PunchInterval)
			continue
		}

		// Also try private address (for LAN peers)
		if peerPrivateAddr != nil {
			h.conn.WriteToUDP(punchData, peerPrivateAddr)
		}

		// Set read deadline
		h.conn.SetReadDeadline(time.Now().Add(h.config.PunchInterval * 2))

		// Wait for response
		buf := make([]byte, 1024)
		n, addr, err := h.conn.ReadFromUDP(buf)
		if err != nil {
			lastErr = err
			time.Sleep(h.config.PunchInterval)
			continue
		}

		// Verify punch response
		if h.verifyPunchResponse(buf[:n], peerID) {
			latency := time.Since(startTime)
			result := &PunchResult{
				Success:    true,
				RemoteAddr: addr,
				LocalAddr:  h.conn.LocalAddr().(*net.UDPAddr),
				Attempts:   attempt + 1,
				Latency:    latency,
			}
			h.mu.Lock()
			h.punchResult[peerID] = result
			h.mu.Unlock()
			return result, nil
		}
	}

	return &PunchResult{
		Success:  false,
		Attempts: h.config.MaxPunchAttempts,
	}, fmt.Errorf("hole punching failed after %d attempts: %w", h.config.MaxPunchAttempts, lastErr)
}

// createPunchPacket creates a punch packet with the given peer ID.
func (h *HolePuncher) createPunchPacket(peerID string) []byte {
	// Simple punch packet format:
	// [4 bytes magic][8 bytes peerID length][peerID bytes][8 bytes timestamp]
	magic := []byte{0x4D, 0x59, 0x43, 0x45} // "MYCE"
	peerIDLen := make([]byte, 8)
	binary.BigEndian.PutUint64(peerIDLen, uint64(len(peerID)))
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixNano()))

	packet := make([]byte, 0, 4+8+len(peerID)+8)
	packet = append(packet, magic...)
	packet = append(packet, peerIDLen...)
	packet = append(packet, []byte(peerID)...)
	packet = append(packet, timestamp...)

	return packet
}

// verifyPunchResponse verifies if the received packet is a valid punch response.
func (h *HolePuncher) verifyPunchResponse(data []byte, expectedPeerID string) bool {
	if len(data) < 20 {
		return false
	}

	// Check magic bytes
	magic := []byte{0x4D, 0x59, 0x43, 0x45}
	for i := 0; i < 4; i++ {
		if data[i] != magic[i] {
			return false
		}
	}

	// Parse peer ID length
	peerIDLen := int(binary.BigEndian.Uint64(data[4:12]))
	if 12+peerIDLen > len(data) {
		return false
	}

	// Verify peer ID
	receivedPeerID := string(data[12 : 12+peerIDLen])
	return receivedPeerID == expectedPeerID
}

// PunchMultiple attempts to punch holes to multiple peers and returns successful connections.
func (h *HolePuncher) PunchMultiple(ctx context.Context, peers map[string]*PeerInfo) ([]*PunchResult, error) {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*PunchResult
	)

	for peerID, info := range peers {
		wg.Add(1)
		go func(id string, peerInfo *PeerInfo) {
			defer wg.Done()
			result, err := h.Punch(ctx, id, peerInfo.PublicAddr, peerInfo.PrivateAddr)
			if err != nil {
				return
			}
			if result.Success {
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}(peerID, info)
	}

	wg.Wait()
	return results, nil
}

// PeerInfo contains information about a peer for hole punching.
type PeerInfo struct {
	PublicAddr  *net.UDPAddr
	PrivateAddr *net.UDPAddr
	NATType     stun.NATType
}

// CanPunch checks if hole punching is likely to succeed based on NAT types.
func CanPunch(localNAT, remoteNAT stun.NATType) bool {
	// Full cone can always punch
	if localNAT == stun.NATFullCone || remoteNAT == stun.NATFullCone {
		return true
	}

	// Symmetric NAT cannot punch with other symmetric NAT
	if localNAT == stun.NATSymmetric && remoteNAT == stun.NATSymmetric {
		return false
	}

	// Other combinations have a chance
	return true
}
