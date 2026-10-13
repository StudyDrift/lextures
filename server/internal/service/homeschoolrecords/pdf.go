package homeschoolrecords

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/lextures/lextures/server/internal/repos/homeschoolrecords"
)

const (
	pdfMargin = 15.0
	pdfPageW  = 215.9
	contentW  = pdfPageW - pdfMargin*2
)

// HoursLogPDFInput drives the attendance/hours log PDF.
type HoursLogPDFInput struct {
	SchoolName   string
	StudentName  string
	SchoolYear   string
	From         string
	To           string
	GeneratedAt  time.Time
	Log          HoursLog
}

// BuildHoursLogPDF renders the daily hours log.
func BuildHoursLogPDF(in HoursLogPDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(true, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	when := in.GeneratedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}

	pdf.AddPage()
	school := strings.TrimSpace(in.SchoolName)
	if school == "" {
		school = "Homeschool"
	}
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(contentW, 8, tr(school), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(contentW, 7, tr("Attendance / Instructional Hours Log"), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(contentW, 5, tr("Student: "+strings.TrimSpace(in.StudentName)), "", 1, "L", false, 0, "")
	if strings.TrimSpace(in.SchoolYear) != "" {
		pdf.CellFormat(contentW, 5, tr("School year: "+in.SchoolYear), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(contentW, 5, tr(fmt.Sprintf("Range: %s – %s", in.From, in.To)), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(contentW, 5, tr("Generated "+when.UTC().Format("Jan 2, 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	colW := []float64{28, 55, 22, 28, contentW - 28 - 55 - 22 - 28}
	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		for i, h := range []string{"Date", "Subject", "Hours", "Source", "Notes"} {
			pdf.CellFormat(colW[i], 7, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
	}
	drawHeader()
	pdf.SetFont("Helvetica", "", 8)
	if len(in.Log.Entries) == 0 {
		pdf.CellFormat(contentW, 8, tr("No hours recorded in this range."), "1", 1, "L", false, 0, "")
	}
	for _, row := range in.Log.Entries {
		vals := []string{row.Date, row.Subject, fmt.Sprintf("%.2f", row.Hours), row.Source, row.Notes}
		h := rowHeight(pdf, tr, colW, vals, 4.0, 7)
		if pdf.GetY()+h > 250 {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont("Helvetica", "", 8)
		}
		x, y := pdf.GetX(), pdf.GetY()
		xx := x
		for i, w := range colW {
			pdf.Rect(xx, y, w, h, "D")
			pdf.SetXY(xx+1, y+1)
			pdf.MultiCell(w-2, 4.0, tr(vals[i]), "", "L", false)
			xx += w
		}
		pdf.SetXY(x, y+h)
	}

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(contentW, 6, tr(fmt.Sprintf("Total hours: %.2f", in.Log.TotalHours)), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(contentW, 6, tr("Totals by subject"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, t := range in.Log.TotalsBySubject {
		pdf.CellFormat(contentW, 5, tr(fmt.Sprintf("%s — %.2f hours", t.Subject, t.Hours)), "", 1, "L", false, 0, "")
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

// TranscriptPDFInput drives the year-end transcript PDF.
type TranscriptPDFInput struct {
	SchoolName  string
	StudentName string
	SchoolYear  string
	GeneratedAt time.Time
	Courses     []homeschoolrecords.TranscriptCourse
}

// BuildTranscriptPDF renders courses, grades, and credits with a signature line.
// Layout follows the Standards gradebook mastery transcript style (gofpdf Letter).
func BuildTranscriptPDF(in TranscriptPDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(true, 28)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	when := in.GeneratedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}

	pdf.AddPage()
	school := strings.TrimSpace(in.SchoolName)
	if school == "" {
		school = "Homeschool"
	}
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(contentW, 8, tr(school), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(contentW, 7, tr("Official Homeschool Transcript"), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(contentW, 5, tr("Student: "+strings.TrimSpace(in.StudentName)), "", 1, "L", false, 0, "")
	if strings.TrimSpace(in.SchoolYear) != "" {
		pdf.CellFormat(contentW, 5, tr("School year: "+in.SchoolYear), "", 1, "L", false, 0, "")
	}
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(contentW, 5, tr("Generated "+when.UTC().Format("Jan 2, 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	colW := []float64{28, 90, 22, 22, contentW - 28 - 90 - 22 - 22}
	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		for i, h := range []string{"Code", "Course", "Credits", "Grade", "Term"} {
			pdf.CellFormat(colW[i], 7, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
	}
	drawHeader()
	pdf.SetFont("Helvetica", "", 8)
	var creditSum float64
	if len(in.Courses) == 0 {
		pdf.CellFormat(contentW, 8, tr("No courses on record for this school year."), "1", 1, "L", false, 0, "")
	}
	for _, c := range in.Courses {
		creditSum += c.Credits
		vals := []string{c.CourseCode, c.Title, fmt.Sprintf("%.2f", c.Credits), c.FinalGrade, c.TermLabel}
		h := rowHeight(pdf, tr, colW, vals, 4.0, 7)
		if pdf.GetY()+h > 230 {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont("Helvetica", "", 8)
		}
		x, y := pdf.GetX(), pdf.GetY()
		xx := x
		for i, w := range colW {
			pdf.Rect(xx, y, w, h, "D")
			pdf.SetXY(xx+1, y+1)
			pdf.MultiCell(w-2, 4.0, tr(vals[i]), "", "L", false)
			xx += w
		}
		pdf.SetXY(x, y+h)
	}

	pdf.Ln(3)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(contentW, 6, tr(fmt.Sprintf("Total credits: %.2f", creditSum)), "", 1, "L", false, 0, "")
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(contentW*0.55, 6, tr("Parent / guardian signature:"), "", 0, "L", false, 0, "")
	pdf.CellFormat(contentW*0.45, 6, "______________________________", "", 1, "L", false, 0, "")
	pdf.Ln(4)
	pdf.CellFormat(contentW*0.55, 6, tr("Date:"), "", 0, "L", false, 0, "")
	pdf.CellFormat(contentW*0.45, 6, "______________________________", "", 1, "L", false, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PortfolioPDFInput drives the portfolio PDF export.
type PortfolioPDFInput struct {
	SchoolName  string
	StudentName string
	SchoolYear  string
	GeneratedAt time.Time
	Samples     []homeschoolrecords.SubmissionSample
	HoursTotals []SubjectTotal
	TotalHours  float64
}

// BuildPortfolioPDF renders selected work samples, comments, and hours summary.
func BuildPortfolioPDF(in PortfolioPDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.SetAutoPageBreak(true, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	when := in.GeneratedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}

	pdf.AddPage()
	school := strings.TrimSpace(in.SchoolName)
	if school == "" {
		school = "Homeschool"
	}
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(contentW, 8, tr(school), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(contentW, 7, tr("Student Portfolio"), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(contentW, 5, tr("Student: "+strings.TrimSpace(in.StudentName)), "", 1, "L", false, 0, "")
	if strings.TrimSpace(in.SchoolYear) != "" {
		pdf.CellFormat(contentW, 5, tr("School year: "+in.SchoolYear), "", 1, "L", false, 0, "")
	}
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(contentW, 5, tr("Generated "+when.UTC().Format("Jan 2, 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(contentW, 6, tr("Hours summary by subject"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	if len(in.HoursTotals) == 0 {
		pdf.CellFormat(contentW, 5, tr("No hours recorded."), "", 1, "L", false, 0, "")
	}
	for _, t := range in.HoursTotals {
		pdf.CellFormat(contentW, 5, tr(fmt.Sprintf("%s — %.2f hours", t.Subject, t.Hours)), "", 1, "L", false, 0, "")
	}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(contentW, 5, tr(fmt.Sprintf("Total: %.2f hours", in.TotalHours)), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(contentW, 6, tr("Work samples"), "", 1, "L", false, 0, "")
	if len(in.Samples) == 0 {
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(contentW, 5, tr("No submissions selected."), "", 1, "L", false, 0, "")
	}
	for i, s := range in.Samples {
		if pdf.GetY() > 230 {
			pdf.AddPage()
		}
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(contentW, 6, tr(fmt.Sprintf("%d. %s", i+1, s.ItemTitle)), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(contentW, 5, tr(fmt.Sprintf("%s (%s) · submitted %s", s.CourseTitle, s.CourseCode, s.SubmittedAt.UTC().Format("2006-01-02"))), "", 1, "L", false, 0, "")
		body := strings.TrimSpace(s.BodyPreview)
		if body != "" {
			pdf.MultiCell(contentW, 4.5, tr(body), "", "L", false)
		}
		comment := strings.TrimSpace(s.InstructorComment)
		if comment != "" {
			pdf.SetFont("Helvetica", "I", 9)
			pdf.MultiCell(contentW, 4.5, tr("Teacher comment: "+comment), "", "L", false)
			pdf.SetFont("Helvetica", "", 9)
		}
		pdf.Ln(2)
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

func rowHeight(pdf *gofpdf.Fpdf, tr func(string) string, colW []float64, vals []string, lineH, minH float64) float64 {
	h := minH
	for i, text := range vals {
		n := len(pdf.SplitLines([]byte(tr(text)), colW[i]-2))
		if n < 1 {
			n = 1
		}
		if cell := float64(n) * lineH; cell > h {
			h = cell
		}
	}
	return h
}
