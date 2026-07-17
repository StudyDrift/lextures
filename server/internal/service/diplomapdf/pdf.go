// Package diplomapdf renders diploma/certificate PDFs (T11).
package diplomapdf

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

// Input describes a diploma/certificate PDF.
type Input struct {
	Kind            string // diploma | certificate
	InstitutionName string
	LearnerName     string
	CredentialTitle string
	Program         string
	Honors          string
	ConferralText   string
	ConferredAt     time.Time
	VerificationURL string
}

// Build renders an accessible landscape diploma/certificate PDF with verification QR.
func Build(in Input) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	title := strings.TrimSpace(in.CredentialTitle)
	if title == "" {
		title = "Credential"
	}
	pdf.SetTitle(title, false)
	pdf.SetAuthor(in.InstitutionName, false)
	pdf.AddPage()

	heading := "Diploma"
	if strings.EqualFold(strings.TrimSpace(in.Kind), "certificate") {
		heading = "Certificate"
	}
	pdf.SetFont("Helvetica", "B", 24)
	pdf.CellFormat(0, 14, heading, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.CellFormat(0, 8, strings.TrimSpace(in.InstitutionName), "", 1, "C", false, 0, "")
	pdf.Ln(6)

	conferral := strings.TrimSpace(in.ConferralText)
	if conferral == "" {
		conferral = "This certifies that"
	}
	pdf.SetFont("Helvetica", "", 11)
	pdf.MultiCell(0, 6, conferral, "", "C", false)
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 12, strings.TrimSpace(in.LearnerName), "", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")

	if p := strings.TrimSpace(in.Program); p != "" {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(0, 7, "Program: "+p, "", 1, "C", false, 0, "")
	}
	if h := strings.TrimSpace(in.Honors); h != "" {
		pdf.SetFont("Helvetica", "I", 11)
		pdf.CellFormat(0, 7, h, "", 1, "C", false, 0, "")
	}
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 7, fmt.Sprintf("Conferred: %s", in.ConferredAt.UTC().Format("January 2, 2006")), "", 1, "C", false, 0, "")

	if strings.TrimSpace(in.VerificationURL) != "" {
		pdf.Ln(6)
		if err := embedQR(pdf, in.VerificationURL); err != nil {
			return nil, err
		}
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(0, 4, "Verify at: "+in.VerificationURL, "", "C", false)
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
	// gofpdf only accepts 8-bit PNG; barcode may produce a paletted/16-bit source.
	bounds := code.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, code.At(x, y))
		}
	}
	var imgBuf bytes.Buffer
	if err := png.Encode(&imgBuf, rgba); err != nil {
		return err
	}
	name := "diploma_qr"
	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(imgBuf.Bytes()))
	pageW, _ := pdf.GetPageSize()
	pdf.ImageOptions(name, (pageW-30)/2, pdf.GetY(), 30, 30, false, opt, 0, "")
	pdf.Ln(32)
	return nil
}
