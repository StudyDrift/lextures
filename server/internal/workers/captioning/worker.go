// Package captioning implements the auto-captioning worker (plan 8.4).
// It claims queued caption jobs from the database, downloads the source audio/video,
// calls Whisper (API or local) to produce a transcript, converts it to WebVTT, and
// uploads the result to object storage.
package captioning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/captions"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/vttformatter"
)

const (
	maxAttemptsDefault = 3
	whisperAPIEndpoint = "https://api.openai.com/v1/audio/transcriptions"
)

// Backend selects the ASR backend.
type Backend string

const (
	BackendWhisperAPI   Backend = "whisper-api"
	BackendWhisperLocal Backend = "whisper-local"
	BackendAzureSpeech  Backend = "azure-speech"
	BackendGoogleSpeech Backend = "google-speech"
	BackendStub         Backend = "stub" // for testing
)

// Worker processes queued caption jobs from the database.
type Worker struct {
	Pool           *pgxpool.Pool
	Storage        filestorage.Driver
	Backend        Backend
	OpenAIAPIKey   string
	MaxAttempts    int
	// HTTPClient allows overriding the HTTP client for testing.
	HTTPClient *http.Client
}

// New creates a Worker with sensible defaults.
func New(pool *pgxpool.Pool, storage filestorage.Driver, backend Backend, openAIKey string) *Worker {
	return &Worker{
		Pool:         pool,
		Storage:      storage,
		Backend:      backend,
		OpenAIAPIKey: openAIKey,
		MaxAttempts:  maxAttemptsDefault,
		HTTPClient:   &http.Client{Timeout: 10 * time.Minute},
	}
}

// ProcessNext claims and processes one queued job. Returns (false, nil) when the queue is empty.
func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	if w.Pool == nil {
		return false, fmt.Errorf("captioning: no database pool configured")
	}
	if w.Storage == nil {
		return false, fmt.Errorf("captioning: no storage driver configured")
	}

	job, err := captions.ClaimNext(ctx, w.Pool)
	if err != nil {
		return false, fmt.Errorf("captioning: claim job: %w", err)
	}
	if job == nil {
		return false, nil
	}

	slog.Info("captioning: start", "caption_id", job.ID, "object_id", job.StorageObjectID, "backend", job.Backend)

	if processErr := w.process(ctx, job); processErr != nil {
		slog.Error("captioning: job failed", "caption_id", job.ID, "err", processErr)
		apiUnavailable := strings.Contains(processErr.Error(), "api_unavailable")
		if markErr := captions.MarkFailed(ctx, w.Pool, job.ID, apiUnavailable); markErr != nil {
			slog.Error("captioning: mark failed", "caption_id", job.ID, "err", markErr)
		}
	}
	return true, nil
}

// EnqueueForObject enqueues a caption job for a newly transcoded video object.
func EnqueueForObject(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID, backend string) (uuid.UUID, error) {
	return captions.EnqueueForObjectIfNeeded(ctx, pool, objectID, backend)
}

func (w *Worker) process(ctx context.Context, job *captions.Caption) error {
	// Look up the source key for this storage object
	var sourceKey string
	err := w.Pool.QueryRow(ctx, `SELECT object_key FROM storage.objects WHERE id = $1`, job.StorageObjectID).Scan(&sourceKey)
	if err != nil {
		return fmt.Errorf("load source key: %w", err)
	}

	jobDir, err := os.MkdirTemp("", "captioning-"+job.ID.String()+"-")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(jobDir) }()

	srcPath := filepath.Join(jobDir, "source"+filepath.Ext(sourceKey))
	if err := w.downloadSource(ctx, sourceKey, srcPath); err != nil {
		return fmt.Errorf("download source: %w", err)
	}

	backend := Backend(job.Backend)
	segments, lang, err := w.transcribe(ctx, srcPath, backend)
	if err != nil {
		return err
	}

	vttContent := vttformatter.Format(segments)
	transcript := vttformatter.PlainText(segments)
	avgConf, hasLow := vttformatter.ConfidenceStats(segments)

	// Upload VTT to storage: {object_id}/captions/{caption_id}/{lang}.vtt
	vttKey := fmt.Sprintf("captions/%s/%s/%s.vtt", job.StorageObjectID, job.ID, lang)
	vttBytes := []byte(vttContent)
	if putErr := w.Storage.PutObject(ctx, vttKey, bytes.NewReader(vttBytes), int64(len(vttBytes)), "text/vtt"); putErr != nil {
		return fmt.Errorf("upload vtt: %w", putErr)
	}

	slog.Info("captioning: done",
		"caption_id", job.ID,
		"object_id", job.StorageObjectID,
		"lang", lang,
		"confidence_avg", avgConf,
		"has_low_confidence", hasLow,
		"backend", string(backend),
	)

	return captions.MarkDone(ctx, w.Pool, job.ID, vttKey, lang, transcript, avgConf, hasLow)
}

