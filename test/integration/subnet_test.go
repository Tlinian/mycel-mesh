package integration

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/mycel/mesh/internal/coordinator/service"
)

// TestSubnetCreation tests subnet creation.
func TestSubnetCreation(t *testing.T) {
	svc := service.NewSubnetService()

	config := service.SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
		Description: "Test subnet",
		Isolated:    false,
	}

	subnet, err := svc.CreateSubnet(context.Background(), "test-001", config)
	if err != nil {
		t.Fatalf("Failed to create subnet: %v", err)
	}

	if subnet.Name != "test-subnet" {
		t.Errorf("Expected name 'test-subnet', got '%s'", subnet.Name)
	}

	if subnet.NetworkCIDR != "10.0.1.0/24" {
		t.Errorf("Expected CIDR '10.0.1.0/24', got '%s'", subnet.NetworkCIDR)
	}

	t.Logf("Subnet created successfully: %s (%s)", subnet.Name, subnet.NetworkCIDR)
}

// TestSubnetIPAllocation tests IP allocation within a subnet.
func TestSubnetIPAllocation(t *testing.T) {
	svc := service.NewSubnetService()

	config := service.SubnetConfig{
		Name:        "alloc-test",
		NetworkCIDR: "10.0.2.0/24",
		Description: "IP allocation test",
		Isolated:    false,
	}

	subnet, err := svc.CreateSubnet(context.Background(), "alloc-001", config)
	if err != nil {
		t.Fatalf("Failed to create subnet: %v", err)
	}

	// Allocate IPs for 5 nodes
	for i := 1; i <= 5; i++ {
		ip, err := subnet.AllocateIP(fmt.Sprintf("node-%d", i))
		if err != nil {
			t.Fatalf("Failed to allocate IP for node-%d: %v", i, err)
		}

		expectedBase := net.IP{10, 0, 2, byte(i + 1)}
		if !ip.Equal(expectedBase) {
			t.Logf("Allocated IP: %s (expected: %s)", ip, expectedBase)
		}
	}

	stats := subnet.GetStats()
	if stats.AllocatedIPs != 5 {
		t.Errorf("Expected 5 allocated IPs, got %d", stats.AllocatedIPs)
	}

	t.Logf("IP allocation test passed: %d IPs allocated", stats.AllocatedIPs)
}

// TestSubnetIsolation tests isolated subnet behavior.
func TestSubnetIsolation(t *testing.T) {
	svc := service.NewSubnetService()

	// Create isolated subnet
	isolatedConfig := service.SubnetConfig{
		Name:        "isolated-subnet",
		NetworkCIDR: "10.0.3.0/24",
		Description: "Isolated subnet",
		Isolated:    true,
	}

	isolatedSubnet, err := svc.CreateSubnet(context.Background(), "isolated-001", isolatedConfig)
	if err != nil {
		t.Fatalf("Failed to create isolated subnet: %v", err)
	}

	// Create normal subnet
	normalConfig := service.SubnetConfig{
		Name:        "normal-subnet",
		NetworkCIDR: "10.0.4.0/24",
		Description: "Normal subnet",
		Isolated:    false,
	}

	normalSubnet, err := svc.CreateSubnet(context.Background(), "normal-001", normalConfig)
	if err != nil {
		t.Fatalf("Failed to create normal subnet: %v", err)
	}

	// Allocate IPs
	isolatedSubnet.AllocateIP("isolated-node-1")
	normalSubnet.AllocateIP("normal-node-1")

	t.Logf("Isolated subnet: %s (isolated=%v)", isolatedSubnet.Name, isolatedSubnet.Isolated)
	t.Logf("Normal subnet: %s (isolated=%v)", normalSubnet.Name, normalSubnet.Isolated)
}

