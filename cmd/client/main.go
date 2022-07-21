package main

import (
	"context"
	"github.com/b1uem0nday/tern/api/migrate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func main() {
	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, "localhost:5000", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal(err)
	}
	client := migrate.NewMigrationServiceClient(conn)

	version, err := client.Migrate(ctx, &migrate.MigrateRequest{
		ConnectionString:   "postgres://user:P@ssw0rd@localhost:5432/test_migration",
		DestinationVersion: nil,
	})
	if err != nil {

		log.Fatal(err)
	}
	log.Println(version.Version)
}
