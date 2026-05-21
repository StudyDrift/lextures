package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/h5p"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/h5pcompletions"
	"github.com/lextures/lextures/server/internal/repos/h5ppackages"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/workers/avscan"
	"github.com/lextures/lextures/server/internal/workers/h5pextract"
)

const h5pMaxUploadBytes = 50 << 20

func (d Deps) h5pEnabled() bool {
	return d.effectiveConfig().H5PEnabled
}

func (d Deps) guardH5PFeature(w http.ResponseWriter) bool {
	if !d.h5pEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	return true
}

type h5pPackageResponse struct {
	PackageID     string `json:"packageId"`
	ItemID        string `json:"itemId,omitempty"`
	Title         string `json:"title"`
	ContentType   string `json:"contentType"`
	ExtractStatus string `json:"extractStatus"`
	AssetsBaseURL string `json:"assetsBaseUrl"`
	DownloadURL   string `json:"downloadUrl,omitempty"`
}

func (d Deps) h5pAssetsBaseURL(courseCode, packageID string) string {
	return fmt.Sprintf("/api/v1/courses/%s/h5p/%s/assets/", courseCode, packageID)
}

func (d Deps) loadH5PForCourse(w http.ResponseWriter, r *http.Request, courseCode string, packageID uuid.UUID) (*h5ppackages.Package, uuid.UUID, bool) {
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return nil, uuid.UUID{}, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return nil, uuid.UUID{}, false
	}
	pkg, err := h5ppackages.LoadByID(r.Context(), d.Pool, *cid, packageID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load H5P package.")
		return nil, uuid.UUID{}, false
	}
	if pkg == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return nil, uuid.UUID{}, false
	}
	return pkg, *cid, true
}

func (d Deps) guardH5PAccess(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID, pkg *h5ppackages.Package, cid uuid.UUID) bool {
	perm := "course:" + courseCode + ":item:create"
	canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if canEdit {
		if pkg.ExtractStatus == "failed" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "H5P package extraction failed.")
			return false
		}
		return true
	}
	if pkg.StructureItemID == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	visible, err := coursestructure.H5PVisibleToStudent(r.Context(), d.Pool, cid, *pkg.StructureItemID, viewer, time.Now().UTC())
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check access.")
		return false
	}
	if !visible {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	if pkg.ExtractStatus != "ready" {
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "H5P package is not ready yet.")
		return false
	}
	cfg := d.effectiveConfig()
	obj, err := storageobjects.LoadByID(r.Context(), d.Pool, pkg.StorageObjectID)
	if err != nil || obj == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify package.")
		return false
	}
	if !obj.IsAccessible(cfg.AvScanningEnabled) {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Package is not available (scan pending or quarantined).")
		return false
	}
	return true
}

