package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// boardBlobCopier copies attachment objects for independent board full-copies (VC.8 AC-3).
type boardBlobCopier struct {
	storage filestorage.Driver
	root    string
}

func (d Deps) boardBlobCopier() *boardBlobCopier {
	return &boardBlobCopier{
		storage: d.Storage,
		root:    strings.TrimSpace(d.effectiveConfig().CourseFilesRoot),
	}
}

func (c *boardBlobCopier) CopyBlob(ctx context.Context, srcKey, destKey string) error {
	srcKey = strings.TrimSpace(srcKey)
	destKey = strings.TrimSpace(destKey)
	if srcKey == "" || destKey == "" {
		return fmt.Errorf("empty storage key")
	}
	if srcKey == destKey {
		return nil
	}
	if c.storage != nil {
		rc, err := c.storage.GetObject(ctx, srcKey)
		if err != nil {
			return err
		}
		defer func() { _ = rc.Close() }()
		data, err := io.ReadAll(io.LimitReader(rc, 512<<20))
		if err != nil {
			return err
		}
		return c.storage.PutObject(ctx, destKey, bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	}
	if c.root == "" {
		return fmt.Errorf("no storage configured for attachment copy")
	}
	srcPath := filepath.Join(c.root, filepath.FromSlash(srcKey))
	destPath := filepath.Join(c.root, filepath.FromSlash(destKey))
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
