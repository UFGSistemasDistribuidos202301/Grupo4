// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.23.4
// source: rpc.proto

package pb

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
	Database_RecvCRDTStates_FullMethodName = "/banco_de_dados.Database/RecvCRDTStates"
)

// DatabaseClient is the client API for Database service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DatabaseClient interface {
	RecvCRDTStates(ctx context.Context, in *RecvCRDTStatesRequest, opts ...grpc.CallOption) (*RecvCRDTStatesReply, error)
}

type databaseClient struct {
	cc grpc.ClientConnInterface
}

func NewDatabaseClient(cc grpc.ClientConnInterface) DatabaseClient {
	return &databaseClient{cc}
}

func (c *databaseClient) RecvCRDTStates(ctx context.Context, in *RecvCRDTStatesRequest, opts ...grpc.CallOption) (*RecvCRDTStatesReply, error) {
	out := new(RecvCRDTStatesReply)
	err := c.cc.Invoke(ctx, Database_RecvCRDTStates_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DatabaseServer is the server API for Database service.
// All implementations must embed UnimplementedDatabaseServer
// for forward compatibility
type DatabaseServer interface {
	RecvCRDTStates(context.Context, *RecvCRDTStatesRequest) (*RecvCRDTStatesReply, error)
	mustEmbedUnimplementedDatabaseServer()
}

// UnimplementedDatabaseServer must be embedded to have forward compatible implementations.
type UnimplementedDatabaseServer struct {
}

func (UnimplementedDatabaseServer) RecvCRDTStates(context.Context, *RecvCRDTStatesRequest) (*RecvCRDTStatesReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecvCRDTStates not implemented")
}
func (UnimplementedDatabaseServer) mustEmbedUnimplementedDatabaseServer() {}

// UnsafeDatabaseServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DatabaseServer will
// result in compilation errors.
type UnsafeDatabaseServer interface {
	mustEmbedUnimplementedDatabaseServer()
}

func RegisterDatabaseServer(s grpc.ServiceRegistrar, srv DatabaseServer) {
	s.RegisterService(&Database_ServiceDesc, srv)
}

func _Database_RecvCRDTStates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RecvCRDTStatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseServer).RecvCRDTStates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Database_RecvCRDTStates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseServer).RecvCRDTStates(ctx, req.(*RecvCRDTStatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Database_ServiceDesc is the grpc.ServiceDesc for Database service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Database_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "banco_de_dados.Database",
	HandlerType: (*DatabaseServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RecvCRDTStates",
			Handler:    _Database_RecvCRDTStates_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rpc.proto",
}