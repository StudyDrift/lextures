package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/oidc"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/meperm"
)

type myPermissionsResponse struct {
	PermissionStrings []string `json:"permissionStrings"`
}

// meSessionUserID requires a login JWT (not an access key). Used for session-only actions.
func (d Deps) meSessionUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if d.JWTSigner == nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.UUID{}, false
	}
	u, err := auth.UserFromRequest(r, d.JWTSigner)
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.UUID{}, false
	}
	userID, err := uuid.Parse(u.UserID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.UUID{}, false
	}
	return d.validateMeUser(w, r, u, userID)
}

// meUserID returns the authenticated user id or writes 401/500 and returns false.
func (d Deps) meUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if d.JWTSigner == nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.UUID{}, false
	}
	u, ctx, err := auth.UserFromRequestOrAccessKey(r, d.JWTSigner, d.Pool)
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.UUID{}, false
	}
	*r = *r.WithContext(ctx)
	userID, err := uuid.Parse(u.UserID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.UUID{}, false
	}
	return d.validateMeUser(w, r, u, userID)
}

func (d Deps) validateMeUser(w http.ResponseWriter, r *http.Request, u auth.AuthUser, userID uuid.UUID) (uuid.UUID, bool) {
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.UUID{}, false
	}
	ctx := r.Context()
	dbOrgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.UUID{}, false
	}
	if u.OrgID != "" && u.OrgID != dbOrgID.String() {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	st, err := organization.OrgStatusForUser(ctx, d.Pool, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.UUID{}, false
	}
	if st == "suspended" {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeOrgSuspended, "This organization has been suspended.")
		return uuid.UUID{}, false
	}
	if st == "deleted" || st == "" {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return userID, true
}

func (d Deps) handleMyPermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		perms, err := meperm.MyPermissions(
			r.Context(), d.Pool, userID, q.Get("courseCode"), q.Get("viewAs"),
		)
		if err != nil {
			st, code, msg := meperm.HTTPErrorFor(err)
			apierr.WriteJSON(w, st, code, msg)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(myPermissionsResponse{PermissionStrings: perms})
	}
}

type oidcIdentityItem struct {
	ID       string  `json:"id"`
	Provider string  `json:"provider"`
	Email    *string `json:"email"`
}

type oidcIdentitiesResponse struct {
	Identities []oidcIdentityItem `json:"identities"`
}

func (d Deps) handleMyOIDCIdentities() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		rows, err := oidc.ListByUserID(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load identities.")
			return
		}
		items := make([]oidcIdentityItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, oidcIdentityItem{
				ID:       row.ID.String(),
				Provider: row.Provider,
				Email:    row.Email,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(oidcIdentitiesResponse{Identities: items})
	}
}

func (d Deps) handleDeleteMyOIDCIdentity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		rawID := chi.URLParam(r, "id")
		identID, err := uuid.Parse(rawID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid identity id.")
			return
		}
		deleted, err := oidc.DeleteByIDForUser(r.Context(), d.Pool, userID, identID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not remove identity.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Identity not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// handleGetMe is GET /api/v1/me — returns the authenticated user's id, email, and display name.
func (d Deps) handleGetMe() http.HandlerFunc {
	type resp struct {
		ID          string  `json:"id"`
		Email       string  `json:"email"`
		DisplayName *string `json:"displayName"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := user.FindByID(r.Context(), d.Pool, userID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName})
	}
}

func (d Deps) registerMeRoutes(r chi.Router) {
	r.Get("/api/v1/me", d.handleGetMe())
	r.Get("/api/v1/me/mfa", d.handleListMyMFA())
	r.Delete("/api/v1/me/mfa/{id}", d.handleDeleteMyMFA())
	r.Get("/api/v1/me/permissions", d.handleMyPermissions())
	r.Get("/api/v1/me/org-role-capabilities", d.handleMeOrgRoleCapabilities())
	r.Get("/api/v1/me/notification-preferences", d.handleGetMyNotificationPreferences())
	r.Put("/api/v1/me/notification-preferences", d.handlePutMyNotificationPreferences())
	r.Get("/api/v1/me/reading-preferences", d.handleGetMyReadingPreferences())
	r.Patch("/api/v1/me/reading-preferences", d.handlePatchMyReadingPreferences())
	// Self-paced enrollments with progress for the Dashboard (plan 15.2)
	r.Get("/api/v1/me/enrollments", d.handleMySelfPacedEnrollments())
	r.Get("/api/unsubscribe", d.handleUnsubscribe())
	r.Get("/api/v1/push/vapid-public-key", d.handleGetVAPIDPublicKey())
	r.Post("/api/v1/me/push-subscriptions", d.handlePostMyPushSubscription())
	r.Delete("/api/v1/me/push-subscriptions/{id}", d.handleDeleteMyPushSubscription())
	r.Get("/api/v1/me/notifications", d.handleGetMyNotifications())
	r.Post("/api/v1/me/notifications/{id}/read", d.handleMarkNotificationRead())
	r.Post("/api/v1/me/notifications/read-all", d.handleMarkAllNotificationsRead())
	r.Get("/api/v1/me/notifications/sse", d.handleNotificationsSSE())
	r.Get("/api/v1/me/sessions", d.handleListMySessions())
	r.Delete("/api/v1/me/sessions", d.handleDeleteMyOtherSessions())
	r.Delete("/api/v1/me/sessions/{id}", d.handleDeleteMySession())
	r.Get("/api/v1/platform/features", d.handleGetPlatformFeatures())
	r.Get("/api/v1/me/oidc-identities", d.handleMyOIDCIdentities())
	r.Delete("/api/v1/me/oidc-identities/{id}", d.handleDeleteMyOIDCIdentity())
	r.Post("/api/v1/me/notebooks/query", d.handleNotebookQuery())
	r.Post("/api/v1/me/notebooks/flashcards", d.handleGenerateNotebookFlashcards())
	d.registerNotebookTaskRoutes(r)
	d.registerStudentNotebookRoutes(r)
	r.Post("/api/v1/stt/transcribe", d.handlePostSTTTranscribe())
	d.registerTTSRoutes(r)
	d.registerSelfReflectionRoutes(r)
	d.registerCCRRoutes(r)
	d.registerIntegrationsRoutes(r)
}
