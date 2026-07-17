package boardexport

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// RenderQRPNG encodes accessURL as a PNG QR code (VC.9 FR-8).
func RenderQRPNG(accessURL string, size int) ([]byte, error) {
	if size < 64 {
		size = 256
	}
	code, err := qr.Encode(accessURL, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("boardexport: qr encode: %w", err)
	}
	code, err = barcode.Scale(code, size, size)
	if err != nil {
		return nil, fmt.Errorf("boardexport: qr scale: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, code); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderQRSVG encodes accessURL as a minimal SVG QR (modules as rects).
func RenderQRSVG(accessURL string, size int) ([]byte, error) {
	if size < 64 {
		size = 256
	}
	code, err := qr.Encode(accessURL, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("boardexport: qr encode: %w", err)
	}
	bounds := code.Bounds()
	modules := bounds.Dx()
	if modules <= 0 {
		return nil, fmt.Errorf("boardexport: empty qr")
	}
	cell := float64(size) / float64(modules)
	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="QR code">`, size, size, size, size)
	b.WriteString(`<rect width="100%" height="100%" fill="#fff"/>`)
	for y := 0; y < modules; y++ {
		for x := 0; x < modules; x++ {
			r, g, bl, _ := code.At(x, y).RGBA()
			if r < 0x8000 && g < 0x8000 && bl < 0x8000 {
				fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="#000"/>`,
					float64(x)*cell, float64(y)*cell, cell, cell)
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.Bytes(), nil
}
