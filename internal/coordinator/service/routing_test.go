package service

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestRoutingTable_New tests routing table creation.
func TestRoutingTable_New(t *testing.T) {
	subnetSvc := NewSubnetService()
	rt := NewRoutingTable(subnetSvc)

	if rt == nil {
		t.Fatal("routing table should not be nil")
	}
	if len(rt.routes) != 0 {
		t.Fatal("new routing table should have no routes")
	}
}

// TestRoutingTable_AddRoute_Success tests successful route addition.
func TestRoutingTable_AddRoute_Success(t *testing.T) {
	subnetSvc := NewSubnetService()

	// Create destination subnet first
	config := SubnetConfig{Name: "dest-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "dest-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)

	gatewayIP := net.ParseIP("10.0.2.1")
	route, err := rt.AddRoute(context.Background(), "route-001", "dest-001", "src-001", gatewayIP, 100)
	if err != nil {
		t.Fatalf("AddRoute() failed: %v", err)
	}

	if route.ID != "route-001" {
		t.Fatalf("expected route ID 'route-001', got '%s'", route.ID)
	}
	if route.DstSubnetID != "dest-001" {
		t.Fatalf("expected DstSubnetID 'dest-001', got '%s'", route.DstSubnetID)
	}
	if route.Metric != 100 {
		t.Fatalf("expected Metric 100, got %d", route.Metric)
	}
	if !route.GatewayIP.Equal(gatewayIP) {
		t.Fatalf("expected GatewayIP '%s', got '%s'", gatewayIP.String(), route.GatewayIP.String())
	}
}

// TestRoutingTable_AddRoute_SubnetNotFound tests error for non-existent subnet.
func TestRoutingTable_AddRoute_SubnetNotFound(t *testing.T) {
	subnetSvc := NewSubnetService()
	rt := NewRoutingTable(subnetSvc)

	gatewayIP := net.ParseIP("10.0.0.1")
	_, err := rt.AddRoute(context.Background(), "route-001", "nonexistent", "src-001", gatewayIP, 100)
	if err == nil {
		t.Fatal("expected error for non-existent subnet")
	}
}

// TestRoutingTable_RemoveRoute_Success tests successful route removal.
func TestRoutingTable_RemoveRoute_Success(t *testing.T) {
	subnetSvc := NewSubnetService()

	config := SubnetConfig{Name: "dest-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "dest-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)
	gatewayIP := net.ParseIP("10.0.2.1")
	_, err = rt.AddRoute(context.Background(), "route-001", "dest-001", "src-001", gatewayIP, 100)
	if err != nil {
		t.Fatalf("AddRoute() failed: %v", err)
	}

	err = rt.RemoveRoute("route-001")
	if err != nil {
		t.Fatalf("RemoveRoute() failed: %v", err)
	}

	routes := rt.ListRoutes()
	if len(routes) != 0 {
		t.Fatalf("expected 0 routes after removal, got %d", len(routes))
	}
}

// TestRoutingTable_RemoveRoute_NotFound tests error for non-existent route.
func TestRoutingTable_RemoveRoute_NotFound(t *testing.T) {
	subnetSvc := NewSubnetService()
	rt := NewRoutingTable(subnetSvc)

	err := rt.RemoveRoute("nonexistent")
	if err != ErrSubnetNotFound {
		t.Fatalf("expected ErrSubnetNotFound, got %v", err)
	}
}

// TestRoutingTable_ListRoutes tests listing all routes.
func TestRoutingTable_ListRoutes(t *testing.T) {
	subnetSvc := NewSubnetService()

	config := SubnetConfig{Name: "dest-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "dest-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)

	// Add multiple routes
	for i := 1; i <= 3; i++ {
		routeID := "route-" + string(rune('0'+i))
		gatewayIP := net.ParseIP("10.0.2." + string(rune('0'+i)))
		_, err := rt.AddRoute(context.Background(), routeID, "dest-001", "src-001", gatewayIP, i*10)
		if err != nil {
			t.Fatalf("AddRoute() failed: %v", err)
		}
	}

	routes := rt.ListRoutes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
}

