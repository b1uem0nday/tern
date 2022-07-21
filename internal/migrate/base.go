package migrate

import (
	"context"
	"fmt"
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

func getCurrentVersion(ctx context.Context, conn *pgx.Conn) (v int, err error) {
	err = conn.QueryRow(ctx, "select version from "+versionTableName).Scan(&v)
	return v, err
}

func ensureRequiredExists(ctx context.Context, conn *pgx.Conn) error {
	err := acquireAdvisoryLock(ctx, conn)
	if err != nil {
		return err
	}
	defer func() {
		unlockErr := releaseAdvisoryLock(ctx, conn)
		if err == nil && unlockErr != nil {
			err = unlockErr
		}
	}()
	var count int
	//check if version table exists
	if err := conn.QueryRow(ctx, checkVersionTableExists, versionTableName).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return err
	}
	_, err = conn.Exec(ctx, fmt.Sprintf(createVersionTable, versionTableName, versionTableName))
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, fmt.Sprintf(createVersionCheckFunc, versionTableName, versionTableName))
	return err

}

func dropService(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, fmt.Sprintf(dropServiceData, versionTableName))

	return err
}
