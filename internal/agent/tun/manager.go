// Package tun provides TUN device management for virtual network interfaces.
package tun

import (
	"fmt"
	"net"
	"sync"
)

// Manager manages TUN device operations.
type Manager struct {
	device    Device
	config    *Config
	mu        sync.RWMutex
	peers     map[string]*Peer // publicKey -> Peer
}

// Config holds TUN device configuration.
type Config struct {
	Name       string      // Interface name (e.g., "Mycel0")
	IP         net.IP      // Assigned IP address
	Subnet     *net.IPNet  // Subnet CIDR
	PrivateKey string      // WireGuard private key
	PublicKey  string      // WireGuard public key
	ListenPort int         // UDP listen port (0 = random)
}

// Peer represents a WireGuard peer.
type Peer struct {
	PublicKey  string
	Endpoint   *net.UDPAddr
	AllowedIPs []*net.IPNet
	LastSeen   int64 // Last handshake timestamp
}

// Device represents a TUN device interface.
type Device interface {
	// Name returns the device name.
	Name() string
	// Close closes the device.
	Close() error
	// File returns the file descriptor (if applicable).
	File() int
	// SetIP sets the device IP address.
	SetIP(ip net.IP, subnet *net.IPNet) error
	// Up brings up the device.
	Up() error
	// Down brings down the device.
	Down() error
	// Read reads a packet from the device.
	Read(buf []byte) (int, error)
	// Write writes a packet to the device.
	Write(buf []byte) (int, error)
}

// NewManager creates a new TUN manager.
func NewManager() *Manager {
	return &Manager{
		peers: make(map[string]*Peer),
	}
}

// CreateInterface creates a new TUN interface.
// On Windows, this uses Wintun driver.
func (m *Manager) CreateInterface(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config == nil {
		return fmt.Errorf("config is required")
	}

	// Create platform-specific device
	device, err := createDevice(config)
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	// Configure IP address
	if err := device.SetIP(config.IP, config.Subnet); err != nil {
		device.Close()
		return fmt.Errorf("failed to set IP: %w", err)
	}

	// Bring up the interface
	if err := device.Up(); err != nil {
		device.Close()
		return fmt.Errorf("failed to bring up interface: %w", err)
	}

	m.device = device
	m.config = config
	return nil
}

// ConfigureInterface reconfigures the interface.
func (m *Manager) ConfigureInterface(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.device == nil {
		return fmt.Errorf("device not created")
	}

	if err := m.device.SetIP(config.IP, config.Subnet); err != nil {
		return fmt.Errorf("failed to set IP: %w", err)
	}

	m.config = config
	return nil
}

// AddPeer adds a WireGuard peer to the interface.
func (m *Manager) AddPeer(publicKey string, endpoint *net.UDPAddr, allowedIPs []*net.IPNet) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.device == nil {
		return fmt.Errorf("device not created")
	}

	peer := &Peer{
		PublicKey:  publicKey,
		Endpoint:   endpoint,
		AllowedIPs: allowedIPs,
	}

	m.peers[publicKey] = peer
	return nil
}

// RemovePeer removes a WireGuard peer from the interface.
func (m *Manager) RemovePeer(publicKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.device == nil {
		return fmt.Errorf("device not created")
	}

	delete(m.peers, publicKey)
	return nil
}

// UpdatePeerEndpoint updates a peer's endpoint.
func (m *Manager) UpdatePeerEndpoint(publicKey string, endpoint *net.UDPAddr) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	peer, exists := m.peers[publicKey]
	if !exists {
		return fmt.Errorf("peer not found")
	}

	peer.Endpoint = endpoint
	return nil
}

// GetPeers returns all configured peers.
func (m *Manager) GetPeers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		peers = append(peers, p)
	}
	return peers
}

// Close closes the TUN interface.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.device == nil {
		return nil
	}

	err := m.device.Close()
	m.device = nil
	return err
}

// Name returns the interface name.
func (m *Manager) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.device == nil {
		return ""
	}
	return m.device.Name()
}

// Read reads a packet from the TUN device.
func (m *Manager) Read(buf []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.device == nil {
		return 0, fmt.Errorf("device not created")
	}
	return m.device.Read(buf)
}

// Write writes a packet to the TUN device.
func (m *Manager) Write(buf []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.device == nil {
		return 0, fmt.Errorf("device not created")
	}
	return m.device.Write(buf)
}

// IsRunning checks if the interface is running.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.device != nil
}

// GetConfig returns the current configuration.
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}