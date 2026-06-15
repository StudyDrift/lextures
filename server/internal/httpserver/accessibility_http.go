package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/accommodations"
	repo "github.com/lextures/lextures/server/internal/repos/accessibilityprofiles"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	svc "github.com/lextures/lextures/server/internal/service/accessibility"
)

const eventAccommodationActive = "accommodation_active"

func (d Deps) accessibilityFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFAccessibilityIntake {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Accessibility services intake is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerAccessibilityRoutes(r chi.Router) {
	r.Post("/api/v1/accessibility/profiles", d.handleCreateAccommodationProfile())
	r.Get("/api/v1/accessibility/profiles", d.handleListAccommodationProfiles())
	r.Get("/api/v1/accessibility/profiles/{id}", d.handleGetAccommodationProfile())
	r.Patch("/api/v1/accessibility/profiles/{id}", d.handleUpdateAccommodationProfile())
	r.Post("/api/v1/accessibility/profiles/{id}/notify-instructors", d.handleNotifyInstructors())
	r.Get("/api/v1/me/accommodation-profiles", d.handleMyAccommodationProfiles())
}

// accessibilityCoordinator authenticates and authorizes a coordinator/admin. The 2.11
// permission (granted to the "Accessibility Coordinator" and "Global Admin" roles) governs
// access — instructors and students fail with 403 (AC-4).
func (d Deps) accessibilityCoordinator(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	has, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, acmodel.PermManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return userID, true
}

type profileJSON struct {
	ID             string          `json:"id"`
	StudentID      string          `json:"studentId"`
	Accommodations []string        `json:"accommodations"`
	CustomParams   json.RawMessage `json:"customParams"`
	EffectiveFrom  string          `json:"effectiveFrom"`
	EffectiveUntil *string         `json:"effectiveUntil,omitempty"`
	Labels         []string        `json:"labels"`
	IsActive       bool            `json:"isActive"`
	NotifiedAt     *string         `json:"notifiedAt,omitempty"`
	CreatedAt      string          `json:"createdAt"`
}

func profileToJSON(p repo.Profile) profileJSON {
	out := profileJSON{
		ID:             p.ID.String(),
		StudentID:      p.StudentID.String(),
		Accommodations: p.Accommodations,
		CustomParams:   p.CustomParams,
		EffectiveFrom:  p.EffectiveFrom.UTC().Format("2006-01-02"),
		Labels:         svc.Labels(p.Accommodations),
		IsActive:       p.IsActive,
		CreatedAt:      p.CreatedAt.UTC().Format(time.RFC3339),
	}
	if len(out.CustomParams) == 0 {
		out.CustomParams = json.RawMessage(`{}`)
	}
	if p.EffectiveUntil != nil {
		s := p.EffectiveUntil.UTC().Format("2006-01-02")
		out.EffectiveUntil = &s
	}
	if p.NotifiedAt != nil {
		s := p.NotifiedAt.UTC().Format(time.RFC3339)
		out.NotifiedAt = &s
	}
	return out
}

func parseAccDate(s string) (*time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, true
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, false
	}
	return &t, true
}

