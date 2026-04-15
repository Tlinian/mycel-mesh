package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/mycel/mesh/api/pb"
	"github.com/mycel/mesh/internal/agent/config"
	"github.com/mycel/mesh/internal/agent/peer"
	"github.com/mycel/mesh/internal/agent/punch"
	"github.com/mycel/mesh/internal/agent/tun"
	"github.com/mycel/mesh/internal/cli/client"
	"github.com/mycel/mesh/internal/pkg/stun"
)

var (
	configPath = flag.String("config", "", "Path to config file (default: ~/.mycel/config.json)")
	showHelp   = flag.Bool("help", false, "Show help")
)

const version = "1.0.0"

func main() {
	flag.Parse()

	if *showHelp {
		fmt.Println("Mycel Agent - Virtual Network Agent")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  agent.exe [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Prerequisites:")
		fmt.Println("  1. Run 'mycelctl join' to register with coordinator")
		fmt.Println("  2. Run as Administrator on Windows")
		fmt.Println()
		return
	}

	// Load configuration
	cfgManager := config.NewManager(*configPath)
	nodeConfig, err := cfgManager.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		log.Fatalf("Run 'mycelctl join' first to register with coordinator")
	}

	log.Printf("Mycel Agent v%s starting...", version)
	log.Printf("Node: %s (%s)", nodeConfig.Name, nodeConfig.AssignedIP)
	log.Printf("Coordinator: %s", nodeConfig.Coordinator)

	// Check admin privileges on Windows
	if !tun.CheckAdminPrivileges() {
		log.Println("WARNING: Not running as Administrator")
		log.Println("         TUN interface creation may fail")
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	var tunManager *tun.Manager
	var peerManager *peer.Manager
	var grpcClient *client.Client
	var punchCoordinator *punch.Coordinator

	// 1. Connect to Coordinator
	log.Println("[1/5] Connecting to Coordinator...")
	grpcClient, err = client.NewClient(nodeConfig.Coordinator)
	if err != nil {
		log.Fatalf("Failed to connect to coordinator: %v", err)
	}
	defer grpcClient.Close()
	log.Printf("     Connected to %s", nodeConfig.Coordinator)

	// 2. Create TUN interface
	log.Println("[2/5] Creating TUN interface...")
	tunManager = tun.NewManager()

	ip := net.ParseIP(nodeConfig.AssignedIP)
	if ip == nil {
		log.Fatalf("Invalid assigned IP: %s", nodeConfig.AssignedIP)
	}

	// Parse subnet CIDR
	_, subnet, err := net.ParseCIDR(nodeConfig.SubnetCIDR)
	if err != nil {
		// Use default subnet if parsing fails
		_, subnet, _ = net.ParseCIDR("10.0.0.0/16")
	}

	tunConfig := &tun.Config{
		Name:       "Mycel0",
		IP:         ip,
		Subnet:     subnet,
		PrivateKey: nodeConfig.PrivateKey,
		PublicKey:  nodeConfig.PublicKey,
		ListenPort: 51820,
	}

	if err := tunManager.CreateInterface(tunConfig); err != nil {
		log.Printf("WARNING: Failed to create TUN interface: %v", err)
		log.Println("         Continuing without TUN (for testing)")
	} else {
		log.Printf("     Interface %s created with IP %s", tunConfig.Name, tunConfig.IP)
	}

	// 3. Initialize peer manager
	log.Println("[3/5] Initializing peer manager...")
	peerManager = peer.NewManager(grpcClient, nodeConfig.NodeID)

	// Set callbacks for peer changes
	peerManager.SetOnPeerAdd(func(p *peer.Peer) {
		log.Printf("New peer: %s (%s)", p.Name, p.IP)
		if tunManager.IsRunning() {
			// Add peer to WireGuard
			allowedIPs := []*net.IPNet{}
			if p.IP != "" {
				peerIP := net.ParseIP(p.IP)
				if peerIP != nil {
					allowedIPs = append(allowedIPs, &net.IPNet{
						IP:   peerIP,
						Mask: net.IPv4Mask(255, 255, 255, 255),
					})
				}
			}
			tunManager.AddPeer(p.PublicKey, p.GetEndpoint(), allowedIPs)
		}

		// Try hole punching with new peer
		if punchCoordinator != nil && punchCoordinator.IsP2PCapable() {
			go func() {
				pctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				result, err := punchCoordinator.TryPunchPeer(pctx, p)
				if err == nil && result != nil && result.Success {
					log.Printf("Hole punch succeeded with %s: %s", p.Name, result.RemoteAddr.String())
					// Update WireGuard endpoint
					if tunManager.IsRunning() {
						tunManager.UpdatePeerEndpoint(p.PublicKey, result.RemoteAddr)
					}
				}
			}()
		}
	})

	peerManager.SetOnPeerRemove(func(nodeID string) {
		log.Printf("Peer removed: %s", nodeID)
		if tunManager.IsRunning() {
			// Find and remove peer from WireGuard
			// Note: We need to track nodeID -> publicKey mapping
			// For now, we'll need to implement this properly
		}
	})

	// 4. Initialize punch coordinator
	log.Println("[4/5] Initializing punch coordinator...")
	punchCoordinator = punch.NewCoordinator(tunManager)
	if err := punchCoordinator.Start(); err != nil {
		log.Printf("WARNING: Failed to start punch coordinator: %v", err)
		log.Println("         NAT hole punching will be disabled")
	} else {
		log.Printf("     Punch coordinator started")
		localNAT := punchCoordinator.GetLocalNATInfo()
		if localNAT != nil {
			log.Printf("     Local NAT type: %s", localNAT.Type)
			if punchCoordinator.IsP2PCapable() {
				log.Printf("     P2P capable: yes")
			} else {
				log.Printf("     P2P capable: no (relay required)")
			}
		}
	}

	// Add existing peers from config
	for _, cfgPeer := range nodeConfig.Peers {
		if tunManager.IsRunning() {
			allowedIPs := []*net.IPNet{}
			if cfgPeer.IP != "" {
				peerIP := net.ParseIP(cfgPeer.IP)
				if peerIP != nil {
					allowedIPs = append(allowedIPs, &net.IPNet{
						IP:   peerIP,
						Mask: net.IPv4Mask(255, 255, 255, 255),
					})
				}
			}
			tunManager.AddPeer(cfgPeer.PublicKey, nil, allowedIPs)
		}
	}
	log.Printf("     %d existing peers configured", len(nodeConfig.Peers))

	// 5. Start heartbeat loop
	log.Println("[5/5] Starting heartbeat loop...")
	natInfoProvider := func() *pb.NATInfo {
		// Use punch coordinator's NAT info if available
		if punchCoordinator != nil {
			localNAT := punchCoordinator.GetLocalNATInfo()
			if localNAT != nil {
				return &pb.NATInfo{
					NatType:  string(localNAT.Type),
					CanPunch: localNAT.CanP2P,
				}
			}
		}

		// Fallback to standalone NAT detection
		natInfo, err := stun.SimpleNATDetection()
		if err != nil {
			return nil
		}

		return &pb.NATInfo{
			NatType:  string(natInfo.Type),
			CanPunch: natInfo.CanP2P,
		}
	}

	peerManager.StartSyncLoop(ctx, 30*time.Second, natInfoProvider)
	log.Println("     Heartbeat loop started (30s interval)")

	// Initial sync
	_, _, err = peerManager.SyncPeers(ctx, natInfoProvider())
	if err != nil {
		log.Printf("WARNING: Initial peer sync failed: %v", err)
	} else {
		log.Printf("     Synced with coordinator, %d peers found", peerManager.PeerCount())
	}

	// All components started
	fmt.Println()
	log.Println("Agent started successfully!")
	log.Println("Virtual IP:", nodeConfig.AssignedIP)
	log.Println("Peers:", peerManager.PeerCount())
	fmt.Println()
	log.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println()
	log.Println("Shutting down...")

	// Cleanup
	cancel()
	if punchCoordinator != nil {
		punchCoordinator.Stop()
	}
	if tunManager != nil {
		tunManager.Close()
	}
	time.Sleep(time.Second)

	log.Println("Agent stopped")
}