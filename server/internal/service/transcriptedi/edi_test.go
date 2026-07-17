package transcriptedi

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

func TestBuildTS130AndValidate(t *testing.T) {
	t.Parallel()
	gpa := 3.5
	rec := &academicrecord.AcademicRecord{
		GeneratedAt: "2026-07-17T12:00:00Z",
		Student:     academicrecord.StudentBlock{Name: "Ada Lovelace", StudentID: "S1"},
		Institution: academicrecord.InstitutionBlock{Name: "State University", CeebActID: "1234"},
		Terms: []academicrecord.TermBlock{{
			Label: "Fall 2025",
			Courses: []academicrecord.CourseLine{{
				Code: "MATH101", Title: "Calc", Grade: "A",
				CreditsAttempted: 3, CreditsEarned: 3,
			}},
		}},
		Cumulative: academicrecord.CumulativeBlock{
			CreditsAttempted: 3, CreditsEarned: 3, GPA: &gpa,
		},
	}
	out, err := BuildTS130(rec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "ST*130*") {
		t.Fatalf("missing ST*130*: %s", s)
	}
	if err := ValidateStructure(out); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStructureRejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := ValidateStructure([]byte("NOPE")); err == nil {
		t.Fatal("expected error")
	}
}
