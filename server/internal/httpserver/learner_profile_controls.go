package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	lpsvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

const learnerProfileControlRateLimitPerMinute = 10

type learnerProfileControlRateEntry struct {
	count int
	reset time.Time
}

var (
	learnerProfileControlRateMu sync.Mutex
	learnerProfileControlRates  = map[uuid.UUID]*learnerProfileControlRateEntry{}
)

func (d Deps) checkLearnerProfileControlRateLimit(userID uuid.UUID) bool {
	learnerProfileControlRateMu.Lock()
	defer learnerProfileControlRateMu.Unlock()
	now := time.Now()
	e, ok := learnerProfileControlRates[userID]
	if !ok || now.After(e.reset) {
		learnerProfileControlRates[userID] = &learnerProfileControlRateEntry{count: 1, reset: now.Add(time.Minute)}
		return true
	}
	if e.count >= learnerProfileControlRateLimitPerMinute {
		return false
	}
	e.count++
	return true
}

func (d Deps) recordLearnerProfileControlAudit(r *http.Request, actorID uuid.UUID, targetUserID uuid.UUID, action string) {
	lpsvc.RecordControl(action)
	if !d.effectiveConfig().AdminAuditLogEnabled {
		return
	}
	orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
	var oid *uuid.UUID
	if orgID != uuid.Nil {
		oid = &orgID
	}
	tt := "learner_profile"
	after, _ := json.Marshal(map[string]string{"action": action, "targetUserId": targetUserID.String()})
	ip := clientIP(r)
	ua := r.UserAgent()
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:      oid,
		EventType:  auditservice.EventLearnerProfileControl,
		ActorID:    actorID,
		ActorIP:    &ip,
		UserAgent:  &ua,
		TargetType: &tt,
		TargetID:   &targetUserID,
		AfterValue: after,
	})
}

func (d Deps) handleLearnerProfileControl(action string, fn func(*lpsvc.Service, uuid.UUID, *http.Request) error, responseStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.learnerProfileEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.checkLearnerProfileControlRateLimit(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many profile control requests. Try again later.")
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		if err := fn(svc, userID, r); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update learner profile.")
			return
		}
		d.recordLearnerProfileControlAudit(r, userID, userID, action)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": responseStatus})
	}
}

func (d Deps) handlePostLearnerProfilePause() http.HandlerFunc {
	return d.handleLearnerProfileControl("pause", func(svc *lpsvc.Service, userID uuid.UUID, r *http.Request) error {
		return svc.Pause(r.Context(), userID)
	}, "paused")
}

func (d Deps) handlePostLearnerProfileResume() http.HandlerFunc {
	return d.handleLearnerProfileControl("resume", func(svc *lpsvc.Service, userID uuid.UUID, r *http.Request) error {
		return svc.Resume(r.Context(), userID)
	}, "active")
}

func (d Deps) handlePostLearnerProfileReset() http.HandlerFunc {
	return d.handleLearnerProfileControl("reset", func(svc *lpsvc.Service, userID uuid.UUID, r *http.Request) error {
		return svc.Reset(r.Context(), userID)
	}, "reset")
}

func (d Deps) handleGetLearnerProfileExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.learnerProfileEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.checkLearnerProfileControlRateLimit(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many profile control requests. Try again later.")
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		doc, err := svc.Export(r.Context(), userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not export learner profile.")
			return
		}
		d.recordLearnerProfileControlAudit(r, userID, userID, "export")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="learner-profile-export.json"`)
		_ = json.NewEncoder(w).Encode(doc)
	}
}