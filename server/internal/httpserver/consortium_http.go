package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/consortium"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	svcConsortium "github.com/lextures/lextures/server/internal/service/consortium"
)

func (d Deps) consortiumFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFConsortiumSharing {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Consortium sharing is not enabled.")
		return true
	}
	return false
}

type consortiumAgreementJSON struct {
	ID           string  `json:"id"`
	HostOrgID    string  `json:"hostOrgId"`
	GuestOrgID   string  `json:"guestOrgId"`
	HostOrgName  string  `json:"hostOrgName,omitempty"`
	GuestOrgName string  `json:"guestOrgName,omitempty"`
	Status       string  `json:"status"`
	SignedAt     *string `json:"signedAt,omitempty"`
	ExpiresAt    *string `json:"expiresAt,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

func agreementToJSON(a consortium.Agreement) consortiumAgreementJSON {
	out := consortiumAgreementJSON{
		ID:           a.ID.String(),
		HostOrgID:    a.HostOrgID.String(),
		GuestOrgID:   a.GuestOrgID.String(),
		HostOrgName:  a.HostOrgName,
		GuestOrgName: a.GuestOrgName,
		Status:       a.Status,
		CreatedAt:    a.CreatedAt.UTC().Format(time.RFC3339),
	}
	if a.SignedAt != nil {
		s := a.SignedAt.UTC().Format(time.RFC3339)
		out.SignedAt = &s
	}
	if a.ExpiresAt != nil {
		s := a.ExpiresAt.UTC().Format(time.RFC3339)
		out.ExpiresAt = &s
	}
	return out
}

func (d Deps) registerConsortiumRoutes(r chi.Router) {
	r.Post("/api/v1/admin/consortium/agreements", d.handleCreateConsortiumAgreement())
	r.Get("/api/v1/admin/consortium/agreements", d.handleListConsortiumAgreements())
	r.Patch("/api/v1/admin/consortium/agreements/{id}", d.handlePatchConsortiumAgreement())
	r.Get("/api/v1/admin/consortium/enrollment-report", d.handleConsortiumEnrollmentReport())
	r.Get("/api/v1/consortium/courses", d.handleListConsortiumCourses())
	r.Post("/api/v1/consortium/courses/{id}/enroll", d.handleConsortiumEnroll())
	r.Get("/api/v1/me/consortium-branding", d.handleMeConsortiumBranding())
	r.Get("/api/v1/courses/{course_code}/consortium-settings", d.handleGetCourseConsortiumSettings())
	r.Patch("/api/v1/courses/{course_code}/consortium-settings", d.handlePatchCourseConsortiumSettings())
}

func (d Deps) handleCreateConsortiumAgreement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			HostOrgID  string `json:"hostOrgId"`
			GuestOrgID string `json:"guestOrgId"`
			Status     string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		hostID, err := uuid.Parse(strings.TrimSpace(body.HostOrgID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "hostOrgId is required.")
			return
		}
		guestID, err := uuid.Parse(strings.TrimSpace(body.GuestOrgID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "guestOrgId is required.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, hostID, true); !ok {
			return
		}
		status := strings.TrimSpace(body.Status)
		if status == "" {
			status = consortium.StatusPending
		}
		switch status {
		case consortium.StatusPending, consortium.StatusActive:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be pending or active.")
			return
		}
		agreement, err := consortium.CreateAgreement(r.Context(), d.Pool, hostID, guestID, status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not create agreement.")
			return
		}
		loaded, err := consortium.GetAgreement(r.Context(), d.Pool, agreement.ID)
		if err != nil || loaded == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load agreement.")
			return
		}
		_ = userID
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"agreement": agreementToJSON(*loaded)})
	}
}

func (d Deps) handleListConsortiumAgreements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		orgIDStr := strings.TrimSpace(r.URL.Query().Get("orgId"))
		if orgIDStr == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "orgId query parameter is required.")
			return
		}
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, false); !ok {
			return
		}
		list, err := consortium.ListAgreementsForOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list agreements.")
			return
		}
		out := make([]consortiumAgreementJSON, 0, len(list))
		for _, a := range list {
			out = append(out, agreementToJSON(a))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"agreements": out})
	}
}

func (d Deps) handlePatchConsortiumAgreement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid agreement id.")
			return
		}
		existing, err := consortium.GetAgreement(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load agreement.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Agreement not found.")
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		status := strings.TrimSpace(body.Status)
		switch status {
		case consortium.StatusActive:
			if _, ok := d.orgRoleAccess(w, r, existing.GuestOrgID, true); !ok {
				return
			}
		case consortium.StatusTerminated:
			if _, ok := d.orgRoleAccess(w, r, existing.HostOrgID, true); !ok {
				if _, ok2 := d.orgRoleAccess(w, r, existing.GuestOrgID, true); !ok2 {
					return
				}
			}
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be active or terminated.")
			return
		}
		updated, err := consortium.UpdateAgreementStatus(r.Context(), d.Pool, id, status)
		if err != nil || updated == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update agreement.")
			return
		}
		loaded, err := consortium.GetAgreement(r.Context(), d.Pool, id)
		if err != nil || loaded == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load agreement.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"agreement": agreementToJSON(*loaded)})
	}
}

func (d Deps) handleConsortiumEnrollmentReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		agreementIDStr := strings.TrimSpace(r.URL.Query().Get("agreementId"))
		if agreementIDStr == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "agreementId is required.")
			return
		}
		agreementID, err := uuid.Parse(agreementIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid agreementId.")
			return
		}
		agreement, err := consortium.GetAgreement(r.Context(), d.Pool, agreementID)
		if err != nil || agreement == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Agreement not found.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, agreement.HostOrgID, false); !ok {
			if _, ok2 := d.orgRoleAccess(w, r, agreement.GuestOrgID, false); !ok2 {
				return
			}
		}
		rows, err := consortium.EnrollmentReport(r.Context(), d.Pool, agreementID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"rows": rows})
	}
}

func (d Deps) handleListConsortiumCourses() http.HandlerFunc {
	type courseOut struct {
		ID          string `json:"id"`
		CourseCode  string `json:"courseCode"`
		Title       string `json:"title"`
		Description string `json:"description"`
		HostOrgID   string `json:"hostOrgId"`
		HostOrgName string `json:"hostOrgName"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		guestOrgIDStr := strings.TrimSpace(r.URL.Query().Get("guestOrgId"))
		var guestOrgID uuid.UUID
		if guestOrgIDStr != "" {
			var err error
			guestOrgID, err = uuid.Parse(guestOrgIDStr)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid guestOrgId.")
				return
			}
		} else {
			var ok2 bool
			guestOrgID, ok2 = d.meOrgID(w, r)
			if !ok2 {
				return
			}
			_ = userID
		}
		courses, err := consortium.ListShareableCoursesForGuest(r.Context(), d.Pool, guestOrgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
			return
		}
		out := make([]courseOut, 0, len(courses))
		for _, c := range courses {
			out = append(out, courseOut{
				ID:          c.ID.String(),
				CourseCode:  c.CourseCode,
				Title:       c.Title,
				Description: c.Description,
				HostOrgID:   c.HostOrgID.String(),
				HostOrgName: c.HostOrgName,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"courses": out})
	}
}

