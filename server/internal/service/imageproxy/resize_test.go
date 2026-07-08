package imageproxy

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestResizeIfNeeded_NoDimensions_Passthrough(t *testing.T) {
	src := mustJPEG(t, 800, 400)
	out, ct, err := ResizeIfNeeded(src, "image/jpeg", ResizeOpts{})
	if err != nil {
		t.Fatalf("ResizeIfNeeded: %v", err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %q want image/jpeg", ct)
	}
	if !bytes.Equal(out, src) {
		t.Fatal("expected passthrough bytes")
	}
}

func TestResizeIfNeeded_DownscalesWidth(t *testing.T) {
	src := mustJPEG(t, 1200, 600)
	out, ct, err := ResizeIfNeeded(src, "image/jpeg", ResizeOpts{MaxWidth: 400, Quality: 80})
	if err != nil {
		t.Fatalf("ResizeIfNeeded: %v", err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %q want image/jpeg", ct)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if cfg.Width > 400 {
		t.Fatalf("width = %d want <= 400", cfg.Width)
	}
	if cfg.Height > 200 {
		t.Fatalf("height = %d want <= 200", cfg.Height)
	}
}

func TestResizeIfNeeded_PNGInput(t *testing.T) {
	src := mustPNG(t, 640, 480)
	out, ct, err := ResizeIfNeeded(src, "image/png", ResizeOpts{MaxWidth: 320, MaxHeight: 240, Quality: 82})
	if err != nil {
		t.Fatalf("ResizeIfNeeded: %v", err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %q want image/jpeg", ct)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if cfg.Width > 320 || cfg.Height > 240 {
		t.Fatalf("size = %dx%d want within 320x240", cfg.Width, cfg.Height)
	}
}

func TestResizeIfNeeded_NotImage(t *testing.T) {
	_, _, err := ResizeIfNeeded([]byte("plain"), "text/plain", ResizeOpts{MaxWidth: 100})
	if err != ErrNotImage {
		t.Fatalf("err = %v want ErrNotImage", err)
	}
}

func TestResizeIfNeeded_SVG(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)
	_, _, err := ResizeIfNeeded(svg, "image/svg+xml", ResizeOpts{MaxWidth: 100})
	if err != ErrNotImage {
		t.Fatalf("err = %v want ErrNotImage", err)
	}
}

func mustJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("jpeg.Encode: %v", err)
	}
	return buf.Bytes()
}

func mustPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}