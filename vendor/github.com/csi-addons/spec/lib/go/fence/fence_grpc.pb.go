// Code generated by make; DO NOT EDIT.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.20.2
// source: fence/fence.proto

package fence

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	FenceController_FenceClusterNetwork_FullMethodName   = "/fence.FenceController/FenceClusterNetwork"
	FenceController_UnfenceClusterNetwork_FullMethodName = "/fence.FenceController/UnfenceClusterNetwork"
	FenceController_ListClusterFence_FullMethodName      = "/fence.FenceController/ListClusterFence"
)

// FenceControllerClient is the client API for FenceController service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FenceControllerClient interface {
	// FenceClusterNetwork RPC call to perform a fencing operation.
	FenceClusterNetwork(ctx context.Context, in *FenceClusterNetworkRequest, opts ...grpc.CallOption) (*FenceClusterNetworkResponse, error)
	// UnfenceClusterNetwork RPC call to remove a list of CIDR blocks from the
	// list of blocklisted/fenced clients.
	UnfenceClusterNetwork(ctx context.Context, in *UnfenceClusterNetworkRequest, opts ...grpc.CallOption) (*UnfenceClusterNetworkResponse, error)
	// ListClusterFence RPC call to provide a list of blocklisted/fenced clients.
	ListClusterFence(ctx context.Context, in *ListClusterFenceRequest, opts ...grpc.CallOption) (*ListClusterFenceResponse, error)
}

type fenceControllerClient struct {
	cc grpc.ClientConnInterface
}

func NewFenceControllerClient(cc grpc.ClientConnInterface) FenceControllerClient {
	return &fenceControllerClient{cc}
}

func (c *fenceControllerClient) FenceClusterNetwork(ctx context.Context, in *FenceClusterNetworkRequest, opts ...grpc.CallOption) (*FenceClusterNetworkResponse, error) {
	out := new(FenceClusterNetworkResponse)
	err := c.cc.Invoke(ctx, FenceController_FenceClusterNetwork_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fenceControllerClient) UnfenceClusterNetwork(ctx context.Context, in *UnfenceClusterNetworkRequest, opts ...grpc.CallOption) (*UnfenceClusterNetworkResponse, error) {
	out := new(UnfenceClusterNetworkResponse)
	err := c.cc.Invoke(ctx, FenceController_UnfenceClusterNetwork_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fenceControllerClient) ListClusterFence(ctx context.Context, in *ListClusterFenceRequest, opts ...grpc.CallOption) (*ListClusterFenceResponse, error) {
	out := new(ListClusterFenceResponse)
	err := c.cc.Invoke(ctx, FenceController_ListClusterFence_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FenceControllerServer is the server API for FenceController service.
// All implementations must embed UnimplementedFenceControllerServer
// for forward compatibility
type FenceControllerServer interface {
	// FenceClusterNetwork RPC call to perform a fencing operation.
	FenceClusterNetwork(context.Context, *FenceClusterNetworkRequest) (*FenceClusterNetworkResponse, error)
	// UnfenceClusterNetwork RPC call to remove a list of CIDR blocks from the
	// list of blocklisted/fenced clients.
	UnfenceClusterNetwork(context.Context, *UnfenceClusterNetworkRequest) (*UnfenceClusterNetworkResponse, error)
	// ListClusterFence RPC call to provide a list of blocklisted/fenced clients.
	ListClusterFence(context.Context, *ListClusterFenceRequest) (*ListClusterFenceResponse, error)
	mustEmbedUnimplementedFenceControllerServer()
}

// UnimplementedFenceControllerServer must be embedded to have forward compatible implementations.
type UnimplementedFenceControllerServer struct {
}

func (UnimplementedFenceControllerServer) FenceClusterNetwork(context.Context, *FenceClusterNetworkRequest) (*FenceClusterNetworkResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FenceClusterNetwork not implemented")
}
func (UnimplementedFenceControllerServer) UnfenceClusterNetwork(context.Context, *UnfenceClusterNetworkRequest) (*UnfenceClusterNetworkResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnfenceClusterNetwork not implemented")
}
func (UnimplementedFenceControllerServer) ListClusterFence(context.Context, *ListClusterFenceRequest) (*ListClusterFenceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListClusterFence not implemented")
}
func (UnimplementedFenceControllerServer) mustEmbedUnimplementedFenceControllerServer() {}

// UnsafeFenceControllerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FenceControllerServer will
// result in compilation errors.
type UnsafeFenceControllerServer interface {
	mustEmbedUnimplementedFenceControllerServer()
}

func RegisterFenceControllerServer(s grpc.ServiceRegistrar, srv FenceControllerServer) {
	s.RegisterService(&FenceController_ServiceDesc, srv)
}

func _FenceController_FenceClusterNetwork_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FenceClusterNetworkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FenceControllerServer).FenceClusterNetwork(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FenceController_FenceClusterNetwork_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FenceControllerServer).FenceClusterNetwork(ctx, req.(*FenceClusterNetworkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FenceController_UnfenceClusterNetwork_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnfenceClusterNetworkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FenceControllerServer).UnfenceClusterNetwork(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FenceController_UnfenceClusterNetwork_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FenceControllerServer).UnfenceClusterNetwork(ctx, req.(*UnfenceClusterNetworkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FenceController_ListClusterFence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListClusterFenceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FenceControllerServer).ListClusterFence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FenceController_ListClusterFence_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FenceControllerServer).ListClusterFence(ctx, req.(*ListClusterFenceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// FenceController_ServiceDesc is the grpc.ServiceDesc for FenceController service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FenceController_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "fence.FenceController",
	HandlerType: (*FenceControllerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FenceClusterNetwork",
			Handler:    _FenceController_FenceClusterNetwork_Handler,
		},
		{
			MethodName: "UnfenceClusterNetwork",
			Handler:    _FenceController_UnfenceClusterNetwork_Handler,
		},
		{
			MethodName: "ListClusterFence",
			Handler:    _FenceController_ListClusterFence_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "fence/fence.proto",
}
