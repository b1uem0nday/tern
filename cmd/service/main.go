package main

import (
	"context"
	"fmt"
	"github.com/b1uem0nday/tern/internal/migrate"
	"github.com/jackc/pgx/v4"
	"log"
)

//проверить хранимые процедуры + функции на корректность создания
//template = 01-blablabla.sql, 02-nanana.sql
func main() {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, "postgres://user:P@ssw0rd@localhost:5432/test_migration")
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.NewMigrator(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	ver, err := m.GetCurrentVersion(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ver)
	err = m.LoadMigrations("../../scripts/migrations/test_schema/")
	if err != nil {
		log.Fatal(err)
	}
	err = m.MigrateTo(ctx, 2)
	if err != nil {
		log.Fatal(err)
	}

}
