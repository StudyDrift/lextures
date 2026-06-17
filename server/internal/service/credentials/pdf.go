package credentials

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

// PDFInput describes a completion certificate PDF.
type PDFInput struct {
	InstitutionName string
	LearnerName     string
	CredentialName  string
	IssuedAt        time.Time
	VerificationURL string
}

// BuildPDF renders an accessible certificate PDF with verification QR code.
func BuildPDF(in PDFInput) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(in.CredentialName, false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(0, 12, "Certificate of Completion", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.CellFormat(0, 8, in.InstitutionName, "", 1, "C", false, 0, "")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, "This certifies that", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, in.LearnerName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, "has successfully completed", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, in.CredentialName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Issued: %s", in.IssuedAt.UTC().Format("January 2, 2006")), "", 1, "C", false, 0, "")

	if strings.TrimSpace(in.VerificationURL) != "" {
		pdf.Ln(6)
		if err := embedQR(pdf, in.VerificationURL); err != nil {
			return nil, err
		}
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(0, 4, fmt.Sprintf("Verify at: %s", in.VerificationURL), "", "C", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func embedQR(pdf *gofpdf.Fpdf, content string) error {
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return err
	}
	code, err = barcode.Scale(code, 120, 120)
	if err != nil {
		return err
	}
	var imgBuf bytes.Buffer
	if err := png.Encode(&imgBuf, code); err != nil {
		return err
	}
	name := "credential_qr"
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(imgBuf.Bytes()))
	pageW, _ := pdf.GetPageSize()
	pdf.ImageOptions(name, (pageW-30)/2, pdf.GetY(), 30, 30, false, opt, 0, "")
	pdf.Ln(32)
	return nil
}