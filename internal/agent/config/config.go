// Package config provides configuration management for the agent.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrInvalidConfig  = errors.New("invalid configuration")
)

// PeerConfig holds configuration for a peer node.
type PeerConfig struct {
	NodeID    string `json:"node_id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	PublicKey string `json:"public_key"`
}

// NodeConfig holds the agent configuration.
type NodeConfig struct {
	NodeID      string        `json:"node_id"`
	Name        string        `json:"name"`
	PublicKey   string        `json:"public_key"`
	PrivateKey  string        `json:"private_key"`
	AssignedIP  string        `json:"assigned_ip"`
	SubnetCIDR  string        `json:"subnet_cidr"`
	Coordinator string        `json:"coordinator"`
	Token       string        `json:"token"`
	Peers       []PeerConfig  `json:"peers"`
}

// Manager handles configuration persistence.
type Manager struct {
	ConfigPath string
	config     *NodeConfig
	mu         sync.RWMutex
}

// NewManager creates a new configuration manager.
func NewManager(configPath string) *Manager {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}
	return &Manager{
		ConfigPath: configPath,
	}
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	// On Windows, use %USERPROFILE%\.mycel\config.json
	// On other platforms, use ~/.mycel/config.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".mycel", "config.json")
}

// Load reads the configuration from disk.
func (m *Manager) Load() (*NodeConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	var config NodeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, ErrInvalidConfig
	}

	m.config = &config
	return &config, nil
}

// Save writes the configuration to disk.
func (m *Manager) Save(config *NodeConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(m.ConfigPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(m.ConfigPath, data, 0600); err != nil {
		return err
	}

	m.config = config
	return nil
}

// Get returns the current configuration.
func (m *Manager) Get() *NodeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// UpdatePeer adds or updates a peer in the configuration.
func (m *Manager) UpdatePeer(peer PeerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return ErrConfigNotFound
	}

	// Find existing peer
	for i, p := range m.config.Peers {
		if p.NodeID == peer.NodeID {
			m.config.Peers[i] = peer
			return m.Save(m.config)
		}
	}

	// Add new peer
	m.config.Peers = append(m.config.Peers, peer)
	return m.Save(m.config)
}

// RemovePeer removes a peer from the configuration.
func (m *Manager) RemovePeer(nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return ErrConfigNotFound
	}

	for i, p := range m.config.Peers {
		if p.NodeID == nodeID {
			m.config.Peers = append(m.config.Peers[:i], m.config.Peers[i+1:]...)
			return m.Save(m.config)
		}
	}

	return nil
}

// Exists checks if a configuration file exists.
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.ConfigPath)
	return err == nil
}

// Delete removes the configuration file.
func (m *Manager) Delete() error {
	return os.Remove(m.ConfigPath)
}

// LoadConfig is a convenience function to load configuration.
func LoadConfig(path string) (*NodeConfig, error) {
	m := NewManager(path)
	return m.Load()
}

// SaveConfig is a convenience function to save configuration.
func SaveConfig(path string, config *NodeConfig) error {
	m := NewManager(path)
	return m.Save(config)
}