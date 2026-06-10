package ccr

import (
	"bytes"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/jung-kurt/gofpdf"
)

// PDFInput describes a CCR PDF export.
type PDFInput struct {
	InstitutionName string
	StudentName     string
	GeneratedAt     time.Time
	VerificationURL string
	Achievements    []AggregatedAchievement
}

// BuildPDF renders an accessible CCR PDF with achievement list and verification QR code.
func BuildPDF(in PDFInput) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Comprehensive Learner Record", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "Comprehensive Learner Record", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, in.InstitutionName, "", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Learner: %s", in.StudentName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s", in.GeneratedAt.UTC().Format(time.RFC3339)), "", 1, "L", false, 0, "")

	if strings.TrimSpace(in.VerificationURL) != "" {
		if err := embedQR(pdf, in.VerificationURL); err != nil {
			return nil, err
		}
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(0, 4, fmt.Sprintf("Verify this credential: %s", in.VerificationURL), "", "L", false)
		pdf.Ln(4)
	}

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Achievements", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(55, 7, "Title", "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, "Type", "1", 0, "L", false, 0, "")
	pdf.CellFormat(35, 7, "Issued", "1", 0, "L", false, 0, "")
	pdf.CellFormat(0, 7, "Description", "1", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, a := range in.Achievements {
		desc := a.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		pdf.CellFormat(55, 6, a.Title, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 6, string(a.Type), "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 6, a.IssuedAt.UTC().Format("2006-01-02"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, desc, "1", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, code); err != nil {
		return err
	}
	name := "ccr_qr"
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(pngBuf.Bytes()))
	pdf.ImageOptions(name, 160, 35, 35, 35, false, opt, 0, "")
	return nil
}
