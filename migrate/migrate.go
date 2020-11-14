package migrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

var migrationPattern = regexp.MustCompile(`\A(\d+)_.+\.sql\z`)

var ErrNoFwMigration = errors.New("no sql in forward migration step")

type BadVersionError string

func (e BadVersionError) Error() string {
	return string(e)
}

type IrreversibleMigrationError struct {
	m *Migration
}

func (e IrreversibleMigrationError) Error() string {
	return fmt.Sprintf("Irreversible migration: %d - %s", e.m.Sequence, e.m.Name)
}

type NoMigrationsFoundError struct {
	Path string
}

func (e NoMigrationsFoundError) Error() string {
	return fmt.Sprintf("No migrations found at %s", e.Path)
}

type MigrationPgError struct {
	Sql string
	*pgconn.PgError
}

type Migration struct {
	Sequence int32
	Name     string
	SQL      string
}

type MigratorOptions struct {
	// DisableTx causes the Migrator not to run migrations in a transaction.
	DisableTx bool
	// MigratorFS is the interface used for collecting the migrations.
	MigratorFS http.FileSystem
}

type Migrator struct {
	conn         *pgx.Conn
	versionTable string
	options      *MigratorOptions
	Migrations   []*Migration
	OnStart      func(ctx context.Context, sequence int32, name string, sql string) // OnStart is called when a migration is run.
	Data         map[string]interface{}                                             // Data available to use in migrations
}

// NewMigrator initializes a new Migrator. It is highly recommended that versionTable be schema qualified.
func NewMigrator(ctx context.Context, conn *pgx.Conn, versionTable string) (m *Migrator, err error) {
	return NewMigratorEx(ctx, conn, versionTable, &MigratorOptions{MigratorFS: defaultMigratorFS{}})
}

// NewMigratorEx initializes a new Migrator. It is highly recommended that versionTable be schema qualified.
func NewMigratorEx(ctx context.Context, conn *pgx.Conn, versionTable string, opts *MigratorOptions) (m *Migrator, err error) {
	m = &Migrator{conn: conn, versionTable: versionTable, options: opts}
	err = m.ensureSchemaVersionTableExists(ctx)
	m.Migrations = make([]*Migration, 0)
	m.Data = make(map[string]interface{})
	return
}

type defaultMigratorFS struct{}

func (defaultMigratorFS) Open(name string) (http.File, error) {
	return os.Open(name)
}

func normalizeDirPath(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		return "/"
	}
	return path
}

func FindMigrationsEx(path string, fs http.FileSystem) ([]string, error) {
	path = normalizeDirPath(path)

	fileInfos, err := fsReadDir(fs, path)
	if err != nil {
		return nil, err
	}

	possiblePaths := make([]string, 0, len(fileInfos))
	for _, fi := range fileInfos {
		if fi.IsDir() {
			continue
		}

		matches := migrationPattern.FindStringSubmatch(fi.Name())
		if len(matches) != 2 {
			continue
		}

		possiblePaths = append(possiblePaths, fi.Name())
	}
	sort.Strings(possiblePaths)

	paths := make([]string, 0, len(fileInfos))
	for _, pp := range possiblePaths {
		matches := migrationPattern.FindStringSubmatch(pp)
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

		paths = append(paths, filepath.Join(path, pp))
	}

	return paths, nil
}

func FindMigrations(path string) ([]string, error) {
	return FindMigrationsEx(path, defaultMigratorFS{})
}

func (m *Migrator) findSharePaths(path string) ([]string, error) {
	filePaths, err := fsFiles(m.options.MigratorFS, path, nil)
	if err != nil {
		return nil, err
	}

	pattern := path + "/*/*.sql"
	var matches []string
	for _, s := range filePaths {
		if matched, _ := filepath.Match(pattern, s); matched {
			matches = append(matches, s)
		}
	}

	return matches, nil
}

