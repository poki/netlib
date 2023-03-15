package migrations

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestMigrations tests if the latest.lock file contains the name of the latest
// migration file. This to ensure that we active merge conflicts when multiple
// branches are adding migration files.
func TestLatestLock(t *testing.T) {
	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatal(err)
	}

	var reg = regexp.MustCompile(`^([0-9]+)_(.*)\.(down|up)\.(.*)$`)
	var highestVersion uint64
	var highest string

	seen := make(map[string]struct{})

	for _, file := range files {
		m := reg.FindStringSubmatch(file)

		if len(m) == 5 {
			version, err := strconv.ParseUint(m[1], 10, 64)
			if err != nil {
				t.Fatal(err)
			}

			timeDirection := m[1] + m[3]
			if _, ok := seen[timeDirection]; ok {
				t.Fatalf("duplicate %q", timeDirection)
			} else {
				seen[timeDirection] = struct{}{}
			}

			if version > highestVersion {
				highestVersion = version
				highest = m[1] + "_" + m[2]
			}
		}
	}

	data, err := os.ReadFile("latest.lock")
	if err != nil {
		t.Fatal(err)
	}

	latest := strings.TrimSpace(string(data))

	if latest != highest {
		t.Fatalf("latest.lock is wrong, %q != %q", latest, highest)
	}
}
