package grpc

import (
	"context"
	p "github.com/b1uem0nday/tern/api/migrate"
	"github.com/b1uem0nday/tern/internal/migrate"
	"google.golang.org/grpc"
	"log"
	"net"
)

type Server struct {
	ctx      context.Context
	gs       *grpc.Server
	listener net.Listener
	p.UnimplementedMigrationServiceServer
	migrationsPool []migrate.Migrate
}

func NewServer(ctx context.Context, path string, port string) (*Server, error) {
	if err := migrate.SetDefaultPath(path); err != nil {
		return nil, err
	}
	gg := Server{
		ctx: ctx,
	}
	var err error
	gg.listener, err = net.Listen("tcp", ":"+port)
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
	if request.MigrationsPath == nil {
		err = m.LoadMigrationWithDefaultPath()
	} else {
		err = m.LoadMigrations(*request.MigrationsPath)
	}

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
