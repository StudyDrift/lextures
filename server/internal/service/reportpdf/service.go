// Package reportpdf generates formatted PDF reports using gofpdf (plans 9.8, 13.4).
package reportpdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

const (
	pageW      = 210.0 // A4 width mm
	marginL    = 15.0
	marginR    = 15.0
	contentW   = pageW - marginL - marginR
	footerY    = 280.0
	ferpaNote  = "Confidential — FERPA protected"
)

// GradebookRow is one student row in a gradebook summary PDF.
type GradebookRow struct {
	DisplayName  string
	FinalGrade   string
	GradePercent float64
}

// GradebookInput describes a gradebook summary report.
type GradebookInput struct {
	InstitutionName string
	CourseName      string
	CourseCode      string
	GeneratedAt     time.Time
	Students        []GradebookRow
}

// ProgressActivity is one item in a per-student progress report.
type ProgressActivity struct {
	ItemTitle string
	ItemType  string
	Status    string
	Grade     string
}

// ProgressInput describes a per-student progress report.
type ProgressInput struct {
	InstitutionName string
	CourseName      string
	CourseCode      string
	StudentName     string
	GeneratedAt     time.Time
	CompletionPct   float64
	Activities      []ProgressActivity
}

// LearningActivityDay is one daily row in a learning activity report.
type LearningActivityDay struct {
	Day         string
	TotalEvents int
}

// LearningActivityInput describes a platform learning activity report.
type LearningActivityInput struct {
	InstitutionName string
	GeneratedAt     time.Time
	From            time.Time
	To              time.Time
	TotalEvents     int
	UniqueUsers     int
	UniqueCourses   int
	ByDay           []LearningActivityDay
}

// BuildGradebookPDF renders a gradebook summary as PDF bytes.
func BuildGradebookPDF(in GradebookInput) ([]byte, error) {
	pdf := newPDF()
	addHeaderPage(pdf, in.InstitutionName, "Gradebook Summary", in.CourseName+" ("+in.CourseCode+")", in.GeneratedAt, true)

	pdf.SetFont("Helvetica", "B", 9)
	colW := []float64{90, 40, 40}
	drawRow(pdf, colW, []string{"Student", "Final Grade", "Percentage"}, true)
	pdf.SetFont("Helvetica", "", 9)
	for _, s := range in.Students {
		drawRow(pdf, colW, []string{s.DisplayName, s.FinalGrade, fmt.Sprintf("%.1f%%", s.GradePercent)}, false)
	}
	return renderPDF(pdf)
}

// BuildProgressPDF renders a per-student progress report as PDF bytes.
func BuildProgressPDF(in ProgressInput) ([]byte, error) {
	pdf := newPDF()
	subtitle := fmt.Sprintf("%s (%s) — %s", in.CourseName, in.CourseCode, in.StudentName)
	addHeaderPage(pdf, in.InstitutionName, "Student Progress Report", subtitle, in.GeneratedAt, true)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(marginL, pdf.GetY()+4)
	pdf.Cell(contentW, 7, fmt.Sprintf("Overall Completion: %.0f%%", in.CompletionPct))
	pdf.Ln(10)

	pdf.SetFont("Helvetica", "B", 9)
	colW := []float64{80, 30, 30, 30}
	drawRow(pdf, colW, []string{"Item", "Type", "Status", "Grade"}, true)
	pdf.SetFont("Helvetica", "", 9)
	for _, a := range in.Activities {
		drawRow(pdf, colW, []string{truncate(a.ItemTitle, 45), a.ItemType, a.Status, a.Grade}, false)
	}
	return renderPDF(pdf)
}