// handleCreateModuleH5P is POST .../structure/modules/{module_id}/h5p (multipart .h5p upload).
func (d Deps) handleCreateModuleH5P() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardH5PFeature(w) {
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, viewer, cid, moduleID, ok := d.beginCreateUnderModule(w, r)
		if !ok {
			return
		}
		if err := r.ParseMultipartForm(h5pMaxUploadBytes + 1<<20); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid multipart form.")
			return
		}
		title := strings.TrimSpace(r.FormValue("title"))
		file, hdr, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing .h5p file.")
			return
		}
		defer func() { _ = file.Close() }()
		if !strings.HasSuffix(strings.ToLower(hdr.Filename), ".h5p") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File must have a .h5p extension.")
			return
		}
		data, err := h5p.ReadZipBytes(file, h5pMaxUploadBytes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		reader := h5p.ZipReaderAt(data)
		manifest, manifestRaw, err := h5p.ParseAndValidateZip(reader, int64(len(data)))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if title == "" {
			title = strings.TrimSpace(manifest.Title)
		}
		if title == "" {
			title = "Interactive H5P Activity"
		}
		cfg := d.effectiveConfig()
		if d.Storage == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Object storage is not configured.")
			return
		}
		tenantID, err := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
			return
		}
		packageID := uuid.New()
		zipKey := filestorage.ObjectKey(tenantID.String(), cid.String(), "h5p", packageID.String()+".h5p")
		bucket := strings.TrimSpace(cfg.StorageBucket)
		if bucket == "" {
			bucket = "local"
		}
		if err := d.Storage.PutObject(r.Context(), zipKey, h5p.ZipReaderAt(data), int64(len(data)), "application/zip"); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store H5P package.")
			return
		}
		courseID := &cid
		objID, err := avscan.RegisterAndEnqueue(r.Context(), d.Pool, tenantID, courseID, zipKey, bucket,
			"application/zip", int64(len(data)), &viewer, cfg.AvScanningEnabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to register storage object.")
			return
		}
		assetsPrefix := filestorage.ObjectKey(tenantID.String(), cid.String(), "h5p", packageID.String()+"/")
		if err := h5ppackages.Insert(r.Context(), d.Pool, packageID, objID, cid, nil, title, manifest.MainLibrary, nil, manifestRaw, assetsPrefix); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save H5P metadata.")
			return
		}
		tmp, err := h5p.WriteTempZip(data)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process package.")
			return
		}
		defer func() { _ = os.Remove(tmp) }()
		if !cfg.AvScanningEnabled {
			if err := h5pextract.ExtractSync(r.Context(), d.Pool, d.Storage, packageID, tmp); err != nil {
				slog.Warn("h5p sync extract failed", "package_id", packageID, "err", err)
			}
		}
		row, err := coursestructure.InsertH5PUnderModule(r.Context(), d.Pool, cid, moduleID, packageID, title)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create module item.")
			return
		}
		slog.Info("h5p package uploaded", "package_id", packageID, "content_type", manifest.MainLibrary, "user_id", viewer)
		d.writeCreatedStructureItem(w, r, cid, row)
	}
}

// handleGetModuleH5PByItem is GET /api/v1/courses/{course_code}/h5p-items/{item_id}.
func (d Deps) handleGetModuleH5PByItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardH5PFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		pkg, err := h5ppackages.LoadByStructureItem(r.Context(), d.Pool, *cid, itemID)
		if err != nil || pkg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if !d.guardH5PAccess(w, r, courseCode, viewer, pkg, *cid) {
			return
		}
		d.writeH5PPackageJSON(w, r, courseCode, pkg)
	}
}

// handleGetH5PPackage is GET /api/v1/courses/{course_code}/h5p/{package_id}.
func (d Deps) handleGetH5PPackage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardH5PFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		packageID, err := uuid.Parse(chi.URLParam(r, "package_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid package id.")
			return
		}
		pkg, cid, ok := d.loadH5PForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardH5PAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		d.writeH5PPackageJSON(w, r, courseCode, pkg)
	}
}

