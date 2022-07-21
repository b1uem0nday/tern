package grpc

import (
	"context"
	p "github.com/b1uem0nday/tern/api/migrate"
	"github.com/b1uem0nday/tern/internal/migrate"
	"google.golang.org/grpc"
	"log"
	"net"
	"strconv"
)

type Server struct {
	ctx      context.Context
	gs       *grpc.Server
	listener net.Listener
	path     string
	p.UnimplementedMigrationServiceServer
}

func NewServer(ctx context.Context, path string, port uint64) (*Server, error) {
	gg := Server{
		ctx:  ctx,
		path: path,
	}
	var err error

	gg.listener, err = net.Listen("tcp", ":"+strconv.FormatUint(port, 10))
	if err != nil {
		return nil, err
	}

	var opts []grpc.ServerOption

	gg.gs = grpc.NewServer(opts...)
	p.RegisterMigrationServiceServer(gg.gs, &gg)
	return &gg, nil
}

func (gg *Server) Run() {
	log.Println("listening", gg.listener.Addr())
	go gg.gs.Serve(gg.listener)
}

func (gg *Server) Migrate(ctx context.Context, request *p.MigrateRequest) (*p.VersionMessage, error) {
	var response p.VersionMessage
	m, err := migrate.NewMigrator(ctx, request.ConnectionString)
	if err != nil {
		return &response, err
	}
	err = m.LoadMigrations(gg.path)
	if err != nil {
		return &response, err
	}
	if request.DestinationVersion == nil {
		err = m.Migrate(ctx)
	} else {
		err = m.MigrateTo(ctx, int(*request.DestinationVersion))
	}

	if err != nil {
		return &response, err
	}
	response.Version = uint64(len(m.Migrations))
	return &response, nil
}
