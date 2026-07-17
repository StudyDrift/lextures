package boardexport

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// RenderPNG builds a simple snapshot PNG of card titles/bodies (VC.9 FR-6).
// Client canvas capture remains preferred for fidelity; this supports headless/job exports.
func RenderPNG(boardTitle string, cards []CardRow) ([]byte, error) {
	const (
		width     = 800
		pad       = 24
		lineH     = 16
		cardGap   = 12
		titleSize = 2 // title uses two lines worth of spacing
	)
	lines := []string{truncate(boardTitle, 90)}
	lines = append(lines, "")
	for i, c := range cards {
		if c.SectionTitle != "" && (i == 0 || cards[i-1].SectionTitle != c.SectionTitle) {
			lines = append(lines, "§ "+truncate(c.SectionTitle, 80))
		}
		title := c.Title
		if title == "" {
			title = fmt.Sprintf("Card %d", i+1)
		}
		lines = append(lines, "• "+truncate(title, 90))
		if c.BodyText != "" {
			lines = append(lines, "  "+truncate(c.BodyText, 95))
		}
		lines = append(lines, "")
	}
	if len(cards) == 0 {
		lines = append(lines, "(empty board)")
	}

	height := pad*2 + len(lines)*lineH + titleSize*lineH
	if height < 200 {
		height = 200
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 250, G: 250, B: 252, A: 255}}, image.Point{}, draw.Src)

	face := basicfont.Face7x13
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 30, G: 41, B: 59, A: 255}),
		Face: face,
	}
	y := pad + face.Metrics().Ascent.Ceil()
	for i, line := range lines {
		if i == 0 {
			drawer.Src = image.NewUniform(color.RGBA{R: 15, G: 23, B: 42, A: 255})
		} else if strings.HasPrefix(line, "§ ") {
			drawer.Src = image.NewUniform(color.RGBA{R: 67, G: 56, B: 202, A: 255})
		} else {
			drawer.Src = image.NewUniform(color.RGBA{R: 30, G: 41, B: 59, A: 255})
		}
		drawer.Dot = fixed.P(pad, y)
		drawer.DrawString(line)
		y += lineH
		_ = cardGap
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
