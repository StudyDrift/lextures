package csvimport

import (
	"strings"
	"testing"
)

func TestParseCSVWithCustomColumns(t *testing.T) {
	csv := `email,first_name,last_name,role,student_id
a@example.com,Ada,Lovelace,student,12345
`
	res, err := ParseCSVWithExtraColumns(strings.NewReader(csv), ProfileLexturesNative, []string{"student_id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(res.Rows))
	}
	if res.Rows[0].CustomFields["student_id"] != "12345" {
		t.Fatalf("expected custom field value, got %#v", res.Rows[0].CustomFields)
	}
}
