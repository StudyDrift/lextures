// Package watermark stamps user-identity text onto PDF pages (plan 8.10 FR-2).
package watermark

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Params describes who is viewing the document and when.
type Params struct {
	UserDisplayName string
	AccessedAt      time.Time
}

// WatermarkPDF reads the PDF bytes from r, stamps every page with the user's
// display name and access timestamp, and writes the result to w.
//
// The watermark is semi-transparent (40% opacity), 10 pt Helvetica, positioned
// at the bottom-right corner so it does not obscure body text (WCAG 2.1 AA safe).
func WatermarkPDF(r io.ReadSeeker, w io.Writer, p Params) error {
	text := Label(p)
	// desc controls font, size, opacity, and placement.
	// "position:br" → bottom-right; offset 10pt from each edge.
	desc := "font:Helvetica, points:10, opacity:0.4, position:br, offset:10 10, rotation:0, scale:1 abs, color:#606060"

	wm, err := api.TextWatermark(text, desc, true, false, types.POINTS)
	if err != nil {
		return fmt.Errorf("watermark: build descriptor: %w", err)
	}

	var buf bytes.Buffer
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	if err := api.AddWatermarks(r, &buf, nil, wm, conf); err != nil {
		return fmt.Errorf("watermark: stamp pages: %w", err)
	}

	_, err = io.Copy(w, &buf)
	return err
}

// Label returns the text string that will be stamped for the given params.
// Used in tests to verify expected watermark content.
func Label(p Params) string {
	return fmt.Sprintf("%s — %s", p.UserDisplayName, p.AccessedAt.UTC().Format("2006-01-02 15:04 UTC"))
}
