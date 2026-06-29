package imageproxy

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"strings"

	"golang.org/x/image/draw"
)

var ErrNotImage = errors.New("not a supported raster image")

// ResizeOpts describes optional downscaling for course-file thumbnails.
type ResizeOpts struct {
	MaxWidth  int
	MaxHeight int
	Quality   int // JPEG quality 1–100; values <= 0 default to 85
}

// ResizeIfNeeded downscales raster image bytes when max width/height are set.
// Returns original bytes when no resize dimension is requested.
func ResizeIfNeeded(data []byte, mime string, opts ResizeOpts) ([]byte, string, error) {
	if opts.MaxWidth <= 0 && opts.MaxHeight <= 0 {
		return data, strings.TrimSpace(mime), nil
	}
	if !isRasterImageMIME(mime) {
		return nil, "", ErrNotImage
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, "", ErrNotImage
	}

	dstW, dstH := fitWithin(srcW, srcH, opts.MaxWidth, opts.MaxHeight)
	if dstW >= srcW && dstH >= srcH {
		return data, strings.TrimSpace(mime), nil
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	quality := opts.Quality
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, "", err
	}
	return out.Bytes(), "image/jpeg", nil
}

func isRasterImageMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
	}
}

func fitWithin(srcW, srcH, maxW, maxH int) (int, int) {
	if maxW <= 0 {
		maxW = srcW
	}
	if maxH <= 0 {
		maxH = srcH
	}
	scale := 1.0
	if sw := float64(maxW) / float64(srcW); sw < scale {
		scale = sw
	}
	if sh := float64(maxH) / float64(srcH); sh < scale {
		scale = sh
	}
	if scale >= 1 {
		return srcW, srcH
	}
	dstW := int(float64(srcW)*scale + 0.5)
	dstH := int(float64(srcH)*scale + 0.5)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}
	return dstW, dstH
}