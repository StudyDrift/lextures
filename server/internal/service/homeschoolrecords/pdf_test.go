package homeschoolrecords

import (
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/repos/homeschoolrecords"
)

func TestBuildHoursLogPDF(t *testing.T) {
	log := HoursLog{
		From: "2025-08-01", To: "2026-07-31", SchoolYear: "2025-2026",
		Entries: []LogRow{{Date: "2025-09-01", Subject: "Reading", Hours: 1.25, Source: "offline", Editable: true, Notes: "Library"}},
		TotalsBySubject: []SubjectTotal{{Subject: "Reading", Hours: 1.25}},
		TotalHours: 1.25,
	}
	b, err := BuildHoursLogPDF(HoursLogPDFInput{
		SchoolName: "Willden Academy", StudentName: "Kid One", SchoolYear: "2025-2026",
		From: log.From, To: log.To, GeneratedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Log: log,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 || !strings.HasPrefix(string(b), "%PDF") {
		t.Fatalf("not a pdf: %d bytes", len(b))
	}
}

func TestBuildTranscriptPDF_Signature(t *testing.T) {
	b, err := BuildTranscriptPDF(TranscriptPDFInput{
		SchoolName: "Willden Academy", StudentName: "Kid One", SchoolYear: "2025-2026",
		GeneratedAt: time.Now().UTC(),
		Courses: []homeschoolrecords.TranscriptCourse{{
			CourseCode: "MATH1", Title: "Math", Credits: 1, FinalGrade: "A", TermLabel: "Fall",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 || !strings.HasPrefix(string(b), "%PDF") {
		t.Fatalf("not a pdf")
	}
}

func TestBuildPortfolioPDFAndZIP(t *testing.T) {
	pdf, err := BuildPortfolioPDF(PortfolioPDFInput{
		SchoolName: "Willden Academy", StudentName: "Kid", SchoolYear: "2025-2026",
		Samples: []homeschoolrecords.SubmissionSample{{
			ItemTitle: "Essay", CourseTitle: "Writing", CourseCode: "WR1",
			SubmittedAt: time.Now().UTC(), BodyPreview: "Hello", InstructorComment: "Nice work",
		}},
		HoursTotals: []SubjectTotal{{Subject: "Writing", Hours: 3}},
		TotalHours:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	zipBytes, err := BuildPortfolioZIP(pdf, HoursLog{Entries: []LogRow{{Date: "2025-09-01", Subject: "Writing", Hours: 3, Source: "activity"}}}, "kid")
	if err != nil {
		t.Fatal(err)
	}
	if len(zipBytes) < 50 {
		t.Fatalf("zip too small")
	}
	csvBytes, err := BuildHoursLogCSV(HoursLog{Entries: []LogRow{{Date: "2025-09-01", Subject: "Writing", Hours: 3, Source: "activity"}}})
	if err != nil || !strings.Contains(string(csvBytes), "Writing") {
		t.Fatalf("csv: %v %s", err, csvBytes)
	}
}
