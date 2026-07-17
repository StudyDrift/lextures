package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	boardLinkResolveLimit   = 60
	boardLinkPasswordLimit  = 10
	boardLinkWindow         = time.Minute
	boardLinkContributeRate = 20
)

type boardLinkAttempt struct {
	count int
	start time.Time
}

var (
	boardLinkMu       sync.Mutex
	boardLinkAttempts = map[string]*boardLinkAttempt{}
)

func boardLinkRateLimited(key string, limit int) bool {
	boardLinkMu.Lock()
	defer boardLinkMu.Unlock()
	now := time.Now()
	e := boardLinkAttempts[key]
	if e == nil || now.Sub(e.start) > boardLinkWindow {
		boardLinkAttempts[key] = &boardLinkAttempt{count: 1, start: now}
		return false
	}
	e.count++
	return e.count > limit
}

func (d Deps) registerBoardLinkRoutes(r chi.Router) {
	r.Get("/api/v1/board-links/{token}", d.handleResolveBoardLink())
	r.Post("/api/v1/board-links/{token}/posts", d.handleBoardLinkCreatePost())
}

func (d Deps) resolveActiveBoardLink(w http.ResponseWriter, r *http.Request) (*board.ResolvedShare, board.Capabilities, bool) {
	token := chi.URLParam(r, "token")
	ip := r.RemoteAddr
	if boardLinkRateLimited("resolve:"+ip, boardLinkResolveLimit) {
		apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Please try again later.")
		return nil, board.Capabilities{}, false
	}
	resolved, err := board.ResolveShareToken(r.Context(), d.Pool, token, time.Now().UTC())
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not resolve share link.")
		return nil, board.Capabilities{}, false
	}
	if resolved == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Share link is invalid, expired, or revoked.")
		return nil, board.Capabilities{}, false
	}
	crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, resolved.CourseCode)
	if err != nil || crow == nil || !crow.VisualBoardsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Share link is invalid, expired, or revoked.")
		return nil, board.Capabilities{}, false
	}
	cfg := d.effectiveConfig()
	blocked, reason, err := board.ExternalSharingBlocked(
		r.Context(), d.Pool, resolved.CourseCode, cfg.FFBoardsExternalSharing, cfg.CoppaWorkflowEnabled,
	)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to evaluate sharing policy.")
		return nil, board.Capabilities{}, false
	}
	if blocked {
		msg := "External board sharing is disabled."
		if reason == "minors_policy" {
			msg = "External board sharing is blocked for this course."
		}
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, msg)
		return nil, board.Capabilities{}, false
	}
	if resolved.Share.HasPassword {
		pw := strings.TrimSpace(r.Header.Get("X-Board-Share-Password"))
		if pw == "" {
			pw = strings.TrimSpace(r.URL.Query().Get("password"))
		}
		if boardLinkRateLimited("pw:"+resolved.Share.ID+":"+ip, boardLinkPasswordLimit) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many password attempts.")
			return nil, board.Capabilities{}, false
		}
		ok, err := board.VerifySharePassword(pw, resolved.Share)
		if err != nil || !ok {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Incorrect password.")
			return nil, board.Capabilities{}, false
		}
	}
	caps := board.ResolveOpts{
		ExternalSharingAllowed:  true,
		ForbidExternalForMinors: false,
		ShareCapability:         resolved.Share.Capability,
		ViaShareLink:            true,
	}
	resolvedCaps, err := board.ResolveAccess(r.Context(), d.Pool, &resolved.Board, uuid.Nil, caps)
	if err != nil || !resolvedCaps.CanView {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Share link does not grant access.")
		return nil, board.Capabilities{}, false
	}
	telemetry.RecordBusinessEvent("board.link.viewed")
	return resolved, resolvedCaps, true
}

// handleResolveBoardLink is GET /api/v1/board-links/{token}
func (d Deps) handleResolveBoardLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resolved, caps, ok := d.resolveActiveBoardLink(w, r)
		if !ok {
			return
		}
		posts, err := board.ListPosts(r.Context(), d.Pool, resolved.CourseCode, resolved.Board.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load posts.")
			return
		}
		avOn := d.effectiveConfig().AvScanningEnabled
		outPosts := make([]map[string]any, 0, len(posts))
		for _, p := range posts {
			row := boardPostJSONWithAttribution(p, resolved.CourseCode, avOn, resolved.Board.Attribution, caps)
			// Public/link views never expose roster PII beyond attribution rules.
			outPosts = append(outPosts, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"board":        boardJSONWithAccess(resolved.Board, caps),
			"capability":   resolved.Share.Capability,
			"requiresPassword": resolved.Share.HasPassword,
			"posts":        outPosts,
		})
	}
}

// handleBoardLinkCreatePost is POST /api/v1/board-links/{token}/posts
func (d Deps) handleBoardLinkCreatePost() http.HandlerFunc {
	type reqBody struct {
		DisplayName string          `json:"displayName"`
		ContentType string          `json:"contentType"`
		Title       string          `json:"title"`
		Body        json.RawMessage `json:"body"`
		LinkURL     string          `json:"linkUrl"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		resolved, caps, ok := d.resolveActiveBoardLink(w, r)
		if !ok {
			return
		}
		if !caps.CanPost || resolved.Share.Capability != board.ShareCapabilityContribute {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This share link does not allow contributing.")
			return
		}
		b := &resolved.Board
		if d.writeGateReject(w, board.CheckWriteAllowed(b, false, board.WritePost, time.Now().UTC())) {
			return
		}
		ip := r.RemoteAddr
		if boardLinkRateLimited("post:"+resolved.Share.ID+":"+ip, boardLinkContributeRate) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many posts. Please try again later.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		matched, term, okFilter := d.screenBoardText(w, r, b, nil, in.Title, in.Body)
		if !okFilter {
			return
		}
		created, err := board.CreateGuestPost(r.Context(), d.Pool, resolved.CourseCode, resolved.Board.ID, in.DisplayName, board.CreatePostInput{
			ContentType: in.ContentType,
			Title:       in.Title,
			Body:        in.Body,
			LinkURL:     in.LinkURL,
			Status:      resolveInitialPostStatus(b, false),
		}, nil)
		if err != nil {
			if strings.HasPrefix(err.Error(), "board:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create post.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		if matched {
			d.flagFilterHit(r.Context(), resolved.CourseCode, b, created.ID, term)
		}
		if created.Status == board.PostStatusPending {
			d.notifyBoardManagers(r.Context(), resolved.CourseCode, b.ID, "board_moderation_pending",
				"Board post awaiting approval",
				"A new post is waiting for approval on a board.")
		}
		telemetry.RecordBusinessEvent("board.link.post.created")
		notifyBoardPeers(r.Context(), b.ID, "post.created", created.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardPostJSONWithAttribution(
			*created, resolved.CourseCode, d.effectiveConfig().AvScanningEnabled, resolved.Board.Attribution, caps,
		))
	}
}
