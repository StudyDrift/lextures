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
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/scormpackages"
	"github.com/lextures/lextures/server/internal/repos/scormregistrations"
	"github.com/lextures/lextures/server/internal/repos/scormscos"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/scorm"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	xapisvc "github.com/lextures/lextures/server/internal/service/xapi"
	"github.com/lextures/lextures/server/internal/workers/avscan"
	"github.com/lextures/lextures/server/internal/workers/scormextract"
)

const scormMaxUploadBytes = 100 << 20

func (d Deps) scormIngestionEnabled() bool {
	return d.effectiveConfig().ScormIngestionEnabled
}

func (d Deps) guardScormFeature(w http.ResponseWriter) bool {
	if !d.scormIngestionEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	return true
}

type scormPackageResponse struct {
	PackageID     string             `json:"packageId"`
	ItemID        string             `json:"itemId,omitempty"`
	Title         string             `json:"title"`
	PackageType   string             `json:"packageType"`
	ExtractStatus string             `json:"extractStatus"`
	AssetsBaseURL string             `json:"assetsBaseUrl"`
	DownloadURL   string             `json:"downloadUrl,omitempty"`
	Scos          []scormScoResponse `json:"scos,omitempty"`
}

type scormScoResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	LaunchHref string `json:"launchHref"`
}

type scormLaunchResponse struct {
	RegistrationID string            `json:"registrationId"`
	LaunchURL      string            `json:"launchUrl"`
	RenderURL      string            `json:"renderUrl"`
	InitialCMI     map[string]string `json:"initialCmi,omitempty"`
}

func (d Deps) scormAssetsBaseURL(courseCode, packageID string) string {
	return fmt.Sprintf("/api/v1/courses/%s/scorm/%s/assets/", courseCode, packageID)
}

func (d Deps) loadScormForCourse(w http.ResponseWriter, r *http.Request, courseCode string, packageID uuid.UUID) (*scormpackages.Package, uuid.UUID, bool) {
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return nil, uuid.UUID{}, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return nil, uuid.UUID{}, false
	}
	pkg, err := scormpackages.LoadByID(r.Context(), d.Pool, *cid, packageID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load SCORM package.")
		return nil, uuid.UUID{}, false
	}
	if pkg == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return nil, uuid.UUID{}, false
	}
	return pkg, *cid, true
}

