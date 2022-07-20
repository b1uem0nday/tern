package main

import (
	"context"
	"github.com/b1uem0nday/tern/internal/migrate"
	"log"
)

//проверить хранимые процедуры + функции на корректность создания
//template = 01-blablabla.sql, 02-nanana.sql
func main() {
	ctx := context.Background()

	m, err := migrate.NewMigrator(ctx, "postgres://user:P@ssw0rd@localhost:5432/test_migration")
	if err != nil {
		log.Fatal(err)
	}

	err = m.Migrate(ctx)
	if err != nil {
		log.Fatal(err)
	}

}
