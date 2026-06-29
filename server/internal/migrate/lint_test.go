package migrate

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestDownMigrationPath(t *testing.T) {
	got := downMigrationPath("migrations/001_users.sql")
	want := "migrations/001_users.down.sql"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsDownMigration(t *testing.T) {
	if !isDownMigration("001_users.down.sql") {
		t.Fatal("expected down migration")
	}
	if isDownMigration("001_users.sql") {
		t.Fatal("expected up migration")
	}
}

func TestRollbackSupported(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{"-- Rollback not supported: restore from backup\n", false},
		{"DROP TABLE foo;\n", true},
		{"-- comment only\n", false},
		{"/* block */ -- Rollback not supported\n", false},
	}
	for _, tc := range cases {
		if got := rollbackSupported([]byte(tc.body)); got != tc.want {
			t.Fatalf("body %q: got %v want %v", tc.body, got, tc.want)
		}
	}
}

func TestParseMigrationName_RejectsDown(t *testing.T) {
	if _, err := parseMigrationName("migrations/001_users.down.sql"); err == nil {
		t.Fatal("expected error for down migration")
	}
}

func TestLintFS_MissingDown(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/001_a.sql": &fstest.MapFile{Data: []byte("CREATE TABLE a (id INT);")},
	}
	res, err := LintFS(fsys, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 1 || !strings.Contains(res.Errors[0], "missing companion") {
		t.Fatalf("errors: %#v", res.Errors)
	}
}

func TestLintFS_DestructiveWarning(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/010_drop.sql":      &fstest.MapFile{Data: []byte("ALTER TABLE t DROP COLUMN old;")},
		"migrations/010_drop.down.sql": &fstest.MapFile{Data: []byte("-- stub\n")},
	}
	res, err := LintFS(fsys, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected destructive warning")
	}
}

func TestLintFS_UnbatchedUpdateWarning(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/020_backfill.sql":      &fstest.MapFile{Data: []byte("UPDATE big SET x = 1;")},
		"migrations/020_backfill.down.sql": &fstest.MapFile{Data: []byte("-- stub\n")},
	}
	res, err := LintFS(fsys, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "full-table UPDATE") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("warnings: %#v", res.Warnings)
	}
}

func TestLintFS_SecretDetection(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/030_bad.sql":      &fstest.MapFile{Data: []byte("SELECT 1; -- postgres://user:pass@host/db")},
		"migrations/030_bad.down.sql": &fstest.MapFile{Data: []byte("-- stub\n")},
	}
	res, err := LintFS(fsys, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected secret error")
	}
}

func TestFormatLintReport(t *testing.T) {
	out := FormatLintReport(LintResult{
		Errors:   []string{"missing down"},
		Warnings: []string{"DROP COLUMN"},
	})
	if !strings.Contains(out, "ERROR:") || !strings.Contains(out, "WARNING:") {
		t.Fatalf("report: %q", out)
	}
}
