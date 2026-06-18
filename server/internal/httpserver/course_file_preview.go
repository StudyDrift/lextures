package httpserver

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/service/officepreview"
)

// handleGetCourseFilePreview is GET /api/v1/courses/{course_code}/course-files/{file_id}/preview
// Converts DOCX, XLSX, or PPTX to HTML for in-browser display (submission attachments, uploads).
func (d Deps) handleGetCourseFilePreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		fileID, err := uuid.Parse(chi.URLParam(r, "file_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid file id.")
			return
		}
		row, err := coursefiles.GetForCourse(r.Context(), d.Pool, courseCode, fileID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load file.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if !d.gateObjectDownload(w, r, row.StorageKey) {
			return
		}
		filename := strings.TrimSpace(row.OriginalFilename)
		if _, supported := officepreview.DetectFormat(filename, row.MimeType); !supported {
			apierr.WriteJSON(w, http.StatusUnsupportedMediaType, apierr.CodeInvalidInput,
				"Preview is only available for .docx, .xlsx, and .pptx files.")
			return
		}
		data, err := d.readCourseFileRowBytes(r.Context(), courseCode, row)
		if err != nil {
			log.Printf("course-file-preview: read course=%q file=%s err=%v", courseCode, fileID, err)
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		html, err := officepreview.ConvertToHTML(data, filename, row.MimeType)
		if err != nil {
			log.Printf("course-file-preview: convert course=%q file=%s err=%v", courseCode, fileID, err)
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInternal, "Could not generate preview for this file.")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}
}