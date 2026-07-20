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
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ssrepo "github.com/lextures/lextures/server/internal/repos/screenshare"
	"github.com/lextures/lextures/server/internal/screenshare/engine"
	ssturn "github.com/lextures/lextures/server/internal/screenshare/turn"
	"github.com/pion/webrtc/v4"
)

func (d Deps) screenShareFlagsOK(w http.ResponseWriter, r *http.Request, courseCode string) (*course.CoursePublic, bool) {
	cfg := d.effectiveConfig()
	if !cfg.ScreenShareEnabled {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Screen sharing is disabled on this platform.")
		return nil, false
	}
	crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || crow == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return nil, false
	}
	if !crow.ScreenShareEnabled {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Screen sharing is not enabled for this course.")
		return nil, false
	}
	return crow, true
}

func (d Deps) turnReady() bool {
	cfg := d.effectiveConfig()
	return strings.TrimSpace(cfg.TURNSharedSecret) != "" && len(cfg.TURNURLs) > 0
}

func (d Deps) mintICEServers(userID string) (map[string]any, error) {
	cfg := d.effectiveConfig()
	creds, err := ssturn.Mint(cfg.TURNSharedSecret, userID, time.Now().UTC(), ssturn.DefaultTTL)
	if err != nil {
		return nil, err
	}
	stunURLs := make([]string, 0)
	turnURLs := make([]string, 0)
	for _, u := range cfg.TURNURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		lower := strings.ToLower(u)
		if strings.HasPrefix(lower, "stun:") {
			stunURLs = append(stunURLs, u)
		} else {
			turnURLs = append(turnURLs, u)
		}
	}
	servers := []map[string]any{}
	if len(stunURLs) > 0 {
		servers = append(servers, map[string]any{"urls": stunURLs})
	}
	if len(turnURLs) > 0 {
		servers = append(servers, map[string]any{
			"urls":       turnURLs,
			"username":   creds.Username,
			"credential": creds.Credential,
		})
	}
	return map[string]any{
		"iceServers": servers,
		"ttlSeconds": creds.TTLSeconds,
	}, nil
}

func (d Deps) webrtcICEServers() []webrtc.ICEServer {
	cfg := d.effectiveConfig()
	out := make([]webrtc.ICEServer, 0, len(cfg.TURNURLs))
	for _, u := range cfg.TURNURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		out = append(out, webrtc.ICEServer{URLs: []string{u}})
	}
	return out
}

func (d Deps) requireScreenShareHost(w http.ResponseWriter, r *http.Request, courseCode string, userID uuid.UUID) bool {
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, userID, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Host permission required.")
		return false
	}
	return true
}

type createScreenShareBody struct {
	Title        string `json:"title"`
	Policy       string `json:"policy"`
	PresentAudio bool   `json:"presentAudio"`
	ViewerCap    int    `json:"viewerCap"`
}

// handleCreateScreenShareSession is POST /api/v1/courses/{course_code}/screen-share/sessions
func (d Deps) handleCreateScreenShareSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		crow, ok := d.screenShareFlagsOK(w, r, courseCode)
		if !ok {
			return
		}
		if !d.turnReady() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Screen sharing TURN is not configured.")
			return
		}
		cid, err := uuid.Parse(crow.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		var body createScreenShareBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		policy := body.Policy
		if policy == "" {
			policy = string(engine.PolicyRequest)
		}

		isHost := false
		if has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create"); err == nil && has {
			isHost = true
		}
		if !isHost {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only a host can start a screen share session.")
			return
		}

		sess, joinToken, err := ssrepo.CreateSession(r.Context(), d.Pool, ssrepo.CreateInput{
			CourseID:     cid,
			HostID:       viewer,
			Title:        body.Title,
			Policy:       policy,
			PresentAudio: body.PresentAudio,
			ViewerCap:    body.ViewerCap,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create session.")
			return
		}
		turn, err := d.mintICEServers(viewer.String())
		if err != nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Failed to mint TURN credentials.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId": sess.ID,
			"joinToken": joinToken,
			"turn":      turn,
			"session":   screenShareSessionJSON(sess),
		})
	}
}

