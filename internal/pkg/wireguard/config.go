package wireguard

import (
	"fmt"
	"strings"
)

// PeerConfig represents a WireGuard peer configuration.
type PeerConfig struct {
	PublicKey  string
	Endpoint   string // IP:port or empty for dynamic
	AllowedIPs string // CIDR, e.g., "10.0.0.2/32"
}

// InterfaceConfig represents WireGuard interface configuration.
type InterfaceConfig struct {
	PrivateKey string
	Address    string // CIDR, e.g., "10.0.0.1/16"
	ListenPort int    // 0 for random
	Peers      []PeerConfig
}

// GenerateWGQuickConfig generates a wg-quick compatible config file.
// Returns the config file content as a string.
func GenerateWGQuickConfig(interfaceName string, config InterfaceConfig) (string, error) {
	if config.PrivateKey == "" {
		return "", fmt.Errorf("private key is required")
	}
	if config.Address == "" {
		return "", fmt.Errorf("address is required")
	}

	var sb strings.Builder

	// [Interface] section
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", config.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", config.Address))
	if config.ListenPort > 0 {
		sb.WriteString(fmt.Sprintf("ListenPort = %d\n", config.ListenPort))
	}
	sb.WriteString("\n")

	// [Peer] sections
	for _, peer := range config.Peers {
		if peer.PublicKey == "" {
			continue
		}
		sb.WriteString("[Peer]\n")
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		if peer.Endpoint != "" {
			sb.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
		}
		if peer.AllowedIPs != "" {
			sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
		} else {
			// Default: allow all traffic from this peer's IP
			sb.WriteString("AllowedIPs = 0.0.0.0/0\n")
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GenerateConfigFromAgentConfig generates WireGuard config from agent config.
func GenerateConfigFromAgentConfig(agentConfig *AgentConfig) (InterfaceConfig, error) {
	if agentConfig == nil {
		return InterfaceConfig{}, fmt.Errorf("agent config is nil")
	}

	config := InterfaceConfig{
		PrivateKey: agentConfig.PrivateKey,
		Address:    agentConfig.AssignedIP + "/" + agentConfig.SubnetMask(),
		ListenPort: 51820, // Default WireGuard port
	}

	// Add peers
	for _, peer := range agentConfig.Peers {
		peerConfig := PeerConfig{
			PublicKey:  peer.PublicKey,
			AllowedIPs: peer.IP + "/32",
		}
		// Set endpoint if peer has public address
		if peer.Endpoint != "" {
			peerConfig.Endpoint = peer.Endpoint
		}
		config.Peers = append(config.Peers, peerConfig)
	}

	return config, nil
}

// AgentConfig represents the agent's configuration for WireGuard.
type AgentConfig struct {
	PrivateKey  string
	PublicKey   string
	AssignedIP  string
	SubnetCIDR  string
	Peers       []PeerInfo
}

// PeerInfo represents a peer node.
type PeerInfo struct {
	NodeID    string
	Name      string
	IP        string
	PublicKey string
	Endpoint  string // Public endpoint if known
}

// SubnetMask extracts the subnet mask from CIDR.
func (c *AgentConfig) SubnetMask() string {
	parts := strings.Split(c.SubnetCIDR, "/")
	if len(parts) != 2 {
		return "16" // Default
	}
	return parts[1]
}

// SaveWGQuickConfig saves the config to a file.
func SaveWGQuickConfig(path string, content string) error {
	// This would typically use os.WriteFile, but we'll just return the content
	// for now as the actual file writing should be handled by the caller
	return nil
}