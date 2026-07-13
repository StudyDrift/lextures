package filestorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds credentials and endpoint for S3-compatible storage.
type S3Config struct {
	Endpoint        string // e.g. "s3.amazonaws.com" or "play.min.io" or R2 endpoint
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	Region          string
	// CDNBaseURL rewrites presigned URL hosts to the CDN edge (plan 17.5 FR-5).
	CDNBaseURL string
}

// S3Driver implements Driver using the minio-go v7 SDK.
// It works with AWS S3, Cloudflare R2, and MinIO via endpoint override.
type S3Driver struct {
	client     *minio.Client
	bucket     string
	cdnBaseURL string
}

// NewS3Driver creates an S3Driver. endpoint should be the host[:port] only (no scheme).
// When AccessKeyID and SecretAccessKey are empty, credentials fall through the
// default chain (env, shared config, ECS/EC2/IRSA task role) — preferred on AWS.
func NewS3Driver(cfg S3Config) (*S3Driver, error) {
	var creds *credentials.Credentials
	if strings.TrimSpace(cfg.AccessKeyID) != "" && strings.TrimSpace(cfg.SecretAccessKey) != "" {
		creds = credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	} else {
		// IAM roles for ECS tasks / EKS IRSA / EC2 instance profiles.
		creds = credentials.NewIAM("")
	}
	opts := &minio.Options{
		Creds:  creds,
		Secure: cfg.UseSSL,
	}
	if cfg.Region != "" {
		opts.Region = cfg.Region
	}
	client, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("filestorage/s3: new client: %w", err)
	}
	return &S3Driver{client: client, bucket: cfg.Bucket, cdnBaseURL: strings.TrimSpace(cfg.CDNBaseURL)}, nil
}

func (d *S3Driver) PutObject(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{ContentType: contentType}
	_, err := d.client.PutObject(ctx, d.bucket, key, r, size, opts)
	if err != nil {
		return fmt.Errorf("filestorage/s3: put %q: %w", key, err)
	}
	return nil
}

func (d *S3Driver) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := d.client.GetObject(ctx, d.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("filestorage/s3: get %q: %w", key, err)
	}
	return obj, nil
}

func (d *S3Driver) GetPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	params := make(url.Values)
	u, err := d.client.PresignedGetObject(ctx, d.bucket, key, ttl, params)
	if err != nil {
		return "", fmt.Errorf("filestorage/s3: presign %q: %w", key, err)
	}
	return d.rewriteCDNHost(u.String()), nil
}

func (d *S3Driver) rewriteCDNHost(raw string) string {
	cdn := strings.TrimSpace(d.cdnBaseURL)
	if cdn == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	cdnURL, err := url.Parse(cdn)
	if err != nil {
		return raw
	}
	parsed.Scheme = cdnURL.Scheme
	parsed.Host = cdnURL.Host
	return parsed.String()
}

func (d *S3Driver) DeleteObject(ctx context.Context, key string) error {
	err := d.client.RemoveObject(ctx, d.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil
		}
		return fmt.Errorf("filestorage/s3: delete %q: %w", key, err)
	}
	return nil
}

func (d *S3Driver) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	var keys []string
	for obj := range d.client.ListObjects(ctx, d.bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("filestorage/s3: list %q: %w", prefix, obj.Err)
		}
		keys = append(keys, obj.Key)
	}
	return keys, nil
}

// PresignedPutURL returns a presigned URL for uploading an object directly from the client.
func (d *S3Driver) PresignedPutURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := d.client.PresignedPutObject(ctx, d.bucket, key, ttl)
	if err != nil {
		return "", fmt.Errorf("filestorage/s3: presign PUT %q: %w", key, err)
	}
	return u.String(), nil
}
