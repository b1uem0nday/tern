package migrate

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

//template - 0001-init.sql
var filePattern = regexp.MustCompile(`\A(\d{1,4})-.+\.sql\z`)

func (m *Migrate) LoadMigrationWithDefaultPath() error {
	log.Printf("use default path: %s", defMigrationPath)
	return m.LoadMigrations(defMigrationPath)
}

/*
	LoadMigrations - opens the specified folder, sorts and loads all pattern-related files to Migrations
*/
func (m *Migrate) LoadMigrations(path string) error {
	path = path + m.scheme
	if err := checkPathExistence(path); err != nil {
		return err
	}

	paths, err := loadValidPaths(path)
	if err != nil {
		return err
	}
	for _, p := range paths {
		body, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		m.appendMigration(filepath.Base(p), string(body))
	}

	return nil
}

func loadValidPaths(path string) ([]string, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
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
			return nil, err
		}

		if n < int64(len(paths)+1) {
			return nil, fmt.Errorf("duplicate migration %d", n)
		}

		if int64(len(paths)+1) < n {
			return nil, fmt.Errorf("missing migration %d", len(paths)+1)
		}

		paths = append(paths, filepath.Join(path, fi.Name()))
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no migrations found at %s", path)
	}
	return paths, err
}
func SetDefaultPath(path string) error {
	if err := checkPathExistence(path); err != nil {
		return err
	}
	defMigrationPath = path
	return nil
}
func checkPathExistence(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	return nil
}
