package service

import (
	"context"
	"errors"
	"flag"
	"github.com/b1uem0nday/tern/internal/config"
	"github.com/b1uem0nday/tern/internal/migrate"
	"log"
)

func RunWithConfig(ctx context.Context) error {
	cfg, err := config.LoadConfig(config.DefConfigPath)
	if err != nil {
		log.Printf("an error was occurred while reading config: %v,\n use default config...\n", err)
	}
	return run(ctx, cfg.ConnectionString(), cfg.Path)

}

func RunWithFlags(ctx context.Context) error {
	connString := flag.String("conn", "", "connection string formatted as postgres://login:password@host:port/scheme")
	path := flag.String("path", "", "path to migration scripts")
	flag.Parse()
	if *connString == "" || *path == "" {
		return errors.New("flags was not properly set up")
	}
	return run(ctx, *connString, *path)

}

func run(ctx context.Context, connString, path string) error {
	migration, err := migrate.NewMigrator(ctx, connString)
	if err != nil {
		return err
	}
	err = migration.LoadMigrations(path)
	if err != nil {
		return err
	}
	return migration.Migrate(ctx)
}
