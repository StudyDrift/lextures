package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/communication"
	"github.com/lextures/lextures/server/internal/repos/gdpr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	platformpeople "github.com/lextures/lextures/server/internal/repos/platformpeople"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/authservice"
	"github.com/lextures/lextures/server/internal/service/introcourse"
	"github.com/lextures/lextures/server/internal/service/coursereviews"
	"github.com/lextures/lextures/server/internal/service/licensesvc"
)

func (d Deps) registerPlatformPeopleRoutes(r chi.Router) {
	r.Get("/api/v1/admin/people", d.handleAdminPeopleSearch())
	r.Post("/api/v1/admin/people/invite", d.handleAdminPeopleInvite())
	r.Get("/api/v1/admin/people/{userId}/report", d.handleAdminPeopleReport())
	r.Patch("/api/v1/admin/people/{userId}", d.handleAdminPeoplePatch())
	r.Delete("/api/v1/admin/people/{userId}", d.handleAdminPeopleDelete())
}

func parsePlatformPeopleListParams(r *http.Request) platformpeople.ListParams {
	p := platformpeople.ListParams{
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if v := strings.TrimSpace(r.URL.Query().Get("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Page = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("per_page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.PerPage = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("perPage")); v != "" && p.PerPage == 0 {
		if n, err := strconv.Atoi(v); err == nil {
			p.PerPage = n
		}
	}
	return p
}

func (d Deps) handleAdminPeopleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		params := parsePlatformPeopleListParams(r)
		if params.Query == "" {
			writeJSON(w, http.StatusOK, platformpeople.ListResult{Items: []platformpeople.PersonRow{}})
			return
		}
		result, err := platformpeople.Search(r.Context(), d.Pool, params)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to search users.")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (d Deps) handleAdminPeopleInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			Email     string  `json:"email"`
			FirstName *string `json:"firstName"`
			LastName  *string `json:"lastName"`
			OrgID     *string `json:"orgId"`
			Role      *string `json:"role"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		email := userrepo.NormalizeEmail(body.Email)
		if email == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "email is required.")
			return
		}
		ctx := r.Context()
		existing, err := userrepo.FindByEmail(ctx, d.Pool, email)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to look up user.")
			return
		}
		if existing != nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "A user with that email already exists.")
			return
		}

		orgID, err := d.resolveInviteOrgID(ctx, actor, body.OrgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		firstName := peopleTrimPtr(body.FirstName)
		lastName := peopleTrimPtr(body.LastName)
		displayName := strings.TrimSpace(peopleStrVal(firstName) + " " + peopleStrVal(lastName))
		if displayName == "" {
			displayName = email
		}

		ph, err := authservice.PlaceholderPasswordHash()
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to provision user.")
			return
		}

		uid, err := platformpeople.InsertUser(ctx, d.Pool, orgID, email, ph, displayName, firstName, lastName)
		if err != nil {
			var pe *pgconn.PgError
			if errors.As(err, &pe) && pe.Code == "23505" {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "A user with that email already exists.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create user.")
			return
		}

		role := "student"
		if body.Role != nil && strings.TrimSpace(*body.Role) != "" {
			role = strings.TrimSpace(strings.ToLower(*body.Role))
		}
		if err := rbac.AssignUserRoleByName(ctx, d.Pool, uid, platformpeople.CliRoleToApp(role)); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to assign role.")
			return
		}
		introcourse.EnsureEnrollmentBestEffort(r.Context(), d.Pool, d.effectiveConfig(), d.Pool, uid, introcourse.PathAdminImport)

		if err := d.licenseService().CheckCanActivate(ctx, uid, orgID); err != nil {
			if errors.Is(err, licensesvc.ErrSeatLimitReached) {
				writeSeatLimitError(w)
				return
			}
		}

		if _, err := authservice.RequestPasswordReset(ctx, d.Pool, d.effectiveConfig(), email); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "User created but invite email could not be sent.")
			return
		}
		communication.SendWelcomeMessage(ctx, d.Pool, email)

		d.recordPlatformPeopleAudit(r, actor, &orgID, auditservice.EventUserCreate, uid, raw)

		result, err := platformpeople.Search(ctx, d.Pool, platformpeople.ListParams{Query: email, Page: 1, PerPage: 1})
		if err != nil || len(result.Items) == 0 {
			writeJSON(w, http.StatusCreated, map[string]any{"id": uid.String(), "email": email})
			return
		}
		writeJSON(w, http.StatusCreated, result.Items[0])
	}
}

func (d Deps) resolveInviteOrgID(ctx context.Context, actor uuid.UUID, orgIDRaw *string) (uuid.UUID, error) {
	if orgIDRaw != nil && strings.TrimSpace(*orgIDRaw) != "" {
		oid, err := uuid.Parse(strings.TrimSpace(*orgIDRaw))
		if err != nil {
			return uuid.UUID{}, errors.New("Invalid orgId.")
		}
		org, err := organization.GetByID(ctx, d.Pool, oid)
		if err != nil || org == nil {
			return uuid.UUID{}, errors.New("Organization not found.")
		}
		return oid, nil
	}
	return organization.OrgIDForUser(ctx, d.Pool, actor)
}

func (d Deps) handleAdminPeopleReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "userId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		rep, err := platformpeople.UserReport(r.Context(), d.Pool, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load user report.")
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func (d Deps) handleAdminPeoplePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		targetID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "userId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			Active *bool `json:"active"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Active == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "active is required.")
			return
		}
		ctx := r.Context()
		if *body.Active {
			var orgID uuid.UUID
			if err := d.Pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, targetID).Scan(&orgID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load user.")
				return
			}
			if err := d.licenseService().CheckCanActivate(ctx, targetID, orgID); err != nil {
				if errors.Is(err, licensesvc.ErrSeatLimitReached) {
					writeSeatLimitError(w)
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify seat license.")
				return
			}
		}
		if err := platformpeople.SetActive(ctx, d.Pool, targetID, *body.Active); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update user.")
			return
		}
		if !*body.Active {
			_ = authservice.RevokeAllSessionsForUser(ctx, d.Pool, targetID)
		}
		var orgID uuid.UUID
		_ = d.Pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, targetID).Scan(&orgID)
		eventType := auditservice.EventUserDeactivate
		if *body.Active {
			eventType = auditservice.EventUserUpdate
		}
		d.recordPlatformPeopleAudit(r, actor, &orgID, eventType, targetID, raw)

		result, err := platformpeople.Search(ctx, d.Pool, platformpeople.ListParams{Query: targetID.String(), Page: 1, PerPage: 1})
		if err != nil || len(result.Items) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{"id": targetID.String(), "active": *body.Active})
			return
		}
		writeJSON(w, http.StatusOK, result.Items[0])
	}
}

