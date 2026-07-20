package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ssrepo "github.com/lextures/lextures/server/internal/repos/screenshare"
	"github.com/lextures/lextures/server/internal/screenshare/engine"
	"github.com/lextures/lextures/server/internal/screenshare/sfu"
	"github.com/lextures/lextures/server/internal/yrelay"
	"github.com/pion/webrtc/v4"
)

const (
	screenShareWSMaxMessageBytes = 256 << 10
	screenShareWSMaxMsgsPerSec   = 40
)

var (
	globalScreenShareRooms = yrelay.NewRegistry()
	globalScreenShareSFU   = sfu.NewRegistry()
)

type screenSharePeerMeta struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	Role      string
	ClientID  uuid.UUID
}

var (
	screenSharePeerMu   sync.RWMutex
	screenSharePeerByID = map[uuid.UUID]screenSharePeerMeta{}
)

func setScreenSharePeer(id uuid.UUID, meta screenSharePeerMeta) {
	screenSharePeerMu.Lock()
	screenSharePeerByID[id] = meta
	screenSharePeerMu.Unlock()
}

func clearScreenSharePeer(id uuid.UUID) {
	screenSharePeerMu.Lock()
	delete(screenSharePeerByID, id)
	screenSharePeerMu.Unlock()
}

type wsSignalBridge struct {
	sessionID uuid.UUID
	seq       *atomic.Int64
}

func (b *wsSignalBridge) peerClient(peerID uuid.UUID) *yrelay.Client {
	room := globalScreenShareRooms.Get(b.sessionID)
	if room == nil {
		return nil
	}
	var found *yrelay.Client
	room.ForEach(func(c *yrelay.Client) {
		if c.ID == peerID {
			found = c
		}
	})
	return found
}

func (b *wsSignalBridge) sendJSON(peerID uuid.UUID, payload map[string]any) {
	c := b.peerClient(peerID)
	if c == nil {
		return
	}
	payload["seq"] = b.seq.Add(1)
	bbytes, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.SendText(ctx, bbytes)
}

func (b *wsSignalBridge) SendOffer(peerID uuid.UUID, sdp string) {
	b.sendJSON(peerID, map[string]any{"type": "offer", "sdp": sdp})
}

func (b *wsSignalBridge) SendAnswer(peerID uuid.UUID, sdp string) {
	b.sendJSON(peerID, map[string]any{"type": "answer", "sdp": sdp})
}

func (b *wsSignalBridge) SendICE(peerID uuid.UUID, candidate webrtc.ICECandidateInit) {
	b.sendJSON(peerID, map[string]any{"type": "ice-candidate", "candidate": candidate})
}

