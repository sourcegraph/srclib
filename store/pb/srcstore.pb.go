// Code generated by protoc-gen-gogo.
// source: srcstore.proto
// DO NOT EDIT!

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	srcstore.proto

It has these top-level messages:
	ImportOp
	IndexOp
*/
package pb

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// discarding unused import gogoproto "github.com/gogo/protobuf/gogoproto"
import unit "sourcegraph.com/sourcegraph/srclib/unit"
import graph3 "sourcegraph.com/sourcegraph/srclib/graph"
import pbtypes "sourcegraph.com/sqs/pbtypes"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type ImportOp struct {
	Repo     string               `protobuf:"bytes,1,opt,name=Repo,proto3" json:"Repo,omitempty"`
	CommitID string               `protobuf:"bytes,2,opt,name=CommitID,proto3" json:"CommitID,omitempty"`
	Unit     *unit.RepoSourceUnit `protobuf:"bytes,3,opt,name=Unit" json:"Unit,omitempty"`
	Data     *graph3.Output       `protobuf:"bytes,4,opt,name=Data" json:"Data,omitempty"`
}

func (m *ImportOp) Reset()         { *m = ImportOp{} }
func (m *ImportOp) String() string { return proto.CompactTextString(m) }
func (*ImportOp) ProtoMessage()    {}

type IndexOp struct {
	Repo     string `protobuf:"bytes,1,opt,name=Repo,proto3" json:"Repo,omitempty"`
	CommitID string `protobuf:"bytes,2,opt,name=CommitID,proto3" json:"CommitID,omitempty"`
}

func (m *IndexOp) Reset()         { *m = IndexOp{} }
func (m *IndexOp) String() string { return proto.CompactTextString(m) }
func (*IndexOp) ProtoMessage()    {}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Client API for MultiRepoImporter service

type MultiRepoImporterClient interface {
	// Import imports srclib build data for a source unit at a
	// specific version into the store.
	Import(ctx context.Context, in *ImportOp, opts ...grpc.CallOption) (*pbtypes.Void, error)
	// Index builds indexes for a specific repo at a specific version.
	Index(ctx context.Context, in *IndexOp, opts ...grpc.CallOption) (*pbtypes.Void, error)
}

type multiRepoImporterClient struct {
	cc *grpc.ClientConn
}

func NewMultiRepoImporterClient(cc *grpc.ClientConn) MultiRepoImporterClient {
	return &multiRepoImporterClient{cc}
}

func (c *multiRepoImporterClient) Import(ctx context.Context, in *ImportOp, opts ...grpc.CallOption) (*pbtypes.Void, error) {
	out := new(pbtypes.Void)
	err := grpc.Invoke(ctx, "/pb.MultiRepoImporter/Import", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *multiRepoImporterClient) Index(ctx context.Context, in *IndexOp, opts ...grpc.CallOption) (*pbtypes.Void, error) {
	out := new(pbtypes.Void)
	err := grpc.Invoke(ctx, "/pb.MultiRepoImporter/Index", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for MultiRepoImporter service

type MultiRepoImporterServer interface {
	// Import imports srclib build data for a source unit at a
	// specific version into the store.
	Import(context.Context, *ImportOp) (*pbtypes.Void, error)
	// Index builds indexes for a specific repo at a specific version.
	Index(context.Context, *IndexOp) (*pbtypes.Void, error)
}

func RegisterMultiRepoImporterServer(s *grpc.Server, srv MultiRepoImporterServer) {
	s.RegisterService(&_MultiRepoImporter_serviceDesc, srv)
}

func _MultiRepoImporter_Import_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(ImportOp)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(MultiRepoImporterServer).Import(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _MultiRepoImporter_Index_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(IndexOp)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(MultiRepoImporterServer).Index(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _MultiRepoImporter_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.MultiRepoImporter",
	HandlerType: (*MultiRepoImporterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Import",
			Handler:    _MultiRepoImporter_Import_Handler,
		},
		{
			MethodName: "Index",
			Handler:    _MultiRepoImporter_Index_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}
