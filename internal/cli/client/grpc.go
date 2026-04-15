// Package client provides the gRPC client for mycelctl.
package client

import (
	"context"
	"time"

	pb "github.com/mycel/mesh/api/pb"
	"github.com/mycel/mesh/internal/encoding"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// init registers the JSON codec at package initialization.
func init() {
	encoding.RegisterAsProto()
}

// Client is the gRPC client for communicating with the Coordinator.
type Client struct {
	conn   *grpc.ClientConn
	client pb.NodeServiceClient
}

// NewClient creates a new gRPC client.
func NewClient(coordinatorAddr string) (*Client, error) {
	conn, err := grpc.Dial(
		coordinatorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewNodeServiceClient(conn),
	}, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Register registers a node with the Coordinator.
func (c *Client) Register(ctx context.Context, nodeID, name, publicKey, token string, natInfo *pb.NATInfo) (*pb.RegisterResponse, error) {
	req := &pb.RegisterRequest{
		NodeId:    nodeID,
		Name:      name,
		PublicKey: publicKey,
		Token:     token,
		NatInfo:   natInfo,
	}

	return c.client.Register(ctx, req)
}

// Heartbeat sends a heartbeat to the Coordinator.
func (c *Client) Heartbeat(ctx context.Context, nodeID string, natInfo *pb.NATInfo) (*pb.HeartbeatResponse, error) {
	req := &pb.HeartbeatRequest{
		NodeId:  nodeID,
		NatInfo: natInfo,
	}

	return c.client.Heartbeat(ctx, req)
}

// ListNodes lists all nodes in the network.
func (c *Client) ListNodes(ctx context.Context, token string) (*pb.ListNodesResponse, error) {
	req := &pb.ListNodesRequest{
		Token: token,
	}

	return c.client.ListNodes(ctx, req)
}

// Unregister unregisters a node from the network.
func (c *Client) Unregister(ctx context.Context, nodeID, token string) (*pb.UnregisterResponse, error) {
	req := &pb.UnregisterRequest{
		NodeId: nodeID,
		Token:  token,
	}

	return c.client.Unregister(ctx, req)
}

// NewClientWithTimeout creates a new gRPC client with custom timeout.
func NewClientWithTimeout(coordinatorAddr string, timeout time.Duration) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		coordinatorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewNodeServiceClient(conn),
	}, nil
}