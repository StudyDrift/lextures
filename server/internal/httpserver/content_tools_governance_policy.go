package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/config"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	"github.com/lextures/lextures/server/internal/ratelimit"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func moderationToAPI(m *ctrepo.ModerationRow) ctmodel.ModerationAction {
	if m == nil {
		return ctmodel.ModerationAction{}
	}
	out := ctmodel.ModerationAction{
		ID:         m.ID,
		InstanceID: m.InstanceID,
		Action:     m.Action,
		CreatedAt:  m.CreatedAt,
	}
	if m.StateID != nil {
		out.StateID = m.StateID
	}
	if m.ContentPath != nil {
		out.ContentPath = m.ContentPath
	}
	if m.Category != nil {
		out.Category = m.Category
	}
	if m.Reason != nil {
		out.Reason = m.Reason
	}
	if m.ActorUserID != nil {
		out.ActorUserID = m.ActorUserID
	}
	if m.SubjectUserID != nil {
		out.SubjectUserID = m.SubjectUserID
	}
	return out
}

func (d Deps) contentToolsReportRateLimited(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	return d.contentToolsGovRateLimited(w, r, userID, "ct_report", 10)
}

func (d Deps) contentToolsModerateRateLimited(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	return d.contentToolsGovRateLimited(w, r, userID, "ct_moderate", 60)
}

func (d Deps) contentToolsConsentRateLimited(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	return d.contentToolsGovRateLimited(w, r, userID, "ct_consent", 10)
}

func (d Deps) contentToolsGovRateLimited(w http.ResponseWriter, r *http.Request, userID uuid.UUID, key string, limit int) bool {
	limiter := d.buildRateLimiter()
	rule := config.RateLimitRule{Limit: limit, Window: time.Minute}
	dec := limiter.Allow(r.Context(), limiter.UserKey(userID.String(), key), rule, ratelimit.LimitTypeToken)
	if dec.Allowed {
		return false
	}
	ratelimit.RecordExceeded("content_tool_"+key, ratelimit.LimitTypeToken)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Try again later.")
	return true
}

// loadContentToolsOrgPolicy is a helper for catalog/state handlers.
func (d Deps) loadContentToolsOrgPolicy(r *http.Request, courseCode string) *ctrepo.PolicyRow {
	if d.Pool == nil {
		return nil
	}
	orgID, err := course.CourseOrgID(r.Context(), d.Pool, courseCode)
	if err != nil || orgID == nil {
		return nil
	}
	pol, err := ctsvc.LoadOrgPolicy(r.Context(), d.Pool, *orgID)
	if err != nil {
		return nil
	}
	return pol
}

// screenContentToolsFreeText applies org filter + crisis escalation (CT.8 FR-8/FR-9).
// Returns true when the handler should stop (blocked).
func (d Deps) screenContentToolsFreeText(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	courseID, instanceID, viewer uuid.UUID,
	toolID string,
	state json.RawMessage,
) bool {
	text := ctsvc.ExtractFreeTextFromState(state)
	if strings.TrimSpace(text) == "" {
		return false
	}
	pol := d.loadContentToolsOrgPolicy(r, courseCode)
	filterAction := "flag"
	crisis := true
	if pol != nil {
		filterAction = pol.FreeTextFilterAction
		crisis = pol.CrisisEscalationEnabled
	}
	res := ctsvc.ScreenFreeText(text, filterAction, crisis)
	if res.Action == ctsvc.FilterActionAllow && !res.Crisis {
		return false
	}
	uid := viewer
	if res.Crisis {
		_ = ctsvc.RecordCrisisEscalation(r.Context(), d.Pool, courseID, instanceID, &uid, toolID)
		_ = ctsvc.RecordFilterFlag(r.Context(), d.Pool, instanceID, courseID, &uid, ctsvc.FilterCategoryCrisis, ctsvc.FilterActionFlag)
		// Crisis never silently drops work — flag and allow persist unless also blocked for other reasons.
	}
	if res.Action == ctsvc.FilterActionFlag || res.Crisis {
		cat := res.Category
		if cat == "" {
			cat = ctsvc.FilterCategoryProfanity
		}
		if !res.Crisis {
			_ = ctsvc.RecordFilterFlag(r.Context(), d.Pool, instanceID, courseID, &uid, cat, ctsvc.FilterActionFlag)
		}
	}
	if res.Action == ctsvc.FilterActionBlock {
		_ = ctsvc.RecordFilterFlag(r.Context(), d.Pool, instanceID, courseID, &uid, ctsvc.FilterCategoryProfanity, ctsvc.FilterActionBlock)
		var body ctmodel.FreeTextBlockedBody
		body.Error.Code = apierr.CodeUnprocessableEntity
		body.Error.Message = "Submission blocked by content policy."
		body.Error.Guidance = res.Guidance
		body.Error.Category = res.Category
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(body)
		return true
	}
	return false
}

// ensureContentToolsPolicyAllowed returns false after writing a typed denial.
func (d Deps) ensureContentToolsPolicyAllowed(w http.ResponseWriter, r *http.Request, courseCode string, m *ctsvc.CompiledManifest, instanceID *uuid.UUID) bool {
	ctsvc.SyncDurableKillsFromDB(r.Context(), d.Pool)
	pol := d.loadContentToolsOrgPolicy(r, courseCode)
	dec := ctsvc.EvaluateToolPolicy(pol, m, instanceID)
	if dec.Allowed {
		return true
	}
	status := http.StatusForbidden
	code := apierr.CodeForbidden
	if strings.HasPrefix(dec.Reason, "kill_") {
		status = http.StatusServiceUnavailable
		code = apierr.CodeServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Tool-Policy-Reason", dec.Reason)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": dec.Detail,
			"reason":  dec.Reason,
		},
	})
	return false
}
