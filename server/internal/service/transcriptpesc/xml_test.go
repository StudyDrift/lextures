package transcriptpesc

import (
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

func TestBuildXML_Validates(t *testing.T) {
	gpa := 3.5
	qp := 21.0
	rec := &academicrecord.AcademicRecord{
		SchemaVersion:   academicrecord.SchemaVersion,
		TemplateVersion: academicrecord.TemplateVersion,
		Variant:         academicrecord.VariantOfficial,
		GeneratedAt:     time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Student:         academicrecord.StudentBlock{Name: "Ada Lovelace", StudentID: "S1"},
		Institution:     academicrecord.InstitutionBlock{Name: "Test University"},
		Terms: []academicrecord.TermBlock{{
			Label: "Fall 2025",
			Courses: []academicrecord.CourseLine{{
				Code: "MATH101", Title: "Calc I", CreditsAttempted: 3, CreditsEarned: 3,
				Grade: "A", QualityPoints: &qp,
			}},
			TermGPA: &gpa, TermCredits: 3,
		}},
		Cumulative: academicrecord.CumulativeBlock{GPA: &gpa, CreditsAttempted: 3, CreditsEarned: 3, QualityPoints: 12},
		Legend:     academicrecord.DefaultLegend(),
	}
	raw, err := BuildXML(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStructure(raw); err != nil {
		t.Fatal(err)
	}
	if !contains(string(raw), "CollegeTranscript") || !contains(string(raw), "MATH") {
		t.Fatalf("unexpected xml: %s", raw)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
