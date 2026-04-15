package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mycel/mesh/api/pb"
	"github.com/mycel/mesh/internal/agent/config"
	"github.com/mycel/mesh/internal/cli/client"
	"github.com/mycel/mesh/internal/pkg/stun"
	"github.com/mycel/mesh/internal/pkg/wireguard"
	"github.com/spf13/cobra"
)

// joinCmd joins a Mycel network.
var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join Mycel network",
	Long:  "Connect to Mycel Coordinator and join the virtual network",
	Example: `  mycelctl join --token <token> --coordinator localhost:51820
  mycelctl join -t <token> -c localhost:51820 --name node-1`,
	RunE: runJoin,
}

func init() {
	joinCmd.Flags().StringP("token", "t", "", "Join token")
	joinCmd.Flags().StringP("coordinator", "c", "", "Coordinator address (host:port)")
	joinCmd.Flags().StringP("name", "n", "", "Node name")
	joinCmd.Flags().StringP("config", "f", "", "Config file path (default: ~/.mycel/config.json)")
	joinCmd.MarkFlagRequired("token")
	joinCmd.MarkFlagRequired("coordinator")
	rootCmd.AddCommand(joinCmd)
}

func runJoin(cmd *cobra.Command, args []string) error {
	token, _ := cmd.Flags().GetString("token")
	coordinator, _ := cmd.Flags().GetString("coordinator")
	name, _ := cmd.Flags().GetString("name")
	configPath, _ := cmd.Flags().GetString("config")

	if name == "" {
		name = fmt.Sprintf("node-%d", time.Now().UnixNano()%10000)
	}

	fmt.Println("Joining Mycel network...")
	fmt.Printf("  Node name: %s\n", name)
	fmt.Printf("  Coordinator: %s\n", coordinator)

	// Step 1: Generate WireGuard keys
	fmt.Println("\n[1/4] Generating WireGuard keys...")
	privateKey, publicKey, err := wireguard.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}
	fmt.Printf("  Public key: %s...\n", publicKey[:20])

	// Step 2: Detect NAT type
	fmt.Println("\n[2/4] Detecting NAT type...")
	natInfo, err := detectNAT()
	if err != nil {
		fmt.Printf("  Warning: NAT detection failed: %v\n", err)
		// Continue with unknown NAT type
		natInfo = &stun.NATInfo{Type: stun.NATUnknown}
	} else {
		fmt.Printf("  NAT type: %s\n", natInfo.Type)
		if natInfo.PublicAddr != nil {
			fmt.Printf("  Public address: %s\n", natInfo.PublicAddr.String())
		}
		fmt.Printf("  P2P capable: %v\n", natInfo.CanP2P)
	}

	// Step 3: Connect to Coordinator and register
	fmt.Println("\n[3/4] Connecting to Coordinator...")
	nodeID := uuid.New().String()

	grpcClient, err := client.NewClientWithTimeout(coordinator, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to coordinator: %w", err)
	}
	defer grpcClient.Close()

	// Convert NAT info to proto format
	pbNATInfo := convertNATInfoToProto(natInfo)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := grpcClient.Register(ctx, nodeID, name, publicKey, token, pbNATInfo)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("registration error: %s", resp.Error)
	}

	fmt.Printf("  Assigned IP: %s\n", resp.AssignedIp)
	fmt.Printf("  Subnet: %s\n", resp.SubnetCidr)
	fmt.Printf("  Existing peers: %d\n", len(resp.Peers))

	// Step 4: Save configuration
	fmt.Println("\n[4/4] Saving configuration...")
	cfgManager := config.NewManager(configPath)

	nodeConfig := &config.NodeConfig{
		NodeID:      nodeID,
		Name:        name,
		PublicKey:   publicKey,
		PrivateKey:  privateKey,
		AssignedIP:  resp.AssignedIp,
		SubnetCIDR:  resp.SubnetCidr,
		Coordinator: coordinator,
		Token:       token,
		Peers:       convertPeersToConfig(resp.Peers),
	}

	if err := cfgManager.Save(nodeConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("  Config saved to: %s\n", cfgManager.ConfigPath)

	// Display success message
	fmt.Println("\nSuccessfully joined network!")
	fmt.Println("\nNext steps:")
	fmt.Println("  View status:  mycelctl status")
	fmt.Println("  List nodes:   mycelctl list -c " + coordinator)
	fmt.Println("  Start agent:  mycelctl start")

	// Display peers if any
	if len(resp.Peers) > 0 {
		fmt.Println("\nExisting peers in network:")
		for _, peer := range resp.Peers {
			fmt.Printf("  - %s (%s) - %s\n", peer.Name, peer.Ip, peer.Status)
		}
	}

	return nil
}

// detectNAT performs NAT detection using STUN.
func detectNAT() (*stun.NATInfo, error) {
	return stun.GetNATTypeWithTimeout(10 * time.Second)
}

// convertNATInfoToProto converts internal NATInfo to proto NATInfo.
func convertNATInfoToProto(info *stun.NATInfo) *pb.NATInfo {
	if info == nil {
		return nil
	}

	pbInfo := &pb.NATInfo{
		NatType:  string(info.Type),
		CanPunch: info.CanP2P,
	}

	if info.PublicAddr != nil {
		pbInfo.PublicIp = info.PublicAddr.IP.String()
		pbInfo.PublicPort = int32(info.PublicAddr.Port)
	}

	// Get local IPs
	addrs, err := getLocalIPs()
	if err == nil {
		pbInfo.LocalIps = addrs
	}

	return pbInfo
}

// getLocalIPs returns local IP addresses.
func getLocalIPs() ([]string, error) {
	var ips []string

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if addr.IsGlobalUnicast() && !addr.IsLoopback() {
			ips = append(ips, addr.String())
		}
	}

	return ips, nil
}

// convertPeersToConfig converts proto peers to config peers.
func convertPeersToConfig(peers []*pb.PeerInfo) []config.PeerConfig {
	var result []config.PeerConfig
	for _, p := range peers {
		result = append(result, config.PeerConfig{
			NodeID:    p.NodeId,
			Name:      p.Name,
			IP:        p.Ip,
			PublicKey: p.PublicKey,
		})
	}
	return result
}