func (d Deps) guardScormAccess(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID, pkg *scormpackages.Package, cid uuid.UUID) bool {
	perm := "course:" + courseCode + ":item:create"
	canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if canEdit {
		if pkg.ExtractStatus == "failed" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "SCORM package extraction failed.")
			return false
		}
		return true
	}
	if pkg.StructureItemID == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	visible, err := coursestructure.ScormVisibleToStudent(r.Context(), d.Pool, cid, *pkg.StructureItemID, viewer, time.Now().UTC())
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check access.")
		return false
	}
	if !visible {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	if pkg.ExtractStatus != "ready" {
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "SCORM package is not ready yet.")
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

// handleCreateModuleScorm is POST .../structure/modules/{module_id}/scorm (multipart .zip upload).
func (d Deps) handleCreateModuleScorm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if !d.guardScormFeature(w) {
			return
		}
		_, viewer, cid, moduleID, ok := d.beginCreateUnderModule(w, r)
		if !ok {
			return
		}
		if err := r.ParseMultipartForm(scormMaxUploadBytes + 1<<20); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid multipart form.")
			return
		}
		title := strings.TrimSpace(r.FormValue("title"))
		file, hdr, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing .zip file.")
			return
		}
		defer func() { _ = file.Close() }()
		if !strings.HasSuffix(strings.ToLower(hdr.Filename), ".zip") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File must have a .zip extension.")
			return
		}
		data, err := scorm.ReadZipBytes(file, scormMaxUploadBytes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		reader := scorm.ZipReaderAt(data)
		manifest, manifestRaw, err := scorm.ParseAndValidateZip(reader, int64(len(data)))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if title == "" {
			title = strings.TrimSpace(manifest.Title)
		}
		if title == "" {
			title = "SCORM Activity"
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
		zipKey := filestorage.ObjectKey(tenantID.String(), cid.String(), "scorm", packageID.String()+".zip")
		bucket := strings.TrimSpace(cfg.StorageBucket)
		if bucket == "" {
			bucket = "local"
		}
		if err := d.Storage.PutObject(r.Context(), zipKey, scorm.ZipReaderAt(data), int64(len(data)), "application/zip"); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store SCORM package.")
			return
		}
		courseID := &cid
		objID, err := avscan.RegisterAndEnqueue(r.Context(), d.Pool, tenantID, courseID, zipKey, bucket,
			"application/zip", int64(len(data)), &viewer, cfg.AvScanningEnabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to register storage object.")
			return
		}
		assetsPrefix := filestorage.ObjectKey(tenantID.String(), cid.String(), "scorm", packageID.String()+"/")
		pkgType := string(manifest.PackageType)
		if pkgType == "" {
			pkgType = "scorm12"
		}
		if err := scormpackages.Insert(r.Context(), d.Pool, packageID, objID, cid, nil, pkgType, title, manifestRaw, assetsPrefix); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save SCORM metadata.")
			return
		}
		if err := scormextract.InsertScosFromManifest(r.Context(), d.Pool, packageID, manifest); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save SCORM SCOs.")
			return
		}
		tmp, err := scorm.WriteTempZip(data)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process package.")
			return
		}
		defer func() { _ = os.Remove(tmp) }()
		if !cfg.AvScanningEnabled {
			if err := scormextract.ExtractSync(r.Context(), d.Pool, d.Storage, packageID, tmp); err != nil {
				slog.Warn("scorm sync extract failed", "package_id", packageID, "err", err)
			}
		}
		row, err := coursestructure.InsertScormUnderModule(r.Context(), d.Pool, cid, moduleID, packageID, title)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create module item.")
			return
		}
		slog.Info("scorm package uploaded", "package_id", packageID, "package_type", pkgType, "user_id", viewer)
		d.writeCreatedStructureItem(w, r, cid, row)
	}
}

// handleGetModuleScormByItem is GET /api/v1/courses/{course_code}/scorm-items/{item_id}.
func (d Deps) handleGetModuleScormByItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardScormFeature(w) {
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
		pkg, err := scormpackages.LoadByStructureItem(r.Context(), d.Pool, *cid, itemID)
		if err != nil || pkg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if !d.guardScormAccess(w, r, courseCode, viewer, pkg, *cid) {
			return
		}
		d.writeScormPackageJSON(w, r, courseCode, pkg)
	}
}