func (m *Migrator) LoadMigrations(path string) error {
	path = normalizeDirPath(path)

	mainTmpl := template.New("main").Funcs(sprig.TxtFuncMap())

	sharedPaths, err := m.findSharePaths(path)
	if err != nil {
		return err
	}

	for _, p := range sharedPaths {
		body, err := fsReadFile(m.options.MigratorFS, p)
		if err != nil {
			return err
		}

		name := strings.Replace(p, path+string(filepath.Separator), "", 1)
		_, err = mainTmpl.New(name).Parse(string(body))
		if err != nil {
			return err
		}
	}

	paths, err := FindMigrationsEx(path, m.options.MigratorFS)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		return NoMigrationsFoundError{Path: path}
	}

	for _, p := range paths {
		body, err := fsReadFile(m.options.MigratorFS, p)
		if err != nil {
			return err
		}

		upSQL := strings.TrimSpace(string(body))
		upSQL, err = m.evalMigration(mainTmpl.New(filepath.Base(p)+" up"), upSQL)
		if err != nil {
			return err
		}
		// Make sure there is SQL in the forward migration step.
		containsSQL := false
		for _, v := range strings.Split(upSQL, "\n") {
			// Only account for regular single line comment, empty line and space/comment combination
			cleanString := strings.TrimSpace(v)
			if len(cleanString) != 0 &&
				!strings.HasPrefix(cleanString, "--") {
				containsSQL = true
				break
			}
		}
		if !containsSQL {
			return ErrNoFwMigration
		}

		m.AppendMigration(filepath.Base(p), upSQL)
	}

	return nil
}

func (m *Migrator) evalMigration(tmpl *template.Template, sql string) (string, error) {
	tmpl, err := tmpl.Parse(sql)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, m.Data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
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

// Migrate runs pending migrations
// It calls m.OnStart when it begins a migration
func (m *Migrator) Migrate(ctx context.Context) error {
	return m.MigrateTo(ctx, int32(len(m.Migrations)))
}

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

// MigrateTo migrates to targetVersion
func (m *Migrator) MigrateTo(ctx context.Context, targetVersion int32) (err error) {
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

	if currentVersion < 0 {
		errMsg := fmt.Sprintf("current version %d is less than 0", currentVersion)
		return BadVersionError(errMsg)
	}

	if int32(len(m.Migrations)) < currentVersion {
		errMsg := fmt.Sprintf("current version %d is greater than last version of %d", currentVersion, len(m.Migrations))
		return BadVersionError(errMsg)
	}

	if targetVersion < currentVersion || int32(len(m.Migrations)) < targetVersion {
		errMsg := fmt.Sprintf("destination version %d is outside the valid versions of %d to %d", targetVersion, currentVersion, len(m.Migrations))
		return BadVersionError(errMsg)
	}

	for ; currentVersion != targetVersion; currentVersion++ {
		var current *Migration
		var sql string
		var sequence int32
		current = m.Migrations[currentVersion]
		sequence = current.Sequence
		sql = current.SQL

		var tx pgx.Tx
		if !m.options.DisableTx {
			tx, err = m.conn.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)
		}

		// Fire on start callback
		if m.OnStart != nil {
			m.OnStart(ctx, current.Sequence, current.Name, sql)
		}

		// Execute the migration
		_, err = m.conn.Exec(ctx, sql)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok {
				return MigrationPgError{Sql: sql, PgError: err}
			}
			return err
		}

		// Reset all database connection settings. Important to do before updating version as search_path may have been changed.
		m.conn.Exec(ctx, "reset all")

		// Add one to the version
		_, err = m.conn.Exec(ctx, "update "+m.versionTable+" set version=$1", sequence)
		if err != nil {
			return err
		}

		if !m.options.DisableTx {
			err = tx.Commit(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Migrator) GetCurrentVersion(ctx context.Context) (v int32, err error) {
	err = m.conn.QueryRow(ctx, "select version from "+m.versionTable).Scan(&v)
	return v, err
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

	_, err = m.conn.Exec(ctx, fmt.Sprintf(`
    create table if not exists %[1]s(
			version int4 not null check (version >= 0)
		);

		create unique index on %[1]s (((1)));

		insert into %[1]s(version) values (0)
		on conflict do nothing;
  `, m.versionTable))
	return err
}

func (m *Migrator) versionTableExists(ctx context.Context) (ok bool, err error) {
	var count int
	if i := strings.IndexByte(m.versionTable, '.'); i == -1 {
		err = m.conn.QueryRow(ctx, "select count(*) from pg_catalog.pg_class where relname=$1 and relkind='r' and pg_table_is_visible(oid)", m.versionTable).Scan(&count)
	} else {
		schema, table := m.versionTable[:i], m.versionTable[i+1:]
		err = m.conn.QueryRow(ctx, "select count(*) from pg_catalog.pg_tables where schemaname=$1 and tablename=$2", schema, table).Scan(&count)
	}
	return count > 0, err
}