func broadcastScreenSharePresentChanged(sessionID uuid.UUID, presenterID string) {
	room := globalScreenShareRooms.Get(sessionID)
	if room == nil {
		return
	}
	var presenter any
	if presenterID != "" {
		presenter = presenterID
	}
	payload, _ := json.Marshal(map[string]any{
		"type":        "present-changed",
		"presenterId": presenter,
		"seq":         time.Now().UnixNano(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	room.BroadcastText(ctx, uuid.Nil, payload)
}

// handleScreenShareWS is GET .../screen-share/sessions/{session_id}/ws
func (d Deps) handleScreenShareWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.JWTSigner == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "server misconfiguration")
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
		if err != nil {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		conn.SetReadLimit(screenShareWSMaxMessageBytes)

		authCtx, cancelAuth := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelAuth()
		typ, b, err := conn.Read(authCtx)
		if err != nil || typ != websocket.MessageText {
			return
		}
		var m struct {
			AuthToken string `json:"authToken"`
			Role      string `json:"role"`
			JoinToken string `json:"joinToken"`
		}
		if err := json.Unmarshal(b, &m); err != nil || m.AuthToken == "" {
			return
		}
		u, err := d.JWTSigner.Verify(r.Context(), m.AuthToken)
		if err != nil {
			return
		}
		userID, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}

		cfg := d.effectiveConfig()
		if !cfg.ScreenShareEnabled {
			_ = writeScreenShareError(conn, "flag_off", "Screen sharing is disabled.")
			return
		}
		crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || crow == nil || !crow.ScreenShareEnabled {
			_ = writeScreenShareError(conn, "flag_off", "Screen sharing is not enabled for this course.")
			return
		}
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, userID)
		if err != nil || !has {
			_ = writeScreenShareError(conn, "not_enrolled", "Not enrolled in this course.")
			return
		}

		sess, err := ssrepo.GetSession(r.Context(), d.Pool, sessionID)
		if err != nil || sess.CourseID != crow.ID {
			_ = writeScreenShareError(conn, "not_found", "Session not found.")
			return
		}
		if sess.Status == "ended" || sess.Status == "abandoned" {
			_ = writeScreenShareError(conn, "ended", "Session has ended.")
			return
		}
		if m.JoinToken != "" && !ssrepo.VerifyJoinToken(sess, m.JoinToken) {
			_ = writeScreenShareError(conn, "bad_token", "Invalid join token.")
			return
		}

		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "host", "presenter", "viewer", "display":
		default:
			role = "viewer"
		}
		isHost := false
		if hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, userID, "course:"+courseCode+":item:create"); err == nil && hasPerm {
			isHost = true
		}
		if role == "host" && !isHost {
			role = "viewer"
		}
		if role == "display" || role == "viewer" {
			n, _ := ssrepo.CountConnectedViewers(r.Context(), d.Pool, sessionID)
			st := ssrepo.EngineState(sess, n, nil)
			if !engine.CanJoinViewer(st) {
				_ = writeScreenShareError(conn, "cap", "Viewer cap reached.")
				return
			}
		}

		_ = ssrepo.UpsertParticipant(r.Context(), d.Pool, sessionID, userID, role)
		_ = ssrepo.AppendEvent(r.Context(), d.Pool, sess.ID, "join", userID.String(), map[string]any{"role": role})

		client := &yrelay.Client{ID: uuid.New(), UserID: userID, Conn: conn}
		room := globalScreenShareRooms.GetOrCreate(sessionID)
		room.Add(client)
		setScreenSharePeer(client.ID, screenSharePeerMeta{
			SessionID: sessionID,
			UserID:    userID,
			Role:      role,
			ClientID:  client.ID,
		})
		defer func() {
			room.Remove(client.ID)
			clearScreenSharePeer(client.ID)
			_ = ssrepo.MarkParticipantLeft(r.Context(), d.Pool, sessionID, userID, role)
			if sfuRoom := globalScreenShareSFU.Get(sessionID); sfuRoom != nil {
				sfuRoom.Detach(client.ID)
			}
			globalScreenShareRooms.MaybeDelete(sessionID)
		}()

		seq := &atomic.Int64{}
		bridge := &wsSignalBridge{sessionID: sessionID, seq: seq}
		sfuRoom := globalScreenShareSFU.GetOrCreate(sessionID, d.webrtcICEServers(), bridge)

		joined := map[string]any{
			"type":     "joined",
			"selfRole": role,
			"seq":      seq.Add(1),
			"status":   sess.Status,
			"policy":   sess.Policy,
		}
		if sess.ActivePresenterID != nil {
			joined["presenterId"] = *sess.ActivePresenterID
		}
		jb, _ := json.Marshal(joined)
		{
			writeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			_ = client.SendText(writeCtx, jb)
			cancel()
		}

		if role == "viewer" || role == "display" || role == "host" {
			_ = sfuRoom.AttachViewer(client.ID)
			if sfuRoom.HasPresenter() {
				_, _ = sfuRoom.CreateViewerOffer(client.ID)
			}
		}

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()
		var msgCount int
		var windowStart = time.Now()

		for {
			msgType, data, err := conn.Read(runCtx)
			if err != nil {
				return
			}
			if msgType != websocket.MessageText {
				continue
			}
			now := time.Now()
			if now.Sub(windowStart) > time.Second {
				windowStart = now
				msgCount = 0
			}
			msgCount++
			if msgCount > screenShareWSMaxMsgsPerSec {
				continue
			}

			var frame map[string]any
			if err := json.Unmarshal(data, &frame); err != nil {
				continue
			}
			ftype, _ := frame["type"].(string)
			switch ftype {
			case "ping":
				pong, _ := json.Marshal(map[string]any{"type": "pong", "seq": seq.Add(1)})
				_ = client.SendText(runCtx, pong)
			case "offer":
				sdp, _ := frame["sdp"].(string)
				fresh, _ := ssrepo.GetSession(runCtx, d.Pool, sessionID)
				isActivePresenter := fresh != nil && fresh.ActivePresenterID != nil && *fresh.ActivePresenterID == userID.String()
				if isActivePresenter || role == "presenter" {
					_ = sfuRoom.AttachPresenter(client.ID)
					_, _ = sfuRoom.HandleOffer(client.ID, sdp)
					// Fan out new offers to existing viewers after presenter publishes.
					room.ForEach(func(c *yrelay.Client) {
						if c.ID == client.ID {
							return
						}
						_ = sfuRoom.AttachViewer(c.ID)
						_, _ = sfuRoom.CreateViewerOffer(c.ID)
					})
				} else {
					_, _ = sfuRoom.HandleOffer(client.ID, sdp)
				}
			case "answer":
				sdp, _ := frame["sdp"].(string)
				_ = sfuRoom.HandleAnswer(client.ID, sdp)
			case "ice-candidate":
				candRaw, _ := json.Marshal(frame["candidate"])
				var cand webrtc.ICECandidateInit
				if err := json.Unmarshal(candRaw, &cand); err == nil {
					_ = sfuRoom.HandleICE(client.ID, cand)
				}
			case "present-request":
				d.handleWSPresentRequest(runCtx, sess, userID, client, room, seq, isHost)
			case "present-stop":
				d.handleWSPresentStop(runCtx, sess, userID, client, room, seq, isHost, sfuRoom)
			case "quality":
				// Reserved for simulcast layer preference (FR-9); acknowledged for protocol completeness.
				ack, _ := json.Marshal(map[string]any{"type": "quality-ack", "seq": seq.Add(1)})
				_ = client.SendText(runCtx, ack)
			default:
				continue
			}
		}
	}
}

