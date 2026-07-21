package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/authservice"
	"github.com/lextures/lextures/server/internal/service/parentassign"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	permParentLinksManage = "org:parent-links:assign:manage"

	parentAssignLimit      = 30
	parentAssignWindow     = time.Minute
	parentResendLimit      = 5
	parentResendWindow     = time.Hour
)

type parentAssignAttempt struct {
	count int
	start time.Time
}

var (
	parentAssignMu       sync.Mutex
	parentAssignAttempts = map[string]*parentAssignAttempt{}
)

func parentAssignRateLimited(key string, limit int, window time.Duration) bool {
	parentAssignMu.Lock()
	defer parentAssignMu.Unlock()
	now := time.Now()
	e := parentAssignAttempts[key]
	if e == nil || now.Sub(e.start) > window {
		parentAssignAttempts[key] = &parentAssignAttempt{count: 1, start: now}
		return false
	}
	e.count++
	return e.count > limit
}

func (d Deps) parentPortalFeatureOff(w http.ResponseWriter) bool {
	if d.effectiveConfig().FFParentPortal {
		return false
	}
	apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Parent portal is not enabled.")
	return true
}

// parentLinksManageAccess requires org:parent-links:manage or Global Admin.
// Dual-accepts org_admin grants for one release so existing org admins are not locked out (PP.1 rollout).
func (d Deps) parentLinksManageAccess(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) (actor uuid.UUID, ok bool) {
	actor, ok = d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	ctx := r.Context()
	ga, err := rbac.UserHasPermission(ctx, d.Pool, actor, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, false
	}
	if ga {
		return actor, true
	}
	has, err := rbac.UserHasPermission(ctx, d.Pool, actor, permParentLinksManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, false
	}
	if has {
		uOrg, oerr := organization.OrgIDForUser(ctx, d.Pool, actor)
		if oerr != nil || uOrg != orgID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return uuid.UUID{}, false
		}
		return actor, true
	}
	// Dual-accept legacy org_admin for this release.
	uOrg, err := organization.OrgIDForUser(ctx, d.Pool, actor)
	if err != nil || uOrg != orgID {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	admin, err := orgroles.UserHasRole(ctx, d.Pool, actor, orgID, orgroles.RoleOrgAdmin)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, false
	}
	if !admin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return actor, true
}

func (d Deps) parentAssignService() parentassign.Deps {
	return parentassign.Deps{
		Pool:   d.Pool,
		Config: d.effectiveConfig(),
		HIBP:   d.passwordChecker(),
	}
}

func (d Deps) registerParentAssignRoutes(r chi.Router) {
	r.Get("/api/v1/orgs/{orgId}/parent-assign/students", d.handleParentAssignStudentSearch())
	r.Get("/api/v1/orgs/{orgId}/parent-assign/students/{studentId}/links", d.handleParentAssignListLinks())
	r.Post("/api/v1/orgs/{orgId}/parent-assign/students/{studentId}/guardians", d.handleParentAssignGuardians())
	r.Post("/api/v1/orgs/{orgId}/parent-assign/links/{linkId}/resend", d.handleParentAssignResend())
}

func (d Deps) handleParentAssignStudentSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.parentPortalFeatureOff(w) {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, ok := d.parentLinksManageAccess(w, r, orgID); !ok {
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		hits, err := parentassign.SearchStudents(r.Context(), d.Pool, orgID, q, 20)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to search students.")
			return
		}
		if hits == nil {
			hits = []parentassign.StudentHit{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"students": hits})
	}
}

func (d Deps) handleParentAssignListLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.parentPortalFeatureOff(w) {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, ok := d.parentLinksManageAccess(w, r, orgID); !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "studentId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		list, err := parentlinks.ListParentsForStudent(r.Context(), d.Pool, studentID, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list links.")
			return
		}
		if list == nil {
			list = []parentlinks.Link{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"links": list})
	}
}

func (d Deps) handleParentAssignGuardians() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.parentPortalFeatureOff(w) {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		actor, ok := d.parentLinksManageAccess(w, r, orgID)
		if !ok {
			return
		}
		if parentAssignRateLimited("assign:"+actor.String(), parentAssignLimit, parentAssignWindow) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many assign requests. Please try again later.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "studentId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		var body struct {
			Guardians []struct {
				Name         string `json:"name"`
				Email        string `json:"email"`
				Relationship string `json:"relationship"`
			} `json:"guardians"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		inputs := make([]parentassign.GuardianInput, 0, len(body.Guardians))
		for _, g := range body.Guardians {
			inputs = append(inputs, parentassign.GuardianInput{
				Name:         g.Name,
				Email:        g.Email,
				Relationship: g.Relationship,
			})
		}
		ip := adminConsoleClientIP(r)
		ua := r.UserAgent()
		results, err := d.parentAssignService().AssignGuardians(r.Context(), orgID, studentID, actor, inputs, &ip, &ua)
		if err != nil {
			if errors.Is(err, parentassign.ErrInvalidInput) || errors.Is(err, parentassign.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to assign guardians.")
			return
		}
		telemetry.RecordBusinessEvent("parent_link_assign")
		writeJSON(w, http.StatusOK, map[string]any{"results": results})
	}
}

func (d Deps) handleParentAssignResend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.parentPortalFeatureOff(w) {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		actor, ok := d.parentLinksManageAccess(w, r, orgID)
		if !ok {
			return
		}
		linkID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "linkId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid link id.")
			return
		}
		ln, err := parentlinks.GetByID(r.Context(), d.Pool, orgID, linkID)
		if err != nil || ln == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Link not found.")
			return
		}
		if parentAssignRateLimited("resend:"+strings.ToLower(ln.ParentEmail), parentResendLimit, parentResendWindow) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many resend requests for this email. Please try again later.")
			return
		}
		ip := adminConsoleClientIP(r)
		ua := r.UserAgent()
		if err := d.parentAssignService().ResendInvite(r.Context(), orgID, linkID, actor, &ip, &ua); err != nil {
			if errors.Is(err, parentassign.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Link not found.")
				return
			}
			if errors.Is(err, parentassign.ErrNotPending) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Only pending invites can be resent.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resend invite.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func (d Deps) handleParentInviteConsume() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.parentPortalFeatureOff(w) {
			return
		}
		var body struct {
			Token    string `json:"token"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		uid, err := d.parentAssignService().ConsumeInvite(r.Context(), body.Token, body.Password)
		if err != nil {
			if errors.Is(err, parentassign.ErrInvalidToken) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidResetToken, "This activate link is invalid or has expired. Ask school staff to resend the invite.")
				return
			}
			if pol, ok := authservice.IsPasswordPolicyViolation(err); ok {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, pol.Detail)
				return
			}
			var fe authservice.FieldError
			if errors.As(err, &fe) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, fe.Message)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to activate account.")
			return
		}
		telemetry.RecordBusinessEvent("parent_link_activate")
		writeJSON(w, http.StatusOK, map[string]any{
			"message":      "Your account is ready. Sign in to open the Family dashboard.",
			"parentUserId": uid.String(),
			"redirectTo":   "/parent",
		})
	}
}
