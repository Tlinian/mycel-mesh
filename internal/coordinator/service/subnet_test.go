package service

import (
	"context"
	"net"
	"testing"
)

// TestNewSubnet_Success tests successful subnet creation.
func TestNewSubnet_Success(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
		Description: "Test subnet",
		Isolated:    false,
	}

	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	if subnet.ID != "test-001" {
		t.Fatalf("expected ID 'test-001', got '%s'", subnet.ID)
	}
	if subnet.Name != "test-subnet" {
		t.Fatalf("expected Name 'test-subnet', got '%s'", subnet.Name)
	}
	if subnet.NetworkCIDR != "10.0.1.0/24" {
		t.Fatalf("expected NetworkCIDR '10.0.1.0/24', got '%s'", subnet.NetworkCIDR)
	}
	if subnet.Isolated {
		t.Fatal("expected Isolated to be false")
	}
}

// TestNewSubnet_InvalidCIDR tests error for invalid CIDR.
func TestNewSubnet_InvalidCIDR(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "invalid-cidr",
	}

	_, err := NewSubnet("test-001", config)
	if err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}

// TestSubnet_AllocateIP_Success tests successful IP allocation.
func TestSubnet_AllocateIP_Success(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	ip, err := subnet.AllocateIP("node-1")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}

	// Verify IP is in subnet range
	if !subnet.Network.Contains(ip) {
		t.Fatalf("allocated IP %s not in subnet range", ip.String())
	}

	// Verify IP is tracked
	allocated := subnet.GetAllocatedIPs()
	if allocated[ip.String()] != "node-1" {
		t.Fatalf("allocated IP not tracked for node-1")
	}
}

// TestSubnet_AllocateIP_Multiple tests multiple IP allocations.
func TestSubnet_AllocateIP_Multiple(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	allocatedIPs := make(map[string]string)
	for i := 1; i <= 10; i++ {
		nodeID := "node-" + string(rune('0'+i))
		ip, err := subnet.AllocateIP(nodeID)
		if err != nil {
			t.Fatalf("AllocateIP() failed for node %d: %v", i, err)
		}
		allocatedIPs[ip.String()] = nodeID
	}

	// Verify all IPs are different
	if len(allocatedIPs) != 10 {
		t.Fatalf("expected 10 unique IPs, got %d", len(allocatedIPs))
	}
}

// TestSubnet_AllocateIP_NoAvailable tests error when no IPs available.
func TestSubnet_AllocateIP_NoAvailable(t *testing.T) {
	// Use a very small subnet (/30 has only 2 usable IPs)
	config := SubnetConfig{
		Name:        "tiny-subnet",
		NetworkCIDR: "10.0.1.0/30",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	// Allocate the 2 available IPs
	_, err = subnet.AllocateIP("node-1")
	if err != nil {
		t.Fatalf("first AllocateIP() failed: %v", err)
	}
	_, err = subnet.AllocateIP("node-2")
	if err != nil {
		t.Fatalf("second AllocateIP() failed: %v", err)
	}

	// Next allocation should fail
	_, err = subnet.AllocateIP("node-3")
	if err != ErrNoAvailableIP {
		t.Fatalf("expected ErrNoAvailableIP, got %v", err)
	}
}

// TestSubnet_AllocateSpecificIP_Success tests specific IP allocation.
func TestSubnet_AllocateSpecificIP_Success(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	requestedIP := net.ParseIP("10.0.1.100")
	err = subnet.AllocateSpecificIP("node-1", requestedIP)
	if err != nil {
		t.Fatalf("AllocateSpecificIP() failed: %v", err)
	}

	// Verify IP is tracked
	nodeIP, found := subnet.GetNodeIP("node-1")
	if !found {
		t.Fatal("allocated IP not found for node-1")
	}
	if !nodeIP.Equal(requestedIP) {
		t.Fatalf("expected IP %s, got %s", requestedIP.String(), nodeIP.String())
	}
}

// TestSubnet_AllocateSpecificIP_NotInSubnet tests error for IP outside subnet.
func TestSubnet_AllocateSpecificIP_NotInSubnet(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	requestedIP := net.ParseIP("192.168.1.1") // Outside subnet
	err = subnet.AllocateSpecificIP("node-1", requestedIP)
	if err != ErrIPNotInSubnet {
		t.Fatalf("expected ErrIPNotInSubnet, got %v", err)
	}
}

// TestSubnet_AllocateSpecificIP_AlreadyAllocated tests error for duplicate allocation.
func TestSubnet_AllocateSpecificIP_AlreadyAllocated(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	requestedIP := net.ParseIP("10.0.1.100")
	err = subnet.AllocateSpecificIP("node-1", requestedIP)
	if err != nil {
		t.Fatalf("first AllocateSpecificIP() failed: %v", err)
	}

	// Try to allocate same IP again
	err = subnet.AllocateSpecificIP("node-2", requestedIP)
	if err != ErrSubnetExists {
		t.Fatalf("expected ErrSubnetExists, got %v", err)
	}
}

// TestSubnet_ReleaseIP_Success tests successful IP release.
func TestSubnet_ReleaseIP_Success(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	ip, err := subnet.AllocateIP("node-1")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}

	err = subnet.ReleaseIP(ip)
	if err != nil {
		t.Fatalf("ReleaseIP() failed: %v", err)
	}

	// Verify IP is released
	allocated := subnet.GetAllocatedIPs()
	if _, exists := allocated[ip.String()]; exists {
		t.Fatal("IP should be released")
	}
}

