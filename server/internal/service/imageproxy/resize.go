package imageproxy

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"strings"

	"github.com/deepteams/webp"
	"golang.org/x/image/draw"
)

const (
	FormatWebP     = "webp"
	FormatJPEG     = "jpeg"
	FormatOriginal = "original"

	// Banner slot is aspect-[5/1]. 1920px covers a typical content column at 2×.
	BannerWidth   = 1920
	BannerHeight  = 384
	BannerQuality = 85

	minEncodeQuality = 80
)

var ErrNotImage = errors.New("not a supported raster image")

// ResizeOpts describes optional downscaling for course-file images.
// The stored master is never rewritten; callers cache the returned bytes separately.
type ResizeOpts struct {
	MaxWidth  int
	MaxHeight int
	Quality   int    // 1–100; values <= 0 default to 85, and values below 80 are raised to 80
	Format    string // FormatWebP or FormatJPEG; empty defaults to JPEG
}

// BannerOpts is the course-page hero derivative (1920×384, quality 85).
func BannerOpts(format string) ResizeOpts {
	return ResizeOpts{
		MaxWidth:  BannerWidth,
		MaxHeight: BannerHeight,
		Quality:   BannerQuality,
		Format:    format,
	}
}

func (opts ResizeOpts) normalized() ResizeOpts {
	if opts.MaxWidth < 0 {
		opts.MaxWidth = 0
	}
	if opts.MaxHeight < 0 {
		opts.MaxHeight = 0
	}
	if opts.Quality <= 0 || opts.Quality > 100 {
		opts.Quality = 85
	}
	if opts.Quality < minEncodeQuality {
		opts.Quality = minEncodeQuality
	}
	switch opts.Format {
	case FormatWebP, FormatJPEG:
	default:
		opts.Format = FormatJPEG
	}
	return opts
}

// ResizeIfNeeded downscales raster image bytes when max width/height are set.
// Images that already fit are returned unchanged (no upscale, no recompress).
// SVG and other non-raster inputs return ErrNotImage so callers can serve the original.
func ResizeIfNeeded(data []byte, mime string, opts ResizeOpts) ([]byte, string, error) {
	opts = opts.normalized()
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
	return encodeScaled(dst, opts)
}

func encodeScaled(dst *image.RGBA, opts ResizeOpts) ([]byte, string, error) {
	if opts.Format == FormatWebP {
		encoded, err := encodeWebP(dst, opts.Quality)
		if err == nil {
			return encoded, "image/webp", nil
		}
		// JPEG keeps the banner visible when WebP encoding fails.
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: opts.Quality}); err != nil {
		return nil, "", err
	}
	return out.Bytes(), "image/jpeg", nil
}

func encodeWebP(img image.Image, quality int) ([]byte, error) {
	encOpts := webp.OptionsForPreset(webp.PresetPhoto, float32(quality))
	encOpts.UseSharpYUV = true
	var out bytes.Buffer
	if err := webp.Encode(&out, img, encOpts); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func isRasterImageMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	case "image/svg+xml", "image/svg":
		return false
	default:
		return false
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
