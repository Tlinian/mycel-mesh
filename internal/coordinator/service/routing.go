// Package service provides inter-subnet routing for Coordinator.
package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Route represents a route between subnets.
type Route struct {
	ID          string
	DstSubnetID string
	DstCIDR     string
	SrcSubnetID string
	GatewayIP   net.IP
	Metric      int
	CreatedAt   time.Time
	mu          sync.RWMutex
}

// RoutingTable manages routes between subnets.
type RoutingTable struct {
	routes    []*Route
	subnetSvc *SubnetService
	mu        sync.RWMutex
}

// NewRoutingTable creates a new routing table.
func NewRoutingTable(subnetSvc *SubnetService) *RoutingTable {
	return &RoutingTable{
		routes:    make([]*Route, 0),
		subnetSvc: subnetSvc,
	}
}

// AddRoute adds a route to the routing table.
func (rt *RoutingTable) AddRoute(ctx context.Context, id, dstSubnetID, srcSubnetID string, gatewayIP net.IP, metric int) (*Route, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Get destination subnet to determine CIDR
	dstSubnet, err := rt.subnetSvc.GetSubnet(dstSubnetID)
	if err != nil {
		return nil, fmt.Errorf("destination subnet not found: %w", err)
	}

	route := &Route{
		ID:          id,
		DstSubnetID: dstSubnetID,
		DstCIDR:     dstSubnet.NetworkCIDR,
		SrcSubnetID: srcSubnetID,
		GatewayIP:   gatewayIP,
		Metric:      metric,
		CreatedAt:   time.Now(),
	}

	rt.routes = append(rt.routes, route)
	return route, nil
}

// RemoveRoute removes a route by ID.
func (rt *RoutingTable) RemoveRoute(routeID string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for i, route := range rt.routes {
		if route.ID == routeID {
			rt.routes = append(rt.routes[:i], rt.routes[i+1:]...)
			return nil
		}
	}
	return ErrSubnetNotFound
}

// ListRoutes returns all routes.
func (rt *RoutingTable) ListRoutes() []*Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	result := make([]*Route, len(rt.routes))
	copy(result, rt.routes)
	return result
}

// GetRouteForDestination finds the best route for a destination subnet.
func (rt *RoutingTable) GetRouteForDestination(dstSubnetID string) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var bestRoute *Route
	for _, route := range rt.routes {
		if route.DstSubnetID == dstSubnetID {
			if bestRoute == nil || route.Metric < bestRoute.Metric {
				bestRoute = route
			}
		}
	}
	return bestRoute
}

// GetRouteForIP finds the best route for a destination IP.
func (rt *RoutingTable) GetRouteForIP(dstIP net.IP) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var bestRoute *Route
	for _, route := range rt.routes {
		_, network, err := net.ParseCIDR(route.DstCIDR)
		if err != nil {
			continue
		}

		if network.Contains(dstIP) {
			if bestRoute == nil || route.Metric < bestRoute.Metric {
				bestRoute = route
			}
		}
	}
	return bestRoute
}

// InterSubnetTraffic represents traffic between subnets.
type InterSubnetTraffic struct {
	SrcSubnetID string
	DstSubnetID string
	SrcNodeID   string
	DstNodeID   string
	Protocol    string
	Port        int
	Allowed     bool
}

// ACLCheckResult represents the result of an ACL check.
type ACLCheckResult struct {
	Allowed bool
	Reason  string
}

// RoutingService handles inter-subnet routing decisions.
type RoutingService struct {
	routingTable *RoutingTable
	subnetSvc    *SubnetService
	aclSvc       interface{} // ACL service interface
	mu           sync.RWMutex
}

// NewRoutingService creates a new routing service.
func NewRoutingService(subnetSvc *SubnetService, routingTable *RoutingTable) *RoutingService {
	return &RoutingService{
		subnetSvc:    subnetSvc,
		routingTable: routingTable,
	}
}

