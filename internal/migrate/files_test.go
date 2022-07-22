package migrate

import (
	"github.com/jackc/pgx/v4"
	"os"
	"path/filepath"
	"testing"
)

const testPath = "test_path"

func TestMigrate_LoadMigrations(t *testing.T) {
	dir, err := os.MkdirTemp("", testPath)
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	tests := []struct {
		name    string
		dir     string
		files   []string
		wantErr bool
	}{
		{
			name:    "no folder",
			wantErr: true,
		},
		{
			name:    "no migration files",
			dir:     dir,
			wantErr: true,
		},
		{
			name:    "duplicate migrations №1",
			dir:     dir,
			files:   []string{"001-file.sql", "0001-file.sql"},
			wantErr: true,
		},
		{
			name:    "missing migration №2",
			dir:     dir,
			files:   []string{"001-file.sql", "003-file.sql"},
			wantErr: true,
		},
		{
			name:    "migrations files and pattern are not match",
			dir:     dir,
			files:   []string{"file1.sql", "file2.sql"},
			wantErr: true,
		},
		{
			name:    "migrations are correct",
			dir:     dir,
			files:   []string{"001-file.sql", "002-file.sql"},
			wantErr: false,
		},
	}
	fakeMigrate := Migrate{}
	fakeMigrate.conn = &pgx.Conn{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.files {
				f, _ := os.Create(filepath.Join(tt.dir, tt.files[i]))
				defer os.Remove(f.Name())
			}

			err := fakeMigrate.LoadMigrations(tt.dir)
			if (err == nil) == tt.wantErr {
				t.Errorf("function: LoadMigrations\n case %s expectations was not met", tt.name)
			}
			if err != nil {
				t.Logf("returning error: %v", err)
			} else {
				t.Logf("error was not expected")
			}

		})
	}
}