func screenShareSessionJSON(s *ssrepo.Session) map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"id":                s.ID,
		"courseId":          s.CourseID,
		"hostId":            s.HostID,
		"title":             s.Title,
		"status":            s.Status,
		"policy":            s.Policy,
		"presentAudio":      s.PresentAudio,
		"viewerCap":         s.ViewerCap,
		"activePresenterId": s.ActivePresenterID,
		"startedAt":         s.StartedAt,
		"endedAt":           s.EndedAt,
		"createdAt":         s.CreatedAt,
	}
}

// handleGetScreenShareSession is GET /api/v1/courses/{course_code}/screen-share/sessions/{id}
func (d Deps) handleGetScreenShareSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if _, ok := d.screenShareFlagsOK(w, r, courseCode); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "session_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		sess, err := ssrepo.GetSession(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
			return
		}
		crow, _ := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if crow == nil || crow.ID != sess.CourseID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(screenShareSessionJSON(sess))
	}
}

// handleEndScreenShareSession is POST .../sessions/{id}/end
func (d Deps) handleEndScreenShareSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if _, ok := d.screenShareFlagsOK(w, r, courseCode); !ok {
			return
		}
		if !d.requireScreenShareHost(w, r, courseCode, viewer) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "session_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		if err := ssrepo.EndSession(r.Context(), d.Pool, id, viewer.String()); err != nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Session already ended.")
			return
		}
		globalScreenShareSFU.Delete(id)
		globalScreenShareRooms.MaybeDelete(id)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}

type presenterActionBody struct {
	Action string `json:"action"` // grant | revoke
	UserID string `json:"userId"`
}

// handleScreenSharePresenter is POST .../sessions/{id}/presenter
func (d Deps) handleScreenSharePresenter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if _, ok := d.screenShareFlagsOK(w, r, courseCode); !ok {
			return
		}
		if !d.requireScreenShareHost(w, r, courseCode, viewer) {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "session_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		var body presenterActionBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		sess, err := ssrepo.GetSession(r.Context(), d.Pool, id)
		if err != nil || sess.Status == "ended" || sess.Status == "abandoned" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Session is not active.")
			return
		}
		st := ssrepo.EngineState(sess, 0, nil)
		var action engine.Action
		switch strings.ToLower(strings.TrimSpace(body.Action)) {
		case "grant":
			action = engine.ActionGrantPresent
		case "revoke":
			action = engine.ActionRevokePresent
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "action must be grant or revoke.")
			return
		}
		next, evs, err := engine.Reduce(st, action, viewer.String(), body.UserID, nil)
		if err != nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, err.Error())
			return
		}
		var presenter *uuid.UUID
		status := string(next.Status)
		if next.ActivePresenterID != "" {
			if u, err := uuid.Parse(next.ActivePresenterID); err == nil {
				presenter = &u
			}
		}
		_ = ssrepo.SetPresenter(r.Context(), d.Pool, id, presenter, status)
		for _, ev := range evs {
			_ = ssrepo.AppendEvent(r.Context(), d.Pool, sess.ID, ev.Type, ev.ActorID, ev.Payload)
		}
		broadcastScreenSharePresentChanged(id, next.ActivePresenterID)
		if next.ActivePresenterID == "" {
			if room := globalScreenShareSFU.Get(id); room != nil {
				room.ClearPresenter()
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"activePresenterId": next.ActivePresenterID,
			"status":            status,
		})
	}
}

// handleScreenShareTurn is POST .../sessions/{id}/turn
func (d Deps) handleScreenShareTurn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if _, ok := d.screenShareFlagsOK(w, r, courseCode); !ok {
			return
		}
		if !d.turnReady() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Screen sharing TURN is not configured.")
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "session_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		sess, err := ssrepo.GetSession(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
			return
		}
		if sess.Status == "ended" || sess.Status == "abandoned" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Session has ended.")
			return
		}
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, viewer)
		if err != nil || !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Not enrolled.")
			return
		}
		turn, err := d.mintICEServers(viewer.String())
		if err != nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Failed to mint TURN credentials.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(turn)
	}
}