// CanCommunicate checks if two nodes in different subnets can communicate.
func (rs *RoutingService) CanCommunicate(srcNodeID, dstNodeID string) ACLCheckResult {
	// Find source node's subnet
	var srcSubnet *Subnet
	var dstSubnet *Subnet

	for _, subnet := range rs.subnetSvc.ListSubnets() {
		ips := subnet.GetAllocatedIPs()
		for _, nodeID := range ips {
			if nodeID == srcNodeID {
				srcSubnet = subnet
			}
			if nodeID == dstNodeID {
				dstSubnet = subnet
			}
		}
	}

	// Same subnet - direct communication
	if srcSubnet != nil && srcSubnet.ID == dstSubnet.ID {
		return ACLCheckResult{Allowed: true, Reason: "same subnet"}
	}

	// Different subnets - check routing
	if srcSubnet == nil || dstSubnet == nil {
		return ACLCheckResult{Allowed: false, Reason: "node not found"}
	}

	// Check if subnets are isolated
	if srcSubnet.Isolated || dstSubnet.Isolated {
		return ACLCheckResult{Allowed: false, Reason: "subnet is isolated"}
	}

	// Check if route exists
	route := rs.routingTable.GetRouteForDestination(dstSubnet.ID)
	if route == nil {
		return ACLCheckResult{Allowed: false, Reason: "no route to destination subnet"}
	}

	return ACLCheckResult{Allowed: true, Reason: "route exists"}
}

// GetRoutePath returns the route path between two subnets.
func (rs *RoutingService) GetRoutePath(srcSubnetID, dstSubnetID string) []*Route {
	var path []*Route

	route := rs.routingTable.GetRouteForDestination(dstSubnetID)
	if route != nil {
		path = append(path, route)
	}

	return path
}

// CreateDefaultRoute creates a default route between two subnets.
func (rs *RoutingService) CreateDefaultRoute(srcSubnetID, dstSubnetID string) (*Route, error) {
	_, err := rs.subnetSvc.GetSubnet(srcSubnetID)
	if err != nil {
		return nil, err
	}

	dstSubnet, err := rs.subnetSvc.GetSubnet(dstSubnetID)
	if err != nil {
		return nil, err
	}

	// Use first available IP in destination subnet as gateway
	gatewayIP, err := dstSubnet.AllocateIP("gateway")
	if err != nil {
		// If allocation fails, use network address + 1
		baseIP := dstSubnet.Network.IP.To4()
		gatewayIP = net.IP{baseIP[0], baseIP[1], baseIP[2], baseIP[3] + 1}
	}

	return rs.routingTable.AddRoute(
		context.Background(),
		fmt.Sprintf("route-%s-to-%s", srcSubnetID, dstSubnetID),
		dstSubnetID,
		srcSubnetID,
		gatewayIP,
		100, // Default metric
	)
}

// SetupInterSubnetRouting sets up routing between all non-isolated subnets.
func (rs *RoutingService) SetupInterSubnetRouting() error {
	subnets := rs.subnetSvc.ListSubnets()

	for i, src := range subnets {
		if src.Isolated {
			continue
		}

		for j, dst := range subnets {
			if i == j || dst.Isolated {
				continue
			}

			// Check if route already exists
			existing := rs.routingTable.GetRouteForDestination(dst.ID)
			if existing != nil {
				continue
			}

			_, err := rs.CreateDefaultRoute(src.ID, dst.ID)
			if err != nil {
				// Continue with other routes
				continue
			}
		}
	}

	return nil
}

// GetRoutingStats returns routing statistics.
func (rs *RoutingService) GetRoutingStats() RoutingStats {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	routes := rs.routingTable.ListRoutes()
	subnets := rs.subnetSvc.ListSubnets()

	var isolatedCount int
	for _, subnet := range subnets {
		if subnet.Isolated {
			isolatedCount++
		}
	}

	return RoutingStats{
		TotalSubnets:    len(subnets),
		IsolatedSubnets: isolatedCount,
		TotalRoutes:     len(routes),
	}
}

// RoutingStats holds routing statistics.
type RoutingStats struct {
	TotalSubnets    int
	IsolatedSubnets int
	TotalRoutes     int
}