func (d Deps) writeH5PPackageJSON(w http.ResponseWriter, r *http.Request, courseCode string, pkg *h5ppackages.Package) {
	out := h5pPackageResponse{
		PackageID:     pkg.ID.String(),
		Title:         pkg.Title,
		ContentType:   pkg.ContentType,
		ExtractStatus: pkg.ExtractStatus,
		AssetsBaseURL: d.h5pAssetsBaseURL(courseCode, pkg.ID.String()),
	}
	if pkg.StructureItemID != nil {
		out.ItemID = pkg.StructureItemID.String()
	}
	if pkg.ExtractStatus != "ready" {
		out.DownloadURL = fmt.Sprintf("/api/v1/courses/%s/h5p/%s/download", courseCode, pkg.ID)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

// handleGetH5PRender is GET /api/v1/courses/{course_code}/h5p/{package_id}/render — sandboxed HTML bootstrap.
func (d Deps) handleGetH5PRender() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !d.guardH5PFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		packageID, err := uuid.Parse(chi.URLParam(r, "package_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid package id.")
			return
		}
		pkg, cid, ok := d.loadH5PForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardH5PAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		if pkg.ExtractStatus != "ready" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><p>This activity could not be loaded.</p></body></html>`))
			return
		}
		assetsBase := d.h5pAssetsBaseURL(courseCode, pkg.ID.String())
		title := strings.ReplaceAll(pkg.Title, `"`, "&quot;")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; connect-src 'self'; frame-ancestors *")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Interactive activity: %s</title>
<style>html,body,#h5p-container{margin:0;height:100%%;}</style></head>
<body>
<div id="h5p-container" data-h5p-json="%sh5p.json" data-assets-base="%s"></div>
<script>
window.addEventListener('message',function(ev){
  if(ev.data&&ev.data.context==='h5p'&&ev.data.statement){
    window.parent.postMessage({type:'h5p-xapi',statement:ev.data.statement,packageId:'%s'},'*');
  }
});
</script>
</body></html>`, title, assetsBase, assetsBase, pkg.ID)
		_, _ = w.Write([]byte(html))
	}
}

// handleGetH5PAsset serves extracted package files.
func (d Deps) handleGetH5PAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardH5PFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		packageID, err := uuid.Parse(chi.URLParam(r, "package_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid package id.")
			return
		}
		assetPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
		if assetPath == "" || strings.Contains(assetPath, "..") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid asset path.")
			return
		}
		pkg, cid, ok := d.loadH5PForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardH5PAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		if d.Storage == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Storage unavailable.")
			return
		}
		key := strings.TrimSuffix(pkg.AssetsPrefix, "/") + "/" + path.Clean(assetPath)
		rc, err := d.Storage.GetObject(r.Context(), key)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Asset not found.")
			return
		}
		defer func() { _ = rc.Close() }()
		w.Header().Set("Content-Type", h5pMimeForAsset(assetPath))
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = io.Copy(w, rc)
	}
}

func h5pMimeForAsset(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".json":
		return "application/json"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// handleGetH5PDownload streams the original .h5p zip (fallback when render fails).
func (d Deps) handleGetH5PDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardH5PFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		packageID, err := uuid.Parse(chi.URLParam(r, "package_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid package id.")
			return
		}
		pkg, cid, ok := d.loadH5PForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardH5PAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		obj, err := storageobjects.LoadByID(r.Context(), d.Pool, pkg.StorageObjectID)
		if err != nil || obj == nil || d.Storage == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		rc, err := d.Storage.GetObject(r.Context(), obj.ObjectKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		defer func() { _ = rc.Close() }()
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+pkg.ID.String()+`.h5p"`)
		_, _ = io.Copy(w, rc)
	}
}

// handleGetH5PCompletions is GET .../h5p/{package_id}/completions (instructor).
func (d Deps) handleGetH5PCompletions() http.HandlerFunc {
	type row struct {
		UserID    string  `json:"userId"`
		Status    string  `json:"status"`
		Label     string  `json:"label"`
		ScoreRaw  *float64 `json:"scoreRaw,omitempty"`
		ScoreMax  *float64 `json:"scoreMax,omitempty"`
		UpdatedAt string  `json:"updatedAt"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardH5PFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		okPerm, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil || !okPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view completions.")
			return
		}
		packageID, err := uuid.Parse(chi.URLParam(r, "package_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid package id.")
			return
		}
		pkg, _, ok := d.loadH5PForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		rows, err := h5pcompletions.ListForPackage(r.Context(), d.Pool, pkg.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load completions.")
			return
		}
		out := make([]row, 0, len(rows))
		for _, c := range rows {
			out = append(out, row{
				UserID:    c.UserID.String(),
				Status:    c.Status,
				Label:     h5p.DisplayLabel(c.Status),
				ScoreRaw:  c.ScoreRaw,
				ScoreMax:  c.ScoreMax,
				UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"completions": out})
	}
}
