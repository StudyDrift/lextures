package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/service/boardexport"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func exportJobJSON(j board.ExportJob, downloadURL string) map[string]any {
	out := map[string]any{
		"id":                j.ID,
		"boardId":           j.BoardID,
		"format":            j.Format,
		"status":            j.Status,
		"error":             j.Error,
		"includeModeration": j.IncludeModeration,
		"createdAt":         j.CreatedAt.UTC().Format(time.RFC3339),
	}
	if j.StorageKey != nil {
		out["storageKey"] = *j.StorageKey
	} else {
		out["storageKey"] = nil
	}
	if j.RequestedBy != nil {
		out["requestedBy"] = *j.RequestedBy
	} else {
		out["requestedBy"] = nil
	}
	if j.CompletedAt != nil {
		out["completedAt"] = j.CompletedAt.UTC().Format(time.RFC3339)
	} else {
		out["completedAt"] = nil
	}
	if downloadURL != "" {
		out["downloadUrl"] = downloadURL
	} else {
		out["downloadUrl"] = nil
	}
	return out
}

func (d Deps) boardExportStorage() filestorage.Driver {
	if d.Storage != nil {
		return d.Storage
	}
	root := strings.TrimSpace(d.effectiveConfig().CourseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	return &filestorage.LocalDriver{Root: root}
}

// handleCreateBoardExport is POST .../boards/{board_id}/export (VC.9).
func (d Deps) handleCreateBoardExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		boardID := chi.URLParam(r, "board_id")
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check permission.")
			return
		}
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !hasPerm && !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Export requires manage permission.")
			return
		}
		var body struct {
			Format            string `json:"format"`
			IncludeModeration bool   `json:"includeModeration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		format, err := board.NormalizeExportFormat(body.Format)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid export format.")
			return
		}
		includeMod := body.IncludeModeration && (hasPerm || caps.CanManage)
		bid, err := uuid.Parse(b.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid board id.")
			return
		}
		job, err := board.CreateExportJob(r.Context(), d.Pool, bid, viewer, format, includeMod)
		if err != nil || job == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create export job.")
			return
		}
		canManage := hasPerm || caps.CanManage
		// Prefer inline render (gofpdf/CSV/PNG are fast for typical boards). If that
		// fails, enqueue for the background worker to retry.
		if runErr := d.runBoardExportInline(r, job.ID, courseCode, b.ID, viewer, format, includeMod, canManage); runErr != nil {
			if _, qerr := background.EnqueueBoardExport(r.Context(), d.Pool, background.BoardExportPayload{
				JobID:             job.ID,
				CourseCode:        courseCode,
				BoardID:           b.ID,
				RequestedBy:       viewer.String(),
				Format:            format,
				IncludeModeration: includeMod,
				CanManage:         canManage,
			}); qerr != nil {
				_ = board.UpdateExportJobStatus(r.Context(), d.Pool, job.ID, board.ExportStatusFailed, nil, runErr.Error())
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start export.")
				return
			}
		}
		job, err = board.GetExportJob(r.Context(), d.Pool, job.ID)
		if err != nil || job == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load export job.")
			return
		}
		telemetry.RecordBusinessEvent("board.export.created")
		downloadURL := ""
		if job.Status == board.ExportStatusDone {
			downloadURL = fmt.Sprintf("/api/v1/courses/%s/boards/%s/export/%s/content", courseCode, boardID, job.ID)
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job": exportJobJSON(*job, downloadURL)})
	}
}

func (d Deps) runBoardExportInline(
	r *http.Request,
	jobID, courseCode, boardID string,
	viewer uuid.UUID,
	format string,
	includeMod, canManage bool,
) error {
	_ = board.UpdateExportJobStatus(r.Context(), d.Pool, jobID, board.ExportStatusRunning, nil, "")
	res, err := boardexport.Build(r.Context(), d.Pool, boardexport.BuildOpts{
		CourseCode:        courseCode,
		BoardID:           boardID,
		ViewerID:          viewer,
		Format:            format,
		IncludeModeration: includeMod,
		Caps:              board.Capabilities{CanView: true, CanManage: canManage},
	})
	if err != nil {
		return err
	}
	key := fmt.Sprintf("boards/%s/%s/exports/%s.%s", courseCode, boardID, jobID, res.Extension)
	if err := d.boardExportStorage().PutObject(r.Context(), key, bytes.NewReader(res.Bytes), int64(len(res.Bytes)), res.ContentType); err != nil {
		return err
	}
	return board.UpdateExportJobStatus(r.Context(), d.Pool, jobID, board.ExportStatusDone, &key, "")
}

// handleGetBoardExport is GET .../export/{job_id} (VC.9).
func (d Deps) handleGetBoardExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode := chi.URLParam(r, "course_code")
		boardID := chi.URLParam(r, "board_id")
		jobID := chi.URLParam(r, "job_id")
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check permission.")
			return
		}
		_, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !hasPerm && !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Export requires manage permission.")
			return
		}
		job, err := board.GetExportJobForBoard(r.Context(), d.Pool, courseCode, boardID, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load export job.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Export job not found.")
			return
		}
		downloadURL := ""
		if job.Status == board.ExportStatusDone && job.StorageKey != nil {
			downloadURL = fmt.Sprintf("/api/v1/courses/%s/boards/%s/export/%s/content", courseCode, boardID, job.ID)
		}
		writeJSON(w, http.StatusOK, exportJobJSON(*job, downloadURL))
	}
}

// handleGetBoardExportContent is GET .../export/{job_id}/content (VC.9).
func (d Deps) handleGetBoardExportContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode := chi.URLParam(r, "course_code")
		boardID := chi.URLParam(r, "board_id")
		jobID := chi.URLParam(r, "job_id")
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check permission.")
			return
		}
		_, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !hasPerm && !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Export requires manage permission.")
			return
		}
		job, err := board.GetExportJobForBoard(r.Context(), d.Pool, courseCode, boardID, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load export job.")
			return
		}
		if job == nil || job.Status != board.ExportStatusDone || job.StorageKey == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Export file not ready.")
			return
		}
		rc, err := d.boardExportStorage().GetObject(r.Context(), *job.StorageKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Export file not found.")
			return
		}
		defer func() { _ = rc.Close() }()
		ct := "application/octet-stream"
		filename := "board-export"
		switch job.Format {
		case board.ExportFormatCSV:
			ct = "text/csv; charset=utf-8"
			filename += ".csv"
		case board.ExportFormatPDF:
			ct = "application/pdf"
			filename += ".pdf"
		case board.ExportFormatImage:
			ct = "image/png"
			filename += ".png"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, rc)
	}
}

// handleGetBoardQR is GET .../boards/{board_id}/qr (VC.9).
func (d Deps) handleGetBoardQR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode := chi.URLParam(r, "course_code")
		boardID := chi.URLParam(r, "board_id")
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		_, _, ok = d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		accessURL, okURL := d.boardAccessURL(w, r, courseCode, boardID)
		if !okURL {
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "png"
		}
		size := 256
		if raw := strings.TrimSpace(r.URL.Query().Get("size")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n >= 64 && n <= 1024 {
				size = n
			}
		}
		var (
			body []byte
			ct   string
			err  error
		)
		switch format {
		case "svg":
			body, err = boardexport.RenderQRSVG(accessURL, size)
			ct = "image/svg+xml"
		default:
			body, err = boardexport.RenderQRPNG(accessURL, size)
			ct = "image/png"
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to render QR code.")
			return
		}
		telemetry.RecordBusinessEvent("board.qr.generated")
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Board-Access-Url", accessURL)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// boardAccessURL returns the in-app board URL, or a caller-supplied share URL that
// must stay on PublicWebOrigin (board-links or this board path).
func (d Deps) boardAccessURL(w http.ResponseWriter, r *http.Request, courseCode, boardID string) (string, bool) {
	origin := strings.TrimRight(strings.TrimSpace(d.effectiveConfig().PublicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	defaultURL := fmt.Sprintf("%s/courses/%s/boards/%s", origin, courseCode, boardID)
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		return defaultURL, true
	}
	if !strings.HasPrefix(raw, origin+"/") {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "QR url must be on the app origin.")
		return "", false
	}
	path := strings.TrimPrefix(raw, origin)
	boardPath := fmt.Sprintf("/courses/%s/boards/%s", courseCode, boardID)
	if path == boardPath || strings.HasPrefix(path, "/board-links/") {
		return raw, true
	}
	apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "QR url is not a permitted board access URL.")
	return "", false
}

// handleGetBoardEmbed is GET .../boards/{board_id}/embed (VC.9).
func (d Deps) handleGetBoardEmbed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode := chi.URLParam(r, "course_code")
		boardID := chi.URLParam(r, "board_id")
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, err := board.Get(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load board.")
			return
		}
		if b == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"mode":    "denied",
				"board":   nil,
				"posts":   []any{},
				"sections": []any{},
				"capabilities": map[string]any{
					"canView": false, "canPost": false, "canInteract": false, "canArrange": false, "canManage": false,
				},
			})
			return
		}
		opts := board.ResolveOpts{
			CourseCode:             courseCode,
			ExternalSharingAllowed: d.effectiveConfig().FFBoardsExternalSharing,
		}
		if d.effectiveConfig().CoppaWorkflowEnabled {
			hasMinors, err := board.CourseHasEnrolledMinors(r.Context(), d.Pool, courseCode)
			if err == nil {
				opts.ForbidExternalForMinors = hasMinors
			}
		}
		caps, err := board.ResolveAccess(r.Context(), d.Pool, b, viewer, opts)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve board access.")
			return
		}
		if !caps.CanView {
			writeJSON(w, http.StatusOK, map[string]any{
				"mode":    "denied",
				"board":   boardJSONWithAccess(*b, caps),
				"posts":   []any{},
				"sections": []any{},
				"capabilities": map[string]any{
					"canView": false, "canPost": false, "canInteract": false, "canArrange": false, "canManage": false,
				},
			})
			return
		}
		mode := "readonly"
		if caps.CanPost || caps.CanInteract {
			mode = "interactive"
		}
		posts, err := board.ListPosts(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load posts.")
			return
		}
		posts = board.FilterVisiblePosts(posts, viewer.String(), caps.CanManage)
		sections, err := board.ListSections(r.Context(), d.Pool, courseCode, boardID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load sections.")
			return
		}
		avOn := d.effectiveConfig().AvScanningEnabled
		postJSON := make([]map[string]any, 0, len(posts))
		for _, p := range posts {
			postJSON = append(postJSON, boardPostJSONWithAttribution(p, courseCode, avOn, b.Attribution, caps))
		}
		secJSON := make([]map[string]any, 0, len(sections))
		for _, s := range sections {
			secJSON = append(secJSON, boardSectionJSON(s))
		}
		telemetry.RecordBusinessEvent("board.embed.rendered")
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":  mode,
			"board": boardJSONWithAccess(*b, caps),
			"posts": postJSON,
			"sections": secJSON,
			"capabilities": map[string]any{
				"canView":     caps.CanView,
				"canPost":     caps.CanPost,
				"canInteract": caps.CanInteract,
				"canArrange":  caps.CanArrange,
				"canManage":   caps.CanManage,
			},
		})
	}
}
