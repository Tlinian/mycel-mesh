package wireguard

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestGenerateKey_Success tests successful key generation.
func TestGenerateKey_Success(t *testing.T) {
	priv, pub, err := GenerateKey()

	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}

	if priv == "" {
		t.Fatal("private key is empty")
	}

	if pub == "" {
		t.Fatal("public key is empty")
	}

	// Verify private key is valid base64
	decodedPriv, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		t.Fatalf("private key is not valid base64: %v", err)
	}
	if len(decodedPriv) != 32 {
		t.Fatalf("private key length should be 32, got %d", len(decodedPriv))
	}

	// Verify public key is valid base64
	decodedPub, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		t.Fatalf("public key is not valid base64: %v", err)
	}
	if len(decodedPub) != 32 {
		t.Fatalf("public key length should be 32, got %d", len(decodedPub))
	}
}

// TestGenerateKey_Uniqueness tests that multiple calls generate different keys.
func TestGenerateKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		priv, pub, err := GenerateKey()
		if err != nil {
			t.Fatalf("GenerateKey() failed at iteration %d: %v", i, err)
		}
		if keys[priv] {
			t.Fatalf("duplicate private key at iteration %d", i)
		}
		if keys[pub] {
			t.Fatalf("duplicate public key at iteration %d", i)
		}
		keys[priv] = true
		keys[pub] = true
	}
}

// TestGenerateKey_Curve25519Clamping tests that private key is properly clamped.
func TestGenerateKey_Curve25519Clamping(t *testing.T) {
	for i := 0; i < 10; i++ {
		priv, _, err := GenerateKey()
		if err != nil {
			t.Fatalf("GenerateKey() failed: %v", err)
		}

		decoded, err := base64.StdEncoding.DecodeString(priv)
		if err != nil {
			t.Fatalf("failed to decode private key: %v", err)
		}

		// Check clamping: byte 0 & 248, byte 31 & 127 | 64
		if decoded[0]&7 != 0 {
			t.Fatalf("private key byte 0 not properly clamped: got %d", decoded[0])
		}
		if decoded[31]&128 != 0 {
			t.Fatalf("private key byte 31 high bit not cleared: got %d", decoded[31])
		}
		if decoded[31]&64 == 0 {
			t.Fatalf("private key byte 31 bit 6 not set: got %d", decoded[31])
		}
	}
}

// TestGenerateWGQuickConfig_ValidConfig tests valid config generation.
func TestGenerateWGQuickConfig_ValidConfig(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "testPrivateKey123",
		Address:    "10.0.0.1/24",
		ListenPort: 51820,
		Peers: []PeerConfig{
			{
				PublicKey:  "peerPubKey1",
				Endpoint:   "192.168.1.1:51820",
				AllowedIPs: "10.0.0.2/32",
			},
		},
	}

	content, err := GenerateWGQuickConfig("wg0", config)
	if err != nil {
		t.Fatalf("GenerateWGQuickConfig() failed: %v", err)
	}

	// Verify content contains expected sections
	if !strings.Contains(content, "[Interface]") {
		t.Fatal("config missing [Interface] section")
	}
	if !strings.Contains(content, "PrivateKey = testPrivateKey123") {
		t.Fatal("config missing PrivateKey")
	}
	if !strings.Contains(content, "Address = 10.0.0.1/24") {
		t.Fatal("config missing Address")
	}
	if !strings.Contains(content, "ListenPort = 51820") {
		t.Fatal("config missing ListenPort")
	}
	if !strings.Contains(content, "[Peer]") {
		t.Fatal("config missing [Peer] section")
	}
	if !strings.Contains(content, "PublicKey = peerPubKey1") {
		t.Fatal("config missing peer PublicKey")
	}
	if !strings.Contains(content, "Endpoint = 192.168.1.1:51820") {
		t.Fatal("config missing peer Endpoint")
	}
}

