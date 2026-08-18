// Package marketingmedia validates, scans, transforms, and stores marketing content images.
package marketingmedia

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "golang.org/x/image/webp"

	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	"github.com/lextures/lextures/server/internal/service/clamav"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

const DefaultMaxBytes int64 = 10 << 20

var (
	ErrAltRequired = errors.New("alt_text_required")
	ErrUnsupported = errors.New("unsupported_media_type")
	ErrTooLarge    = errors.New("media_too_large")
	ErrInfected    = errors.New("media_infected")
)

type Scanner interface {
	ScanStream(context.Context, io.Reader) (clamav.ScanResult, error)
}
type Service struct {
	Pool     *pgxpool.Pool
	Storage  filestorage.Driver
	Scanner  Scanner
	MaxBytes int64
}
type Upload struct {
	Data       []byte
	AltText    string
	Decorative bool
	Title      string
	Credit     string
	ActorID    uuid.UUID
}

func (s *Service) Create(ctx context.Context, in Upload) (*repo.MediaAsset, bool, error) {
	if !in.Decorative && strings.TrimSpace(in.AltText) == "" {
		return nil, false, ErrAltRequired
	}
	if in.Decorative {
		in.AltText = ""
	}
	if len([]rune(in.AltText)) > 300 {
		return nil, false, ErrAltRequired
	}
	max := s.MaxBytes
	if max <= 0 {
		max = DefaultMaxBytes
	}
	if int64(len(in.Data)) > max {
		return nil, false, ErrTooLarge
	}
	if len(in.Data) == 0 {
		return nil, false, ErrUnsupported
	}
	mime := http.DetectContentType(in.Data)
	if mime == "image/svg+xml" || bytes.HasPrefix(bytes.TrimSpace(in.Data), []byte("<svg")) {
		return nil, false, ErrUnsupported
	}
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/gif" && mime != "image/webp" && mime != "image/avif" {
		return nil, false, ErrUnsupported
	}
	if mime == "image/gif" && bytes.Count(in.Data, []byte{0x21, 0xF9, 0x04}) > 1 {
		return nil, false, ErrUnsupported
	}
	if s.Scanner != nil {
		res, e := s.Scanner.ScanStream(ctx, bytes.NewReader(in.Data))
		if e != nil {
			return nil, false, e
		}
		if !res.Clean {
			return nil, false, fmt.Errorf("%w: %s", ErrInfected, res.VirusName)
		}
	}
	sum := sha256.Sum256(in.Data)
	checksum := hex.EncodeToString(sum[:])
	if existing, e := repo.GetMediaByChecksum(ctx, s.Pool, checksum); e == nil {
		return existing, true, nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return nil, false, e
	}
	cfg, _, e := image.DecodeConfig(bytes.NewReader(in.Data))
	if e != nil {
		return nil, false, ErrUnsupported
	}
	img, _, e := image.Decode(bytes.NewReader(in.Data))
	if e != nil {
		return nil, false, ErrUnsupported
	}
	id := uuid.New()
	ext := extension(mime)
	originalKey := fmt.Sprintf("marketing/media/%s/original.%s", id, ext)
	if e = s.Storage.PutObject(ctx, originalKey, bytes.NewReader(in.Data), int64(len(in.Data)), mime); e != nil {
		return nil, false, e
	}
	rends := []repo.MediaRendition{{Name: "original", Ext: ext, MIME: mime, Width: cfg.Width, Height: cfg.Height, Key: originalKey, URL: publicURL(id, "original", ext), Bytes: int64(len(in.Data))}}
	written := []string{originalKey}
	cleanup := func() {
		for _, k := range written {
			_ = s.Storage.DeleteObject(context.Background(), k)
		}
	}
	if social, encErr := encodeSocialPNG(img); encErr == nil {
		socialKey := fmt.Sprintf("marketing/media/%s/social.png", id)
		if e = s.Storage.PutObject(ctx, socialKey, bytes.NewReader(social), int64(len(social)), "image/png"); e != nil {
			cleanup()
			return nil, false, e
		}
		written = append(written, socialKey)
		rends = append(rends, repo.MediaRendition{Name: "social", Ext: "png", MIME: "image/png", Width: socialCardWidth, Height: socialCardHeight, Key: socialKey, URL: publicURL(id, "social", "png"), Bytes: int64(len(social))})
	}
	for _, target := range []int{1600, 800, 400} {
		if cfg.Width <= target {
			continue
		}
		// Pure-Go encoders only (no CGO/libwebp). Prefer JPEG for photographic
		// uploads; otherwise PNG. Original bytes remain available as "original".
		format := "png"
		if ext == "jpg" || ext == "jpeg" {
			format = "jpg"
		}
		if ext == "gif" || ext == "avif" {
			continue
		}
		scaled := resize(img, target, cfg.Height*target/cfg.Width)
		var b bytes.Buffer
		outMime := mimeForExt(format)
		var encErr error
		switch format {
		case "jpg":
			encErr = jpeg.Encode(&b, scaled, &jpeg.Options{Quality: 90})
		default:
			format = "png"
			outMime = "image/png"
			encErr = png.Encode(&b, scaled)
		}
		if encErr != nil {
			cleanup()
			return nil, false, encErr
		}
		name := fmt.Sprintf("%dw", target)
		key := fmt.Sprintf("marketing/media/%s/%s.%s", id, name, format)
		if e = s.Storage.PutObject(ctx, key, bytes.NewReader(b.Bytes()), int64(b.Len()), outMime); e != nil {
			cleanup()
			return nil, false, e
		}
		written = append(written, key)
		rends = append(rends, repo.MediaRendition{Name: name, Ext: format, MIME: outMime, Width: target, Height: cfg.Height * target / cfg.Width, Key: key, URL: publicURL(id, name, format), Bytes: int64(b.Len())})
	}
	w, h := cfg.Width, cfg.Height
	m := repo.MediaAsset{ID: id, Checksum: checksum, MIMEType: mime, ByteSize: int64(len(in.Data)), Width: &w, Height: &h, AltText: strings.TrimSpace(in.AltText), Decorative: in.Decorative, Title: strings.TrimSpace(in.Title), Credit: strings.TrimSpace(in.Credit), StorageKey: originalKey, Renditions: rends, UploadedBy: &in.ActorID}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		cleanup()
		return nil, false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	created, e := repo.InsertMedia(ctx, tx, m)
	if e != nil {
		cleanup()
		return nil, false, e
	}
	if e = tx.Commit(ctx); e != nil {
		cleanup()
		return nil, false, e
	}
	return created, false, nil
}
func extension(m string) string {
	switch m {
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/avif":
		return "avif"
	case "image/gif":
		return "gif"
	}
	return "png"
}
func mimeForExt(e string) string {
	if e == "jpg" {
		return "image/jpeg"
	}
	return "image/" + e
}
func publicURL(id uuid.UUID, name, ext string) string {
	return "/api/v1/public/content/media/" + id.String() + "/" + name + "." + ext
}

