// Package transcriptpdf renders official/unofficial academic transcript PDFs (T01).
package transcriptpdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

// BuildPDF renders an academic record to PDF bytes.
// Unofficial and preview variants receive a diagonal UNOFFICIAL watermark.
func BuildPDF(rec *academicrecord.AcademicRecord) ([]byte, error) {
	if rec == nil {
		return nil, fmt.Errorf("transcriptpdf: nil record")
	}
	pdf := gofpdf.New("P", "mm", "Letter", "")
	docTitle := "Official Academic Transcript"
	if rec.Variant != academicrecord.VariantOfficial {
		docTitle = "UNOFFICIAL Academic Transcript"
	}
	pdf.SetTitle(docTitle, false)
	pdf.SetAuthor(rec.Institution.Name, false)
	pdf.SetCreator("Lextures", false)
	pdf.SetSubject(docTitle, false)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	unofficial := rec.Variant != academicrecord.VariantOfficial
	if unofficial {
		stampUnofficialWatermark(pdf)
	}

	// Institution header / letterhead block
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 8, rec.Institution.Name, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 13)
	title := "Official Academic Transcript"
	if unofficial {
		title = "UNOFFICIAL Academic Transcript"
		pdf.SetTextColor(160, 0, 0)
	}
	if rec.Variant == academicrecord.VariantPartial {
		title = "Partial Academic Transcript"
	}
	if rec.Variant == academicrecord.VariantInProgress {
		title = "In-Progress Academic Transcript"
		if unofficial {
			pdf.SetTextColor(160, 0, 0)
		}
	}
	pdf.CellFormat(0, 7, title, "", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	if unofficial {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(160, 0, 0)
		pdf.CellFormat(0, 6, "UNOFFICIAL — NOT AN OFFICIAL RECORD", "", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 5, fmt.Sprintf("Student: %s", rec.Student.Name), "", 1, "L", false, 0, "")
	if rec.Student.StudentID != "" {
		pdf.CellFormat(0, 5, fmt.Sprintf("Student ID: %s", rec.Student.StudentID), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated: %s", rec.GeneratedAt), "", 1, "L", false, 0, "")
	if rec.Standing != "" {
		pdf.CellFormat(0, 5, fmt.Sprintf("Academic standing: %s", rec.Standing), "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	for _, term := range rec.Terms {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 6, term.Label, "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 8)
		colCode, colTitle, colAtt, colErn, colGr, colQP := 22.0, 80.0, 18.0, 18.0, 14.0, 0.0
		pdf.CellFormat(colCode, 5, "Code", "1", 0, "L", false, 0, "")
		pdf.CellFormat(colTitle, 5, "Title", "1", 0, "L", false, 0, "")
		pdf.CellFormat(colAtt, 5, "Att", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colErn, 5, "Ern", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colGr, 5, "Gr", "1", 0, "C", false, 0, "")
		pdf.CellFormat(colQP, 5, "QP", "1", 1, "R", false, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		for _, c := range term.Courses {
			qp := ""
			if c.QualityPoints != nil {
				qp = fmt.Sprintf("%.1f", *c.QualityPoints)
			}
			pdf.CellFormat(colCode, 5, truncate(c.Code, 12), "1", 0, "L", false, 0, "")
			pdf.CellFormat(colTitle, 5, truncate(c.Title, 48), "1", 0, "L", false, 0, "")
			pdf.CellFormat(colAtt, 5, fmt.Sprintf("%.1f", c.CreditsAttempted), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colErn, 5, fmt.Sprintf("%.1f", c.CreditsEarned), "1", 0, "R", false, 0, "")
			pdf.CellFormat(colGr, 5, c.Grade, "1", 0, "C", false, 0, "")
			pdf.CellFormat(colQP, 5, qp, "1", 1, "R", false, 0, "")
		}
		pdf.SetFont("Helvetica", "", 9)
		termLine := fmt.Sprintf("Term credits earned: %.2f", term.TermCredits)
		if term.TermGPA != nil {
			termLine += fmt.Sprintf("  |  Term GPA: %.3f", *term.TermGPA)
		}
		pdf.CellFormat(0, 5, termLine, "", 1, "R", false, 0, "")
		pdf.Ln(2)
	}

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "Cumulative totals", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 5, fmt.Sprintf("Credits attempted: %.2f", rec.Cumulative.CreditsAttempted), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Credits earned: %.2f", rec.Cumulative.CreditsEarned), "", 1, "L", false, 0, "")
	if rec.Cumulative.GPA != nil {
		pdf.CellFormat(0, 5, fmt.Sprintf("Cumulative GPA: %.3f", *rec.Cumulative.GPA), "", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	if len(rec.Legend) > 0 {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(0, 5, "Grade legend", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		keys := sortedKeys(rec.Legend)
		for _, k := range keys {
			pdf.CellFormat(0, 4, fmt.Sprintf("%s — %s", k, rec.Legend[k]), "", 1, "L", false, 0, "")
		}
	}

	if rec.Variant == academicrecord.VariantOfficial {
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.MultiCell(0, 4, "This official transcript is sealed by the issuing institution. Alteration voids the document.", "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func stampUnofficialWatermark(pdf *gofpdf.Fpdf) {
	pdf.SetFont("Helvetica", "B", 48)
	pdf.SetTextColor(200, 200, 200)
	// Diagonal watermark across the page body.
	pdf.TransformBegin()
	pdf.TransformRotate(35, 50, 140)
	pdf.Text(30, 140, "UNOFFICIAL")
	pdf.TransformEnd()
	pdf.SetTextColor(0, 0, 0)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
