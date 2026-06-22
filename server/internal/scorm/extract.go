package scorm

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// ExtractZipToStorage unpacks a SCORM zip from disk into object storage under assetsPrefix.
func ExtractZipToStorage(ctx context.Context, storage filestorage.Driver, zipPath, assetsPrefix string) error {
	if storage == nil {
		return fmt.Errorf("scorm extract: no storage driver")
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("scorm extract: open zip: %w", err)
	}
	defer func() { _ = r.Close() }()
	if len(r.File) > maxZipEntries {
		return fmt.Errorf("scorm extract: too many zip entries")
	}
	prefix := strings.TrimSuffix(assetsPrefix, "/") + "/"
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(f.Name)
		if strings.Contains(name, "..") {
			continue
		}
		if f.UncompressedSize64 > uint64(maxZipFileBytes) {
			return fmt.Errorf("scorm extract: entry too large: %s", name)
		}
		destKey := prefix + name
		rc, openErr := f.Open()
		if openErr != nil {
			return openErr
		}
		ct := mimeForPath(name)
		putErr := storage.PutObject(ctx, destKey, rc, int64(f.UncompressedSize64), ct)
		_ = rc.Close()
		if putErr != nil {
			return fmt.Errorf("scorm extract: put %s: %w", name, putErr)
		}
	}
	return nil
}

// WriteTempZip writes data to a temp file and returns its path (caller removes).
func WriteTempZip(data []byte) (string, error) {
	f, err := os.CreateTemp("", "scorm-upload-*.zip")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// DownloadToTemp downloads an object from storage to a local temp file.
func DownloadToTemp(ctx context.Context, storage filestorage.Driver, objectKey string) (string, error) {
	rc, err := storage.GetObject(ctx, objectKey)
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()
	f, err := os.CreateTemp("", "scorm-src-*.zip")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, rc); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func mimeForPath(name string) string {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".html", ".htm":
		return "text/html"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".woff", ".woff2":
		return "font/woff2"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}
