package httpserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/imageproxy"
	"golang.org/x/sync/singleflight"
)

// courseImageDerivativeFlight collapses concurrent misses for the same derivative.
var courseImageDerivativeFlight singleflight.Group

type driverObjectStore struct {
	driver filestorage.Driver
}

func (s driverObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	rc, err := s.driver.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

func (s driverObjectStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	err := s.driver.PutObject(ctx, key, bytes.NewReader(data), int64(len(data)), contentType)
	if err != nil {
		log.Printf("image derivative put key=%q err=%v", key, err)
	}
	return err
}

func (d Deps) derivativeDriver() filestorage.Driver {
	if d.Storage != nil {
		return d.Storage
	}
	root := strings.TrimSpace(d.effectiveConfig().CourseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	return &filestorage.LocalDriver{Root: root}
}

func (d Deps) courseFileDerivative(
	ctx context.Context,
	courseCode string,
	row *coursefiles.Row,
	opts imageproxy.ResizeOpts,
	format string,
) ([]byte, string, error) {
	if row == nil {
		return nil, "", &imageproxy.SourceError{Err: errors.New("missing file")}
	}
	opts.Format = format
	mime := strings.TrimSpace(row.MimeType)
	cache := imageproxy.Cache{
		Store: driverObjectStore{driver: d.derivativeDriver()},
		Group: &courseImageDerivativeFlight,
	}
	return cache.GetOrCreate(ctx, row.ID, opts, mime, func(ctx context.Context) ([]byte, error) {
		return d.readCourseFileRowBytes(ctx, courseCode, row)
	})
}

// scheduleBannerDerivative builds the banner-sized WebP and JPEG copies after a hero is saved
// so the first course-page view does not decode the master.
func (d Deps) scheduleBannerDerivative(courseCode, imageURL string) {
	fileID, ok := courseFileIDFromHeroURL(imageURL)
	if !ok || d.Pool == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := d.pregenerateBannerDerivative(ctx, courseCode, fileID); err != nil {
			log.Printf("hero banner derivative: course=%s file=%s err=%v", courseCode, fileID, err)
		}
	}()
}

func (d Deps) pregenerateBannerDerivative(ctx context.Context, courseCode string, fileID uuid.UUID) error {
	if d.Pool == nil {
		return nil
	}
	row, err := coursefiles.GetForCourse(ctx, d.Pool, courseCode, fileID)
	if err != nil || row == nil {
		return err
	}
	for _, format := range []string{imageproxy.FormatWebP, imageproxy.FormatJPEG} {
		if _, _, err := d.courseFileDerivative(ctx, courseCode, row, imageproxy.BannerOpts(format), format); err != nil {
			return err
		}
	}
	return nil
}

// courseFileIDFromHeroURL returns the course-files id embedded in a hero content URL.
func courseFileIDFromHeroURL(raw string) (uuid.UUID, bool) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return uuid.Nil, false
	}
	if strings.Contains(path, "://") {
		u, err := url.Parse(path)
		if err != nil {
			return uuid.Nil, false
		}
		path = u.Path
	} else if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	const marker = "/course-files/"
	i := strings.Index(path, marker)
	if i < 0 {
		return uuid.Nil, false
	}
	rest := strings.TrimSuffix(path[i+len(marker):], "/content")
	if rest == "" || strings.Contains(rest, "/") {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(rest)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