// TestSubnet_ReleaseIP_NotAllocated tests error for releasing unallocated IP.
func TestSubnet_ReleaseIP_NotAllocated(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	ip := net.ParseIP("10.0.1.100")
	err = subnet.ReleaseIP(ip)
	if err != ErrSubnetNotFound {
		t.Fatalf("expected ErrSubnetNotFound, got %v", err)
	}
}

// TestSubnet_GetStats tests subnet statistics.
func TestSubnet_GetStats(t *testing.T) {
	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}
	subnet, err := NewSubnet("test-001", config)
	if err != nil {
		t.Fatalf("NewSubnet() failed: %v", err)
	}

	// Initial stats
	stats := subnet.GetStats()
	if stats.TotalIPs != 256 {
		t.Fatalf("expected TotalIPs 256, got %d", stats.TotalIPs)
	}
	if stats.AllocatedIPs != 0 {
		t.Fatalf("expected AllocatedIPs 0, got %d", stats.AllocatedIPs)
	}

	// Allocate 5 IPs
	for i := 1; i <= 5; i++ {
		_, err := subnet.AllocateIP("node-" + string(rune('0'+i)))
		if err != nil {
			t.Fatalf("AllocateIP() failed: %v", err)
		}
	}

	stats = subnet.GetStats()
	if stats.AllocatedIPs != 5 {
		t.Fatalf("expected AllocatedIPs 5, got %d", stats.AllocatedIPs)
	}
	if stats.AvailableIPs != 249 { // 256 - 5 - 2 (network/broadcast)
		t.Fatalf("expected AvailableIPs 249, got %d", stats.AvailableIPs)
	}
}

// TestSubnetService_CreateSubnet_Success tests successful subnet creation.
func TestSubnetService_CreateSubnet_Success(t *testing.T) {
	service := NewSubnetService()

	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}

	subnet, err := service.CreateSubnet(context.Background(), "subnet-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	if subnet.ID != "subnet-001" {
		t.Fatalf("expected ID 'subnet-001', got '%s'", subnet.ID)
	}
}

// TestSubnetService_CreateSubnet_DuplicateName tests error for duplicate name.
func TestSubnetService_CreateSubnet_DuplicateName(t *testing.T) {
	service := NewSubnetService()

	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}

	_, err := service.CreateSubnet(context.Background(), "subnet-001", config)
	if err != nil {
		t.Fatalf("first CreateSubnet() failed: %v", err)
	}

	// Create subnet with same name
	_, err = service.CreateSubnet(context.Background(), "subnet-002", config)
	if err != ErrSubnetExists {
		t.Fatalf("expected ErrSubnetExists, got %v", err)
	}
}

// TestSubnetService_GetSubnet_Success tests successful subnet retrieval.
func TestSubnetService_GetSubnet_Success(t *testing.T) {
	service := NewSubnetService()

	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}

	_, err := service.CreateSubnet(context.Background(), "subnet-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	subnet, err := service.GetSubnet("subnet-001")
	if err != nil {
		t.Fatalf("GetSubnet() failed: %v", err)
	}

	if subnet.Name != "test-subnet" {
		t.Fatalf("expected Name 'test-subnet', got '%s'", subnet.Name)
	}
}

// TestSubnetService_GetSubnet_NotFound tests error for non-existent subnet.
func TestSubnetService_GetSubnet_NotFound(t *testing.T) {
	service := NewSubnetService()

	_, err := service.GetSubnet("non-existent")
	if err != ErrSubnetNotFound {
		t.Fatalf("expected ErrSubnetNotFound, got %v", err)
	}
}

// TestSubnetService_ListSubnets tests listing all subnets.
func TestSubnetService_ListSubnets(t *testing.T) {
	service := NewSubnetService()

	// Create multiple subnets
	for i := 1; i <= 3; i++ {
		config := SubnetConfig{
			Name:        "subnet-" + string(rune('0'+i)),
			NetworkCIDR: "10.0." + string(rune('0'+i)) + ".0/24",
		}
		_, err := service.CreateSubnet(context.Background(), "id-"+string(rune('0'+i)), config)
		if err != nil {
			t.Fatalf("CreateSubnet() failed: %v", err)
		}
	}

	subnets := service.ListSubnets()
	if len(subnets) != 3 {
		t.Fatalf("expected 3 subnets, got %d", len(subnets))
	}
}