// TestGenerateWGQuickConfig_MissingPrivateKey tests error when private key is missing.
func TestGenerateWGQuickConfig_MissingPrivateKey(t *testing.T) {
	config := InterfaceConfig{
		Address: "10.0.0.1/24",
	}

	_, err := GenerateWGQuickConfig("wg0", config)
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
	if !strings.Contains(err.Error(), "private key is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGenerateWGQuickConfig_MissingAddress tests error when address is missing.
func TestGenerateWGQuickConfig_MissingAddress(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "testKey",
	}

	_, err := GenerateWGQuickConfig("wg0", config)
	if err == nil {
		t.Fatal("expected error for missing address")
	}
	if !strings.Contains(err.Error(), "address is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGenerateWGQuickConfig_MultiplePeers tests config with multiple peers.
func TestGenerateWGQuickConfig_MultiplePeers(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "testPrivateKey",
		Address:    "10.0.0.1/24",
		Peers: []PeerConfig{
			{PublicKey: "peer1", AllowedIPs: "10.0.0.2/32"},
			{PublicKey: "peer2", AllowedIPs: "10.0.0.3/32"},
			{PublicKey: "peer3", Endpoint: "1.2.3.4:51820", AllowedIPs: "10.0.0.4/32"},
		},
	}

	content, err := GenerateWGQuickConfig("wg0", config)
	if err != nil {
		t.Fatalf("GenerateWGQuickConfig() failed: %v", err)
	}

	// Count peer sections
	peerCount := strings.Count(content, "[Peer]")
	if peerCount != 3 {
		t.Fatalf("expected 3 peer sections, got %d", peerCount)
	}
}

// TestGenerateWGQuickConfig_NoListenPort tests config without explicit listen port.
func TestGenerateWGQuickConfig_NoListenPort(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "testKey",
		Address:    "10.0.0.1/24",
		ListenPort: 0, // 0 means no explicit port
	}

	content, err := GenerateWGQuickConfig("wg0", config)
	if err != nil {
		t.Fatalf("GenerateWGQuickConfig() failed: %v", err)
	}

	if strings.Contains(content, "ListenPort") {
		t.Fatal("ListenPort should not be included when set to 0")
	}
}

// TestGenerateWGQuickConfig_PeerWithoutPublicKey tests peer with empty public key is skipped.
func TestGenerateWGQuickConfig_PeerWithoutPublicKey(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "testKey",
		Address:    "10.0.0.1/24",
		Peers: []PeerConfig{
			{PublicKey: "", Endpoint: "1.2.3.4:51820"}, // Should be skipped
			{PublicKey: "validPeer", AllowedIPs: "10.0.0.2/32"},
		},
	}

	content, err := GenerateWGQuickConfig("wg0", config)
	if err != nil {
		t.Fatalf("GenerateWGQuickConfig() failed: %v", err)
	}

	peerCount := strings.Count(content, "[Peer]")
	if peerCount != 1 {
		t.Fatalf("expected 1 peer section (empty pubkey should be skipped), got %d", peerCount)
	}
}

// TestGenerateConfigFromAgentConfig_Success tests agent config conversion.
func TestGenerateConfigFromAgentConfig_Success(t *testing.T) {
	agentConfig := &AgentConfig{
		PrivateKey: "agentPrivKey",
		PublicKey:  "agentPubKey",
		AssignedIP: "10.0.0.5",
		SubnetCIDR: "10.0.0.0/24",
		Peers: []PeerInfo{
			{Name: "peer1", IP: "10.0.0.2", PublicKey: "peer1Pub", Endpoint: "1.2.3.4:51820"},
			{Name: "peer2", IP: "10.0.0.3", PublicKey: "peer2Pub"},
		},
	}

	config, err := GenerateConfigFromAgentConfig(agentConfig)
	if err != nil {
		t.Fatalf("GenerateConfigFromAgentConfig() failed: %v", err)
	}

	if config.PrivateKey != "agentPrivKey" {
		t.Fatalf("expected PrivateKey 'agentPrivKey', got '%s'", config.PrivateKey)
	}
	if config.Address != "10.0.0.5/24" {
		t.Fatalf("expected Address '10.0.0.5/24', got '%s'", config.Address)
	}
	if config.ListenPort != 51820 {
		t.Fatalf("expected ListenPort 51820, got %d", config.ListenPort)
	}
	if len(config.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(config.Peers))
	}
}

// TestGenerateConfigFromAgentConfig_NilConfig tests error for nil agent config.
func TestGenerateConfigFromAgentConfig_NilConfig(t *testing.T) {
	_, err := GenerateConfigFromAgentConfig(nil)
	if err == nil {
		t.Fatal("expected error for nil agent config")
	}
	if !strings.Contains(err.Error(), "agent config is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestAgentConfig_SubnetMask tests subnet mask extraction.
func TestAgentConfig_SubnetMask(t *testing.T) {
	tests := []struct {
		cidr     string
		expected string
	}{
		{"10.0.0.0/24", "24"},
		{"10.0.0.0/16", "16"},
		{"192.168.1.0/28", "28"},
		{"", "16"},       // Default
		{"10.0.0.0", "16"}, // No mask
	}

	for _, tt := range tests {
		config := &AgentConfig{SubnetCIDR: tt.cidr}
		mask := config.SubnetMask()
		if mask != tt.expected {
			t.Fatalf("SubnetMask() for CIDR '%s' expected '%s', got '%s'", tt.cidr, tt.expected, mask)
		}
	}
}

// TestPeerConfig_DefaultAllowedIPs tests default AllowedIPs when not specified.
func TestPeerConfig_DefaultAllowedIPs(t *testing.T) {
	config := InterfaceConfig{
		PrivateKey: "testKey",
		Address:    "10.0.0.1/24",
		Peers: []PeerConfig{
			{PublicKey: "peer1"}, // No AllowedIPs
		},
	}

	content, err := GenerateWGQuickConfig("wg0", config)
	if err != nil {
		t.Fatalf("GenerateWGQuickConfig() failed: %v", err)
	}

	if !strings.Contains(content, "AllowedIPs = 0.0.0.0/0") {
		t.Fatal("expected default AllowedIPs '0.0.0.0/0' when not specified")
	}
}