package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/repos/userimport"
	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
	"github.com/lextures/lextures/server/internal/service/clamav"
	"github.com/lextures/lextures/server/internal/service/csvimport"
)

const (
	maxUserImportFileBytes = 50 << 20 // 50 MB
	userImportRateLimit    = 5
)

func (d Deps) bulkCsvImportEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().BulkCsvImportEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Bulk CSV user import is not enabled.")
		return false
	}
	if !d.adminConsoleEnabled(w) {
		return false
	}
	return true
}

func (d Deps) registerAdminImportRoutes(r chi.Router) {
	r.Post("/api/v1/admin-console/imports", d.handleAdminImportUpload())
	r.Get("/api/v1/admin-console/imports", d.handleAdminImportList())
	r.Get("/api/v1/admin-console/imports/{jobId}", d.handleAdminImportStatus())
	r.Get("/api/v1/admin-console/imports/{jobId}/result", d.handleAdminImportResult())
}

type importJobStatusJSON struct {
	JobID            string              `json:"jobId"`
	Status           string              `json:"status"`
	MergeStrategy    string              `json:"mergeStrategy"`
	ImportProfile    string              `json:"importProfile"`
	DryRun           bool                `json:"dryRun"`
	TotalRows        *int                `json:"totalRows,omitempty"`
	ProcessedRows    int                 `json:"processedRows"`
	ErrorRows        int                 `json:"errorRows"`
	CreatedCount     int                 `json:"createdCount"`
	UpdatedCount     int                 `json:"updatedCount"`
	DeactivatedCount int                 `json:"deactivatedCount"`
	SkippedCount     int                 `json:"skippedCount"`
	Errors           []csvimport.RowError `json:"errors"`
	CreatedAt        string              `json:"createdAt"`
	UpdatedAt        string              `json:"updatedAt"`
	HasResult        bool                `json:"hasResult"`
}

type importJobSummaryJSON struct {
	JobID         string `json:"jobId"`
	Status        string `json:"status"`
	MergeStrategy string `json:"mergeStrategy"`
	ImportProfile string `json:"importProfile"`
	DryRun        bool   `json:"dryRun"`
	TotalRows     *int   `json:"totalRows,omitempty"`
	ProcessedRows int    `json:"processedRows"`
	ErrorRows     int    `json:"errorRows"`
	CreatedAt     string `json:"createdAt"`
}

func jobToStatusJSON(j *userimport.Job) importJobStatusJSON {
	var errs []csvimport.RowError
	if len(j.ErrorsJSON) > 0 {
		_ = json.Unmarshal(j.ErrorsJSON, &errs)
	}
	if errs == nil {
		errs = []csvimport.RowError{}
	}
	return importJobStatusJSON{
		JobID:            j.ID.String(),
		Status:           string(j.Status),
		MergeStrategy:    string(j.MergeStrategy),
		ImportProfile:    string(j.ImportProfile),
		DryRun:           j.DryRun,
		TotalRows:        j.TotalRows,
		ProcessedRows:    j.ProcessedRows,
		ErrorRows:        j.ErrorRows,
		CreatedCount:     j.CreatedCount,
		UpdatedCount:     j.UpdatedCount,
		DeactivatedCount: j.DeactivatedCount,
		SkippedCount:     j.SkippedCount,
		Errors:           errs,
		CreatedAt:        j.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        j.UpdatedAt.UTC().Format(time.RFC3339),
		HasResult:        j.ResultFilePath != nil && *j.ResultFilePath != "",
	}
}