func (d Deps) handleAdminPeopleDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		targetID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "userId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		if targetID == actor {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "You cannot delete your own account.")
			return
		}
		ctx := r.Context()
		var orgID uuid.UUID
		var email string
		err = d.Pool.QueryRow(ctx, `SELECT org_id, email FROM "user".users WHERE id = $1`, targetID).Scan(&orgID, &email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load user.")
			return
		}
		if platformpeople.IsErased(email) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This account has already been deleted.")
			return
		}

		_ = authservice.RevokeAllSessionsForUser(ctx, d.Pool, targetID)
		if err := platformpeople.SetActive(ctx, d.Pool, targetID, false); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to deactivate user.")
			return
		}
		if d.LearnerProfileService != nil {
			_ = d.LearnerProfileService.Erase(ctx, targetID)
		}
		if err := gdpr.AnonymiseUser(ctx, d.Pool, targetID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete user account.")
			return
		}
		_ = coursereviews.AnonymizeReviewerReviews(ctx, d.Pool, targetID)

		d.recordPlatformPeopleAudit(r, actor, &orgID, auditservice.EventUserDeactivate, targetID, nil)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": targetID.String()})
	}
}

func (d Deps) recordPlatformPeopleAudit(r *http.Request, actor uuid.UUID, orgID *uuid.UUID, eventType string, targetID uuid.UUID, after []byte) {
	if !d.effectiveConfig().AdminAuditLogEnabled {
		return
	}
	tt := "user"
	ip := adminConsoleClientIP(r)
	ua := r.UserAgent()
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:      orgID,
		EventType:  eventType,
		ActorID:    actor,
		ActorIP:    &ip,
		UserAgent:  &ua,
		TargetType: &tt,
		TargetID:   &targetID,
		AfterValue: after,
	})
}

func peopleTrimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func peopleStrVal(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}