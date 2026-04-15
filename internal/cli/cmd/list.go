package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/mycel/mesh/api/pb"
	"github.com/mycel/mesh/internal/agent/config"
	"github.com/mycel/mesh/internal/cli/client"
	"github.com/spf13/cobra"
)

// listCmd lists all nodes in the network.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List network nodes",
	Long:  "Display all nodes currently in the Mycel network",
	Example: `  mycelctl list --coordinator localhost:51820
  mycelctl list -c localhost:51820`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringP("coordinator", "c", "", "Coordinator address (host:port)")
	listCmd.Flags().StringP("token", "t", "", "Join token (optional, uses saved config if available)")
	listCmd.Flags().StringP("config", "f", "", "Config file path (default: ~/.mycel/config.json)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	coordinator, _ := cmd.Flags().GetString("coordinator")
	token, _ := cmd.Flags().GetString("token")
	configPath, _ := cmd.Flags().GetString("config")

	// Try to load config if coordinator not specified
	if coordinator == "" {
		cfgManager := config.NewManager(configPath)
		cfg, err := cfgManager.Load()
		if err != nil {
			return fmt.Errorf("coordinator address required (use -c flag or join first to save config)")
		}
		coordinator = cfg.Coordinator
		if token == "" {
			token = cfg.Token
		}
	}

	if coordinator == "" {
		return fmt.Errorf("coordinator address required")
	}

	// Connect to Coordinator
	grpcClient, err := client.NewClientWithTimeout(coordinator, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to coordinator: %w", err)
	}
	defer grpcClient.Close()

	// Get node list
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := grpcClient.ListNodes(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("list error: %s", resp.Error)
	}

	// Display results
	if len(resp.Nodes) == 0 {
		fmt.Println("No nodes in network.")
		return nil
	}

	fmt.Printf("Nodes in network (%d total):\n\n", len(resp.Nodes))

	// Use tabwriter for nice formatting
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tIP\tSTATUS\tNAT TYPE\tPUBLIC ADDR")
	fmt.Fprintln(w, "----\t--\t------\t--------\t-----------")

	for _, node := range resp.Nodes {
		natType := "unknown"
		publicAddr := "-"
		if node.NatInfo != nil {
			natType = node.NatInfo.NatType
			if node.NatInfo.PublicIp != "" {
				publicAddr = fmt.Sprintf("%s:%d", node.NatInfo.PublicIp, node.NatInfo.PublicPort)
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", node.Name, node.Ip, node.Status, natType, publicAddr)
	}

	w.Flush()

	// Show summary
	online := 0
	for _, n := range resp.Nodes {
		if n.Status == "online" {
			online++
		}
	}
	fmt.Printf("\nSummary: %d online, %d offline\n", online, len(resp.Nodes)-online)

	return nil
}

// displayNodeDetails shows detailed information for a specific node.
func displayNodeDetails(node *pb.PeerInfo) {
	fmt.Printf("\nNode: %s\n", node.Name)
	fmt.Printf("  ID: %s\n", node.NodeId)
	fmt.Printf("  IP: %s\n", node.Ip)
	fmt.Printf("  Status: %s\n", node.Status)
	fmt.Printf("  Public Key: %s...\n", node.PublicKey[:20])

	if node.NatInfo != nil {
		fmt.Printf("  NAT Type: %s\n", node.NatInfo.NatType)
		fmt.Printf("  Public Address: %s:%d\n", node.NatInfo.PublicIp, node.NatInfo.PublicPort)
		fmt.Printf("  P2P Capable: %v\n", node.NatInfo.CanPunch)
		if len(node.NatInfo.LocalIps) > 0 {
			fmt.Printf("  Local IPs: %v\n", node.NatInfo.LocalIps)
		}
	}
}