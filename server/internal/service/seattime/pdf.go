package seattime

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	reposeattime "github.com/lextures/lextures/server/internal/repos/seattime"
)

// TranscriptRow is one CEU entry on the learner transcript.
type TranscriptRow struct {
	CourseTitle  string
	CEUCredit    float64
	ContactHours float64
	CompletedAt  time.Time
}

// TranscriptInput describes a CE transcript PDF export.
type TranscriptInput struct {
	InstitutionName string
	LearnerName     string
	GeneratedAt     time.Time
	Rows            []TranscriptRow
}

// BuildTranscriptPDF renders an accessible CE transcript PDF.
func BuildTranscriptPDF(in TranscriptInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Continuing Education Transcript", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "Continuing Education Transcript", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	institution := strings.TrimSpace(in.InstitutionName)
	if institution == "" {
		institution = "Lextures"
	}
	pdf.CellFormat(0, 7, institution, "", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Learner: %s", in.LearnerName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s", in.GeneratedAt.UTC().Format(time.RFC3339)), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(70, 7, "Course", "1", 0, "L", false, 0, "")
	pdf.CellFormat(25, 7, "CEU Credit", "1", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, "Contact Hrs", "1", 0, "R", false, 0, "")
	pdf.CellFormat(0, 7, "Completed", "1", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, row := range in.Rows {
		pdf.CellFormat(70, 6, row.CourseTitle, "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", row.CEUCredit), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.1f", row.ContactHours), "1", 0, "R", false, 0, "")
		pdf.CellFormat(0, 6, row.CompletedAt.UTC().Format("2006-01-02"), "1", 1, "L", false, 0, "")
	}
	if len(in.Rows) == 0 {
		pdf.CellFormat(0, 6, "No CEU awards on record.", "1", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildTranscriptRows maps awards to display rows.
func BuildTranscriptRows(awards []reposeattime.CEUAward, titles map[string]string) []TranscriptRow {
	rows := make([]TranscriptRow, 0, len(awards))
	for _, a := range awards {
		title := titles[a.CourseID.String()]
		if title == "" {
			title = "Course"
		}
		rows = append(rows, TranscriptRow{
			CourseTitle:  title,
			CEUCredit:    a.CEUCredit,
			ContactHours: a.ContactHours,
			CompletedAt:  a.IssuedAt,
		})
	}
	return rows
}

// BuildCertificatePDF renders a CEU completion certificate.
func BuildCertificatePDF(institution, learnerName, courseTitle string, ceuCredit, contactHours float64, issuedAt time.Time) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetTitle("CEU Certificate", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(0, 14, "Certificate of Completion", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	if institution == "" {
		institution = "Lextures"
	}
	pdf.CellFormat(0, 8, institution, "", 1, "C", false, 0, "")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, "This certifies that", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, learnerName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, fmt.Sprintf("has completed %s", courseTitle), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("and earned %.2f CEU (%.1f contact hours)", ceuCredit, contactHours), "", 1, "C", false, 0, "")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Issued: %s", issuedAt.UTC().Format("January 2, 2006")), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
