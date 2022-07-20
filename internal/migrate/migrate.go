package migrate

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//template - 0001-init.sql
var migrationPattern = regexp.MustCompile(`\A(\d{1,4})-.+\.sql\z`)

const migrationPath = "/home/irina/Documents/tern/scripts/migrations/"

type (
	Migration struct {
		Sequence int32
		Name     string
		SQL      string
	}
	DefaultMigratorFS struct{}

	Migrator struct {
		conn       *pgx.Conn
		Migrations []*Migration
	}
)

func NewMigrator(ctx context.Context, connString string) (m *Migrator, err error) {
	m = &Migrator{}
	m.conn, err = pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	m.Migrations = make([]*Migration, 0)

	err = m.ensureSchemaVersionTableExists(ctx)
	if err != nil {
		m.conn.Close(ctx)
		return nil, err
	}

	if err = m.LoadMigrations(migrationPath + m.conn.Config().Database); err != nil {
		m.conn.Close(ctx)
		return nil, err
	}
	return
}

func (m *Migrator) ensureSchemaVersionTableExists(ctx context.Context) (err error) {
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

	if ok, err := m.versionTableExists(ctx); err != nil || ok {
		return err
	}

	_, err = m.conn.Exec(ctx, fmt.Sprintf(createVersionTable, versionTableName, versionTableName, versionTableName))
	return err
}

func (m *Migrator) versionTableExists(ctx context.Context) (ok bool, err error) {
	var count int
	err = m.conn.QueryRow(ctx, "select count(*) from pg_catalog.pg_class where relname=$1 and relkind='r' and pg_table_is_visible(oid)", versionTableName).Scan(&count)
	return count > 0, err
}

func (m *Migrator) ForceVersion(ctx context.Context, version uint) error {
	if int(version) > len(m.Migrations) {
		return fmt.Errorf("version %d is higher than existent one", version)
	}

	_, err := m.conn.Exec(ctx, forceInsertVersionTable, version)

	return err
}

func (m *Migrator) LoadMigrations(path string) error {

	paths, err := FindMigrationsEx(path)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		return fmt.Errorf("no migrations found at %s", path)
	}

	for _, p := range paths {
		body, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}

		m.AppendMigration(filepath.Base(p), string(body))
	}

	return nil
}

func FindMigrationsEx(path string) ([]string, error) {
	path = strings.TrimRight(path, string(filepath.Separator))

	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(fileInfos))
	for _, fi := range fileInfos {
		if fi.IsDir() {
			continue
		}

		matches := migrationPattern.FindStringSubmatch(fi.Name())
		if len(matches) != 2 {
			continue
		}

		n, err := strconv.ParseInt(matches[1], 10, 32)
		if err != nil {
			// The regexp already validated that the prefix is all digits so this *should* never fail
			return nil, err
		}

		if n < int64(len(paths)+1) {
			return nil, fmt.Errorf("Duplicate migration %d", n)
		}

		if int64(len(paths)+1) < n {
			return nil, fmt.Errorf("Missing migration %d", len(paths)+1)
		}

		paths = append(paths, filepath.Join(path, fi.Name()))
	}

	return paths, nil
}

func (m *Migrator) AppendMigration(name, upSQL string) {
	m.Migrations = append(
		m.Migrations,
		&Migration{
			Sequence: int32(len(m.Migrations)) + 1,
			Name:     name,
			SQL:      upSQL,
		})
	return
}

func (m *Migrator) Migrate(ctx context.Context) error {
	return m.MigrateTo(ctx, len(m.Migrations))
}

// MigrateTo migrates to targetVersion
func (m *Migrator) MigrateTo(ctx context.Context, targetVersion int) (err error) {
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

	currentVersion, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return err
	}

	if targetVersion < 0 || len(m.Migrations) < targetVersion {
		return fmt.Errorf(fmt.Sprintf("destination version %d is outside the valid versions of 0 to %d", targetVersion, len(m.Migrations)))
	}

	if currentVersion < 0 || len(m.Migrations) < currentVersion {
		return fmt.Errorf("current version %d is outside the valid versions of 0 to %d", currentVersion, len(m.Migrations))
	}

	var direction int
	if currentVersion < targetVersion {
		direction = 1
	} else {
		direction = -1
	}

	for currentVersion != targetVersion {
		var current *Migration
		var sql string
		var sequence int32
		if direction == 1 {
			current = m.Migrations[currentVersion]
			sequence = current.Sequence
			sql = current.SQL
		}

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

		currentVersion = currentVersion + direction
	}

	return nil
}
