// Package grpc provides the gRPC server implementation for the Coordinator.
package grpc

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/mycel/mesh/api/pb"
	"github.com/mycel/mesh/internal/coordinator/node"
	"github.com/mycel/mesh/internal/coordinator/service"
	"github.com/mycel/mesh/internal/encoding"
	"google.golang.org/grpc"
)

// init registers the JSON codec at package initialization.
func init() {
	encoding.RegisterAsProto()
}

// NodeServiceServer implements the NodeService gRPC service.
type NodeServiceServer struct {
	pb.UnimplementedNodeServiceServer
	registry     *node.NodeRegistry
	subnetService *service.SubnetService
}

// NewNodeServiceServer creates a new NodeServiceServer.
func NewNodeServiceServer(registry *node.NodeRegistry, subnetService *service.SubnetService) *NodeServiceServer {
	return &NodeServiceServer{
		registry:     registry,
		subnetService: subnetService,
	}
}

// Register handles node registration requests.
func (s *NodeServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Register request from node %s (%s)", req.NodeId, req.Name)

	// Validate token (simple validation - in production use JWT)
	if req.Token == "" {
		return &pb.RegisterResponse{
			Error: "token is required",
		}, nil
	}

	// Validate node ID
	if req.NodeId == "" {
		return &pb.RegisterResponse{
			Error: "node_id is required",
		}, nil
	}

	// Validate public key
	if req.PublicKey == "" {
		return &pb.RegisterResponse{
			Error: "public_key is required",
		}, nil
	}

	// Get or create default subnet
	subnet, err := s.subnetService.GetOrCreateDefaultSubnet()
	if err != nil {
		return &pb.RegisterResponse{
			Error: "failed to get subnet: " + err.Error(),
		}, nil
	}

	// Allocate IP address
	ip, err := subnet.AllocateIP(req.NodeId)
	if err != nil {
		return &pb.RegisterResponse{
			Error: "failed to allocate IP: " + err.Error(),
		}, nil
	}

	// Convert NAT info from proto to internal type
	var natInfo *node.NATInfo
	if req.NatInfo != nil {
		natInfo = &node.NATInfo{
			NATType:    req.NatInfo.NatType,
			PublicIP:   req.NatInfo.PublicIp,
			PublicPort: int(req.NatInfo.PublicPort),
			CanPunch:   req.NatInfo.CanPunch,
			LocalIPs:   req.NatInfo.LocalIps,
		}
	}

	// Register node
	_, err = s.registry.Register(req.NodeId, req.Name, req.PublicKey, ip.String(), natInfo)
	if err != nil {
		// Release IP if registration fails
		subnet.ReleaseIP(ip)
		return &pb.RegisterResponse{
			Error: "failed to register node: " + err.Error(),
		}, nil
	}

	log.Printf("Node %s registered with IP %s", req.NodeId, ip.String())

	// Get existing peers
	peers := s.registry.ListNodes()
	var peerInfos []*pb.PeerInfo
	for _, peer := range peers {
		if peer.ID != req.NodeId {
			peerInfo := &pb.PeerInfo{
				NodeId:    peer.ID,
				Name:      peer.Name,
				Ip:        peer.AssignedIP,
				PublicKey: peer.PublicKey,
				Status:    string(peer.Status),
			}
			if peer.NATInfo != nil {
				peerInfo.NatInfo = &pb.NATInfo{
					NatType:    peer.NATInfo.NATType,
					PublicIp:   peer.NATInfo.PublicIP,
					PublicPort: int32(peer.NATInfo.PublicPort),
					CanPunch:   peer.NATInfo.CanPunch,
					LocalIps:   peer.NATInfo.LocalIPs,
				}
			}
			peerInfos = append(peerInfos, peerInfo)
		}
	}

	return &pb.RegisterResponse{
		AssignedIp:  ip.String(),
		SubnetCidr:  subnet.NetworkCIDR,
		Peers:       peerInfos,
	}, nil
}

