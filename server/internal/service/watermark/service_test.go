package watermark_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	gofpdf "github.com/jung-kurt/gofpdf"
	"github.com/lextures/lextures/server/internal/service/watermark"
)

// simplePDF creates a minimal single-page PDF using gofpdf.
func simplePDF(t *testing.T) []byte {
	t.Helper()
	f := gofpdf.New("P", "mm", "A4", "")
	f.AddPage()
	f.SetFont("Arial", "", 12)
	f.Cell(40, 10, "Test lecture notes")
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		t.Fatalf("create test PDF: %v", err)
	}
	return buf.Bytes()
}

func TestLabel(t *testing.T) {
	p := watermark.Params{
		UserDisplayName: "Alice Johnson",
		AccessedAt:      time.Date(2026, 4, 17, 14, 35, 0, 0, time.UTC),
	}
	got := watermark.Label(p)
	want := "Alice Johnson — 2026-04-17 14:35 UTC"
	if got != want {
		t.Fatalf("Label: got %q want %q", got, want)
	}
}

func TestWatermarkPDF_StampsPages(t *testing.T) {
	src := simplePDF(t)
	p := watermark.Params{
		UserDisplayName: "Alice Johnson",
		AccessedAt:      time.Date(2026, 4, 17, 14, 35, 0, 0, time.UTC),
	}

	var out bytes.Buffer
	if err := watermark.WatermarkPDF(bytes.NewReader(src), &out, p); err != nil {
		t.Fatalf("WatermarkPDF: %v", err)
	}

	if out.Len() == 0 {
		t.Fatal("output is empty")
	}
	if !strings.HasPrefix(out.String(), "%PDF") {
		t.Fatal("output does not start with %PDF")
	}
}

func TestWatermarkPDF_InvalidInput(t *testing.T) {
	var out bytes.Buffer
	err := watermark.WatermarkPDF(strings.NewReader("not a pdf"), &out, watermark.Params{
		UserDisplayName: "Bob",
		AccessedAt:      time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for invalid PDF input")
	}
}

func TestWatermarkPDF_PreservesOutput(t *testing.T) {
	src := simplePDF(t)
	p := watermark.Params{
		UserDisplayName: "Bob Smith",
		AccessedAt:      time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
	}

	var out bytes.Buffer
	if err := watermark.WatermarkPDF(bytes.NewReader(src), &out, p); err != nil {
		t.Fatalf("WatermarkPDF: %v", err)
	}
	// Output should be larger than input (watermark stream added).
	if out.Len() < len(src) {
		t.Fatalf("watermarked PDF (%d bytes) is smaller than original (%d bytes)", out.Len(), len(src))
	}
}
