package httpserver

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/filemanager"
	"github.com/lextures/lextures/server/internal/service/officepreview"
)

// handleGetCourseFileItemPreview is GET /api/v1/courses/{course_code}/files/items/{item_id}/preview
// Converts DOCX, XLSX, or PPTX to HTML for in-browser display (no client-side Office libraries).
func (d Deps) handleGetCourseFileItemPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseFilesManage(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		courseID, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil || !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		item, err := filemanager.GetFileItem(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load file.")
			return
		}
		if item == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		filename := strings.TrimSpace(item.DisplayName)
		if filename == "" {
			filename = item.OriginalFilename
		}
		if _, supported := officepreview.DetectFormat(filename, item.MimeType); !supported {
			apierr.WriteJSON(w, http.StatusUnsupportedMediaType, apierr.CodeInvalidInput,
				"Preview is only available for .docx, .xlsx, and .pptx files.")
			return
		}
		data, err := d.readCourseFileItemBytes(r.Context(), courseCode, item)
		if err != nil {
			log.Printf("course-file-item-preview: read course=%q item=%s err=%v", courseCode, itemID, err)
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "File not found.")
			return
		}
		html, err := officepreview.ConvertToHTML(data, filename, item.MimeType)
		if err != nil {
			log.Printf("course-file-item-preview: convert course=%q item=%s err=%v", courseCode, itemID, err)
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInternal, "Could not generate preview for this file.")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}
}