// Heartbeat handles heartbeat requests from nodes.
func (s *NodeServiceServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	log.Printf("Heartbeat from node %s", req.NodeId)

	// Update node's last seen time
	var natInfo *node.NATInfo
	if req.NatInfo != nil {
		natInfo = &node.NATInfo{
			NATType:    req.NatInfo.NatType,
			PublicIP:   req.NatInfo.PublicIp,
			PublicPort: int(req.NatInfo.PublicPort),
			CanPunch:   req.NatInfo.CanPunch,
			LocalIPs:   req.NatInfo.LocalIps,
		}
	}

	err := s.registry.Heartbeat(req.NodeId, natInfo)
	if err != nil {
		return &pb.HeartbeatResponse{
			Error: "node not found: " + err.Error(),
		}, nil
	}

	// Get peer updates (nodes that changed since last heartbeat)
	// For simplicity, return all online peers
	peers := s.registry.GetOnlineNodes()
	var peerInfos []*pb.PeerInfo
	for _, peer := range peers {
		if peer.ID != req.NodeId {
			peerInfo := &pb.PeerInfo{
				NodeId:    peer.ID,
				Name:      peer.Name,
				Ip:        peer.AssignedIP,
				PublicKey: peer.PublicKey,
				Status:    string(peer.Status),
			}
			if peer.NATInfo != nil {
				peerInfo.NatInfo = &pb.NATInfo{
					NatType:    peer.NATInfo.NATType,
					PublicIp:   peer.NATInfo.PublicIP,
					PublicPort: int32(peer.NATInfo.PublicPort),
					CanPunch:   peer.NATInfo.CanPunch,
					LocalIps:   peer.NATInfo.LocalIPs,
				}
			}
			peerInfos = append(peerInfos, peerInfo)
		}
	}

	return &pb.HeartbeatResponse{
		PeerUpdates: peerInfos,
		Timestamp:   time.Now().Unix(),
	}, nil
}

// ListNodes handles list nodes requests.
func (s *NodeServiceServer) ListNodes(ctx context.Context, req *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	log.Printf("ListNodes request")

	// Validate token
	if req.Token == "" {
		return &pb.ListNodesResponse{
			Error: "token is required",
		}, nil
	}

	nodes := s.registry.ListNodes()
	var nodeInfos []*pb.PeerInfo
	for _, n := range nodes {
		nodeInfo := &pb.PeerInfo{
			NodeId:    n.ID,
			Name:      n.Name,
			Ip:        n.AssignedIP,
			PublicKey: n.PublicKey,
			Status:    string(n.Status),
		}
		if n.NATInfo != nil {
			nodeInfo.NatInfo = &pb.NATInfo{
				NatType:    n.NATInfo.NATType,
				PublicIp:   n.NATInfo.PublicIP,
				PublicPort: int32(n.NATInfo.PublicPort),
				CanPunch:   n.NATInfo.CanPunch,
				LocalIps:   n.NATInfo.LocalIPs,
			}
		}
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return &pb.ListNodesResponse{
		Nodes: nodeInfos,
	}, nil
}

// Unregister handles node unregistration requests.
func (s *NodeServiceServer) Unregister(ctx context.Context, req *pb.UnregisterRequest) (*pb.UnregisterResponse, error) {
	log.Printf("Unregister request for node %s", req.NodeId)

	// Validate token
	if req.Token == "" {
		return &pb.UnregisterResponse{
			Error: "token is required",
		}, nil
	}

	// Get node to find its IP
	node, err := s.registry.GetNode(req.NodeId)
	if err != nil {
		return &pb.UnregisterResponse{
			Error: "node not found: " + err.Error(),
		}, nil
	}

	// Release IP from subnet
	subnet, err := s.subnetService.GetOrCreateDefaultSubnet()
	if err == nil {
		subnet.ReleaseIP(net.ParseIP(node.AssignedIP))
	}

	// Remove node from registry
	err = s.registry.RemoveNode(req.NodeId)
	if err != nil {
		return &pb.UnregisterResponse{
			Error: "failed to remove node: " + err.Error(),
		}, nil
	}

	return &pb.UnregisterResponse{
		Success: true,
	}, nil
}

// StartGRPCServer starts the gRPC server.
func StartGRPCServer(addr string, registry *node.NodeRegistry, subnetService *service.SubnetService) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	RegisterNodeServiceServer(grpcServer, registry, subnetService)

	log.Printf("gRPC server listening on %s", addr)
	go grpcServer.Serve(lis)
	return grpcServer, nil
}

// RegisterNodeServiceServer registers the NodeService with a gRPC server.
func RegisterNodeServiceServer(s *grpc.Server, registry *node.NodeRegistry, subnetService *service.SubnetService) {
	pb.RegisterNodeServiceServer(s, NewNodeServiceServer(registry, subnetService))
}