// BuildLearningActivityPDF renders a platform learning activity report as PDF bytes.
func BuildLearningActivityPDF(in LearningActivityInput) ([]byte, error) {
	pdf := newPDF()
	dateRange := fmt.Sprintf("%s – %s", in.From.Format("Jan 2, 2006"), in.To.Format("Jan 2, 2006"))
	addHeaderPage(pdf, in.InstitutionName, "Learning Activity Report", dateRange, in.GeneratedAt, false)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(marginL, pdf.GetY()+4)
	pdf.Cell(60, 7, fmt.Sprintf("Total Events: %d", in.TotalEvents))
	pdf.Ln(7)
	pdf.SetXY(marginL, pdf.GetY())
	pdf.Cell(60, 7, fmt.Sprintf("Active Learners: %d", in.UniqueUsers))
	pdf.Ln(7)
	pdf.SetXY(marginL, pdf.GetY())
	pdf.Cell(60, 7, fmt.Sprintf("Active Courses: %d", in.UniqueCourses))
	pdf.Ln(10)

	if len(in.ByDay) > 0 {
		pdf.SetFont("Helvetica", "B", 9)
		colW := []float64{60, 40}
		drawRow(pdf, colW, []string{"Date", "Events"}, true)
		pdf.SetFont("Helvetica", "", 9)
		for _, d := range in.ByDay {
			drawRow(pdf, colW, []string{d.Day, fmt.Sprintf("%d", d.TotalEvents)}, false)
		}
	}
	return renderPDF(pdf)
}

func newPDF() *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginL, 15, marginR)
	pdf.SetAutoPageBreak(true, 20)
	return pdf
}

func addHeaderPage(pdf *gofpdf.Fpdf, institution, title, subtitle string, generatedAt time.Time, ferpa bool) {
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetXY(marginL, 18)
	pdf.Cell(contentW, 8, title)
	pdf.Ln(9)
	if strings.TrimSpace(institution) != "" {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(marginL, pdf.GetY())
		pdf.Cell(contentW, 6, institution)
		pdf.Ln(6)
	}
	if strings.TrimSpace(subtitle) != "" {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(marginL, pdf.GetY())
		pdf.Cell(contentW, 6, subtitle)
		pdf.Ln(6)
	}
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetXY(marginL, pdf.GetY()+1)
	pdf.Cell(contentW, 5, "Generated: "+generatedAt.UTC().Format("Jan 2, 2006 15:04 UTC"))
	pdf.Ln(8)

	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(marginL, pdf.GetY(), pageW-marginR, pdf.GetY())
	pdf.Ln(4)

	// Footer on all pages
	pdf.SetFooterFunc(func() {
		pdf.SetY(footerY)
		pdf.SetFont("Helvetica", "I", 7)
		if ferpa {
			pdf.Cell(contentW/2, 5, ferpaNote)
		} else {
			pdf.Cell(contentW/2, 5, "")
		}
		pdf.Cell(contentW/2, 5, fmt.Sprintf("Page %d", pdf.PageNo()))
	})
}

func drawRow(pdf *gofpdf.Fpdf, colWidths []float64, cells []string, header bool) {
	if header {
		pdf.SetFillColor(220, 220, 220)
	} else {
		pdf.SetFillColor(255, 255, 255)
	}
	x := marginL
	y := pdf.GetY()
	h := 6.0
	for i, w := range colWidths {
		var cell string
		if i < len(cells) {
			cell = truncate(cells[i], int(w/2.2))
		}
		pdf.SetXY(x, y)
		pdf.CellFormat(w, h, cell, "1", 0, "L", true, 0, "")
		x += w
	}
	pdf.Ln(h)
}

