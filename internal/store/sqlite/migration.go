package sqlite

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Migration is one forward-only schema change.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// LoadMigrations reads "<version>_<name>.sql" files from the root of fsys,
// sorted by version. Versions must be positive and unique.
func LoadMigrations(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []Migration
	seen := map[int]string{}
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".sql" {
			continue
		}
		version, name, err := parseMigrationName(e.Name())
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("migration version %d used by %s and %s", version, prev, e.Name())
		}
		seen[version] = e.Name()

		body, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: version, Name: name, SQL: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func parseMigrationName(file string) (int, string, error) {
	base := strings.TrimSuffix(file, ".sql")
	num, name, ok := strings.Cut(base, "_")
	if !ok || name == "" {
		return 0, "", fmt.Errorf("migration %q: want <version>_<name>.sql", file)
	}
	version, err := strconv.Atoi(num)
	if err != nil || version <= 0 {
		return 0, "", fmt.Errorf("migration %q: version must be a positive integer", file)
	}
	return version, name, nil
}