func (d Deps) handleCreateAccommodationProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.accessibilityFeatureOff(w) {
			return
		}
		coordinatorID, ok := d.accessibilityCoordinator(w, r)
		if !ok {
			return
		}
		var body struct {
			StudentID      string          `json:"studentId"`
			Accommodations []string        `json:"accommodations"`
			CustomParams   json.RawMessage `json:"customParams"`
			EffectiveFrom  string          `json:"effectiveFrom"`
			EffectiveUntil string          `json:"effectiveUntil"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "A valid studentId is required.")
			return
		}
		if !svc.ValidTypes(body.Accommodations) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "accommodations must be a non-empty list of valid accommodation types.")
			return
		}
		from, okFrom := parseAccDate(body.EffectiveFrom)
		until, okUntil := parseAccDate(body.EffectiveUntil)
		if !okFrom || !okUntil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Dates must be in YYYY-MM-DD format.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, studentID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Student organization not found.")
			return
		}
		prof, err := repo.Create(r.Context(), d.Pool, repo.CreateInput{
			StudentID:      studentID,
			OrgID:          orgID,
			Accommodations: body.Accommodations,
			CustomParams:   body.CustomParams,
			EffectiveFrom:  from,
			EffectiveUntil: until,
			CreatedBy:      coordinatorID,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create accommodation profile.")
			return
		}
		// FR-3 / AC-1: propagate to the 2.11 override engine immediately.
		if _, err := svc.Apply(r.Context(), d.Pool, prof.ID, studentID, prof.Accommodations, prof.CustomParams, from, until, coordinatorID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Profile created but accommodation propagation failed.")
			return
		}
		if reloaded, err := repo.Get(r.Context(), d.Pool, prof.ID); err == nil && reloaded != nil {
			prof = reloaded
		}
		writeJSON(w, http.StatusCreated, map[string]any{"profile": profileToJSON(*prof)})
	}
}

func (d Deps) handleListAccommodationProfiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.accessibilityFeatureOff(w) {
			return
		}
		coordinatorID, ok := d.accessibilityCoordinator(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, coordinatorID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Organization not found.")
			return
		}
		profiles, err := repo.ListForOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load accommodation profiles.")
			return
		}
		out := make([]profileJSON, 0, len(profiles))
		for i := range profiles {
			out = append(out, profileToJSON(profiles[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"profiles": out})
	}
}

func (d Deps) handleGetAccommodationProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.accessibilityFeatureOff(w) {
			return
		}
		if _, ok := d.accessibilityCoordinator(w, r); !ok {
			return
		}
		prof, ok := d.loadProfile(w, r)
		if !ok {
			return
		}
		courses, _ := repo.CoursesForStudent(r.Context(), d.Pool, prof.StudentID)
		writeJSON(w, http.StatusOK, map[string]any{
			"profile":         profileToJSON(*prof),
			"affectedCourses": affectedCoursesJSON(courses),
		})
	}
}

func (d Deps) handleUpdateAccommodationProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.accessibilityFeatureOff(w) {
			return
		}
		coordinatorID, ok := d.accessibilityCoordinator(w, r)
		if !ok {
			return
		}
		prof, ok := d.loadProfile(w, r)
		if !ok {
			return
		}
		var body struct {
			Accommodations *[]string       `json:"accommodations"`
			CustomParams   json.RawMessage `json:"customParams"`
			EffectiveFrom  *string         `json:"effectiveFrom"`
			EffectiveUntil *string         `json:"effectiveUntil"`
			IsActive       *bool           `json:"isActive"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		patch := repo.UpdatePatch{CustomParams: body.CustomParams, IsActive: body.IsActive}
		if body.Accommodations != nil {
			if !svc.ValidTypes(*body.Accommodations) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "accommodations must be a non-empty list of valid types.")
				return
			}
			patch.Accommodations = body.Accommodations
		}
		if body.EffectiveFrom != nil {
			from, okFrom := parseAccDate(*body.EffectiveFrom)
			if !okFrom {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "effectiveFrom must be YYYY-MM-DD.")
				return
			}
			patch.EffectiveFrom = from
		}
		if body.EffectiveUntil != nil {
			until, okUntil := parseAccDate(*body.EffectiveUntil)
			if !okUntil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "effectiveUntil must be YYYY-MM-DD.")
				return
			}
			patch.EffectiveUntil = until
		}

		deactivating := body.IsActive != nil && !*body.IsActive && prof.IsActive
		updated, err := repo.Update(r.Context(), d.Pool, prof.ID, patch)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update accommodation profile.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Accommodation profile not found.")
			return
		}

		switch {
		case deactivating:
			// AC-5: removing the profile removes the propagated override.
			if err := svc.Deactivate(r.Context(), d.Pool, prof.StudentID, prof.AppliedID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to remove accommodation override.")
				return
			}
			_ = repo.SetApplied(r.Context(), d.Pool, prof.ID, nil)
		case updated.IsActive:
			// Re-propagate: drop any prior override, then apply the updated settings.
			_ = svc.Deactivate(r.Context(), d.Pool, prof.StudentID, prof.AppliedID)
			if _, err := svc.Apply(r.Context(), d.Pool, updated.ID, updated.StudentID, updated.Accommodations, updated.CustomParams, ptrTime(updated.EffectiveFrom), updated.EffectiveUntil, coordinatorID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to re-apply accommodation override.")
				return
			}
		}
		if reloaded, err := repo.Get(r.Context(), d.Pool, updated.ID); err == nil && reloaded != nil {
			updated = reloaded
		}
		writeJSON(w, http.StatusOK, map[string]any{"profile": profileToJSON(*updated)})
	}
}

