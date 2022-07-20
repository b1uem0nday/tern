package grpc

import (
	"context"
	"github.com/b1uem0nday/tern/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"strconv"
)

type GrpcServer struct {
	ctx      context.Context
	gs       *grpc.Server
	listener net.Listener
	api.UnimplementedMigrationServiceServer
}

func NewGrpcServer(ctx context.Context, port uint64) (*GrpcServer, error) {
	var gg GrpcServer
	var err error
	gg.ctx = ctx

	gg.listener, err = net.Listen("tcp", ":"+strconv.FormatUint(port, 10))
	if err != nil {
		return nil, err
	}

	var opts []grpc.ServerOption

	gg.gs = grpc.NewServer(opts...)
	api.RegisterMigrationServiceServer(gg.gs, &gg)
	return &gg, nil
}

func (gg *GrpcServer) Run() (err error) {
	log.Println("listening", gg.listener.Addr())
	go gg.gs.Serve(gg.listener)
	return nil
}

func (gg *GrpcServer) Connect(ctx context.Context, request *api.ConnectRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (gg *GrpcServer) ConnectAndUpdate(ctx context.Context, request *api.ConnectAndUpdateRequest) (*api.VersionMessage, error) {
	return &api.VersionMessage{}, nil
}

func (gg *GrpcServer) ForceVersion(ctx context.Context, message *api.VersionMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
