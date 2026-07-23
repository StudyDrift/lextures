package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
)

const (
	sttRateLimitPerH  = 30
	sttMaxUploadBytes = 10 << 20
	whisperSTTEndpoint = "https://api.openai.com/v1/audio/transcriptions"
)

type sttRateEntry struct {
	count  int
	window time.Time
}

var (
	sttRateMu     sync.Mutex
	sttRateByUser = map[uuid.UUID]sttRateEntry{}
)

func (d Deps) checkSTTRateLimit(userID uuid.UUID) bool {
	sttRateMu.Lock()
	defer sttRateMu.Unlock()
	now := time.Now()
	e, ok := sttRateByUser[userID]
	if !ok || now.Sub(e.window) >= time.Hour {
		sttRateByUser[userID] = sttRateEntry{count: 1, window: now}
		return true
	}
	if e.count >= sttRateLimitPerH {
		return false
	}
	e.count++
	sttRateByUser[userID] = e
	return true
}

type sttTranscribeResponse struct {
	Transcript string `json:"transcript"`
}

type whisperTextResponse struct {
	Text string `json:"text"`
}

func (d Deps) handlePostSTTTranscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireSpeechToText(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.checkSTTRateLimit(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Speech-to-text rate limit exceeded.")
			return
		}
		if err := r.ParseMultipartForm(sttMaxUploadBytes); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid multipart form.")
			return
		}
		file, header, err := r.FormFile("audio")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing audio file.")
			return
		}
		defer func() { _ = file.Close() }()

		cfg := d.effectiveConfig()
		if cfg.OpenAIAPIKey == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Speech-to-text fallback is not configured.")
			return
		}

		transcript, err := transcribeAudioWhisper(r.Context(), cfg.OpenAIAPIKey, file, header.Filename)
		if err != nil {
			writeAIGenerationFailed(w, r, "Transcription failed.", err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sttTranscribeResponse{Transcript: transcript})
	}
}

func transcribeAudioWhisper(ctx context.Context, apiKey string, audio io.Reader, filename string) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", "whisper-1")
	_ = mw.WriteField("response_format", "json")
	name := filename
	if name == "" {
		name = "audio.webm"
	}
	fw, err := mw.CreateFormFile("file", filepath.Base(name))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, audio); err != nil {
		return "", fmt.Errorf("copy audio: %w", err)
	}
	_ = mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whisperSTTEndpoint, &buf)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("whisper status %d: %s", resp.StatusCode, body)
	}
	var result whisperTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode whisper response: %w", err)
	}
	return strings.TrimSpace(result.Text), nil
}
