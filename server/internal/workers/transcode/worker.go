// Package transcode implements the video transcoding worker (plan 8.3).
// It invokes FFmpeg as a subprocess to produce multi-bitrate HLS renditions and a poster thumbnail.
package transcode

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/transcodejobs"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/workers/captioning"
)

const (
	maxAttemptsDefault = 3
	defaultFFmpegPath  = "ffmpeg"
)

// Rendition describes a single HLS quality level.
type Rendition struct {
	Name      string
	Height    int
	VideoBR   string
	AudioBR   string
	Bandwidth int // bits/s for EXT-X-STREAM-INF
}

// DefaultRenditions are the three required HLS quality levels (FR-2).
var DefaultRenditions = []Rendition{
	{Name: "360p", Height: 360, VideoBR: "500k", AudioBR: "96k", Bandwidth: 600000},
	{Name: "720p", Height: 720, VideoBR: "2000k", AudioBR: "128k", Bandwidth: 2200000},
	{Name: "1080p", Height: 1080, VideoBR: "4000k", AudioBR: "192k", Bandwidth: 4300000},
}

// Worker processes queued transcode jobs from the database.
type Worker struct {
	Pool                 *pgxpool.Pool
	Storage              filestorage.Driver
	FFmpegPath           string
	MaxAttempts          int
	AutoCaptionOnComplete bool
	CaptionBackend       string
}

// New creates a Worker with sensible defaults.
func New(pool *pgxpool.Pool, storage filestorage.Driver) *Worker {
	return &Worker{
		Pool:        pool,
		Storage:     storage,
		FFmpegPath:  defaultFFmpegPath,
		MaxAttempts: maxAttemptsDefault,
	}
}

// ProcessNext claims and processes one queued job. Returns (false, nil) when the queue is empty.
func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	if w.Pool == nil {
		return false, fmt.Errorf("transcode: no database pool configured")
	}
	if w.Storage == nil {
		return false, fmt.Errorf("transcode: no storage driver configured")
	}
	job, err := transcodejobs.ClaimNext(ctx, w.Pool)
	if err != nil {
		return false, fmt.Errorf("transcode: claim job: %w", err)
	}
	if job == nil {
		return false, nil
	}

	slog.Info("transcode: start", "job_id", job.ID, "source_key", job.SourceKey, "attempt", job.Attempts)

	if processErr := w.process(ctx, job); processErr != nil {
		slog.Error("transcode: job failed", "job_id", job.ID, "err", processErr)
		if markErr := transcodejobs.MarkFailed(ctx, w.Pool, job.ID, processErr.Error(), w.MaxAttempts); markErr != nil {
			slog.Error("transcode: mark failed", "job_id", job.ID, "err", markErr)
		}
	}
	return true, nil
}

func (w *Worker) process(ctx context.Context, job *transcodejobs.Job) error {
	jobDir, err := os.MkdirTemp("", "transcode-"+job.ID.String()+"-")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(jobDir) }()

	srcPath := filepath.Join(jobDir, "source"+filepath.Ext(job.SourceKey))
	if err := w.downloadSource(ctx, job.SourceKey, srcPath); err != nil {
		return fmt.Errorf("download source: %w", err)
	}

	hlsDir := filepath.Join(jobDir, "hls")
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		return fmt.Errorf("create hls dir: %w", err)
	}

	outputPrefix := strings.TrimSuffix(job.SourceKey, filepath.Ext(job.SourceKey)) + "/hls"

	// Extract poster thumbnail at 5 s (FR-5)
	posterPath := filepath.Join(jobDir, "poster.jpg")
	if err := w.extractPoster(ctx, srcPath, posterPath); err != nil {
		slog.Warn("transcode: poster extraction failed (non-fatal)", "job_id", job.ID, "err", err)
		posterPath = ""
	}

	// Transcode to multi-bitrate HLS (FR-2, FR-3)
	if err := w.transcodeHLS(ctx, srcPath, hlsDir); err != nil {
		return fmt.Errorf("hls transcode: %w", err)
	}

	// Upload HLS segments and manifests
	if err := w.uploadDir(ctx, hlsDir, outputPrefix); err != nil {
		return fmt.Errorf("upload hls: %w", err)
	}

	// Upload poster if it was generated
	var posterKey string
	if posterPath != "" {
		posterKey = strings.TrimSuffix(job.SourceKey, filepath.Ext(job.SourceKey)) + "/poster.jpg"
		if upErr := w.uploadFile(ctx, posterPath, posterKey, "image/jpeg"); upErr != nil {
			slog.Warn("transcode: poster upload failed (non-fatal)", "job_id", job.ID, "err", upErr)
			posterKey = ""
		}
	}

	masterPlaylist := outputPrefix + "/master.m3u8"
	slog.Info("transcode: done",
		"job_id", job.ID,
		"source_key", job.SourceKey,
		"master_playlist", masterPlaylist,
	)

	if err := transcodejobs.MarkDone(ctx, w.Pool, job.ID, outputPrefix, masterPlaylist, posterKey, nil); err != nil {
		return err
	}
	if w.AutoCaptionOnComplete && job.StorageObjectID != nil {
		backend := w.CaptionBackend
		if backend == "" {
			backend = string(captioning.BackendWhisperAPI)
		}
		if _, enqErr := captioning.EnqueueForObject(ctx, w.Pool, *job.StorageObjectID, backend); enqErr != nil {
			slog.Warn("transcode: enqueue caption", "object_id", *job.StorageObjectID, "err", enqErr)
		}
	}
	return nil
}

