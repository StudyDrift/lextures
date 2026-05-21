package filestorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
)

// MoveObject copies the object to destKey then deletes srcKey.
func MoveObject(ctx context.Context, d Driver, srcKey, destKey string) error {
	if srcKey == destKey {
		return nil
	}
	rc, err := d.GetObject(ctx, srcKey)
	if err != nil {
		return fmt.Errorf("move: get %q: %w", srcKey, err)
	}
	defer func() { _ = rc.Close() }()

	// Buffer small objects; stream large via temp file would be heavier — uploads are capped in practice.
	data, err := io.ReadAll(io.LimitReader(rc, 512<<20))
	if err != nil {
		return fmt.Errorf("move: read %q: %w", srcKey, err)
	}
	if err := d.PutObject(ctx, destKey, bytes.NewReader(data), int64(len(data)), "application/octet-stream"); err != nil {
		return fmt.Errorf("move: put %q: %w", destKey, err)
	}
	if err := d.DeleteObject(ctx, srcKey); err != nil {
		return fmt.Errorf("move: delete %q: %w", srcKey, err)
	}
	return nil
}

// CopyObjectS3 uses server-side copy when the driver is S3Driver.
func CopyObjectS3(ctx context.Context, d *S3Driver, srcKey, destKey string) error {
	src := minio.CopySrcOptions{Bucket: d.bucket, Object: srcKey}
	dst := minio.CopyDestOptions{Bucket: d.bucket, Object: destKey}
	_, err := d.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("filestorage/s3: copy %q -> %q: %w", srcKey, destKey, err)
	}
	return d.DeleteObject(ctx, srcKey)
}

// MoveObjectLocal renames a file under the local driver root.
func MoveObjectLocal(root, srcKey, destKey string) error {
	srcPath := filepath.Join(root, filepath.FromSlash(srcKey))
	destPath := filepath.Join(root, filepath.FromSlash(destKey))
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(srcPath, destPath); err != nil {
		// Cross-device: copy then delete
		in, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		return os.Remove(srcPath)
	}
	return nil
}
