package migrate

import "testing"

func TestMigrate_Migrator_BaseConn(t *testing.T) {
	tests := []struct {
		caseName string
		connStr  string
		wantErr  bool
	}{{
		caseName: "database is not responding",
		connStr:  "",
		wantErr:  true,
	}, {
		caseName: "invalid connection string",
		connStr:  "",
		wantErr:  false,
	}}
	for _, tt := range tests {
		t.Run(tt.caseName, func(t *testing.T) {

		})
	}

}
