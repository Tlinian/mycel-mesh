// Package node provides node registration and management for the Coordinator.
package node

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNodeNotFound    = errors.New("node not found")
	ErrNodeExists      = errors.New("node already exists")
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidNodeID   = errors.New("invalid node ID")
)

// NodeStatus represents the current status of a node.
type NodeStatus string

const (
	StatusOnline  NodeStatus = "online"
	StatusOffline NodeStatus = "offline"
)

// NATInfo contains NAT detection results.
type NATInfo struct {
	NATType    string   `json:"nat_type"`
	PublicIP   string   `json:"public_ip"`
	PublicPort int      `json:"public_port"`
	CanPunch   bool     `json:"can_punch"`
	LocalIPs   []string `json:"local_ips,omitempty"`
}

// Node represents a registered node in the network.
type Node struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	PublicKey  string     `json:"public_key"`
	AssignedIP string     `json:"assigned_ip"`
	NATInfo    *NATInfo   `json:"nat_info,omitempty"`
	LastSeen   time.Time  `json:"last_seen"`
	Status     NodeStatus `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

// NodeRegistry manages all registered nodes.
type NodeRegistry struct {
	nodes    map[string]*Node
	tokenMap map[string]string // token -> nodeID
	mu       sync.RWMutex
}

// NewNodeRegistry creates a new NodeRegistry.
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		nodes:    make(map[string]*Node),
		tokenMap: make(map[string]string),
	}
}

// Register adds a new node to the registry.
func (r *NodeRegistry) Register(nodeID, name, publicKey, assignedIP string, natInfo *NATInfo) (*Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if node already exists
	if _, exists := r.nodes[nodeID]; exists {
		return nil, ErrNodeExists
	}

	now := time.Now()
	node := &Node{
		ID:         nodeID,
		Name:       name,
		PublicKey:  publicKey,
		AssignedIP: assignedIP,
		NATInfo:    natInfo,
		LastSeen:   now,
		Status:     StatusOnline,
		CreatedAt:  now,
	}

	r.nodes[nodeID] = node
	return node, nil
}

// Heartbeat updates the last seen time and NAT info for a node.
func (r *NodeRegistry) Heartbeat(nodeID string, natInfo *NATInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, exists := r.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.LastSeen = time.Now()
	node.Status = StatusOnline
	if natInfo != nil {
		node.NATInfo = natInfo
	}

	return nil
}

// GetNode retrieves a node by ID.
func (r *NodeRegistry) GetNode(nodeID string) (*Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, exists := r.nodes[nodeID]
	if !exists {
		return nil, ErrNodeNotFound
	}

	return node, nil
}

// ListNodes returns all registered nodes.
func (r *NodeRegistry) ListNodes() []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*Node, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetOnlineNodes returns all nodes that are currently online.
func (r *NodeRegistry) GetOnlineNodes() []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*Node, 0)
	for _, node := range r.nodes {
		if node.Status == StatusOnline {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// RemoveNode removes a node from the registry.
func (r *NodeRegistry) RemoveNode(nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[nodeID]; !exists {
		return ErrNodeNotFound
	}

	delete(r.nodes, nodeID)
	return nil
}

// MarkOffline marks a node as offline.
func (r *NodeRegistry) MarkOffline(nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, exists := r.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.Status = StatusOffline
	return nil
}

// CleanupStaleNodes marks nodes as offline if they haven't sent a heartbeat.
func (r *NodeRegistry) CleanupStaleNodes(timeout time.Duration) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var staleNodes []string
	now := time.Now()

	for id, node := range r.nodes {
		if node.Status == StatusOnline && now.Sub(node.LastSeen) > timeout {
			node.Status = StatusOffline
			staleNodes = append(staleNodes, id)
		}
	}

	return staleNodes
}

// GetNodeCount returns the total number of registered nodes.
func (r *NodeRegistry) GetNodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// GetOnlineCount returns the number of online nodes.
func (r *NodeRegistry) GetOnlineCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, node := range r.nodes {
		if node.Status == StatusOnline {
			count++
		}
	}
	return count
}