func (d Deps) handleNotifyInstructors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.accessibilityFeatureOff(w) {
			return
		}
		if _, ok := d.accessibilityCoordinator(w, r); !ok {
			return
		}
		prof, ok := d.loadProfile(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		studentName := "the student"
		if u, err := user.FindByID(ctx, d.Pool, prof.StudentID); err == nil && u != nil {
			studentName = user.DisplayLabel(u.DisplayName, u.Email)
		}
		effDate := prof.EffectiveFrom.UTC().Format("2006-01-02")
		letter := svc.RenderLetter(studentName, effDate, prof.Accommodations)

		instructors, err := repo.InstructorIDsForStudent(ctx, d.Pool, prof.StudentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve instructors.")
			return
		}
		orgID, _ := organization.OrgIDForUser(ctx, d.Pool, prof.StudentID)
		var orgPtr *uuid.UUID
		if orgID != uuid.Nil {
			orgPtr = &orgID
		}
		ns := d.notificationsService()
		vars := map[string]string{
			"subject":     svc.LetterSubject(studentName),
			"studentName": studentName,
			"labels":      strings.Join(svc.Labels(prof.Accommodations), ", "),
			"letter":      letter,
		}
		for _, id := range instructors {
			if err := ns.EnqueueEmail(ctx, id, eventAccommodationActive, "accommodation_active", vars, orgPtr); err != nil {
				slog.Warn("accessibility.notify_instructors", "err", err, "instructor_id", id)
			}
		}
		if err := repo.MarkNotified(ctx, d.Pool, prof.ID); err != nil {
			slog.Warn("accessibility.mark_notified", "err", err, "profile_id", prof.ID)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"notifiedInstructorCount": len(instructors),
			"letter":                  letter,
		})
	}
}

func (d Deps) handleMyAccommodationProfiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.accessibilityFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		profiles, err := repo.ListActiveForStudent(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load your accommodations.")
			return
		}
		courses, _ := repo.CoursesForStudent(r.Context(), d.Pool, userID)
		out := make([]profileJSON, 0, len(profiles))
		for i := range profiles {
			out = append(out, profileToJSON(profiles[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"profiles":        out,
			"affectedCourses": affectedCoursesJSON(courses),
		})
	}
}

// loadProfile parses {id}, loads the profile, and enforces the feature gate already done
// by the caller. Returns the profile or writes an error and reports false.
func (d Deps) loadProfile(w http.ResponseWriter, r *http.Request) (*repo.Profile, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid profile id.")
		return nil, false
	}
	prof, err := repo.Get(r.Context(), d.Pool, id)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load accommodation profile.")
		return nil, false
	}
	if prof == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Accommodation profile not found.")
		return nil, false
	}
	return prof, true
}

func affectedCoursesJSON(courses []repo.AffectedCourse) []map[string]string {
	out := make([]map[string]string, 0, len(courses))
	for _, c := range courses {
		out = append(out, map[string]string{
			"courseId":   c.CourseID.String(),
			"courseCode": c.CourseCode,
			"title":      c.Title,
		})
	}
	return out
}

func ptrTime(t time.Time) *time.Time { return &t }
