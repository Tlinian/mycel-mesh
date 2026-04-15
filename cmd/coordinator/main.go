package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "github.com/mycel/mesh/internal/coordinator/grpc"
	"github.com/mycel/mesh/internal/coordinator/metrics"
	"github.com/mycel/mesh/internal/coordinator/node"
	"github.com/mycel/mesh/internal/coordinator/relay"
	"github.com/mycel/mesh/internal/coordinator/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

var (
	version     = "1.0.0"
	httpAddr    = flag.String("http", ":8080", "HTTP server address")
	grpcAddr    = flag.String("grpc", ":51820", "gRPC server address")
	relayAddr   = flag.String("relay", ":51821", "Relay server address")
	initMode    = flag.Bool("init", false, "Initialize coordinator and generate tokens")
	configFile  = flag.String("config", "", "Configuration file path")
	showVersion = flag.Bool("version", false, "Show version")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Mycel Coordinator v%s\n", version)
		os.Exit(0)
	}

	if *initMode {
		initCoordinator()
		os.Exit(0)
	}

	startCoordinator()
}

// initCoordinator initializes the coordinator and generates admin token.
func initCoordinator() {
	fmt.Println("🔧 Initializing Mycel Coordinator...")
	fmt.Println()

	// Generate admin token
	token := generateAdminToken()

	fmt.Println("✅ Initialization complete!")
	fmt.Println()
	fmt.Println("📋 Admin Token (save this securely):")
	fmt.Println("   " + token)
	fmt.Println()
	fmt.Println("🚀 Start coordinator with:")
	fmt.Println("   ./coordinator.exe")
	fmt.Println()
	fmt.Println("📝 Join network with:")
	fmt.Println("   ./mycelctl.exe join --token <token>")
}

// generateAdminToken generates a simple admin token.
func generateAdminToken() string {
	// Simple token generation (in production, use proper JWT)
	return fmt.Sprintf("mycel-admin-%d", time.Now().UnixNano())
}

// startCoordinator starts the coordinator server.
func startCoordinator() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize services
	subnetService := service.NewSubnetService()
	_, _ = subnetService.GetOrCreateDefaultSubnet()

	// Initialize node registry
	nodeRegistry := node.NewNodeRegistry()

	// Initialize relay service
	relayService := relay.NewService(relay.Config{Port: 51821})
	if err := relayService.Start(); err != nil {
		log.Printf("Failed to start relay service: %v", err)
	} else {
		log.Printf("Relay service started on :51821")
		relayService.StartCleanupLoop(30*time.Second, 5*time.Minute)
	}

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Start background metrics collection
	go collectMetrics(ctx, metricsCollector, subnetService, nodeRegistry)

	// Start background stale node cleanup
	go cleanupStaleNodes(ctx, nodeRegistry)

	// Setup HTTP handlers
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","version":"%s"}`, version)
	})

	// API endpoints
	mux.HandleFunc("/api/v1/status", handleStatus(subnetService, nodeRegistry))
	mux.HandleFunc("/api/v1/nodes", handleNodes(nodeRegistry))
	mux.HandleFunc("/api/v1/subnets", handleSubnets(subnetService))
	mux.HandleFunc("/api/v1/token", handleToken())
	mux.HandleFunc("/api/v1/relay", handleRelay(relayService))

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Start HTTP server
	go func() {
		log.Printf("HTTP server listening on %s", *httpAddr)
		log.Printf("Metrics available at http://localhost%s/metrics", *httpAddr)
		log.Printf("Health check at http://localhost%s/health", *httpAddr)
		if err := http.ListenAndServe(*httpAddr, mux); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start gRPC server
	grpcServer, grpcLis := startGRPCServer(*grpcAddr, nodeRegistry, subnetService)
	if grpcServer != nil {
		go func() {
			log.Printf("gRPC server listening on %s", *grpcAddr)
			if err := grpcServer.Serve(grpcLis); err != nil {
				log.Printf("gRPC server error: %v", err)
			}
		}()
	}

	fmt.Println()
	fmt.Println("Mycel Coordinator v" + version + " started successfully!")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("   1. Get token:   curl http://localhost:8080/api/v1/token")
	fmt.Println("   2. Join network: ./mycelctl.exe join -t <token> -c localhost:51820")
	fmt.Println("   3. List nodes:   ./mycelctl.exe list -c localhost:51820")
	fmt.Println()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println()
	fmt.Println("Shutting down...")
	cancel()
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}
	if relayService != nil {
		relayService.Stop()
	}
	time.Sleep(time.Second)
	fmt.Println("Goodbye!")
}

