package imageproxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// SourceError means the original object could not be read. Resize failures are plain errors.
type SourceError struct {
	Err error
}

func (e *SourceError) Error() string {
	if e == nil || e.Err == nil {
		return "image source unavailable"
	}
	return e.Err.Error()
}

func (e *SourceError) Unwrap() error { return e.Err }

// ObjectStore is the durable cache for resized derivatives.
type ObjectStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, data []byte, contentType string) error
}

// Cache serves a derivative keyed by file id, width, height, quality, and format.
// A hit does not call load, so the original object is not read or decoded again.
type Cache struct {
	Store ObjectStore
	Group *singleflight.Group
}

type cachedImage struct {
	data []byte
	mime string
}

// DerivativeKey is the storage key for one derivative variant.
func DerivativeKey(fileID uuid.UUID, opts ResizeOpts, kind string) string {
	opts = opts.normalized()
	switch kind {
	case FormatWebP, FormatJPEG, FormatOriginal:
	default:
		kind = FormatOriginal
	}
	return fmt.Sprintf("image-derivatives/v1/%s/%dx%d/q%d/%s", fileID.String(), opts.MaxWidth, opts.MaxHeight, opts.Quality, kind)
}

// GetOrCreate returns a cached derivative. load is invoked only on a miss and
// must return the untouched master bytes.
func (c Cache) GetOrCreate(
	ctx context.Context,
	fileID uuid.UUID,
	opts ResizeOpts,
	sourceMIME string,
	load func(context.Context) ([]byte, error),
) ([]byte, string, error) {
	opts = opts.normalized()
	sourceMIME = normalizeMIME(sourceMIME)
	if opts.MaxWidth <= 0 && opts.MaxHeight <= 0 {
		data, err := load(ctx)
		if err != nil {
			return nil, "", &SourceError{Err: err}
		}
		return data, sourceMIME, nil
	}

	encodedKey := DerivativeKey(fileID, opts, opts.Format)
	originalKey := DerivativeKey(fileID, opts, FormatOriginal)
	if data, mime, ok := c.hit(ctx, encodedKey, originalKey, opts.Format, sourceMIME); ok {
		return data, mime, nil
	}

	flightKey := encodedKey
	fetch := func() (any, error) {
		if data, mime, ok := c.hit(ctx, encodedKey, originalKey, opts.Format, sourceMIME); ok {
			return cachedImage{data: data, mime: mime}, nil
		}
		master, err := load(ctx)
		if err != nil {
			return nil, &SourceError{Err: err}
		}
		out, mime, err := ResizeIfNeeded(master, sourceMIME, opts)
		if err != nil {
			if errors.Is(err, ErrNotImage) {
				c.store(ctx, originalKey, master, sourceMIME)
				return cachedImage{data: master, mime: sourceMIME}, nil
			}
			return nil, err
		}
		if mime == sourceMIME && bytes.Equal(out, master) {
			c.store(ctx, originalKey, master, sourceMIME)
			return cachedImage{data: master, mime: sourceMIME}, nil
		}
		// Store under the format we actually encoded. A WebP failure falls back to
		// JPEG and must not be served later with an image/webp content type.
		c.store(ctx, DerivativeKey(fileID, opts, kindForMIME(mime)), out, mime)
		return cachedImage{data: out, mime: mime}, nil
	}

	var (
		v   any
		err error
	)
	if c.Group == nil {
		v, err = fetch()
	} else {
		v, err, _ = c.Group.Do(flightKey, fetch)
	}
	if err != nil {
		return nil, "", err
	}
	img, _ := v.(cachedImage)
	return img.data, img.mime, nil
}

func (c Cache) hit(ctx context.Context, encodedKey, originalKey, format, sourceMIME string) ([]byte, string, bool) {
	if c.Store == nil {
		return nil, "", false
	}
	if data, ok := c.get(ctx, encodedKey); ok {
		return data, contentTypeForFormat(format), true
	}
	if data, ok := c.get(ctx, originalKey); ok {
		return data, sourceMIME, true
	}
	return nil, "", false
}

func (c Cache) get(ctx context.Context, key string) ([]byte, bool) {
	data, err := c.Store.Get(ctx, key)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func (c Cache) store(ctx context.Context, key string, data []byte, contentType string) {
	if c.Store == nil || len(data) == 0 {
		return
	}
	_ = c.Store.Put(ctx, key, data, contentType)
}

func contentTypeForFormat(format string) string {
	if format == FormatWebP {
		return "image/webp"
	}
	return "image/jpeg"
}

func kindForMIME(mime string) string {
	if mime == "image/webp" {
		return FormatWebP
	}
	return FormatJPEG
}

func normalizeMIME(mime string) string {
	mime = trimMIME(mime)
	if mime == "" {
		return "application/octet-stream"
	}
	return mime
}

func trimMIME(mime string) string {
	for i := 0; i < len(mime); i++ {
		if mime[i] == ';' {
			mime = mime[:i]
			break
		}
	}
	start, end := 0, len(mime)
	for start < end && (mime[start] == ' ' || mime[start] == '\t') {
		start++
	}
	for end > start && (mime[end-1] == ' ' || mime[end-1] == '\t') {
		end--
	}
	return mime[start:end]
}
