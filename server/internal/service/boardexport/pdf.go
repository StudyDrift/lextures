package boardexport

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// RenderPDF builds a print-ready PDF of cards in reading order (VC.9 FR-4).
// Uses gofpdf (no network, no HTML fetch — SSRF-safe).
func RenderPDF(boardTitle string, cards []CardRow) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(boardTitle, true)
	pdf.SetAuthor("Lextures", true)
	pdf.SetCreationDate(time.Now())
	pdf.SetAutoPageBreak(true, 15)

	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(0, 8, sanitizePDF(boardTitle), "", 1, "L", false, 0, "")
		pdf.Ln(2)
	})

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.MultiCell(0, 8, sanitizePDF(boardTitle), "", "L", false)
	pdf.Ln(4)

	currentSection := "\x00" // sentinel so first empty section still headers once
	for i, c := range cards {
		sec := c.SectionTitle
		if sec != currentSection {
			currentSection = sec
			if sec != "" {
				pdf.SetFont("Helvetica", "B", 13)
				pdf.MultiCell(0, 7, sanitizePDF(sec), "", "L", false)
				pdf.Ln(2)
			}
		}
		pdf.SetFont("Helvetica", "B", 11)
		title := c.Title
		if title == "" {
			title = fmt.Sprintf("Card %d", i+1)
		}
		pdf.MultiCell(0, 6, sanitizePDF(title), "", "L", false)

		pdf.SetFont("Helvetica", "", 10)
		meta := c.ContentType
		if c.Author != "" {
			meta += " · " + c.Author
		}
		meta += " · " + c.CreatedAt.UTC().Format("2006-01-02 15:04")
		pdf.MultiCell(0, 5, sanitizePDF(meta), "", "L", false)

		if c.BodyText != "" {
			pdf.MultiCell(0, 5, sanitizePDF(c.BodyText), "", "L", false)
		}
		if c.Link != "" {
			pdf.SetTextColor(0, 0, 180)
			pdf.MultiCell(0, 5, sanitizePDF(c.Link), "", "L", false)
			pdf.SetTextColor(0, 0, 0)
		}
		if c.AttachmentFilename != "" {
			alt := c.AttachmentAltText
			if alt == "" {
				alt = c.AttachmentFilename
			}
			pdf.MultiCell(0, 5, sanitizePDF(fmt.Sprintf("Attachment: %s (alt: %s)", c.AttachmentFilename, alt)), "", "L", false)
		}
		pdf.SetFont("Helvetica", "I", 9)
		pdf.MultiCell(0, 4, sanitizePDF(fmt.Sprintf("Reactions: %d · Comments: %d", c.ReactionCount, c.CommentCount)), "", "L", false)
		pdf.Ln(4)
	}

	if len(cards) == 0 {
		pdf.SetFont("Helvetica", "I", 11)
		pdf.MultiCell(0, 6, "No cards to export.", "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sanitizePDF(s string) string {
	// gofpdf core fonts are Latin-1; replace unsupported runes.
	var b strings.Builder
	for _, r := range s {
		if r > 255 {
			b.WriteRune('?')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