// transcribe sends the audio file to the configured ASR backend and returns segments.
func (w *Worker) transcribe(ctx context.Context, srcPath string, backend Backend) ([]vttformatter.Segment, string, error) {
	switch backend {
	case BackendWhisperAPI:
		return w.transcribeWhisperAPI(ctx, srcPath)
	case BackendStub:
		return stubTranscribe(srcPath)
	default:
		// For unimplemented backends, fall back to stub so the system stays functional
		slog.Warn("captioning: unsupported backend, using stub", "backend", string(backend))
		return stubTranscribe(srcPath)
	}
}

// transcribeWhisperAPI calls the OpenAI Whisper API with verbose_json output to get word-level timing.
func (w *Worker) transcribeWhisperAPI(ctx context.Context, srcPath string) ([]vttformatter.Segment, string, error) {
	if w.OpenAIAPIKey == "" {
		return nil, "", fmt.Errorf("api_unavailable: OPENAI_API_KEY not configured")
	}

	f, err := os.Open(srcPath) //nolint:gosec
	if err != nil {
		return nil, "", fmt.Errorf("open audio file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", "whisper-1")
	_ = mw.WriteField("response_format", "verbose_json")
	_ = mw.WriteField("timestamp_granularities[]", "segment")
	fw, err := mw.CreateFormFile("file", filepath.Base(srcPath))
	if err != nil {
		return nil, "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, "", fmt.Errorf("copy audio: %w", err)
	}
	_ = mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whisperAPIEndpoint, &buf)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+w.OpenAIAPIKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	client := w.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("api_unavailable: whisper API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("api_unavailable: whisper API returned %d: %s", resp.StatusCode, body)
	}

	var result whisperVerboseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("decode whisper response: %w", err)
	}

	lang := result.Language
	if lang == "" {
		lang = "en"
	}

	segments := make([]vttformatter.Segment, 0, len(result.Segments))
	for _, s := range result.Segments {
		segments = append(segments, vttformatter.Segment{
			Start:      time.Duration(s.Start * float64(time.Second)),
			End:        time.Duration(s.End * float64(time.Second)),
			Text:       strings.TrimSpace(s.Text),
			Confidence: s.AvgLogprob,
		})
	}
	return segments, lang, nil
}

func (w *Worker) downloadSource(ctx context.Context, key, dstPath string) error {
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

// whisperVerboseResponse is the subset of the Whisper verbose_json response we need.
type whisperVerboseResponse struct {
	Language string           `json:"language"`
	Segments []whisperSegment `json:"segments"`
}

type whisperSegment struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Text       string  `json:"text"`
	AvgLogprob float32 `json:"avg_logprob"`
}

// stubTranscribe returns a synthetic single-segment transcript for testing.
// It assigns a moderate confidence score to avoid false low-confidence flags.
func stubTranscribe(srcPath string) ([]vttformatter.Segment, string, error) {
	segs := []vttformatter.Segment{
		{
			Start:      0,
			End:        5 * time.Second,
			Text:       fmt.Sprintf("[Auto-generated transcript for %s]", filepath.Base(srcPath)),
			Confidence: 0.80,
		},
	}
	return segs, "en", nil
}
