package gradecomment

import "testing"

func TestParseLegacyFlat_authorPrefix(t *testing.T) {
	got := ParseLegacyFlat("User 1: Nice work.\n\nStudent: Student reply")
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].DisplayName != "User 1" || got[0].Body != "Nice work." {
		t.Fatalf("first=%+v", got[0])
	}
	if got[1].DisplayName != "Student" || got[1].Body != "Student reply" {
		t.Fatalf("second=%+v", got[1])
	}
}

func TestAppend_setsCreatedAtAndFlat(t *testing.T) {
	_, raw, flat, err := Append(nil, Comment{
		DisplayName: "Prof Kim",
		Body:        "Strong analysis.",
		Source:      "lextures",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("expected json")
	}
	if flat == nil || *flat != "Prof Kim: Strong analysis." {
		t.Fatalf("flat=%v", flat)
	}
}