package stun

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// NATType represents the type of NAT.
type NATType string

const (
	// NATUnknown indicates NAT type could not be determined.
	NATUnknown NATType = "unknown"
	// NATFullCone indicates full cone NAT (most permissive).
	NATFullCone NATType = "full_cone"
	// NATRestrictedCone indicates restricted cone NAT.
	NATRestrictedCone NATType = "restricted_cone"
	// NATPortRestrictedCone indicates port restricted cone NAT.
	NATPortRestrictedCone NATType = "port_restricted_cone"
	// NATSymmetric indicates symmetric NAT (most restrictive).
	NATSymmetric NATType = "symmetric"
)

// NATInfo contains information about the detected NAT.
type NATInfo struct {
	Type       NATType
	PublicAddr *net.UDPAddr
	CanP2P     bool
	Details    string
}

// IsP2PCapable checks if P2P connection is likely to succeed based on NAT type.
func (info *NATInfo) IsP2PCapable() bool {
	if info == nil {
		return false
	}
	return info.CanP2P
}

// IsSymmetric checks if the NAT is symmetric.
func (info *NATInfo) IsSymmetric() bool {
	if info == nil {
		return false
	}
	return info.Type == NATSymmetric
}

// NATDetector performs NAT type detection using multiple STUN servers.
type NATDetector struct {
	client     *Client
	mu         sync.Mutex
	detections map[string]*NATInfo
}

// NewNATDetector creates a new NAT detector with the given STUN client.
func NewNATDetector(client *Client) *NATDetector {
	if client == nil {
		client = NewDefaultClient()
	}
	return &NATDetector{
		client:     client,
		detections: make(map[string]*NATInfo),
	}
}

// DetectNATType performs NAT type detection using the RFC 8445 procedure.
// Returns NATInfo with detected NAT type and P2P capability.
func (d *NATDetector) DetectNATType() (*NATInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Query multiple STUN servers to detect NAT behavior
	server1 := DefaultSTUNServers[0] // stun.l.google.com:19302
	server2 := DefaultSTUNServers[1] // stun1.l.google.com:19302

	// First query to server1
	result1, err := d.client.QuerySTUN(server1)
	if err != nil {
		return &NATInfo{
			Type:    NATUnknown,
			Details: fmt.Sprintf("STUN query failed: %v", err),
		}, err
	}

	// Second query to server1 (check if same public address)
	result2, err := d.client.QuerySTUN(server1)
	if err != nil {
		return &NATInfo{
			Type:    NATUnknown,
			Details: fmt.Sprintf("Second STUN query failed: %v", err),
		}, err
	}

	// Third query to server2 (check if public address changes)
	result3, err := d.client.QuerySTUN(server2)
	if err != nil {
		return &NATInfo{
			Type:    NATUnknown,
			Details: fmt.Sprintf("Query to second STUN server failed: %v", err),
		}, err
	}

	// Analyze results
	info := &NATInfo{
		PublicAddr: result1.PublicAddr,
	}

	// Check if NAT is symmetric (different public address for different STUN servers)
	if !result1.PublicAddr.IP.Equal(result3.PublicAddr.IP) ||
		result1.PublicAddr.Port != result3.PublicAddr.Port {
		info.Type = NATSymmetric
		info.CanP2P = false
		info.Details = "Symmetric NAT: different public address for different STUN servers"
		return info, nil
	}

	// Check if NAT is stable (same public address for same STUN server)
	if !result1.PublicAddr.IP.Equal(result2.PublicAddr.IP) ||
		result1.PublicAddr.Port != result2.PublicAddr.Port {
		info.Type = NATSymmetric
		info.CanP2P = false
		info.Details = "Symmetric NAT: public address changes between queries"
		return info, nil
	}

	// For full cone detection, we would need a second STUN server with different IP
	// to test if the mapping is created for any external host.
	// Simplified: if we get consistent results, assume it's cone NAT.

	// Check port consistency
	if result1.PublicAddr.Port != result2.PublicAddr.Port {
		info.Type = NATSymmetric
		info.CanP2P = false
		info.Details = "Symmetric NAT: port changes between queries"
		return info, nil
	}

	// Default to full cone for consistent results
	// In a full implementation, we would test binding from different ports
	info.Type = NATFullCone
	info.CanP2P = true
	info.Details = "Full cone NAT: consistent public address, P2P should work"

	return info, nil
}

// SimpleNATDetection performs a simple NAT detection using default STUN servers.
// This is a convenience function for quick NAT type detection.
func SimpleNATDetection() (*NATInfo, error) {
	client := NewDefaultClient()
	detector := NewNATDetector(client)
	return detector.DetectNATType()
}

// GetPublicIP is a convenience function to get the public IP address.
func GetPublicIP() (string, error) {
	client := NewDefaultClient()
	addr, err := client.GetPublicAddress()
	if err != nil {
		return "", err
	}
	return addr.IP.String(), nil
}

// GetNATTypeWithTimeout performs NAT detection with a timeout.
func GetNATTypeWithTimeout(timeout time.Duration) (*NATInfo, error) {
	done := make(chan *NATInfo, 1)
	errChan := make(chan error, 1)

	go func() {
		info, err := SimpleNATDetection()
		if err != nil {
			errChan <- err
			return
		}
		done <- info
	}()

	select {
	case info := <-done:
		return info, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
		return &NATInfo{
			Type:    NATUnknown,
			Details: "NAT detection timeout",
		}, fmt.Errorf("NAT detection timeout after %v", timeout)
	}
}