// TestSubnetService_DeleteSubnet_Success tests successful subnet deletion.
func TestSubnetService_DeleteSubnet_Success(t *testing.T) {
	service := NewSubnetService()

	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}

	_, err := service.CreateSubnet(context.Background(), "subnet-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	err = service.DeleteSubnet("subnet-001")
	if err != nil {
		t.Fatalf("DeleteSubnet() failed: %v", err)
	}

	// Verify subnet is deleted
	_, err = service.GetSubnet("subnet-001")
	if err != ErrSubnetNotFound {
		t.Fatal("subnet should be deleted")
	}
}

// TestSubnetService_DeleteSubnet_NotEmpty tests error for non-empty subnet.
func TestSubnetService_DeleteSubnet_NotEmpty(t *testing.T) {
	service := NewSubnetService()

	config := SubnetConfig{
		Name:        "test-subnet",
		NetworkCIDR: "10.0.1.0/24",
	}

	subnet, err := service.CreateSubnet(context.Background(), "subnet-001", config)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	// Allocate an IP
	_, err = subnet.AllocateIP("node-1")
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}

	// Try to delete non-empty subnet
	err = service.DeleteSubnet("subnet-001")
	if err != ErrSubnetNotEmpty {
		t.Fatalf("expected ErrSubnetNotEmpty, got %v", err)
	}
}

// TestSubnetService_FindSubnetByIP tests finding subnet by IP.
func TestSubnetService_FindSubnetByIP(t *testing.T) {
	service := NewSubnetService()

	// Create subnets
	config1 := SubnetConfig{Name: "subnet-1", NetworkCIDR: "10.0.1.0/24"}
	config2 := SubnetConfig{Name: "subnet-2", NetworkCIDR: "10.0.2.0/24"}

	_, err := service.CreateSubnet(context.Background(), "id-1", config1)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}
	_, err = service.CreateSubnet(context.Background(), "id-2", config2)
	if err != nil {
		t.Fatalf("CreateSubnet() failed: %v", err)
	}

	// Find IP in subnet-1
	ip := net.ParseIP("10.0.1.100")
	found := service.FindSubnetByIP(ip)
	if found == nil {
		t.Fatal("subnet not found for IP")
	}
	if found.Name != "subnet-1" {
		t.Fatalf("expected subnet-1, got '%s'", found.Name)
	}

	// Find IP in subnet-2
	ip2 := net.ParseIP("10.0.2.50")
	found2 := service.FindSubnetByIP(ip2)
	if found2 == nil {
		t.Fatal("subnet not found for IP")
	}
	if found2.Name != "subnet-2" {
		t.Fatalf("expected subnet-2, got '%s'", found2.Name)
	}

	// IP not in any subnet
	ip3 := net.ParseIP("192.168.1.1")
	found3 := service.FindSubnetByIP(ip3)
	if found3 != nil {
		t.Fatal("should not find subnet for IP outside range")
	}
}

// TestSubnetService_GetOrCreateDefaultSubnet tests default subnet creation.
func TestSubnetService_GetOrCreateDefaultSubnet(t *testing.T) {
	service := NewSubnetService()

	subnet, err := service.GetOrCreateDefaultSubnet()
	if err != nil {
		t.Fatalf("GetOrCreateDefaultSubnet() failed: %v", err)
	}

	if subnet.Name != "default" {
		t.Fatalf("expected Name 'default', got '%s'", subnet.Name)
	}
	if subnet.NetworkCIDR != "10.0.0.0/16" {
		t.Fatalf("expected NetworkCIDR '10.0.0.0/16', got '%s'", subnet.NetworkCIDR)
	}

	// Call again - should return existing
	subnet2, err := service.GetOrCreateDefaultSubnet()
	if err != nil {
		t.Fatalf("second GetOrCreateDefaultSubnet() failed: %v", err)
	}

	if subnet.ID != subnet2.ID {
		t.Fatal("should return same subnet instance")
	}
}

// TestIPConversion tests IP to uint32 conversion.
func TestIPConversion(t *testing.T) {
	tests := []struct {
		ip       string
		expected uint32
	}{
		{"10.0.0.1", 0x0A000001},
		{"192.168.1.1", 0xC0A80101},
		{"0.0.0.0", 0x00000000},
		{"255.255.255.255", 0xFFFFFFFF},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		result := ipToUint32(ip)
		if result != tt.expected {
			t.Fatalf("ipToUint32(%s) expected %08X, got %08X", tt.ip, tt.expected, result)
		}

		// Reverse conversion
		reversed := uint32ToIP(result)
		if !reversed.Equal(ip) {
			t.Fatalf("uint32ToIP(%08X) expected %s, got %s", tt.expected, tt.ip, reversed.String())
		}
	}
}