func (d Deps) writeScormPackageJSON(w http.ResponseWriter, r *http.Request, courseCode string, pkg *scormpackages.Package) {
	scos, _ := scormscos.ListForPackage(r.Context(), d.Pool, pkg.ID)
	out := scormPackageResponse{
		PackageID:     pkg.ID.String(),
		Title:         pkg.Title,
		PackageType:   pkg.PackageType,
		ExtractStatus: pkg.ExtractStatus,
		AssetsBaseURL: d.scormAssetsBaseURL(courseCode, pkg.ID.String()),
	}
	if pkg.StructureItemID != nil {
		out.ItemID = pkg.StructureItemID.String()
	}
	if pkg.ExtractStatus != "ready" {
		out.DownloadURL = fmt.Sprintf("/api/v1/courses/%s/scorm/%s/download", courseCode, pkg.ID)
	}
	for _, s := range scos {
		out.Scos = append(out.Scos, scormScoResponse{
			ID:         s.ID.String(),
			Identifier: s.Identifier,
			Title:      s.Title,
			LaunchHref: s.LaunchHref,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

// handlePostScormLaunch is POST /api/v1/courses/{course_code}/scorm/{sco_id}/launch.
func (d Deps) handlePostScormLaunch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardScormFeature(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		scoID, err := uuid.Parse(chi.URLParam(r, "sco_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sco id.")
			return
		}
		sco, err := scormscos.LoadByID(r.Context(), d.Pool, scoID)
		if err != nil || sco == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		pkg, cid, ok := d.loadScormForCourse(w, r, courseCode, sco.PackageID)
		if !ok {
			return
		}
		if !d.guardScormAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		enrollmentID, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, cid, viewer)
		if err != nil || enrollmentID == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Student enrollment required to launch SCORM content.")
			return
		}
		reg, err := scormregistrations.LoadOrCreate(r.Context(), d.Pool, scoID, *enrollmentID)
		if err != nil || reg == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create registration.")
			return
		}
		u, _ := user.FindByID(r.Context(), d.Pool, viewer)
		studentID := viewer.String()
		studentName := "Learner"
		if u != nil {
			if u.DisplayName != nil && strings.TrimSpace(*u.DisplayName) != "" {
				studentName = strings.TrimSpace(*u.DisplayName)
			} else {
				studentName = u.Email
			}
		}
		state := reg.ToState()
		hasResume := strings.TrimSpace(state.SuspendData) != "" || strings.TrimSpace(state.Location) != ""
		initial := scorm.InitialCMI(state, studentID, studentName, hasResume)
		renderURL := fmt.Sprintf("/api/v1/courses/%s/scorm/%s/render?registration=%s",
			courseCode, pkg.ID.String(), reg.ID.String())
		launchPath := strings.TrimPrefix(sco.LaunchHref, "/")
		launchURL := d.scormAssetsBaseURL(courseCode, pkg.ID.String()) + launchPath
		out := scormLaunchResponse{
			RegistrationID: reg.ID.String(),
			LaunchURL:      launchURL,
			RenderURL:      renderURL,
			InitialCMI:     initial,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

type scormCommitBody struct {
	CMI    scorm.CMIUpdate `json:"cmi"`
	Finish bool            `json:"finish"`
}

// handlePostScormRTECommit is POST /api/v1/scorm/rte/{registration_id}/commit.
func (d Deps) handlePostScormRTECommit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !d.guardScormFeature(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		regID, err := uuid.Parse(chi.URLParam(r, "registration_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid registration id.")
			return
		}
		reg, err := scormregistrations.LoadByID(r.Context(), d.Pool, regID)
		if err != nil || reg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		en, err := enrollment.GetByID(r.Context(), d.Pool, reg.EnrollmentID)
		if err != nil || en == nil || en.UserID != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Not allowed.")
			return
		}
		var body scormCommitBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		state := reg.ToState()
		scorm.ApplyCMIUpdate(&state, body.CMI)
		if err := scormregistrations.UpdateState(r.Context(), d.Pool, regID, state); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save SCORM state.")
			return
		}
		payload, _ := json.Marshal(body)
		_ = scormregistrations.LogEvent(r.Context(), d.Pool, regID, "commit", payload)

		sco, _ := scormscos.LoadByID(r.Context(), d.Pool, reg.ScoID)
		pkg, _ := scormpackages.LoadByIDGlobal(r.Context(), d.Pool, sco.PackageID)
		if pkg != nil && pkg.StructureItemID != nil && scorm.IsAttemptComplete(state.CompletionStatus) {
			pointsWorth := 100
			if pts, max, ok := scorm.GradePoints(state, pointsWorth); ok {
				_ = coursegrades.UpsertPointsFromLTI(r.Context(), d.Pool, pkg.CourseID, viewer, *pkg.StructureItemID, pts, max)
			}
			if d.effectiveConfig().XAPIEmissionEnabled {
				if u, _ := user.FindByID(r.Context(), d.Pool, viewer); u != nil {
					orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
					courseID := pkg.CourseID
					stmt := map[string]string{
						"verb": xapisvc.VerbCompleted,
						"completion": state.CompletionStatus,
					}
					raw, _ := json.Marshal(stmt)
					objID := fmt.Sprintf("scorm:%s", regID.String())
					_ = d.learningEmitter().StoreExternalStatement(r.Context(), orgID, &courseID, u.Email, "", xapisvc.VerbCompleted, objID, pkg.Title, raw)
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetScormRender serves sandboxed HTML with SCORM 1.2 API shim.
func (d Deps) handleGetScormRender() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !d.guardScormFeature(w) {
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
		pkg, cid, ok := d.loadScormForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardScormAccess(w, r, courseCode, viewer, pkg, cid) {
			return
		}
		if pkg.ExtractStatus != "ready" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><p>This activity could not be loaded.</p></body></html>`))
			return
		}
		regParam := strings.TrimSpace(r.URL.Query().Get("registration"))
		regID, err := uuid.Parse(regParam)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "registration query param required.")
			return
		}
		reg, err := scormregistrations.LoadByID(r.Context(), d.Pool, regID)
		if err != nil || reg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		en, err := enrollment.GetByID(r.Context(), d.Pool, reg.EnrollmentID)
		if err != nil || en == nil || en.UserID != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Not allowed.")
			return
		}
		sco, err := scormscos.LoadByID(r.Context(), d.Pool, reg.ScoID)
		if err != nil || sco == nil || sco.PackageID != pkg.ID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		u, _ := user.FindByID(r.Context(), d.Pool, viewer)
		studentID := viewer.String()
		studentName := "Learner"
		if u != nil {
			if u.DisplayName != nil && strings.TrimSpace(*u.DisplayName) != "" {
				studentName = strings.TrimSpace(*u.DisplayName)
			} else {
				studentName = u.Email
			}
		}
		state := reg.ToState()
		hasResume := strings.TrimSpace(state.SuspendData) != "" || strings.TrimSpace(state.Location) != ""
		initial := scorm.InitialCMI(state, studentID, studentName, hasResume)
		initialJSON, _ := json.Marshal(initial)
		launchPath := strings.TrimPrefix(sco.LaunchHref, "/")
		scoSrc := d.scormAssetsBaseURL(courseCode, pkg.ID.String()) + launchPath
		commitURL := fmt.Sprintf("/api/v1/scorm/rte/%s/commit", regID.String())
		title := strings.ReplaceAll(pkg.Title, `"`, "&quot;")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; connect-src 'self'; frame-ancestors *")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>SCORM: %s</title>
<style>html,body,#sco-frame{margin:0;height:100%%;width:100%%;border:0;}</style></head>
<body>
<iframe id="sco-frame" title="%s"></iframe>
<script>
var cmiData = %s;
var commitUrl = %q;
var initialized = false;
var finished = false;
function apiResponse(v){return String(v);}
function LMSInitialize(p){if(initialized)return "false";initialized=true;return "true";}
function LMSFinish(p){if(!initialized||finished)return "false";finished=true;commitCMI(true);return "true";}
function LMSGetValue(el){if(!initialized)return "";return cmiData[el]!==undefined?String(cmiData[el]):"";}
function LMSSetValue(el,val){if(!initialized)return "false";cmiData[el]=val;return "true";}
function LMSCommit(p){if(!initialized)return "false";commitCMI(false);return "true";}
function LMSGetLastError(){return "0";}
function LMSGetErrorString(c){return "";}
function LMSGetDiagnostic(c){return "";}
var API = {LMSInitialize,LMSFinish,LMSGetValue,LMSSetValue,LMSCommit,LMSGetLastError,LMSGetErrorString,LMSGetDiagnostic};
window.API = API;
function commitCMI(finish){
  fetch(commitUrl,{method:"POST",credentials:"include",headers:{"Content-Type":"application/json"},body:JSON.stringify({cmi:cmiData,finish:finish})}).catch(function(){});
}
document.getElementById("sco-frame").src = %q;
</script></body></html>`, title, title, string(initialJSON), commitURL, scoSrc)
		_, _ = w.Write([]byte(html))
	}
}

// handleGetScormAsset serves extracted package files.
func (d Deps) handleGetScormAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardScormFeature(w) {
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
		pkg, cid, ok := d.loadScormForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardScormAccess(w, r, courseCode, viewer, pkg, cid) {
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
		w.Header().Set("Content-Type", scormMimeForAsset(assetPath))
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = io.Copy(w, rc)
	}
}

func scormMimeForAsset(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".html", ".htm":
		return "text/html"
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

// handleGetScormDownload streams the original zip.
func (d Deps) handleGetScormDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.guardScormFeature(w) {
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
		pkg, cid, ok := d.loadScormForCourse(w, r, courseCode, packageID)
		if !ok {
			return
		}
		if !d.guardScormAccess(w, r, courseCode, viewer, pkg, cid) {
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
		w.Header().Set("Content-Disposition", `attachment; filename="`+pkg.ID.String()+`.zip"`)
		_, _ = io.Copy(w, rc)
	}
}
