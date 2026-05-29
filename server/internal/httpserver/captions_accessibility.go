package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	captionsrepo "github.com/lextures/lextures/server/internal/repos/captions"
	captionssvc "github.com/lextures/lextures/server/internal/service/captions"
	"github.com/lextures/lextures/server/internal/service/vttformatter"
)

func (d Deps) captionsAccessible() bool {
	cfg := d.effectiveConfig()
	return cfg.VideoCaptionsEnabled || cfg.AutoCaptioningEnabled
}

func (d Deps) registerCaptionAccessibilityRoutes(r chi.Router) {
	r.Post("/api/v1/files/{object_id}/captions/import", d.handleImportCaption())
	r.Delete("/api/v1/files/{object_id}/captions/{caption_id}", d.handleDeleteCaption())
	r.Patch("/api/v1/files/{object_id}/captions/{caption_id}", d.handlePatchCaptionVTT())
	r.Get("/api/v1/files/{object_id}/captions/{caption_id}/export", d.handleExportCaption())
	r.Get("/api/v1/admin/captions/compliance", d.handleCaptionComplianceReport())
}

type patchCaptionVTTBody struct {
	VTTContent string `json:"vtt_content"`
}

func (d Deps) handlePatchCaptionVTT() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.captionsAccessible() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Video captions are not enabled.")
			return
		}
		objectID, captionID, ok := d.parseCaptionIDs(w, r)
		if !ok {
			return
		}
		var body patchCaptionVTTBody
		if decErr := json.NewDecoder(r.Body).Decode(&body); decErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.VTTContent) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "vtt_content is required.")
			return
		}
		cap, err := captionsrepo.Load(r.Context(), d.Pool, captionID)
		if err != nil || cap == nil || cap.StorageObjectID != objectID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Caption not found.")
			return
		}
		segments, parseErr := captionssvc.ParseVTT(body.VTTContent)
		if parseErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid WebVTT content.")
			return
		}
		vttKey := "captions/" + objectID.String() + "/" + captionID.String() + "/" + cap.Lang + ".vtt"
		if cap.VTTKey != nil && *cap.VTTKey != "" {
			vttKey = *cap.VTTKey
		}
		transcript := captionssvc.PlainTranscript(segments)
		if storeErr := d.storeVTT(r, vttKey, body.VTTContent); storeErr != nil {
			slog.Error("captions: store vtt", "err", storeErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store captions.")
			return
		}
		if updateErr := captionsrepo.UpdateVTTContent(r.Context(), d.Pool, captionID, userID, body.VTTContent, transcript, vttKey); updateErr != nil {
			slog.Error("captions: patch vtt", "err", updateErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update caption.")
			return
		}
		updated, _ := captionsrepo.Load(r.Context(), d.Pool, captionID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(captionToResponse(updated))
	}
}

func (d Deps) handleImportCaption() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.captionsAccessible() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Video captions are not enabled.")
			return
		}
		objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
			return
		}
		if parseErr := r.ParseMultipartForm(8 << 20); parseErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid multipart form.")
			return
		}
		file, header, fileErr := r.FormFile("file")
		if fileErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "file is required.")
			return
		}
		defer func() { _ = file.Close() }()
		raw, readErr := io.ReadAll(file)
		if readErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to read upload.")
			return
		}
		content := string(raw)
		var vtt string
		name := strings.ToLower(header.Filename)
		switch {
		case strings.HasSuffix(name, ".srt"):
			vtt, err = captionssvc.VTTFromSRT(content)
		case strings.HasSuffix(name, ".vtt"):
			vtt = content
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Only .vtt and .srt files are supported.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not parse caption file.")
			return
		}
		segments, parseErr := captionssvc.ParseVTT(vtt)
		if parseErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid caption content.")
			return
		}
		transcript := captionssvc.PlainTranscript(segments)
		backend := "import"
		capID, enqErr := captionsrepo.Enqueue(r.Context(), d.Pool, objectID, backend)
		if enqErr != nil {
			slog.Error("captions: import enqueue", "err", enqErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create caption record.")
			return
		}
		vttKey := "captions/" + objectID.String() + "/" + capID.String() + "/en.vtt"
		if storeErr := d.storeVTT(r, vttKey, vtt); storeErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store captions.")
			return
		}
		if updateErr := captionsrepo.UpdateVTTContent(r.Context(), d.Pool, capID, userID, vtt, transcript, vttKey); updateErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save imported captions.")
			return
		}
		updated, _ := captionsrepo.Load(r.Context(), d.Pool, capID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(captionToResponse(updated))
	}
}

func (d Deps) handleDeleteCaption() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.captionsAccessible() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Video captions are not enabled.")
			return
		}
		_, captionID, ok := d.parseCaptionIDs(w, r)
		if !ok {
			return
		}
		if err := captionsrepo.Delete(r.Context(), d.Pool, captionID); err != nil {
			slog.Error("captions: delete", "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete caption.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleExportCaption() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.captionsAccessible() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Video captions are not enabled.")
			return
		}
		objectID, captionID, ok := d.parseCaptionIDs(w, r)
		if !ok {
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "vtt"
		}
		cap, err := captionsrepo.Load(r.Context(), d.Pool, captionID)
		if err != nil || cap == nil || cap.StorageObjectID != objectID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Caption not found.")
			return
		}
		vttBody, loadErr := d.loadVTTBody(r, cap)
		if loadErr != nil || vttBody == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "VTT not available.")
			return
		}
		switch format {
		case "vtt":
			w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="captions.vtt"`)
			_, _ = w.Write([]byte(vttBody))
		case "srt":
			segments, parseErr := vttformatter.ParseVTT(vttBody)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to convert captions.")
				return
			}
			w.Header().Set("Content-Type", "application/x-subrip; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="captions.srt"`)
			_, _ = w.Write([]byte(captionssvc.SRTFromVTT(segments)))
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "format must be vtt or srt.")
		}
	}
}

// handleCaptionComplianceReport is GET /api/v1/admin/captions/compliance (alias for caption-coverage).
func (d Deps) handleCaptionComplianceReport() http.HandlerFunc {
	return d.handleCaptionCoverageReport()
}

func (d Deps) parseCaptionIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	objectID, err := uuid.Parse(chi.URLParam(r, "object_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid object_id.")
		return uuid.Nil, uuid.Nil, false
	}
	captionID, err := uuid.Parse(chi.URLParam(r, "caption_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid caption_id.")
		return uuid.Nil, uuid.Nil, false
	}
	return objectID, captionID, true
}

func (d Deps) storeVTT(r *http.Request, key, content string) error {
	if d.Storage == nil {
		return nil
	}
	return d.Storage.PutObject(r.Context(), key, strings.NewReader(content), int64(len(content)), "text/vtt")
}

func (d Deps) loadVTTBody(r *http.Request, cap *captionsrepo.Caption) (string, error) {
	if cap.TranscriptText != nil && cap.VTTKey != nil {
		if d.Storage != nil {
			rc, err := d.Storage.GetObject(r.Context(), *cap.VTTKey)
			if err == nil {
				defer func() { _ = rc.Close() }()
				b, readErr := io.ReadAll(rc)
				if readErr == nil {
					return string(b), nil
				}
			}
		}
	}
	if cap.TranscriptText != nil {
		return vttformatter.Format([]vttformatter.Segment{{Text: *cap.TranscriptText}}), nil
	}
	return "", nil
}