// startGRPCServer creates and returns the gRPC server and listener.
func startGRPCServer(addr string, registry *node.NodeRegistry, subnetService *service.SubnetService) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to listen on %s: %v", addr, err)
		return nil, nil
	}

	grpcServer := grpc.NewServer()
	grpcserver.RegisterNodeServiceServer(grpcServer, registry, subnetService)

	return grpcServer, lis
}

// cleanupStaleNodes periodically marks nodes as offline if they haven't sent heartbeats.
func cleanupStaleNodes(ctx context.Context, registry *node.NodeRegistry) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	heartbeatTimeout := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stale := registry.CleanupStaleNodes(heartbeatTimeout)
			for _, nodeID := range stale {
				log.Printf("Node %s marked as offline (heartbeat timeout)", nodeID)
			}
		}
	}
}

// RegisterNodeServiceServer is a helper function to register the gRPC server.
func RegisterNodeServiceServer(s *grpc.Server, registry *node.NodeRegistry, subnetService *service.SubnetService) {
	grpcserver.RegisterNodeServiceServer(s, registry, subnetService)
}

// collectMetrics collects metrics in background.
func collectMetrics(ctx context.Context, collector *metrics.Collector, subnetService *service.SubnetService, registry *node.NodeRegistry) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			subnets := subnetService.ListSubnets()
			totalNodes := registry.GetNodeCount()
			onlineNodes := registry.GetOnlineCount()
			offlineNodes := totalNodes - onlineNodes

			for _, subnet := range subnets {
				stats := subnet.GetStats()
				collector.UpdateSubnetMetrics(subnet.ID, stats.AllocatedIPs, stats.AvailableIPs)
			}

			collector.UpdateNodeMetrics(totalNodes, onlineNodes, offlineNodes)
		}
	}
}

// HTTP Handlers

func handleStatus(subnetService *service.SubnetService, registry *node.NodeRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		subnets := subnetService.ListSubnets()
		totalNodes := registry.GetNodeCount()
		onlineNodes := registry.GetOnlineCount()
		fmt.Fprintf(w, `{"status":"running","version":"%s","nodes":%d,"online":%d,"subnets":%d}`, version, totalNodes, onlineNodes, len(subnets))
	}
}

func handleNodes(registry *node.NodeRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		nodes := registry.ListNodes()
		result := `{"nodes":[`
		for i, n := range nodes {
			if i > 0 {
				result += ","
			}
			status := n.Status
			if status == "" {
				status = "offline"
			}
			result += fmt.Sprintf(`{"id":"%s","name":"%s","ip":"%s","status":"%s"}`, n.ID, n.Name, n.AssignedIP, status)
		}
		result += `]}`
		fmt.Fprint(w, result)
	}
}

func handleSubnets(subnetService *service.SubnetService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		subnets := subnetService.ListSubnets()
		result := "["
		for i, s := range subnets {
			if i > 0 {
				result += ","
			}
			stats := s.GetStats()
			result += fmt.Sprintf(`{"id":"%s","name":"%s","cidr":"%s","allocated":%d}`, s.ID, s.Name, s.NetworkCIDR, stats.AllocatedIPs)
		}
		result += "]"
		fmt.Fprint(w, result)
	}
}

func handleToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := generateAdminToken()
		fmt.Fprintf(w, `{"token":"%s"}`, token)
	}
}

func handleRelay(relayService *relay.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if relayService == nil {
			fmt.Fprintf(w, `{"status":"disabled"}`)
			return
		}
		stats := relayService.GetStats()
		fmt.Fprintf(w, `{"status":"running","active_connections":%d,"bytes_sent":%d,"bytes_recv":%d}`,
			stats.ActiveConnections, stats.TotalBytesSent, stats.TotalBytesRecv)
	}
}
