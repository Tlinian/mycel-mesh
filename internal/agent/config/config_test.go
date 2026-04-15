package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewManager_DefaultPath tests manager creation with default path.
func TestNewManager_DefaultPath(t *testing.T) {
	manager := NewManager("")
	if manager.ConfigPath == "" {
		t.Fatal("ConfigPath should not be empty")
	}
	if !filepath.IsAbs(manager.ConfigPath) {
		t.Fatal("ConfigPath should be absolute")
	}
}

// TestNewManager_CustomPath tests manager creation with custom path.
func TestNewManager_CustomPath(t *testing.T) {
	customPath := "/tmp/mycel-config.json"
	manager := NewManager(customPath)
	if manager.ConfigPath != customPath {
		t.Fatalf("expected ConfigPath '%s', got '%s'", customPath, manager.ConfigPath)
	}
}

// TestDefaultConfigPath tests default configuration path.
func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Fatal("default path should not be empty")
	}
	if filepath.Base(path) != "config.json" {
		t.Fatalf("expected filename 'config.json', got '%s'", filepath.Base(path))
	}
}

// TestManager_SaveLoad tests save and load roundtrip.
func TestManager_SaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	manager := NewManager(configPath)

	config := &NodeConfig{
		NodeID:      "test-node-001",
		Name:        "test-node",
		PublicKey:   "testPubKey",
		PrivateKey:  "testPrivKey",
		AssignedIP:  "10.0.0.5",
		SubnetCIDR:  "10.0.0.0/16",
		Coordinator: "coordinator.example.com:51820",
		Token:       "test-token",
		Peers: []PeerConfig{
			{NodeID: "peer-1", Name: "peer-one", IP: "10.0.0.2", PublicKey: "peer1Pub"},
		},
	}

	err = manager.Save(config)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	if !manager.Exists() {
		t.Fatal("config file should exist after save")
	}

	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.NodeID != config.NodeID {
		t.Fatalf("expected NodeID '%s', got '%s'", config.NodeID, loaded.NodeID)
	}
	if loaded.Name != config.Name {
		t.Fatalf("expected Name '%s', got '%s'", config.Name, loaded.Name)
	}
	if len(loaded.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(loaded.Peers))
	}
}

// TestManager_Load_NotFound tests load error when file doesn't exist.
func TestManager_Load_NotFound(t *testing.T) {
	manager := NewManager("/nonexistent/path/config.json")

	_, err := manager.Load()
	if err != ErrConfigNotFound {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

// TestManager_Load_InvalidJSON tests load error for invalid JSON.
func TestManager_Load_InvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	err = os.WriteFile(configPath, []byte("not valid json"), 0600)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	manager := NewManager(configPath)
	_, err = manager.Load()
	if err != ErrInvalidConfig {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

// TestManager_Get tests getting current config.
func TestManager_Get(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	manager := NewManager(configPath)

	config := manager.Get()
	if config != nil {
		t.Fatal("initial Get() should return nil")
	}

	testConfig := &NodeConfig{NodeID: "test-001", Name: "test"}
	manager.Save(testConfig)

	config = manager.Get()
	if config == nil {
		t.Fatal("Get() should return config after Save()")
	}
}

// TestManager_Exists tests existence check.
func TestManager_Exists(t *testing.T) {
	manager := NewManager("/nonexistent/path/config.json")

	if manager.Exists() {
		t.Fatal("Exists() should return false for nonexistent file")
	}

	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(configPath, []byte("{}"), 0600)

	manager = NewManager(configPath)
	if !manager.Exists() {
		t.Fatal("Exists() should return true for existing file")
	}
}

// TestManager_Delete tests deleting config file.
func TestManager_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	manager := NewManager(configPath)

	config := &NodeConfig{NodeID: "test-001"}
	manager.Save(config)

	err = manager.Delete()
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if manager.Exists() {
		t.Fatal("config file should not exist after Delete()")
	}
}

// TestLoadConfig tests convenience LoadConfig function.
func TestLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	testConfig := &NodeConfig{NodeID: "test-001", Name: "test"}
	SaveConfig(configPath, testConfig)

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if loaded.NodeID != "test-001" {
		t.Fatalf("expected NodeID 'test-001', got '%s'", loaded.NodeID)
	}
}

// TestSaveConfig tests convenience SaveConfig function.
func TestSaveConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	testConfig := &NodeConfig{NodeID: "test-001"}

	err = SaveConfig(configPath, testConfig)
	if err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatal("config file should exist")
	}
}

// TestPeerConfig tests peer configuration structure.
func TestPeerConfig(t *testing.T) {
	peer := PeerConfig{
		NodeID:    "peer-001",
		Name:      "test-peer",
		IP:        "10.0.0.2",
		PublicKey: "peerPubKey",
		Endpoint:  "192.168.1.1:51820",
	}

	if peer.NodeID != "peer-001" {
		t.Fatalf("expected NodeID 'peer-001', got '%s'", peer.NodeID)
	}
	if peer.Endpoint != "192.168.1.1:51820" {
		t.Fatalf("expected Endpoint '192.168.1.1:51820', got '%s'", peer.Endpoint)
	}
}

// TestNodeConfig tests node configuration structure.
func TestNodeConfig(t *testing.T) {
	config := NodeConfig{
		NodeID:      "node-001",
		Name:        "test-node",
		PublicKey:   "pubKey",
		PrivateKey:  "privKey",
		AssignedIP:  "10.0.0.5",
		SubnetCIDR:  "10.0.0.0/16",
		Coordinator: "coord.example.com:51820",
		Token:       "token123",
	}

	if config.NodeID != "node-001" {
		t.Fatalf("expected NodeID 'node-001', got '%s'", config.NodeID)
	}
	if config.AssignedIP != "10.0.0.5" {
		t.Fatalf("expected AssignedIP '10.0.0.5', got '%s'", config.AssignedIP)
	}
}

// TestManager_Save_CreatesDirectory tests that Save creates parent directory.
func TestManager_Save_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a path with non-existent parent directory
	configPath := filepath.Join(tmpDir, "subdir", "config.json")
	manager := NewManager(configPath)

	config := &NodeConfig{NodeID: "test-001"}
	err = manager.Save(config)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("parent directory should be created")
	}
}

// TestManager_Save_WithEmptyPeers tests saving config with empty peers.
func TestManager_Save_WithEmptyPeers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	manager := NewManager(configPath)

	config := &NodeConfig{
		NodeID: "test-001",
		Peers:  []PeerConfig{}, // Empty peers
	}

	err = manager.Save(config)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(loaded.Peers) != 0 {
		t.Fatalf("expected 0 peers, got %d", len(loaded.Peers))
	}
}

// TestManager_Save_WithMultiplePeers tests saving config with multiple peers.
func TestManager_Save_WithMultiplePeers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mycel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	manager := NewManager(configPath)

	config := &NodeConfig{
		NodeID: "test-001",
		Peers: []PeerConfig{
			{NodeID: "peer-1", Name: "peer-one", IP: "10.0.0.2"},
			{NodeID: "peer-2", Name: "peer-two", IP: "10.0.0.3"},
			{NodeID: "peer-3", Name: "peer-three", IP: "10.0.0.4"},
		},
	}

	err = manager.Save(config)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(loaded.Peers) != 3 {
		t.Fatalf("expected 3 peers, got %d", len(loaded.Peers))
	}
}