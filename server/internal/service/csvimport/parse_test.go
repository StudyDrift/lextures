package csvimport

import (
	"strings"
	"testing"
)

func TestParseCSV_LexturesNative(t *testing.T) {
	csv := `email,first_name,last_name,role,external_id
good@example.edu,Jane,Smith,teacher,T001
bad-email,Jim,Bob,student,S002
missing@example.edu,Ann,Lee,invalidrole,X1
`
	res, err := ParseCSV(strings.NewReader(csv), ProfileLexturesNative)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("rows: got %d want 1", len(res.Rows))
	}
	if len(res.Errors) != 2 {
		t.Fatalf("errors: got %d want 2", len(res.Errors))
	}
	row := res.Rows[0]
	if row.Email != "good@example.edu" || row.ExternalID != "T001" || row.Role != "teacher" {
		t.Fatalf("row: %+v", row)
	}
}

func TestParseCSV_OneRoster(t *testing.T) {
	csv := `sourcedId,givenName,familyName,role,email
OR-1,Pat,Lee,student,pat@example.edu
`
	res, err := ParseCSV(strings.NewReader(csv), ProfileOneRosterV12)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("rows: %d", len(res.Rows))
	}
	if res.Rows[0].ExternalID != "OR-1" {
		t.Fatalf("external_id: %q", res.Rows[0].ExternalID)
	}
}

func TestSanitizeField(t *testing.T) {
	if got := sanitizeField("=cmd"); got != "'=cmd" {
		t.Fatalf("got %q", got)
	}
}

func TestParseMergeStrategy(t *testing.T) {
	for _, s := range []string{"create_only", "upsert", "sync"} {
		if _, err := ParseMergeStrategy(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	if _, err := ParseMergeStrategy("nope"); err == nil {
		t.Fatal("expected error")
	}
}
