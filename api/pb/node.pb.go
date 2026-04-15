// Code generated manually. DO NOT EDIT.
package pb

import (
	"context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// NATInfo contains NAT detection results.
type NATInfo struct {
	NatType    string   `json:"nat_type,omitempty"`
	PublicIp   string   `json:"public_ip,omitempty"`
	PublicPort int32    `json:"public_port,omitempty"`
	CanPunch   bool     `json:"can_punch,omitempty"`
	LocalIps   []string `json:"local_ips,omitempty"`
}

func (x *NATInfo) Reset()            { *x = NATInfo{} }
func (x *NATInfo) String() string    { return "" }
func (*NATInfo) ProtoMessage()       {}
func (x *NATInfo) GetNatType() string {
	if x != nil { return x.NatType }
	return ""
}
func (x *NATInfo) GetPublicIp() string {
	if x != nil { return x.PublicIp }
	return ""
}
func (x *NATInfo) GetPublicPort() int32 {
	if x != nil { return x.PublicPort }
	return 0
}
func (x *NATInfo) GetCanPunch() bool {
	if x != nil { return x.CanPunch }
	return false
}
func (x *NATInfo) GetLocalIps() []string {
	if x != nil { return x.LocalIps }
	return nil
}

// RegisterRequest is sent by a node to join the network.
type RegisterRequest struct {
	NodeId    string   `json:"node_id,omitempty"`
	Name      string   `json:"name,omitempty"`
	PublicKey string   `json:"public_key,omitempty"`
	Token     string   `json:"token,omitempty"`
	NatInfo   *NATInfo `json:"nat_info,omitempty"`
}

func (x *RegisterRequest) Reset()         { *x = RegisterRequest{} }
func (x *RegisterRequest) String() string { return "" }
func (*RegisterRequest) ProtoMessage()    {}
func (x *RegisterRequest) GetNodeId() string {
	if x != nil { return x.NodeId }
	return ""
}
func (x *RegisterRequest) GetName() string {
	if x != nil { return x.Name }
	return ""
}
func (x *RegisterRequest) GetPublicKey() string {
	if x != nil { return x.PublicKey }
	return ""
}
func (x *RegisterRequest) GetToken() string {
	if x != nil { return x.Token }
	return ""
}
func (x *RegisterRequest) GetNatInfo() *NATInfo {
	if x != nil { return x.NatInfo }
	return nil
}

// PeerInfo contains information about a peer node.
type PeerInfo struct {
	NodeId    string   `json:"node_id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Ip        string   `json:"ip,omitempty"`
	PublicKey string   `json:"public_key,omitempty"`
	NatInfo   *NATInfo `json:"nat_info,omitempty"`
	Status    string   `json:"status,omitempty"`
}

func (x *PeerInfo) Reset()         { *x = PeerInfo{} }
func (x *PeerInfo) String() string { return "" }
func (*PeerInfo) ProtoMessage()    {}
func (x *PeerInfo) GetNodeId() string {
	if x != nil { return x.NodeId }
	return ""
}
func (x *PeerInfo) GetName() string {
	if x != nil { return x.Name }
	return ""
}
func (x *PeerInfo) GetIp() string {
	if x != nil { return x.Ip }
	return ""
}
func (x *PeerInfo) GetPublicKey() string {
	if x != nil { return x.PublicKey }
	return ""
}
func (x *PeerInfo) GetNatInfo() *NATInfo {
	if x != nil { return x.NatInfo }
	return nil
}
func (x *PeerInfo) GetStatus() string {
	if x != nil { return x.Status }
	return ""
}

// RegisterResponse is returned after successful registration.
type RegisterResponse struct {
	AssignedIp string     `json:"assigned_ip,omitempty"`
	SubnetCidr string     `json:"subnet_cidr,omitempty"`
	Peers      []*PeerInfo `json:"peers,omitempty"`
	Error      string     `json:"error,omitempty"`
}

func (x *RegisterResponse) Reset()         { *x = RegisterResponse{} }
func (x *RegisterResponse) String() string { return "" }
func (*RegisterResponse) ProtoMessage()    {}
func (x *RegisterResponse) GetAssignedIp() string {
	if x != nil { return x.AssignedIp }
	return ""
}
func (x *RegisterResponse) GetSubnetCidr() string {
	if x != nil { return x.SubnetCidr }
	return ""
}
func (x *RegisterResponse) GetPeers() []*PeerInfo {
	if x != nil { return x.Peers }
	return nil
}
func (x *RegisterResponse) GetError() string {
	if x != nil { return x.Error }
	return ""
}

// HeartbeatRequest is sent periodically to maintain node status.
type HeartbeatRequest struct {
	NodeId  string   `json:"node_id,omitempty"`
	NatInfo *NATInfo `json:"nat_info,omitempty"`
}

func (x *HeartbeatRequest) Reset()         { *x = HeartbeatRequest{} }
func (x *HeartbeatRequest) String() string { return "" }
func (*HeartbeatRequest) ProtoMessage()    {}
func (x *HeartbeatRequest) GetNodeId() string {
	if x != nil { return x.NodeId }
	return ""
}
func (x *HeartbeatRequest) GetNatInfo() *NATInfo {
	if x != nil { return x.NatInfo }
	return nil
}

// HeartbeatResponse confirms heartbeat received.
type HeartbeatResponse struct {
	PeerUpdates   []*PeerInfo `json:"peer_updates,omitempty"`
	OfflineNodes  []string    `json:"offline_nodes,omitempty"`
	Timestamp     int64       `json:"timestamp,omitempty"`
	Error         string      `json:"error,omitempty"`
}

func (x *HeartbeatResponse) Reset()         { *x = HeartbeatResponse{} }
func (x *HeartbeatResponse) String() string { return "" }
func (*HeartbeatResponse) ProtoMessage()    {}
func (x *HeartbeatResponse) GetPeerUpdates() []*PeerInfo {
	if x != nil { return x.PeerUpdates }
	return nil
}
func (x *HeartbeatResponse) GetOfflineNodes() []string {
	if x != nil { return x.OfflineNodes }
	return nil
}
func (x *HeartbeatResponse) GetTimestamp() int64 {
	if x != nil { return x.Timestamp }
	return 0
}
func (x *HeartbeatResponse) GetError() string {
	if x != nil { return x.Error }
	return ""
}

// ListNodesRequest requests all nodes in the network.
type ListNodesRequest struct {
	Token string `json:"token,omitempty"`
}

func (x *ListNodesRequest) Reset()         { *x = ListNodesRequest{} }
func (x *ListNodesRequest) String() string { return "" }
func (*ListNodesRequest) ProtoMessage()    {}
func (x *ListNodesRequest) GetToken() string {
	if x != nil { return x.Token }
	return ""
}

// ListNodesResponse contains all nodes.
type ListNodesResponse struct {
	Nodes []*PeerInfo `json:"nodes,omitempty"`
	Error string      `json:"error,omitempty"`
}

func (x *ListNodesResponse) Reset()         { *x = ListNodesResponse{} }
func (x *ListNodesResponse) String() string { return "" }
func (*ListNodesResponse) ProtoMessage()    {}
func (x *ListNodesResponse) GetNodes() []*PeerInfo {
	if x != nil { return x.Nodes }
	return nil
}
func (x *ListNodesResponse) GetError() string {
	if x != nil { return x.Error }
	return ""
}

// UnregisterRequest removes a node from the network.
type UnregisterRequest struct {
	NodeId string `json:"node_id,omitempty"`
	Token  string `json:"token,omitempty"`
}

func (x *UnregisterRequest) Reset()         { *x = UnregisterRequest{} }
func (x *UnregisterRequest) String() string { return "" }
func (*UnregisterRequest) ProtoMessage()    {}
func (x *UnregisterRequest) GetNodeId() string {
	if x != nil { return x.NodeId }
	return ""
}
func (x *UnregisterRequest) GetToken() string {
	if x != nil { return x.Token }
	return ""
}

// UnregisterResponse confirms unregistration.
type UnregisterResponse struct {
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (x *UnregisterResponse) Reset()         { *x = UnregisterResponse{} }
func (x *UnregisterResponse) String() string { return "" }
func (*UnregisterResponse) ProtoMessage()    {}
func (x *UnregisterResponse) GetSuccess() bool {
	if x != nil { return x.Success }
	return false
}
func (x *UnregisterResponse) GetError() string {
	if x != nil { return x.Error }
	return ""
}

// NodeServiceClient is the client API for NodeService service.
type NodeServiceClient interface {
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	Heartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error)
	ListNodes(ctx context.Context, in *ListNodesRequest, opts ...grpc.CallOption) (*ListNodesResponse, error)
	Unregister(ctx context.Context, in *UnregisterRequest, opts ...grpc.CallOption) (*UnregisterResponse, error)
}

type nodeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeServiceClient(cc grpc.ClientConnInterface) NodeServiceClient {
	return &nodeServiceClient{cc}
}

func (c *nodeServiceClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, "/mycel.NodeService/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) Heartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error) {
	out := new(HeartbeatResponse)
	err := c.cc.Invoke(ctx, "/mycel.NodeService/Heartbeat", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) ListNodes(ctx context.Context, in *ListNodesRequest, opts ...grpc.CallOption) (*ListNodesResponse, error) {
	out := new(ListNodesResponse)
	err := c.cc.Invoke(ctx, "/mycel.NodeService/ListNodes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) Unregister(ctx context.Context, in *UnregisterRequest, opts ...grpc.CallOption) (*UnregisterResponse, error) {
	out := new(UnregisterResponse)
	err := c.cc.Invoke(ctx, "/mycel.NodeService/Unregister", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NodeServiceServer is the server API for NodeService service.
type NodeServiceServer interface {
	Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
	Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
	ListNodes(context.Context, *ListNodesRequest) (*ListNodesResponse, error)
	Unregister(context.Context, *UnregisterRequest) (*UnregisterResponse, error)
}

// UnimplementedNodeServiceServer can be embedded to have forward compatible implementations.
type UnimplementedNodeServiceServer struct{}

func (UnimplementedNodeServiceServer) Register(context.Context, *RegisterRequest) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedNodeServiceServer) Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}
func (UnimplementedNodeServiceServer) ListNodes(context.Context, *ListNodesRequest) (*ListNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListNodes not implemented")
}
func (UnimplementedNodeServiceServer) Unregister(context.Context, *UnregisterRequest) (*UnregisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unregister not implemented")
}

func RegisterNodeServiceServer(s grpc.ServiceRegistrar, srv NodeServiceServer) {
	s.RegisterService(&NodeService_ServiceDesc, srv)
}

func _NodeService_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mycel.NodeService/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).Register(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartbeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mycel.NodeService/Heartbeat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).Heartbeat(ctx, req.(*HeartbeatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_ListNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListNodesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).ListNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mycel.NodeService/ListNodes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).ListNodes(ctx, req.(*ListNodesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_Unregister_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnregisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).Unregister(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mycel.NodeService/Unregister",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).Unregister(ctx, req.(*UnregisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var NodeService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "mycel.NodeService",
	HandlerType: (*NodeServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Register",
			Handler:    _NodeService_Register_Handler,
		},
		{
			MethodName: "Heartbeat",
			Handler:    _NodeService_Heartbeat_Handler,
		},
		{
			MethodName: "ListNodes",
			Handler:    _NodeService_ListNodes_Handler,
		},
		{
			MethodName: "Unregister",
			Handler:    _NodeService_Unregister_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/proto/node.proto",
}