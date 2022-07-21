package migrate

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

var defMigrationPath string

type (
	Migrator interface {
		ForceVersion(ctx context.Context, version uint) error
		Migrate(ctx context.Context) error
		MigrateTo(ctx context.Context, targetVersion int) error
		DropServiceData(ctx context.Context) error
	}
	Migration struct {
		Sequence int32
		Name     string
		SQL      string
	}
	Migrate struct {
		scheme     string
		conn       *pgx.Conn
		Migrations []*Migration
	}
)

func NewMigrator(ctx context.Context, connString string) (m *Migrate, err error) {
	m = &Migrate{}
	m.conn, err = pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	m.Migrations = make([]*Migration, 0)
	m.scheme = m.conn.Config().Database
	err = ensureRequiredExists(ctx, m.conn)
	if err != nil {
		m.conn.Close(ctx)
		return nil, err
	}
	return m, nil
}

func (m *Migrate) ForceVersion(ctx context.Context, version uint) error {
	if int(version) > len(m.Migrations) {
		return fmt.Errorf("version %d is higher than existent one", version)
	}
	_, err := m.conn.Exec(ctx, forceInsertVersionTable, version)
	return err
}

func (m *Migrate) Migrate(ctx context.Context) error {
	return m.MigrateTo(ctx, len(m.Migrations))
}

// MigrateTo migrates to targetVersion
func (m *Migrate) MigrateTo(ctx context.Context, targetVersion int) (err error) {
	err = acquireAdvisoryLock(ctx, m.conn)
	if err != nil {
		return err
	}
	defer func() {
		unlockErr := releaseAdvisoryLock(ctx, m.conn)
		if err == nil && unlockErr != nil {
			err = unlockErr
		}
	}()

	currentVersion, err := getCurrentVersion(ctx, m.conn)
	if err != nil {
		return err
	}

	if targetVersion < 0 || len(m.Migrations) < targetVersion {
		return fmt.Errorf(fmt.Sprintf("destination version %d is outside the valid versions of 0 to %d", targetVersion, len(m.Migrations)))
	}

	if currentVersion < 0 || len(m.Migrations) < currentVersion {
		return fmt.Errorf("current version %d is outside the valid versions of 0 to %d", currentVersion, len(m.Migrations))
	}

	for currentVersion != targetVersion {
		var current *Migration
		var sql string
		var sequence int32

		current = m.Migrations[currentVersion]
		sequence = current.Sequence
		sql = current.SQL

		var tx pgx.Tx
		tx, err = m.conn.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		// Execute the migration
		_, err = m.conn.Exec(ctx, sql)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok {
				return fmt.Errorf("%s: %v", current.Name, err)
			}
			return err
		}

		// Reset all database connection settings. Important to do before updating version as search_path may have been changed.
		m.conn.Exec(ctx, "reset all")

		// Add one to the version
		_, err = m.conn.Exec(ctx, "update "+versionTableName+" set version=$1", sequence)
		if err != nil {
			return err
		}

		err = tx.Commit(ctx)
		if err != nil {
			return err
		}

		currentVersion++
	}

	return nil
}

/*
	DropServiceData deletes existent tables & functions related to service
*/
func (m *Migrate) DropServiceData(ctx context.Context) error {
	return dropService(ctx, m.conn)
}

func (m *Migrate) appendMigration(name, SQL string) {
	m.Migrations = append(
		m.Migrations,
		&Migration{
			Sequence: int32(len(m.Migrations)) + 1,
			Name:     name,
			SQL:      SQL,
		})
	return
}