func renderPDF(pdf *gofpdf.Fpdf) ([]byte, error) {
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// ── Report Card PDF (plan 13.4) ───────────────────────────────────────────────

// ReportCardStudent is one student row in a report card PDF.
type ReportCardStudent struct {
	DisplayName   string
	StudentID     string
	FinalGradePct *float64
	LetterGrade   string
	Comment       string
	Absences      int
}

// ReportCardInput describes one report card document (one student, one course, one period).
type ReportCardInput struct {
	InstitutionName string
	CourseName      string
	CourseCode      string
	GradingPeriod   string
	GeneratedAt     time.Time
	Student         ReportCardStudent
}

// BuildReportCardPDF renders a single student report card as PDF bytes.
func BuildReportCardPDF(in ReportCardInput) ([]byte, error) {
	pdf := newPDF()
	title := "Report Card"
	subtitle := fmt.Sprintf("%s (%s)  —  %s", in.CourseName, in.CourseCode, in.GradingPeriod)
	addHeaderPage(pdf, in.InstitutionName, title, subtitle, in.GeneratedAt, true)

	// Student info block
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(marginL, pdf.GetY()+4)
	pdf.Cell(40, 7, "Student:")
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(contentW-40, 7, in.Student.DisplayName)
	pdf.Ln(7)

	if in.Student.StudentID != "" {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetXY(marginL, pdf.GetY())
		pdf.Cell(40, 7, "Student ID:")
		pdf.SetFont("Helvetica", "", 10)
		pdf.Cell(contentW-40, 7, in.Student.StudentID)
		pdf.Ln(7)
	}

	pdf.SetXY(marginL, pdf.GetY()+2)
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(marginL, pdf.GetY(), pageW-marginR, pdf.GetY())
	pdf.Ln(4)

	// Grade summary table
	pdf.SetFont("Helvetica", "B", 9)
	colW := []float64{70, 40, 40, 30}
	drawRow(pdf, colW, []string{"Course", "Final %", "Letter Grade", "Absences"}, true)
	pdf.SetFont("Helvetica", "", 9)

	pctStr := "—"
	if in.Student.FinalGradePct != nil {
		pctStr = fmt.Sprintf("%.1f%%", *in.Student.FinalGradePct)
	}
	letterStr := in.Student.LetterGrade
	if letterStr == "" {
		letterStr = "—"
	}
	absStr := fmt.Sprintf("%d", in.Student.Absences)
	drawRow(pdf, colW, []string{truncate(in.CourseName, 38), pctStr, letterStr, absStr}, false)
	pdf.Ln(6)

	// Comment section
	if strings.TrimSpace(in.Student.Comment) != "" {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetXY(marginL, pdf.GetY()+2)
		pdf.Cell(contentW, 7, "Teacher Comment:")
		pdf.Ln(7)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(marginL, pdf.GetY())
		pdf.MultiCell(contentW, 5, in.Student.Comment, "1", "L", false)
		pdf.Ln(4)
	}

	// Signature line
	pdf.SetXY(marginL, pdf.GetY()+10)
	pdf.SetFont("Helvetica", "", 9)
	pdf.Cell(80, 5, "Teacher Signature: ____________________")
	pdf.Cell(80, 5, "Date: ____________________")
	pdf.Ln(10)

	return renderPDF(pdf)
}

// ── CCR PDF (plan 14.13) ──────────────────────────────────────────────────────

// CCRAchievementRow is one achievement line in a co-curricular transcript PDF.
type CCRAchievementRow struct {
	Type    string
	Title   string
	Issued  string
	Details string
}

// CCRInput describes a comprehensive learner record PDF export.
type CCRInput struct {
	InstitutionName string
	StudentName     string
	GeneratedAt     time.Time
	Achievements    []CCRAchievementRow
	VerificationURL string
	QRCodePNG       []byte
}

// BuildCCRPDF renders a student co-curricular transcript with optional verification QR code.
func BuildCCRPDF(in CCRInput) ([]byte, error) {
	pdf := newPDF()
	subtitle := in.StudentName
	addHeaderPage(pdf, in.InstitutionName, "Comprehensive Learner Record", subtitle, in.GeneratedAt, true)

	if len(in.QRCodePNG) > 0 && strings.TrimSpace(in.VerificationURL) != "" {
		opt := gofpdf.ImageOptions{ImageType: "PNG"}
		name := "ccr-qr.png"
		pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(in.QRCodePNG))
		pdf.ImageOptions(name, pageW-marginR-35, 18, 30, 30, false, opt, 0, "")
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetXY(pageW-marginR-35, 50)
		pdf.MultiCell(35, 3, "Scan to verify", "", "C", false)
	}

	pdf.SetFont("Helvetica", "B", 9)
	colW := []float64{35, 70, 25, 40}
	drawRow(pdf, colW, []string{"Type", "Achievement", "Date", "Details"}, true)
	pdf.SetFont("Helvetica", "", 9)
	for _, a := range in.Achievements {
		drawRow(pdf, colW, []string{
			truncate(a.Type, 16),
			truncate(a.Title, 42),
			a.Issued,
			truncate(a.Details, 24),
		}, false)
	}

	if strings.TrimSpace(in.VerificationURL) != "" {
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(contentW, 4, "Verification URL: "+in.VerificationURL, "", "L", false)
	}

	return renderPDF(pdf)
}
