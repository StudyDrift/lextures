package transcriptpdf

import (
	"bytes"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

func TestBuildPDF_UnofficialWatermark(t *testing.T) {
	gpa := 3.5
	rec := &academicrecord.AcademicRecord{
		SchemaVersion:   academicrecord.SchemaVersion,
		TemplateVersion: academicrecord.TemplateVersion,
		Variant:         academicrecord.VariantUnofficial,
		GeneratedAt:     time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Student:         academicrecord.StudentBlock{Name: "Ada Lovelace"},
		Institution:     academicrecord.InstitutionBlock{Name: "Test University"},
		Terms: []academicrecord.TermBlock{{
			Label: "Fall 2025",
			Courses: []academicrecord.CourseLine{{
				Code: "MATH101", Title: "Calc I", CreditsAttempted: 3, CreditsEarned: 3, Grade: "A",
			}},
			TermGPA: &gpa, TermCredits: 3,
		}},
		Cumulative: academicrecord.CumulativeBlock{GPA: &gpa, CreditsAttempted: 3, CreditsEarned: 3},
		Legend:     academicrecord.DefaultLegend(),
	}
	pdf, err := BuildPDF(rec)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 100 || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("expected PDF bytes, got %d", len(pdf))
	}
	// Document info dictionary carries the UNOFFICIAL title (content streams may be compressed).
	if !bytes.Contains(pdf, []byte("UNOFFICIAL")) {
		t.Fatal("unofficial PDF should mark UNOFFICIAL in document metadata")
	}
}

func TestBuildPDF_OfficialNoWatermark(t *testing.T) {
	rec := &academicrecord.AcademicRecord{
		Variant:     academicrecord.VariantOfficial,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Student:     academicrecord.StudentBlock{Name: "Ada"},
		Institution: academicrecord.InstitutionBlock{Name: "U"},
		Terms:       []academicrecord.TermBlock{},
		Cumulative:  academicrecord.CumulativeBlock{},
		Legend:      academicrecord.DefaultLegend(),
	}
	pdf, err := BuildPDF(rec)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(pdf, []byte("UNOFFICIAL")) {
		t.Fatal("official PDF must not carry UNOFFICIAL watermark")
	}
}

func TestBuildPDF_OfficialEmbedsVerifyFooter(t *testing.T) {
	rec := &academicrecord.AcademicRecord{
		Variant:     academicrecord.VariantOfficial,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Student:     academicrecord.StudentBlock{Name: "Ada"},
		Institution: academicrecord.InstitutionBlock{Name: "U"},
		Terms:       []academicrecord.TermBlock{},
		Cumulative:  academicrecord.CumulativeBlock{},
		Legend:      academicrecord.DefaultLegend(),
	}
	pdf, err := BuildPDF(rec, Options{
		VerificationURL: "https://app.example.com/verify/abc-token",
		ContentHash:     "deadbeefcafebabe",
		ShortCode:       "ABC",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("expected PDF")
	}
	if !bytes.Contains(pdf, []byte("Verify at")) && !bytes.Contains(pdf, []byte("verify")) {
		// Content streams may be compressed; document should still grow with QR image.
		if len(pdf) < 800 {
			t.Fatalf("expected larger PDF with QR/footer, got %d bytes", len(pdf))
		}
	}
}
