// Package transcriptpdf renders official/unofficial academic transcript PDFs (T01).
package transcriptpdf

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"sort"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/jung-kurt/gofpdf"
	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

// Options controls tamper-evidence embedding on official PDFs (T08).
type Options struct {
	VerificationURL string
	ContentHash     string
	ShortCode       string
}

// BuildPDF renders an academic record to PDF bytes.
// Unofficial and preview variants receive a diagonal UNOFFICIAL watermark.
func BuildPDF(rec *academicrecord.AcademicRecord, opts ...Options) ([]byte, error) {
	if rec == nil {
		return nil, fmt.Errorf("transcriptpdf: nil record")
	}
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
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
	pdf.SetAutoPageBreak(true, 28)
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
		if err := embedVerificationFooter(pdf, opt); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func embedVerificationFooter(pdf *gofpdf.Fpdf, opt Options) error {
	url := strings.TrimSpace(opt.VerificationURL)
	hash := strings.TrimSpace(opt.ContentHash)
	code := strings.TrimSpace(opt.ShortCode)
	if url == "" && hash == "" {
		return nil
	}
	pdf.Ln(6)
	if url != "" {
		if err := embedQR(pdf, url); err != nil {
			return err
		}
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(140, 4, fmt.Sprintf("Verify at %s", url), "", "L", false)
		if code != "" {
			pdf.SetFont("Helvetica", "B", 8)
			pdf.CellFormat(0, 4, fmt.Sprintf("Verification code: %s", code), "", 1, "L", false, 0, "")
		}
	}
	if hash != "" {
		pdf.SetFont("Helvetica", "", 7)
		pdf.MultiCell(0, 3.5, fmt.Sprintf("Content hash (SHA-256): %s", hash), "", "L", false)
	}
	return nil
}

func embedQR(pdf *gofpdf.Fpdf, link string) error {
	code, err := qr.Encode(link, qr.M, qr.Auto)
	if err != nil {
		return err
	}
	code, err = barcode.Scale(code, 120, 120)
	if err != nil {
		return err
	}
	// gofpdf only accepts 8-bit PNG; barcode may produce a paletted/16-bit source.
	bounds := code.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, code.At(x, y))
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, rgba); err != nil {
		return err
	}
	name := "transcript_verify_qr"
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(pngBuf.Bytes()))
	y := pdf.GetY()
	pdf.ImageOptions(name, 160, y, 30, 30, false, opt, 0, "")
	return nil
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
