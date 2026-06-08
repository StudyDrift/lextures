package gradeexport

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateCSV(t *testing.T) {
	grades := []StudentGrade{
		{
			UserID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			EnrollmentID:  uuid.MustParse("00000000-0000-0000-0000-000000000010"),
			DisplayName:   "Alice Smith",
			ExternalSISID: "SRN001",
			State:         "active",
			ComputedGrade: "A",
			FinalGrade:    "A",
		},
		{
			UserID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			EnrollmentID:  uuid.MustParse("00000000-0000-0000-0000-000000000020"),
			DisplayName:   "Bob Jones",
			ExternalSISID: "",
			State:         "withdrawn",
			ComputedGrade: "W",
			FinalGrade:    "W",
		},
	}
	data, err := GenerateCSV(grades)
	if err != nil {
		t.Fatalf("GenerateCSV: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty CSV output")
	}
	s := string(data)
	if !strings.Contains(s, "StudentID") {
		t.Error("missing CSV header StudentID")
	}
	if !strings.Contains(s, "SRN001") {
		t.Error("missing external SIS ID in CSV")
	}
	if !strings.Contains(s, "Alice Smith") {
		t.Error("missing student name in CSV")
	}
	if !strings.Contains(s, "withdrawn") {
		t.Error("missing enrollment state in CSV")
	}
	// Bob has no ExternalSISID; should fall back to UUID
	if !strings.Contains(s, "00000000-0000-0000-0000-000000000002") {
		t.Error("expected UUID fallback for Bob who has no external SIS ID")
	}
}

func TestGenerateCSV_OverrideGrade(t *testing.T) {
	uid := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
	grades := []StudentGrade{
		{
			UserID:        uid,
			EnrollmentID:  uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000010"),
			DisplayName:   "Carol",
			ExternalSISID: "SRN999",
			State:         "active",
			ComputedGrade: "B",
			FinalGrade:    "B+", // instructor override
		},
	}
	data, err := GenerateCSV(grades)
	if err != nil {
		t.Fatalf("GenerateCSV: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "B+") {
		t.Errorf("expected overridden grade B+ in CSV, got: %s", s)
	}
}

func TestGenerateCSV_Empty(t *testing.T) {
	data, err := GenerateCSV(nil)
	if err != nil {
		t.Fatalf("GenerateCSV with no grades: %v", err)
	}
	s := string(data)
	// Header row should always be present.
	if !strings.Contains(s, "StudentID") {
		t.Errorf("expected header row in empty CSV, got: %q", s)
	}
}

func TestParsePoints(t *testing.T) {
	cases := []struct {
		in  string
		out float64
	}{
		{"", 0},
		{"10", 10},
		{"9.5", 9.5},
		{"bad", 0},
		{"0", 0},
	}
	for _, tc := range cases {
		got := parsePoints(tc.in)
		if got != tc.out {
			t.Errorf("parsePoints(%q) = %v, want %v", tc.in, got, tc.out)
		}
	}
}