func writeScreenShareError(conn *websocket.Conn, code, message string) error {
	payload, _ := json.Marshal(map[string]any{"type": "error", "code": code, "message": message})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, payload)
}

func (d Deps) handleWSPresentRequest(ctx context.Context, sess *ssrepo.Session, userID uuid.UUID, client *yrelay.Client, room *yrelay.Room, seq *atomic.Int64, isHost bool) {
	fresh, err := ssrepo.GetSession(ctx, d.Pool, mustParseUUID(sess.ID))
	if err != nil {
		return
	}
	st := ssrepo.EngineState(fresh, 0, nil)
	action := engine.ActionRequestPresent
	if st.Policy == engine.PolicyFreeForAll {
		action = engine.ActionSelfPromote
	}
	if isHost {
		action = engine.ActionGrantPresent
	}
	target := userID.String()
	next, evs, err := engine.Reduce(st, action, userID.String(), target, nil)
	if err != nil {
		errFrame, _ := json.Marshal(map[string]any{"type": "error", "code": "denied", "message": err.Error(), "seq": seq.Add(1)})
		_ = client.SendText(ctx, errFrame)
		return
	}
	var presenter *uuid.UUID
	if next.ActivePresenterID != "" {
		if u, e := uuid.Parse(next.ActivePresenterID); e == nil {
			presenter = &u
		}
	}
	_ = ssrepo.SetPresenter(ctx, d.Pool, mustParseUUID(sess.ID), presenter, string(next.Status))
	for _, ev := range evs {
		_ = ssrepo.AppendEvent(ctx, d.Pool, sess.ID, ev.Type, ev.ActorID, ev.Payload)
	}
	if next.ActivePresenterID == userID.String() {
		grant, _ := json.Marshal(map[string]any{"type": "present-grant", "seq": seq.Add(1)})
		_ = client.SendText(ctx, grant)
		broadcastScreenSharePresentChanged(mustParseUUID(sess.ID), next.ActivePresenterID)
	} else {
		ack, _ := json.Marshal(map[string]any{"type": "present-request-queued", "seq": seq.Add(1)})
		_ = client.SendText(ctx, ack)
	}
	_ = room // silence if unused in grant path
}

func (d Deps) handleWSPresentStop(ctx context.Context, sess *ssrepo.Session, userID uuid.UUID, client *yrelay.Client, room *yrelay.Room, seq *atomic.Int64, isHost bool, sfuRoom *sfu.Room) {
	fresh, err := ssrepo.GetSession(ctx, d.Pool, mustParseUUID(sess.ID))
	if err != nil {
		return
	}
	st := ssrepo.EngineState(fresh, 0, nil)
	if st.ActivePresenterID != userID.String() && !isHost {
		errFrame, _ := json.Marshal(map[string]any{"type": "error", "code": "denied", "message": "not presenter", "seq": seq.Add(1)})
		_ = client.SendText(ctx, errFrame)
		return
	}
	next, evs, err := engine.Reduce(st, engine.ActionStopPresent, userID.String(), "", nil)
	if err != nil {
		return
	}
	_ = ssrepo.SetPresenter(ctx, d.Pool, mustParseUUID(sess.ID), nil, string(next.Status))
	for _, ev := range evs {
		_ = ssrepo.AppendEvent(ctx, d.Pool, sess.ID, ev.Type, ev.ActorID, ev.Payload)
	}
	if sfuRoom != nil {
		sfuRoom.ClearPresenter()
	}
	broadcastScreenSharePresentChanged(mustParseUUID(sess.ID), "")
	stop, _ := json.Marshal(map[string]any{"type": "present-revoke", "seq": seq.Add(1)})
	_ = client.SendText(ctx, stop)
	_ = room
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
