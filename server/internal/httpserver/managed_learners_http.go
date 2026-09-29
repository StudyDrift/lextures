package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	managedlearners "github.com/lextures/lextures/server/internal/service/managedlearners"
)

func (d Deps) managedLearnersEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFHomeschoolManagedLearners {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Managed learners are not enabled.")
		return false
	}
	return true
}

func (d Deps) registerManagedLearnersRoutes(r chi.Router) {
	r.Get("/api/v1/me/dependents", d.handleListDependents())
	r.Post("/api/v1/me/dependents", d.handleCreateDependent())
	// Static session exit before parameterized routes.
	r.Delete("/api/v1/me/dependents/sessions/current", d.handleEndDependentSession())
	r.Post("/api/v1/me/dependents/{id}/sessions", d.handleStartDependentSession())
	r.Patch("/api/v1/me/dependents/{id}", d.handlePatchDependent())
	r.Delete("/api/v1/me/dependents/{id}", d.handleDeleteDependent())
}

// managedLearnerGuardMiddleware blocks sensitive writes during Learn-as sessions.
func (d Deps) managedLearnerGuardMiddleware() func(http.Handler) http.Handler {
	exitPath := "/api/v1/me/dependents/sessions/current"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := auth.BearerToken(r.Header)
			if !ok || d.JWTSigner == nil || auth.JWTType(token) != "managed_learner" {
				next.ServeHTTP(w, r)
				return
			}
			path := r.URL.Path
			method := r.Method
			// Always allow exiting Learn-as.
			if method == http.MethodDelete && path == exitPath {
				next.ServeHTTP(w, r)
				return
			}
			// Block managing dependents, billing, admin, password changes while learning as child.
			blocked := false
			switch {
			case strings.HasPrefix(path, "/api/v1/me/dependents"):
				blocked = true
			case strings.HasPrefix(path, "/api/v1/admin"), strings.HasPrefix(path, "/api/v1/admin-console"):
				blocked = true
			case strings.Contains(path, "/billing"), strings.HasPrefix(path, "/api/v1/me/billing"):
				blocked = true
			case path == "/api/v1/me/password" || strings.HasSuffix(path, "/change-password"):
				blocked = method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
			}
			if blocked {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Action not allowed while learning as a managed learner.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (d Deps) handleListDependents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		list, err := managedlearners.List(r.Context(), d.Pool, actor)
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		if list == nil {
			list = []managedlearners.Dependent{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"dependents": list})
	}
}

func (d Deps) handleCreateDependent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			DisplayName  string  `json:"displayName"`
			GradeLevel   *string `json:"gradeLevel"`
			Relationship string  `json:"relationship"`
			Under13      bool    `json:"under13"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		dep, err := managedlearners.Create(r.Context(), d.Pool, managedlearners.CreateParams{
			ActorID:      actor,
			DisplayName:  body.DisplayName,
			GradeLevel:   body.GradeLevel,
			Relationship: body.Relationship,
			Under13:      body.Under13,
		})
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, dep)
	}
}

func (d Deps) handlePatchDependent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dependent id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			DisplayName *string `json:"displayName"`
			GradeLevel  *string `json:"gradeLevel"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		dep, err := managedlearners.Patch(r.Context(), d.Pool, managedlearners.PatchParams{
			ActorID:     actor,
			DependentID: id,
			DisplayName: body.DisplayName,
			GradeLevel:  body.GradeLevel,
		})
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, dep)
	}
}

func (d Deps) handleDeleteDependent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dependent id.")
			return
		}
		if err := managedlearners.Deactivate(r.Context(), d.Pool, actor, id); err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleStartDependentSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		if token, ok := auth.BearerToken(r.Header); ok {
			typ := auth.JWTType(token)
			if typ == "managed_learner" || typ == "impersonation" {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Nested Learn-as is not allowed.")
				return
			}
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dependent id.")
			return
		}
		result, err := managedlearners.StartSession(r.Context(), d.Pool, d.JWTSigner, actor, id)
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"accessToken": result.Token,
			"expiresAt":   result.ExpiresAt,
			"target":      result.Target,
		})
	}
}

func (d Deps) handleEndDependentSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
			return
		}
		token, ok := auth.BearerToken(r.Header)
		if !ok || auth.JWTType(token) != "managed_learner" {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Not in a Learn-as session.")
			return
		}
		ml, err := d.JWTSigner.VerifyManagedLearner(token)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Learn-as session expired.")
				return
			}
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid Learn-as session.")
			return
		}
		actorID, err := uuid.Parse(ml.ActorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid Learn-as session.")
			return
		}
		targetID, err := uuid.Parse(ml.TargetUserID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid Learn-as session.")
			return
		}
		if err := managedlearners.EndSession(r.Context(), d.Pool, managedlearners.EndSessionParams{
			JTI:          ml.JTI,
			ActorID:      actorID,
			TargetUserID: targetID,
		}); err != nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Failed to end Learn-as session.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeManagedLearnersErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, managedlearners.ErrFeatureDisabled):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Managed learners are not enabled.")
	case errors.Is(err, managedlearners.ErrForbidden), errors.Is(err, managedlearners.ErrActorBlocked):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
	case errors.Is(err, managedlearners.ErrNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Learner not found.")
	case errors.Is(err, managedlearners.ErrCapExceeded):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "You have reached the maximum number of managed learners.")
	case errors.Is(err, managedlearners.ErrInvalidInput):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, managedlearners.ErrStoreDown):
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Learn-as is temporarily unavailable.")
	default:
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process managed learner request.")
	}
}