// TestRoutingTable tests inter-subnet routing.
func TestRoutingTable(t *testing.T) {
	svc := service.NewSubnetService()
	rt := service.NewRoutingTable(svc)

	// Create subnets
	subnet1, _ := svc.CreateSubnet(context.Background(), "subnet-1", service.SubnetConfig{
		Name:        "subnet-1",
		NetworkCIDR: "10.0.10.0/24",
		Isolated:    false,
	})

	subnet2, _ := svc.CreateSubnet(context.Background(), "subnet-2", service.SubnetConfig{
		Name:        "subnet-2",
		NetworkCIDR: "10.0.20.0/24",
		Isolated:    false,
	})

	// Allocate gateway IPs
	gatewayIP1, _ := subnet1.AllocateIP("gateway")
	gatewayIP2, _ := subnet2.AllocateIP("gateway")

	// Add routes
	route1, err := rt.AddRoute(context.Background(), "route-1-to-2", "subnet-2", "subnet-1", gatewayIP2, 100)
	if err != nil {
		t.Fatalf("Failed to add route: %v", err)
	}

	route2, err := rt.AddRoute(context.Background(), "route-2-to-1", "subnet-1", "subnet-2", gatewayIP1, 100)
	if err != nil {
		t.Fatalf("Failed to add return route: %v", err)
	}

	t.Logf("Route 1: %s -> %s (metric: %d)", route1.SrcSubnetID, route1.DstSubnetID, route1.Metric)
	t.Logf("Route 2: %s -> %s (metric: %d)", route2.SrcSubnetID, route2.DstSubnetID, route2.Metric)

	// Verify routes
	routes := rt.ListRoutes()
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}
}

// TestRoutingService tests the routing service.
func TestRoutingService(t *testing.T) {
	svc := service.NewSubnetService()
	rt := service.NewRoutingTable(svc)
	rs := service.NewRoutingService(svc, rt)

	// Create subnets
	svc.CreateSubnet(context.Background(), "dev-subnet", service.SubnetConfig{
		Name:        "dev",
		NetworkCIDR: "10.1.0.0/24",
		Isolated:    false,
	})

	svc.CreateSubnet(context.Background(), "prod-subnet", service.SubnetConfig{
		Name:        "prod",
		NetworkCIDR: "10.2.0.0/24",
		Isolated:    false,
	})

	// Setup inter-subnet routing
	err := rs.SetupInterSubnetRouting()
	if err != nil {
		t.Logf("Warning: SetupInterSubnetRouting returned: %v", err)
	}

	// Test communication check
	result := rs.CanCommunicate("dev-node-1", "prod-node-1")
	t.Logf("Communication dev->prod: allowed=%v, reason=%s", result.Allowed, result.Reason)

	// Get stats
	stats := rs.GetRoutingStats()
	t.Logf("Routing stats: TotalSubnets=%d, TotalRoutes=%d, IsolatedSubnets=%d",
		stats.TotalSubnets, stats.TotalRoutes, stats.IsolatedSubnets)
}

// TestSubnetService_GetOrCreateDefaultSubnet tests default subnet creation.
func TestSubnetService_GetOrCreateDefaultSubnet(t *testing.T) {
	svc := service.NewSubnetService()

	// Get or create default subnet
	subnet, err := svc.GetOrCreateDefaultSubnet()
	if err != nil {
		t.Fatalf("Failed to get/create default subnet: %v", err)
	}

	if subnet.Name != "default" {
		t.Errorf("Expected default subnet name 'default', got '%s'", subnet.Name)
	}

	// Call again - should return existing
	subnet2, err := svc.GetOrCreateDefaultSubnet()
	if err != nil {
		t.Fatalf("Failed to get existing default subnet: %v", err)
	}

	if subnet.ID != subnet2.ID {
		t.Error("Expected same subnet instance")
	}

	t.Logf("Default subnet: %s (%s)", subnet.Name, subnet.NetworkCIDR)
}

// BenchmarkSubnetIPAllocation benchmarks IP allocation performance.
func BenchmarkSubnetIPAllocation(b *testing.B) {
	svc := service.NewSubnetService()

	config := service.SubnetConfig{
		Name:        "bench-subnet",
		NetworkCIDR: "10.10.0.0/16",
		Isolated:    false,
	}

	subnet, _ := svc.CreateSubnet(context.Background(), "bench-001", config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := fmt.Sprintf("bench-node-%d", i)
		subnet.AllocateIP(nodeID)
	}
}

// BenchmarkRoutingLookup benchmarks routing lookup performance.
func BenchmarkRoutingLookup(b *testing.B) {
	svc := service.NewSubnetService()
	rt := service.NewRoutingTable(svc)

	// Create multiple subnets and routes
	for i := 0; i < 10; i++ {
		svc.CreateSubnet(context.Background(),
			fmt.Sprintf("subnet-%d", i),
			service.SubnetConfig{
				Name:        fmt.Sprintf("subnet-%d", i),
				NetworkCIDR: fmt.Sprintf("10.%d.0.0/24", i),
				Isolated:    false,
			})

		rt.AddRoute(context.Background(),
			fmt.Sprintf("route-%d", i),
			fmt.Sprintf("subnet-%d", i),
			"default",
			net.IP{10, 0, 0, 1},
			100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.GetRouteForDestination(fmt.Sprintf("subnet-%d", i%10))
	}
}