func (d Deps) handleConsortiumEnroll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		homeOrgID, ok2 := d.meOrgID(w, r)
		if !ok2 {
			return
		}
		err = svcConsortium.EnrollGuestStudent(r.Context(), d.Pool, courseID, userID, homeOrgID)
		if err != nil {
			switch err {
			case svcConsortium.ErrAgreementNotActive:
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "No active consortium agreement.")
			case svcConsortium.ErrCourseNotShareable:
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Course is not available for consortium enrollment.")
			case svcConsortium.ErrAlreadyEnrolled:
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Already enrolled.")
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"enrolled": true})
	}
}

func (d Deps) handleGetCourseConsortiumSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		settings, err := course.GetConsortiumSettings(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		if settings == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

func (d Deps) handlePatchCourseConsortiumSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		var body struct {
			ConsortiumShareable bool `json:"consortiumShareable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		settings, err := course.SetConsortiumShareable(r.Context(), d.Pool, courseCode, body.ConsortiumShareable)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update settings.")
			return
		}
		if settings == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

func (d Deps) handleMeConsortiumBranding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.consortiumFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := strings.TrimSpace(r.URL.Query().Get("courseCode"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseCode is required.")
			return
		}
		var homeOrgID uuid.UUID
		err := d.Pool.QueryRow(r.Context(), `
SELECT ce.home_org_id
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
WHERE c.course_code = $1 AND ce.user_id = $2 AND ce.active AND ce.home_org_id IS NOT NULL
LIMIT 1
`, courseCode, userID).Scan(&homeOrgID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"active": false})
			return
		}
		var orgName string
		_ = d.Pool.QueryRow(r.Context(), `SELECT name FROM tenant.organizations WHERE id = $1`, homeOrgID).Scan(&orgName)
		row, err := orgbranding.Get(r.Context(), d.Pool, homeOrgID)
		primary := orgbranding.DefaultPrimaryHex
		secondary := orgbranding.DefaultSecondaryHex
		var logoURL *string
		if err == nil && row != nil {
			if row.PrimaryColor != "" {
				primary = row.PrimaryColor
			}
			if row.SecondaryColor != "" {
				secondary = row.SecondaryColor
			}
			logoURL = row.LogoURL
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active":         true,
			"orgName":        orgName,
			"primaryColor":   primary,
			"secondaryColor": secondary,
			"logoUrl":        logoURL,
		})
	}
}
