package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/portfolios"
)

// handlePostPortfolioArtifactUpload is POST /api/v1/me/portfolios/{pid}/artifacts/upload
// Multipart form: file (required), title (required), description (optional),
// outcomeIds (optional JSON array string), isPublic (optional "true"/"false").
func (d Deps) handlePostPortfolioArtifactUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pid, ok := parsePathUUID(w, r, "pid")
		if !ok {
			return
		}
		if err := r.ParseMultipartForm(12 << 20); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid form or file too large.")
			return
		}
		f, header, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing 'file' part.")
			return
		}
		defer func() { _ = f.Close() }()

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
			return
		}
		description := strings.TrimSpace(r.FormValue("description"))
		isPublic := strings.EqualFold(strings.TrimSpace(r.FormValue("isPublic")), "true")

		var outcomeIDs []uuid.UUID
		if raw := strings.TrimSpace(r.FormValue("outcomeIds")); raw != "" {
			var ids []string
			if err := json.Unmarshal([]byte(raw), &ids); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcomeIds JSON.")
				return
			}
			outcomeIDs, err = parseUUIDList(ids)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outcome id.")
				return
			}
		}

		ct := strings.TrimSpace(header.Header.Get("Content-Type"))
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(header.Filename))
		}
		if ct == "" {
			ct = "application/octet-stream"
		}
		allowed := strings.HasPrefix(ct, "text/") ||
			ct == "application/pdf" ||
			strings.HasPrefix(ct, "image/") ||
			strings.HasPrefix(ct, "video/") ||
			ct == "application/octet-stream"
		if !allowed {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported file type.")
			return
		}
		if header.Size <= 0 || header.Size > 20<<20 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File must be between 1 byte and 20MB.")
			return
		}

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".bin"
		}
		fileUUID := uuid.New().String()
		storageKey := fmt.Sprintf("portfolios/%s/%s%s", uid.String(), fileUUID, ext)

		cfg := d.effectiveConfig()
		if d.Storage != nil {
			if perr := d.Storage.PutObject(r.Context(), storageKey, f, header.Size, ct); perr != nil {
				log.Printf("portfolio-artifact-upload: PutObject key=%s err=%v", storageKey, perr)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store file.")
				return
			}
		} else {
			root := strings.TrimSpace(cfg.CourseFilesRoot)
			if root == "" {
				root = "data/course-files"
			}
			p := coursefiles.BlobDiskPath(root, uid.String(), fileUUID+ext)
			if werr := writeLocalFile(p, f); werr != nil {
				log.Printf("portfolio-artifact-upload: local write key=%s err=%v", storageKey, werr)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store file.")
				return
			}
			storageKey = fileUUID + ext
		}

		a, err := portfolios.CreateArtifact(r.Context(), d.Pool, uid, pid, portfolios.CreateArtifactInput{
			ArtifactType: "upload",
			Title:        title,
			Description:  description,
			FileKey:      storageKey,
			FileName:     header.Filename,
			FileMime:     ct,
			OutcomeIDs:   outcomeIDs,
			IsPublic:     isPublic,
		})
		if err != nil {
			d.writePortfolioRepoErr(w, err, "portfolio")
			return
		}
		writeJSON(w, http.StatusCreated, artifactToJSON(a))
	}
}

// handleGetPortfolioArtifactContent is GET /api/v1/me/portfolios/{pid}/artifacts/{aid}/content
func (d Deps) handleGetPortfolioArtifactContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireEportfolioEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		aid, ok := parsePathUUID(w, r, "aid")
		if !ok {
			return
		}
		art, err := portfolios.GetArtifactOwned(r.Context(), d.Pool, uid, aid)
		if err != nil {
			d.writePortfolioRepoErr(w, err, "artifact")
			return
		}
		if strings.TrimSpace(art.FileKey) == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "This artifact has no file.")
			return
		}

		cfg := d.effectiveConfig()
		if d.Storage != nil {
			ttlSecs := cfg.StoragePresignTTL
			if ttlSecs <= 0 {
				ttlSecs = 3600
			}
			presignURL, presignErr := d.Storage.GetPresignedURL(r.Context(), art.FileKey, time.Duration(ttlSecs)*time.Second)
			if presignErr == nil && presignURL != "" {
				http.Redirect(w, r, presignURL, http.StatusFound)
				return
			}
		}

		b, err := d.readPortfolioArtifactBytes(r.Context(), art)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		ct := strings.TrimSpace(art.FileMime)
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		if art.FileName != "" {
			w.Header().Set("Content-Disposition", "inline; filename=\""+art.FileName+"\"")
		}
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

func (d Deps) readPortfolioArtifactBytes(ctx context.Context, art *portfolios.Artifact) ([]byte, error) {
	if art == nil || strings.TrimSpace(art.FileKey) == "" {
		return nil, fmt.Errorf("no file")
	}
	if d.Storage != nil {
		rc, err := d.Storage.GetObject(ctx, art.FileKey)
		if err == nil {
			defer func() { _ = rc.Close() }()
			return io.ReadAll(rc)
		}
	}
	cfg := d.effectiveConfig()
	root := strings.TrimSpace(cfg.CourseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	if art.SourceCourseID != nil {
		if code, err := repoCourse.GetCourseCodeByID(ctx, d.Pool, *art.SourceCourseID); err == nil && code != nil {
			if b, err := os.ReadFile(coursefiles.BlobDiskPath(root, *code, art.FileKey)); err == nil {
				return b, nil
			}
			if b, err := os.ReadFile(filepath.Join(root, *code, art.FileKey)); err == nil {
				return b, nil
			}
		}
	}
	var ownerID uuid.UUID
	if err := d.Pool.QueryRow(ctx, `SELECT owner_id FROM portfolio.portfolios WHERE id = $1`, art.PortfolioID).Scan(&ownerID); err == nil {
		if b, err := os.ReadFile(coursefiles.BlobDiskPath(root, ownerID.String(), art.FileKey)); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("file not found")
}