func (w *Worker) downloadSource(ctx context.Context, key, dstPath string) error {
	if w.Storage == nil {
		return fmt.Errorf("no storage driver configured")
	}
	rc, err := w.Storage.GetObject(ctx, key)
	if err != nil {
		return fmt.Errorf("get object %q: %w", key, err)
	}
	defer func() { _ = rc.Close() }()

	f, err := os.Create(dstPath) //nolint:gosec
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, rc); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// extractPoster extracts a JPEG frame at 5 seconds (FR-5).
func (w *Worker) extractPoster(ctx context.Context, srcPath, dstPath string) error {
	args := []string{
		"-y", "-ss", "5",
		"-i", srcPath,
		"-vframes", "1",
		"-vf", "scale='min(1280,iw)':'min(720,ih)':force_original_aspect_ratio=decrease",
		"-q:v", "2",
		dstPath,
	}
	out, err := exec.CommandContext(ctx, w.ffmpegBin(), args...).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// transcodeHLS generates multi-bitrate HLS segments and manifests in outDir.
func (w *Worker) transcodeHLS(ctx context.Context, srcPath, outDir string) error {
	n := len(DefaultRenditions)

	// Build filter_complex split + scale graph
	var fb strings.Builder
	fmt.Fprintf(&fb, "[0:v]split=%d", n)
	for i := range DefaultRenditions {
		fmt.Fprintf(&fb, "[v%d]", i)
	}
	for i, r := range DefaultRenditions {
		fb.WriteString(";")
		fmt.Fprintf(&fb, "[v%d]scale=-2:%d[v%dout]", i, r.Height, i)
	}

	args := []string{"-y", "-i", srcPath, "-filter_complex", fb.String()}
	for i, r := range DefaultRenditions {
		segPattern := filepath.Join(outDir, r.Name+"_%03d.ts")
		playlistPath := filepath.Join(outDir, r.Name+".m3u8")
		args = append(args,
			"-map", fmt.Sprintf("[v%dout]", i), "-map", "0:a",
			"-c:v", "libx264", "-preset", "fast", "-crf", "23",
			"-b:v", r.VideoBR,
			"-c:a", "aac", "-b:a", r.AudioBR,
			"-hls_time", "6",
			"-hls_list_size", "0",
			"-hls_segment_filename", segPattern,
			"-f", "hls", playlistPath,
		)
	}

	out, err := exec.CommandContext(ctx, w.ffmpegBin(), args...).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}

	return writeMasterPlaylist(outDir)
}

// writeMasterPlaylist writes master.m3u8 referencing each rendition playlist.
func writeMasterPlaylist(outDir string) error {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n")
	for _, r := range DefaultRenditions {
		fmt.Fprintf(&sb,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=x%d,CODECS=\"avc1.64001f,mp4a.40.2\"\n%s.m3u8\n",
			r.Bandwidth, r.Height, r.Name)
	}
	return os.WriteFile(filepath.Join(outDir, "master.m3u8"), []byte(sb.String()), 0o644)
}

// uploadDir walks dir and uploads all .m3u8 and .ts files to storage under outputPrefix.
func (w *Worker) uploadDir(ctx context.Context, dir, outputPrefix string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		var mime string
		switch ext {
		case ".m3u8":
			mime = "application/vnd.apple.mpegurl"
		case ".ts":
			mime = "video/mp2t"
		default:
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		key := outputPrefix + "/" + strings.ReplaceAll(rel, "\\", "/")
		return w.uploadFile(ctx, path, key, mime)
	})
}

func (w *Worker) uploadFile(ctx context.Context, localPath, key, mime string) error {
	f, err := os.Open(localPath) //nolint:gosec
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	return w.Storage.PutObject(ctx, key, f, info.Size(), mime)
}

func (w *Worker) ffmpegBin() string {
	if w.FFmpegPath != "" {
		return w.FFmpegPath
	}
	return defaultFFmpegPath
}

// EnqueueForObject enqueues a transcode job for a newly uploaded video object.
func EnqueueForObject(ctx context.Context, pool *pgxpool.Pool, sourceKey string, objectID *uuid.UUID) (uuid.UUID, error) {
	return transcodejobs.Enqueue(ctx, pool, sourceKey, objectID)
}

// IsVideoMIME returns true for MIME types that should be transcoded.
func IsVideoMIME(mime string) bool {
	return strings.HasPrefix(mime, "video/")
}

// BuildMasterPlaylistContent returns the text content of a master.m3u8 manifest.
// Useful for tests and validation without writing to disk.
func BuildMasterPlaylistContent() string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n")
	for _, r := range DefaultRenditions {
		fmt.Fprintf(&sb,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=x%d,CODECS=\"avc1.64001f,mp4a.40.2\"\n%s.m3u8\n",
			r.Bandwidth, r.Height, r.Name)
	}
	return sb.String()
}
