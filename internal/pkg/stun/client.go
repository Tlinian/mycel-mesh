// Package stun provides STUN client functionality for NAT traversal.
package stun

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/stun/v2"
)

// DefaultSTUNServers is a list of public STUN servers.
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun3.l.google.com:19302",
	"stun4.l.google.com:19302",
	"stun.stunprotocol.org:3478",
}

// ClientConfig holds configuration for the STUN client.
type ClientConfig struct {
	STUNServers []string
	Timeout     time.Duration
	Retries     int
}

// Client represents a STUN client for NAT traversal.
type Client struct {
	config ClientConfig
	mu     sync.Mutex
}

// NATResult contains the result of a STUN query.
type NATResult struct {
	PublicAddr *net.UDPAddr
	XORMapped  *net.UDPAddr
	Changed    bool
	RTT        time.Duration
}

// NewClient creates a new STUN client with the given configuration.
func NewClient(config ClientConfig) *Client {
	if len(config.STUNServers) == 0 {
		config.STUNServers = DefaultSTUNServers
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.Retries == 0 {
		config.Retries = 3
	}
	return &Client{
		config: config,
	}
}

// NewDefaultClient creates a STUN client with default configuration.
func NewDefaultClient() *Client {
	return NewClient(ClientConfig{})
}

// QuerySTUN sends a STUN binding request to the specified server and returns the result.
func (c *Client) QuerySTUN(server string) (*NATResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	serverAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		return nil, fmt.Errorf("resolve server address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(c.config.Timeout))

	// Build STUN binding request
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, fmt.Errorf("build STUN request: %w", err)
	}

	startTime := time.Now()

	// Send request
	if _, err := conn.WriteTo(req.Raw, serverAddr); err != nil {
		return nil, fmt.Errorf("send STUN request: %w", err)
	}

	// Read response
	buf := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, fmt.Errorf("read STUN response: %w", err)
	}

	rtt := time.Since(startTime)

	// Parse response
	res := &stun.Message{Raw: buf[:n]}
	if err := res.Decode(); err != nil {
		return nil, fmt.Errorf("parse STUN response: %w", err)
	}

	result := &NATResult{
		RTT: rtt,
	}

	// Extract XOR-MAPPED-ADDRESS
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(res); err == nil {
		result.XORMapped = &net.UDPAddr{
			IP:   xorAddr.IP,
			Port: xorAddr.Port,
		}
		result.PublicAddr = result.XORMapped
	}

	// Extract MAPPED-ADDRESS (if available)
	var mappedAddr stun.MappedAddress
	if err := mappedAddr.GetFrom(res); err == nil {
		result.PublicAddr = &net.UDPAddr{
			IP:   mappedAddr.IP,
			Port: mappedAddr.Port,
		}
	}

	return result, nil
}

// QueryAllServers queries all configured STUN servers and returns the first successful result.
func (c *Client) QueryAllServers() (*NATResult, string, error) {
	var lastErr error

	for _, server := range c.config.STUNServers {
		result, err := c.QuerySTUN(server)
		if err == nil {
			return result, server, nil
		}
		lastErr = err
	}

	return nil, "", fmt.Errorf("all STUN servers failed: %w", lastErr)
}

// GetPublicAddress returns the public IP address and port as seen by STUN servers.
func (c *Client) GetPublicAddress() (*net.UDPAddr, error) {
	result, _, err := c.QueryAllServers()
	if err != nil {
		return nil, err
	}
	return result.PublicAddr, nil
}