const (
	socialCardWidth  = 1200
	socialCardHeight = 630
)

func encodeSocialPNG(src image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, coverCrop(src, socialCardWidth, socialCardHeight)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func coverCrop(src image.Image, tw, th int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw < 1 || sh < 1 || tw < 1 || th < 1 {
		return image.NewRGBA(image.Rect(0, 0, tw, th))
	}
	scale := float64(tw) / float64(sw)
	if float64(sh)*scale < float64(th) {
		scale = float64(th) / float64(sh)
	}
	nw := int(float64(sw)*scale + 0.5)
	nh := int(float64(sh)*scale + 0.5)
	if nw < tw {
		nw = tw
	}
	if nh < th {
		nh = th
	}
	scaled := resize(src, nw, nh)
	x0 := (nw - tw) / 2
	y0 := (nh - th) / 2
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			dst.Set(x, y, scaled.At(x0+x, y0+y))
		}
	}
	return dst
}

func resize(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y*sb.Dy()/h
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*sb.Dx()/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
func Rendition(a *repo.MediaAsset, file string) (repo.MediaRendition, bool) {
	file = path.Base(file)
	for _, r := range a.Renditions {
		if r.Name+"."+r.Ext == file {
			return r, true
		}
	}
	return repo.MediaRendition{}, false
}
