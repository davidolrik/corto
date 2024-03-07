package migrations

import (
	"strings"
	"testing"
)

func TestMigrationsAreEmbedded(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("reading embedded migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded migration files")
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			t.Errorf("unexpected non-SQL file embedded: %s", entry.Name())
		}
	}
}