// TestRoutingTable_GetRouteForDestination tests finding route by destination.
func TestRoutingTable_GetRouteForDestination(t *testing.T) {
	subnetSvc := NewSubnetService()

	config := SubnetConfig{Name: "dest-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "dest-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)

	gatewayIP := net.ParseIP("10.0.2.1")
	_, err = rt.AddRoute(context.Background(), "route-001", "dest-001", "src-001", gatewayIP, 100)
	if err != nil {
		t.Fatalf("AddRoute() failed: %v", err)
	}

	// Add route with lower metric
	gatewayIP2 := net.ParseIP("10.0.2.2")
	_, err = rt.AddRoute(context.Background(), "route-002", "dest-001", "src-001", gatewayIP2, 50)
	if err != nil {
		t.Fatalf("AddRoute() failed: %v", err)
	}

	// Should return route with lowest metric
	route := rt.GetRouteForDestination("dest-001")
	if route == nil {
		t.Fatal("route should be found")
	}
	if route.Metric != 50 {
		t.Fatalf("expected route with lowest metric 50, got %d", route.Metric)
	}
}

// TestRoutingTable_GetRouteForDestination_NoRoute tests when no route exists.
func TestRoutingTable_GetRouteForDestination_NoRoute(t *testing.T) {
	subnetSvc := NewSubnetService()
	rt := NewRoutingTable(subnetSvc)

	route := rt.GetRouteForDestination("nonexistent")
	if route != nil {
		t.Fatal("should return nil for non-existent destination")
	}
}

// TestRoutingTable_GetRouteForIP tests finding route by IP.
func TestRoutingTable_GetRouteForIP(t *testing.T) {
	subnetSvc := NewSubnetService()

	config := SubnetConfig{Name: "dest-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "dest-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)

	gatewayIP := net.ParseIP("10.0.2.1")
	_, err = rt.AddRoute(context.Background(), "route-001", "dest-001", "src-001", gatewayIP, 100)
	if err != nil {
		t.Fatalf("AddRoute() failed: %v", err)
	}

	// IP within destination subnet
	dstIP := net.ParseIP("10.0.2.50")
	route := rt.GetRouteForIP(dstIP)
	if route == nil {
		t.Fatal("route should be found for IP in destination subnet")
	}

	// IP outside destination subnet
	dstIP2 := net.ParseIP("192.168.1.1")
	route2 := rt.GetRouteForIP(dstIP2)
	if route2 != nil {
		t.Fatal("should return nil for IP outside route destinations")
	}
}

// TestRoutingService_New tests routing service creation.
func TestRoutingService_New(t *testing.T) {
	subnetSvc := NewSubnetService()
	rt := NewRoutingTable(subnetSvc)
	rs := NewRoutingService(subnetSvc, rt)

	if rs == nil {
		t.Fatal("routing service should not be nil")
	}
}

// TestRoutingService_CanCommunicate_SameSubnet tests same subnet communication.
func TestRoutingService_CanCommunicate_SameSubnet(t *testing.T) {
	subnetSvc := NewSubnetService()

	config := SubnetConfig{Name: "test-subnet", NetworkCIDR: "10.0.1.0/24"}
	subnet, err := subnetSvc.CreateSubnet(context.Background(), "subnet-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	// Allocate IPs for nodes
	_, err = subnet.AllocateIP("node-1")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}
	_, err = subnet.AllocateIP("node-2")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)
	rs := NewRoutingService(subnetSvc, rt)

	result := rs.CanCommunicate("node-1", "node-2")
	if !result.Allowed {
		t.Fatalf("same subnet should be allowed: %s", result.Reason)
	}
}

// TestRoutingService_CanCommunicate_IsolatedSubnet tests isolated subnet blocking.
func TestRoutingService_CanCommunicate_IsolatedSubnet(t *testing.T) {
	subnetSvc := NewSubnetService()

	// Create isolated subnet
	config := SubnetConfig{Name: "isolated-subnet", NetworkCIDR: "10.0.1.0/24", Isolated: true}
	subnet1, err := subnetSvc.CreateSubnet(context.Background(), "isolated-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	// Create non-isolated subnet for comparison
	config2 := SubnetConfig{Name: "normal-subnet", NetworkCIDR: "10.0.2.0/24", Isolated: false}
	subnet2, err := subnetSvc.CreateSubnet(context.Background(), "normal-001", config2)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	// Allocate IPs
	_, err = subnet1.AllocateIP("node-1")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}
	_, err = subnet2.AllocateIP("node-2")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)
	rs := NewRoutingService(subnetSvc, rt)

	// Node in isolated subnet trying to communicate with node in normal subnet
	result := rs.CanCommunicate("node-1", "node-2")
	if result.Allowed {
		t.Fatal("isolated subnet should block external communication")
	}
}

