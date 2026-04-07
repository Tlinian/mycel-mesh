// Package service provides subnet management for Coordinator.
package service

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrSubnetNotFound    = errors.New("subnet not found")
	ErrSubnetExists      = errors.New("subnet already exists")
	ErrIPNotInSubnet     = errors.New("IP not in subnet range")
	ErrNoAvailableIP     = errors.New("no available IP addresses")
	ErrSubnetNotEmpty    = errors.New("subnet is not empty")
)

// SubnetConfig holds configuration for a subnet.
type SubnetConfig struct {
	// Name is the subnet name.
	Name string
	// NetworkCIDR is the CIDR notation (e.g., "10.0.1.0/24").
	NetworkCIDR string
	// Description is an optional description.
	Description string
	// Isolated indicates if the subnet is isolated from others.
	Isolated bool
}

// Subnet represents a virtual subnet.
type Subnet struct {
	ID          string
	Name        string
	Network     *net.IPNet
	NetworkCIDR string
	Description string
	Isolated    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Allocated IPs
	allocatedIPs map[string]string // IP -> NodeID
	mu           sync.RWMutex
}

// NewSubnet creates a new subnet from configuration.
func NewSubnet(id string, config SubnetConfig) (*Subnet, error) {
	_, network, err := net.ParseCIDR(config.NetworkCIDR)
	if err != nil {
		return nil, err
	}

	return &Subnet{
		ID:          id,
		Name:        config.Name,
		Network:     network,
		NetworkCIDR: config.NetworkCIDR,
		Description: config.Description,
		Isolated:    config.Isolated,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		allocatedIPs: make(map[string]string),
	}, nil
}

// AllocateIP allocates the next available IP in the subnet.
func (s *Subnet) AllocateIP(nodeID string) (net.IP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse network base IP
	baseIP := s.Network.IP

	// Calculate network size
	ones, bits := s.Network.Mask.Size()
	size := uint32(1 << (bits - ones))

	// Convert base IP to uint32
	base := ipToUint32(baseIP)

	// Search for available IP
	for i := uint32(1); i < size-1; i++ {
		ip := uint32ToIP(base + i)
		ipStr := ip.String()

		if _, exists := s.allocatedIPs[ipStr]; !exists {
			s.allocatedIPs[ipStr] = nodeID
			return ip, nil
		}
	}

	return nil, ErrNoAvailableIP
}

// AllocateSpecificIP allocates a specific IP in the subnet.
func (s *Subnet) AllocateSpecificIP(nodeID string, requestedIP net.IP) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if IP is in subnet range
	if !s.Network.Contains(requestedIP) {
		return ErrIPNotInSubnet
	}

	ipStr := requestedIP.String()
	if _, exists := s.allocatedIPs[ipStr]; exists {
		return ErrSubnetExists
	}

	s.allocatedIPs[ipStr] = nodeID
	return nil
}

// ReleaseIP releases an IP address.
func (s *Subnet) ReleaseIP(ip net.IP) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ipStr := ip.String()
	if _, exists := s.allocatedIPs[ipStr]; !exists {
		return ErrSubnetNotFound
	}

	delete(s.allocatedIPs, ipStr)
	return nil
}

// GetAllocatedIPs returns all allocated IPs.
func (s *Subnet) GetAllocatedIPs() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string, len(s.allocatedIPs))
	for k, v := range s.allocatedIPs {
		result[k] = v
	}
	return result
}

// GetNodeIP returns the allocated IP for a node.
func (s *Subnet) GetNodeIP(nodeID string) (net.IP, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for ipStr, nid := range s.allocatedIPs {
		if nid == nodeID {
			return net.ParseIP(ipStr), true
		}
	}
	return nil, false
}

// GetStats returns subnet statistics.
func (s *Subnet) GetStats() SubnetStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ones, bits := s.Network.Mask.Size()
	totalIPs := uint32(1 << (bits - ones))

	return SubnetStats{
		TotalIPs:     int(totalIPs),
		AllocatedIPs: len(s.allocatedIPs),
		AvailableIPs: int(totalIPs) - len(s.allocatedIPs) - 2, // Exclude network and broadcast
	}
}

// SubnetStats holds statistics about a subnet.
type SubnetStats struct {
	TotalIPs     int
	AllocatedIPs int
	AvailableIPs int
}

// Service manages subnets.
type SubnetService struct {
	subnets map[string]*Subnet
	mu      sync.RWMutex
}

// NewSubnetService creates a new subnet service.
func NewSubnetService() *SubnetService {
	return &SubnetService{
		subnets: make(map[string]*Subnet),
	}
}

// CreateSubnet creates a new subnet.
func (s *SubnetService) CreateSubnet(ctx context.Context, id string, config SubnetConfig) (*Subnet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if subnet with same name exists
	for _, existing := range s.subnets {
		if existing.Name == config.Name {
			return nil, ErrSubnetExists
		}
	}

	subnet, err := NewSubnet(id, config)
	if err != nil {
		return nil, err
	}

	s.subnets[id] = subnet
	return subnet, nil
}

// GetSubnet retrieves a subnet by ID.
func (s *SubnetService) GetSubnet(id string) (*Subnet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subnet, exists := s.subnets[id]
	if !exists {
		return nil, ErrSubnetNotFound
	}
	return subnet, nil
}

// ListSubnets returns all subnets.
func (s *SubnetService) ListSubnets() []*Subnet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Subnet, 0, len(s.subnets))
	for _, subnet := range s.subnets {
		result = append(result, subnet)
	}
	return result
}

// DeleteSubnet deletes a subnet.
func (s *SubnetService) DeleteSubnet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subnet, exists := s.subnets[id]
	if !exists {
		return ErrSubnetNotFound
	}

	// Check if subnet is empty
	if len(subnet.allocatedIPs) > 0 {
		return ErrSubnetNotEmpty
	}

	delete(s.subnets, id)
	return nil
}

// FindSubnetByIP finds which subnet contains an IP.
func (s *SubnetService) FindSubnetByIP(ip net.IP) *Subnet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, subnet := range s.subnets {
		if subnet.Network.Contains(ip) {
			return subnet
		}
	}
	return nil
}

// GetOrCreateDefaultSubnet gets the default subnet or creates one.
func (s *SubnetService) GetOrCreateDefaultSubnet() (*Subnet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing default subnet
	for _, subnet := range s.subnets {
		if subnet.Name == "default" {
			return subnet, nil
		}
	}

	// Create default subnet
	defaultConfig := SubnetConfig{
		Name:        "default",
		NetworkCIDR: "10.0.0.0/16",
		Description: "Default subnet",
		Isolated:    false,
	}

	subnet, err := NewSubnet("default", defaultConfig)
	if err != nil {
		return nil, err
	}

	s.subnets["default"] = subnet
	return subnet, nil
}

// ipToUint32 converts an IP address to uint32.
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// uint32ToIP converts a uint32 to an IP address.
func uint32ToIP(n uint32) net.IP {
	return net.IP{
		byte(n >> 24),
		byte(n >> 16),
		byte(n >> 8),
		byte(n),
	}
}
