//go:build windows

// Package tun provides Windows-specific TUN device management using Wintun.
package tun

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// WindowsDevice implements Device interface for Windows using Wintun.
type WindowsDevice struct {
	name     string
	ip       net.IP
	subnet   *net.IPNet
	fd       int
	mu       sync.Mutex
	isUp     bool
}

// createDevice creates a Windows TUN device.
func createDevice(config *Config) (Device, error) {
	if config.Name == "" {
		config.Name = "Mycel0"
	}

	device := &WindowsDevice{
		name: config.Name,
	}

	// Try to create the interface using Windows commands
	// This requires Administrator privileges
	if err := device.createInterface(); err != nil {
		return nil, err
	}

	return device, nil
}

// createInterface creates a new network interface on Windows.
func (d *WindowsDevice) createInterface() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if interface already exists
	if d.interfaceExists() {
		// Remove existing interface first
		d.removeInterface()
	}

	// Create a new interface using netsh
	// Note: This creates a regular network interface, not a true TUN device
	// For production, use golang.zx2c4.com/wintun for proper Wintun support
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "interface", d.name, "enabled")
	if err := cmd.Run(); err != nil {
		// Interface might not exist, try to add it
		// On Windows, we need to use a different approach
		// For now, we'll assume the interface will be created when we set IP
	}

	return nil
}

// interfaceExists checks if the interface already exists.
func (d *WindowsDevice) interfaceExists() bool {
	cmd := exec.Command("netsh", "interface", "show", "interface", d.name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), d.name)
}

// removeInterface removes the interface.
func (d *WindowsDevice) removeInterface() error {
	// On Windows, we can't easily remove interfaces via netsh
	// This is a placeholder for proper Wintun cleanup
	return nil
}

// Name returns the device name.
func (d *WindowsDevice) Name() string {
	return d.name
}

// Close closes the device.
func (d *WindowsDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isUp {
		d.Down()
	}

	// Remove IP configuration
	cmd := exec.Command("netsh", "interface", "ipv4", "delete", "address", d.name)
	cmd.Run()

	return nil
}

// File returns the file descriptor.
func (d *WindowsDevice) File() int {
	return d.fd
}

// SetIP sets the device IP address.
func (d *WindowsDevice) SetIP(ip net.IP, subnet *net.IPNet) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if ip == nil || subnet == nil {
		return fmt.Errorf("IP and subnet are required")
	}

	d.ip = ip
	d.subnet = subnet

	// Calculate subnet mask
	mask := subnet.Mask
	maskStr := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])

	// Set IP address using netsh
	// This requires Administrator privileges
	cmd := exec.Command("netsh", "interface", "ipv4", "add", "address",
		d.name, ip.String(), maskStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternative method
		cmd = exec.Command("netsh", "interface", "ipv4", "set", "address",
			d.name, "static", ip.String(), maskStr)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to set IP: %w, output: %s", err, string(output))
		}
	}

	return nil
}

// Up brings up the device.
func (d *WindowsDevice) Up() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cmd := exec.Command("netsh", "interface", "set", "interface", d.name, "admin=enable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring up interface: %w, output: %s", err, string(output))
	}

	d.isUp = true
	return nil
}

// Down brings down the device.
func (d *WindowsDevice) Down() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cmd := exec.Command("netsh", "interface", "set", "interface", d.name, "admin=disable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring down interface: %w, output: %s", err, string(output))
	}

	d.isUp = false
	return nil
}

// Read reads a packet from the device.
// Note: This is a stub implementation. For real packet I/O,
// use golang.zx2c4.com/wintun and wireguard-go.
func (d *WindowsDevice) Read(buf []byte) (int, error) {
	// Stub: In production, this would read from the Wintun device
	// using the wireguard-go library
	return 0, fmt.Errorf("not implemented - requires wireguard-go library")
}

// Write writes a packet to the device.
// Note: This is a stub implementation. For real packet I/O,
// use golang.zx2c4.com/wintun and wireguard-go.
func (d *WindowsDevice) Write(buf []byte) (int, error) {
	// Stub: In production, this would write to the Wintun device
	// using the wireguard-go library
	return 0, fmt.Errorf("not implemented - requires wireguard-go library")
}

// AddRoute adds a route to the routing table.
func AddRoute(destination string, gateway string, interfaceName string) error {
	cmd := exec.Command("route", "add", destination, "mask", "255.255.255.255", gateway, "metric", "1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add route: %w, output: %s", err, string(output))
	}
	return nil
}

// DeleteRoute removes a route from the routing table.
func DeleteRoute(destination string) error {
	cmd := exec.Command("route", "delete", destination)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete route: %w, output: %s", err, string(output))
	}
	return nil
}

// CheckAdminPrivileges checks if the process has admin privileges.
func CheckAdminPrivileges() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

// GetInterfaceIndex returns the interface index.
func GetInterfaceIndex(name string) (int, error) {
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "interface", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	// Parse output to find interface index
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, name) {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				// First field is usually the index
				var idx int
				if _, err := fmt.Sscanf(fields[0], "%d", &idx); err == nil {
					return idx, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("interface not found")
}

// WaitForInterface waits for the interface to be ready.
func WaitForInterface(name string, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		d := &WindowsDevice{name: name}
		if d.interfaceExists() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("interface not ready after %v", timeout)
}