// TestRoutingService_GetRoutePath tests getting route path.
func TestRoutingService_GetRoutePath(t *testing.T) {
	subnetSvc := NewSubnetService()

	config := SubnetConfig{Name: "dest-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "dest-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)
	gatewayIP := net.ParseIP("10.0.2.1")
	_, err = rt.AddRoute(context.Background(), "route-001", "dest-001", "src-001", gatewayIP, 100)
	if err != nil {
		t.Fatalf("AddRoute() failed: %v", err)
	}

	rs := NewRoutingService(subnetSvc, rt)

	path := rs.GetRoutePath("src-001", "dest-001")
	if len(path) != 1 {
		t.Fatalf("expected 1 route in path, got %d", len(path))
	}
}

// TestRoutingService_CreateDefaultRoute tests default route creation.
func TestRoutingService_CreateDefaultRoute(t *testing.T) {
	subnetSvc := NewSubnetService()

	srcConfig := SubnetConfig{Name: "src-subnet", NetworkCIDR: "10.0.1.0/24"}
	_, err := subnetSvc.CreateSubnet(context.Background(), "src-001", srcConfig)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	dstConfig := SubnetConfig{Name: "dst-subnet", NetworkCIDR: "10.0.2.0/24"}
	_, err = subnetSvc.CreateSubnet(context.Background(), "dst-001", dstConfig)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)
	rs := NewRoutingService(subnetSvc, rt)

	route, err := rs.CreateDefaultRoute("src-001", "dst-001")
	if err != nil {
		t.Fatalf("CreateDefaultRoute() failed: %v", err)
	}

	if route.DstSubnetID != "dst-001" {
		t.Fatalf("expected DstSubnetID 'dst-001', got '%s'", route.DstSubnetID)
	}
}

// TestRoutingService_SetupInterSubnetRouting tests automatic routing setup.
func TestRoutingService_SetupInterSubnetRouting(t *testing.T) {
	subnetSvc := NewSubnetService()

	// Create non-isolated subnets
	for i := 1; i <= 3; i++ {
		config := SubnetConfig{
			Name:        "subnet-" + string(rune('0'+i)),
			NetworkCIDR: "10.0." + string(rune('0'+i)) + ".0/24",
			Isolated:    false,
		}
		_, err := subnetSvc.CreateSubnet(context.Background(), "id-"+string(rune('0'+i)), config)
		if err != nil {
			t.Fatalf("CreateSubnet() failed: %v", err)
		}
	}

	// Create isolated subnet
	isolatedConfig := SubnetConfig{Name: "isolated", NetworkCIDR: "10.0.99.0/24", Isolated: true}
	_, err := subnetSvc.CreateSubnet(context.Background(), "isolated-001", isolatedConfig)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	rt := NewRoutingTable(subnetSvc)
	rs := NewRoutingService(subnetSvc, rt)

	err = rs.SetupInterSubnetRouting()
	if err != nil {
		t.Fatalf("SetupInterSubnetRouting() failed: %v", err)
	}

	// Verify routes were created for non-isolated subnets
	stats := rs.GetRoutingStats()
	if stats.TotalSubnets != 4 {
		t.Fatalf("expected 4 total subnets, got %d", stats.TotalSubnets)
	}
	if stats.IsolatedSubnets != 1 {
		t.Fatalf("expected 1 isolated subnet, got %d", stats.IsolatedSubnets)
	}
}

// TestRoutingService_GetRoutingStats tests statistics retrieval.
func TestRoutingService_GetRoutingStats(t *testing.T) {
	subnetSvc := NewSubnetService()
	rt := NewRoutingTable(subnetSvc)
	rs := NewRoutingService(subnetSvc, rt)

	stats := rs.GetRoutingStats()
	if stats.TotalSubnets != 0 {
		t.Fatalf("expected 0 subnets initially, got %d", stats.TotalSubnets)
	}
}

// TestRoute_Struct tests route struct fields.
func TestRoute_Struct(t *testing.T) {
	route := &Route{
		ID:          "test-route",
		DstSubnetID: "dest",
		SrcSubnetID: "src",
		GatewayIP:   net.ParseIP("10.0.0.1"),
		Metric:      100,
		CreatedAt:   time.Now(),
	}

	if route.ID != "test-route" {
		t.Fatalf("expected ID 'test-route', got '%s'", route.ID)
	}
}

// TestACLCheckResult tests ACL check result.
func TestACLCheckResult(t *testing.T) {
	result := ACLCheckResult{Allowed: true, Reason: "test reason"}

	if !result.Allowed {
		t.Fatal("Allowed should be true")
	}
	if result.Reason != "test reason" {
		t.Fatalf("expected Reason 'test reason', got '%s'", result.Reason)
	}
}