func (d Deps) handleAdminImportUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.bulkCsvImportEnabled(w) {
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		ctx := r.Context()

		since := time.Now().UTC().Add(-time.Hour)
		count, err := userimport.CountRecentUploads(ctx, d.Pool, orgID, since)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check rate limit.")
			return
		}
		if count >= userImportRateLimit {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Import rate limit exceeded (5 per hour per org).")
			return
		}

		if err := r.ParseMultipartForm(maxUserImportFileBytes); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid multipart form or file too large (max 50 MB).")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing file field.")
			return
		}
		defer func() { _ = file.Close() }()

		data, err := io.ReadAll(io.LimitReader(file, maxUserImportFileBytes+1))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read uploaded file.")
			return
		}
		if len(data) > maxUserImportFileBytes {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File exceeds 50 MB limit.")
			return
		}

		cfg := d.effectiveConfig()
		if cfg.AvScanningEnabled {
			client := clamav.NewClient(d.Config.ClamAVAddr, cfg.ClamAVStub)
			scan, err := client.ScanStream(r.Context(), strings.NewReader(string(data)))
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Virus scan failed.")
				return
			}
			if !scan.Clean {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Uploaded file failed virus scan.")
				return
			}
		}

		mergeStr := strings.TrimSpace(r.FormValue("merge_strategy"))
		if mergeStr == "" {
			mergeStr = strings.TrimSpace(r.FormValue("mergeStrategy"))
		}
		merge, err := csvimport.ParseMergeStrategy(mergeStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		profileStr := strings.TrimSpace(r.FormValue("profile"))
		profile, err := csvimport.ParseProfile(profileStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		dryRun := strings.EqualFold(strings.TrimSpace(r.FormValue("dry_run")), "true") ||
			strings.EqualFold(strings.TrimSpace(r.FormValue("dryRun")), "true")

		var extraCols []string
		if cfg.CustomFieldsEnabled {
			defs, defErr := cfrepo.ListDefinitions(ctx, d.Pool, orgID, cfrepo.EntityUser, false)
			if defErr == nil {
				for _, def := range defs {
					extraCols = append(extraCols, def.Key)
				}
			}
		}

		parsed, err := csvimport.ParseCSVWithExtraColumns(strings.NewReader(string(data)), profile, extraCols)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		root := strings.TrimSpace(d.Config.CourseFilesRoot)
		if root == "" {
			root = "data/course-files"
		}
		jobDir := filepath.Join(root, "import-jobs", uuid.New().String())
		if err := os.MkdirAll(jobDir, 0o750); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store import file.")
			return
		}
		inputPath := filepath.Join(jobDir, "input.csv")
		if err := os.WriteFile(inputPath, data, 0o600); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store import file.")
			return
		}

		job, err := userimport.Insert(ctx, d.Pool, userimport.InsertParams{
			OrgID:         orgID,
			ActorID:       actor,
			MergeStrategy: merge,
			ImportProfile: profile,
			DryRun:        dryRun,
			TotalRows:     len(parsed.Rows),
			InputFilePath: inputPath,
		})
		if err != nil {
			_ = os.Remove(inputPath)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create import job.")
			return
		}

		if len(parsed.Errors) > 0 {
			_ = userimport.UpdateProgress(ctx, d.Pool, job.ID, userimport.ProgressUpdate{
				ErrorRows: len(parsed.Errors),
				Errors:    parsed.Errors,
			})
		}

		queueID, err := background.EnqueueUserImport(ctx, d.Pool, job.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue import job.")
			return
		}
		_ = userimport.SetQueueJobID(ctx, d.Pool, job.ID, queueID)
		if !d.effectiveConfig().BackgroundJobsEnabled {
			go d.runUserImportJob(job.ID)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jobId":      job.ID.String(),
			"parseErrors": parsed.Errors,
			"totalRows":  len(parsed.Rows),
			"filename":   header.Filename,
		})
	}
}

func (d Deps) handleAdminImportStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.bulkCsvImportEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid jobId.")
			return
		}
		job, err := userimport.Get(r.Context(), d.Pool, orgID, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load import job.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Import job not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(jobToStatusJSON(job))
	}
}

func (d Deps) handleAdminImportList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.bulkCsvImportEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		page := 1
		perPage := 25
		if s := strings.TrimSpace(r.URL.Query().Get("page")); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 {
				page = n
			}
		}
		if s := strings.TrimSpace(r.URL.Query().Get("perPage")); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
				perPage = n
			}
		}
		offset := (page - 1) * perPage
		jobs, total, err := userimport.ListRecent(r.Context(), d.Pool, orgID, perPage, offset)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list import jobs.")
			return
		}
		items := make([]importJobSummaryJSON, 0, len(jobs))
		for _, j := range jobs {
			items = append(items, importJobSummaryJSON{
				JobID:         j.ID.String(),
				Status:        string(j.Status),
				MergeStrategy: string(j.MergeStrategy),
				ImportProfile: string(j.ImportProfile),
				DryRun:        j.DryRun,
				TotalRows:     j.TotalRows,
				ProcessedRows: j.ProcessedRows,
				ErrorRows:     j.ErrorRows,
				CreatedAt:     j.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		totalPages := (total + perPage - 1) / perPage
		if totalPages < 1 {
			totalPages = 1
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":      items,
			"total":      total,
			"page":       page,
			"perPage":    perPage,
			"totalPages": totalPages,
		})
	}
}

func (d Deps) handleAdminImportResult() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.bulkCsvImportEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid jobId.")
			return
		}
		job, err := userimport.Get(r.Context(), d.Pool, orgID, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load import job.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Import job not found.")
			return
		}
		if job.ResultFilePath == nil || *job.ResultFilePath == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Result file not available.")
			return
		}
		if job.Status != userimport.StatusComplete {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Import job is not complete.")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="import-result-%s.csv"`, jobID))
		http.ServeFile(w, r, *job.ResultFilePath)
	}
}

func (d Deps) runUserImportJob(jobID uuid.UUID) {
	if d.Pool == nil {
		return
	}
	h := background.UserImportHandler(d.Pool, d.Config)
	_ = h.Execute(context.Background(), mustJSON(background.UserImportPayload{ImportJobID: jobID}))
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
