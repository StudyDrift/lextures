package transcriptpesc

import (
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

func TestParseXML_RoundTrip(t *testing.T) {
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
	parsed, err := ParseXML(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Student.Name != "Ada Lovelace" {
		t.Fatalf("student name: %q", parsed.Student.Name)
	}
	if parsed.Institution.Name != "Test University" {
		t.Fatalf("school: %q", parsed.Institution.Name)
	}
	if len(parsed.Terms) != 1 || len(parsed.Terms[0].Courses) != 1 {
		t.Fatalf("terms/courses: %+v", parsed.Terms)
	}
	if parsed.Terms[0].Courses[0].Code != "MATH101" || parsed.Terms[0].Courses[0].Grade != "A" {
		t.Fatalf("course: %+v", parsed.Terms[0].Courses[0])
	}
	if !parsed.Terms[0].Courses[0].Transfer {
		t.Fatal("expected transfer flag on inbound-mapped courses")
	}
}

func TestParseXML_RejectsXXE(t *testing.T) {
	xxe := []byte(`<?xml version="1.0"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<CollegeTranscript xmlns="urn:org:pesc:message:CollegeTranscript:v1.0.0">
  <TransmissionData>
    <DocumentID>1</DocumentID>
    <CreatedDateTime>2026-01-01T00:00:00Z</CreatedDateTime>
    <DocumentTypeCode>CollegeTranscript</DocumentTypeCode>
  </TransmissionData>
  <Student>
    <Person><Name><CompositeName>&xxe;</CompositeName></Name></Person>
    <AcademicRecord><School><OrganizationName>Evil U</OrganizationName></School></AcademicRecord>
  </Student>
</CollegeTranscript>`)
	// Go's encoding/xml does not expand external entities; parse should not leak file contents.
	rec, err := ParseXML(xxe)
	if err == nil && rec != nil && strings.Contains(rec.Student.Name, "root:") {
		t.Fatal("XXE expanded unexpectedly")
	}
}

func TestParseXML_RejectsOversized(t *testing.T) {
	big := make([]byte, MaxXMLBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	if _, err := ParseXML(big); err == nil {
		t.Fatal("expected oversized error")
	}
}

func TestParseXML_MissingName(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?>
<CollegeTranscript xmlns="urn:org:pesc:message:CollegeTranscript:v1.0.0">
  <TransmissionData>
    <DocumentID>1</DocumentID>
    <CreatedDateTime>2026-01-01T00:00:00Z</CreatedDateTime>
    <DocumentTypeCode>CollegeTranscript</DocumentTypeCode>
  </TransmissionData>
  <Student>
    <Person><Name></Name></Person>
    <AcademicRecord><School><OrganizationName>Test U</OrganizationName></School></AcademicRecord>
  </Student>
</CollegeTranscript>`)
	if _, err := ParseXML(raw); err == nil {
		t.Fatal("expected missing name error")
	}
}
