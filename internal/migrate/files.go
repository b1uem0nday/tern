package migrate

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
)

//template - 0001-init.sql
var filePattern = regexp.MustCompile(`\A(\d{1,4})-.+\.sql\z`)

/*
LoadMigrations - opens the specified folder, sorts and loads all pattern-related files to Migrations
*/
func (m *Migrate) LoadMigrations(path string) error {
	path = path + m.conn.Config().Database

	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(fileInfos))
	for _, fi := range fileInfos {
		if fi.IsDir() {
			continue
		}

		matches := filePattern.FindStringSubmatch(fi.Name())
		if len(matches) != 2 {
			continue
		}

		n, err := strconv.ParseInt(matches[1], 10, 32)
		if err != nil {
			// The regexp already validated that the prefix is all digits so this *should* never fail
			return err
		}

		if n < int64(len(paths)+1) {
			return fmt.Errorf("duplicate migration %d", n)
		}

		if int64(len(paths)+1) < n {
			return fmt.Errorf("missing migration %d", len(paths)+1)
		}

		paths = append(paths, filepath.Join(path, fi.Name()))
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
