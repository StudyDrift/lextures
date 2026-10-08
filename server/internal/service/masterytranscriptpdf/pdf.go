package masterytranscriptpdf

import (
	"bytes"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// Row is one standard on a mastery transcript.
type Row struct {
	Code         string
	Description  string
	Level        string
	LastAssessed *time.Time
}

// Input is the data rendered into a mastery transcript PDF.
type Input struct {
	CourseTitle string
	CourseCode  string
	StudentName string
	GeneratedAt time.Time
	Rows        []Row
}

const (
	pdfMargin = 15.0
	pdfPageW  = 215.9
	contentW  = pdfPageW - pdfMargin*2
)

// Build renders a per-student standards mastery transcript.
func Build(in Input) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(true, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	when := in.GeneratedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(contentW, 8, tr("Mastery Transcript"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	title := strings.TrimSpace(in.CourseTitle)
	if title == "" {
		title = in.CourseCode
	} else if strings.TrimSpace(in.CourseCode) != "" {
		title = title + " (" + in.CourseCode + ")"
	}
	pdf.CellFormat(contentW, 6, tr(title), "", 1, "L", false, 0, "")
	pdf.CellFormat(contentW, 6, tr("Student: "+strings.TrimSpace(in.StudentName)), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(contentW, 5, tr("Generated "+when.UTC().Format("Jan 2, 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Standard | Description | Level | Assessed
	colW := []float64{36, 95.9, 28, 26}
	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		headers := []string{"Standard", "Description", "Level", "Assessed"}
		for i, h := range headers {
			pdf.CellFormat(colW[i], 7, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
	}
	drawHeader()

	pdf.SetFont("Helvetica", "", 8)
	if len(in.Rows) == 0 {
		pdf.CellFormat(contentW, 8, tr("No standards are attached to this course."), "1", 1, "L", false, 0, "")
	}
	for _, row := range in.Rows {
		code := dashIfEmpty(row.Code)
		level := dashIfEmpty(row.Level)
		assessed := "-"
		if row.LastAssessed != nil && !row.LastAssessed.IsZero() {
			assessed = row.LastAssessed.UTC().Format("2006-01-02")
		}
		desc := strings.TrimSpace(row.Description)
		if desc == "" {
			desc = "-"
		}
		lineH := 4.2
		h := lineH
		for i, text := range []string{code, desc, level, assessed} {
			n := len(pdf.SplitLines([]byte(tr(text)), colW[i]-2))
			if n < 1 {
				n = 1
			}
			if cell := float64(n) * lineH; cell > h {
				h = cell
			}
		}
		if h < 7 {
			h = 7
		}
		if pdf.GetY()+h > 250 {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont("Helvetica", "", 8)
		}
		x, y := pdf.GetX(), pdf.GetY()
		pdf.Rect(x, y, colW[0], h, "D")
		pdf.Rect(x+colW[0], y, colW[1], h, "D")
		pdf.Rect(x+colW[0]+colW[1], y, colW[2], h, "D")
		pdf.Rect(x+colW[0]+colW[1]+colW[2], y, colW[3], h, "D")
		pdf.SetXY(x+1, y+1)
		pdf.MultiCell(colW[0]-2, lineH, tr(code), "", "L", false)
		pdf.SetXY(x+colW[0]+1, y+1)
		pdf.MultiCell(colW[1]-2, lineH, tr(desc), "", "L", false)
		pdf.SetXY(x+colW[0]+colW[1]+1, y+1)
		pdf.MultiCell(colW[2]-2, lineH, tr(level), "", "L", false)
		pdf.SetXY(x+colW[0]+colW[1]+colW[2]+1, y+1)
		pdf.MultiCell(colW[3]-2, lineH, tr(assessed), "", "L", false)
		pdf.SetXY(x, y+h)
	}
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dashIfEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
