package migrate

import (
	"context"
	"github.com/jackc/pgx/v4"
)

const (
	versionTableName = "version"
)

// Lock to ensure multiple migrations cannot occur simultaneously
const lockNum = int64(9628173550095224) // arbitrary random number

func acquireAdvisoryLock(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "select pg_advisory_lock($1)", lockNum)
	return err
}

func releaseAdvisoryLock(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "select pg_advisory_unlock($1)", lockNum)
	return err
}

func (m *Migrator) GetCurrentVersion(ctx context.Context) (v int, err error) {
	err = m.conn.QueryRow(ctx, "select version from "+versionTableName).Scan(&v)
	return v, err
}
