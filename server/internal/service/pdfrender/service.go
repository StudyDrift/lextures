// Package pdfrender renders completion certificate PDFs (plan 15.5).
package pdfrender

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/jung-kurt/gofpdf"
)

// CertificateInput describes a completion certificate PDF.
type CertificateInput struct {
	InstitutionName string
	LearnerName     string
	AchievementName string
	Description     string
	IssuedAt        time.Time
	VerificationURL string
}

// BuildCertificate renders a landscape certificate with QR verification link.
func BuildCertificate(in CertificateInput) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(in.AchievementName, false)
	pdf.AddPage()

	pdf.SetFillColor(15, 23, 42)
	pdf.Rect(0, 0, 297, 210, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetXY(20, 24)
	pdf.CellFormat(0, 8, strings.TrimSpace(in.InstitutionName), "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "B", 28)
	pdf.SetXY(20, 52)
	pdf.CellFormat(0, 14, "Certificate of Completion", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 14)
	pdf.SetXY(20, 78)
	pdf.CellFormat(0, 8, "This certifies that", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(20, 92)
	pdf.CellFormat(0, 12, strings.TrimSpace(in.LearnerName), "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 14)
	pdf.SetXY(20, 110)
	pdf.CellFormat(0, 8, "has successfully completed", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(20, 122)
	pdf.CellFormat(0, 10, strings.TrimSpace(in.AchievementName), "", 1, "C", false, 0, "")

	if desc := strings.TrimSpace(in.Description); desc != "" {
		pdf.SetFont("Helvetica", "", 11)
		pdf.SetXY(40, 136)
		pdf.MultiCell(217, 6, desc, "", "C", false)
	}

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(20, 158)
	pdf.CellFormat(0, 7, fmt.Sprintf("Completed: %s", in.IssuedAt.UTC().Format("January 2, 2006")), "", 1, "C", false, 0, "")

	if strings.TrimSpace(in.VerificationURL) != "" {
		if err := embedQR(pdf, in.VerificationURL); err != nil {
			return nil, err
		}
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetXY(220, 182)
		pdf.MultiCell(65, 4, "Scan to verify", "", "C", false)
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
	rgba := image.NewRGBA(code.Bounds())
	for y := code.Bounds().Min.Y; y < code.Bounds().Max.Y; y++ {
		for x := code.Bounds().Min.X; x < code.Bounds().Max.X; x++ {
			rgba.Set(x, y, code.At(x, y))
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, rgba); err != nil {
		return err
	}
	name := "cert_qr"
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(pngBuf.Bytes()))
	pdf.ImageOptions(name, 232, 150, 30, 30, false, opt, 0, "")
	return nil
}