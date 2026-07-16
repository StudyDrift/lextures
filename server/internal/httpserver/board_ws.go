package httpserver

// Y.js WebSocket relay for collaboration boards (plan VC.4).
// Protocol matches collab docs: byte 0 = sync (persist + relay), byte 1 = awareness (relay only).

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/telemetry"
	"github.com/lextures/lextures/server/internal/yrelay"
)

const (
	boardWSMaxMessageBytes = 1 << 20 // 1 MiB
	boardWSMaxMsgsPerSec   = 40
	boardWSBurst           = 80
)

var globalBoardRooms = yrelay.NewRegistry()

// handleBoardWS is GET /api/v1/courses/{course_code}/boards/{board_id}/ws.
// First message must be text JSON: {"authToken":"..."}.
func (d Deps) handleBoardWS() http.HandlerFunc {
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
		cfg := d.effectiveConfig()
		if !cfg.FFVisualBoards || !cfg.FFBoardsRealtime {
			http.Error(w, "boards realtime disabled", http.StatusNotFound)
			return
		}

		courseCode := chi.URLParam(r, "course_code")
		rawBoardID := chi.URLParam(r, "board_id")
		boardID, err := uuid.Parse(rawBoardID)
		if err != nil {
			http.Error(w, "invalid board id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		authCtx, cancelAuth := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelAuth()
		typ, b, err := conn.Read(authCtx)
		if err != nil || typ != websocket.MessageText {
			return
		}
		var m struct {
			AuthToken string `json:"authToken"`
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
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, userID)
		if err != nil || !has {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			return
		}
		crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || crow == nil || !crow.VisualBoardsEnabled {
			return
		}
		row, err := board.Get(r.Context(), d.Pool, courseCode, boardID.String())
		if err != nil || row == nil || row.Archived {
			return
		}
		caps, err := board.ResolveAccess(r.Context(), d.Pool, row, userID, board.ResolveOpts{
			CourseCode:             courseCode,
			ExternalSharingAllowed: cfg.FFBoardsExternalSharing,
		})
		if err != nil || !caps.CanView {
			return
		}

		replay, err := board.GetReplayState(r.Context(), d.Pool, boardID)
		if err == nil {
			writeCtx, cancelWrite := context.WithTimeout(r.Context(), 30*time.Second)
			if len(replay.Snapshot) > 0 {
				_ = conn.Write(writeCtx, websocket.MessageBinary, yrelay.EncodeSyncUpdate(replay.Snapshot))
			}
			for _, upd := range replay.Updates {
				_ = conn.Write(writeCtx, websocket.MessageBinary, yrelay.EncodeSyncUpdate(upd))
			}
			cancelWrite()
		}
		telemetry.RecordBusinessEvent("board.ws.replay")

		{
			writeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			_ = conn.Write(writeCtx, websocket.MessageBinary, yrelay.EncodeEmptySyncStep1())
			cancel()
		}

		client := &yrelay.Client{
			ID:     uuid.New(),
			UserID: userID,
			Conn:   conn,
		}
		room := globalBoardRooms.GetOrCreate(boardID)
		room.Add(client)
		telemetry.RecordBusinessEvent("board.ws.join")
		defer func() {
			room.Remove(client.ID)
			globalBoardRooms.MaybeDelete(boardID)
			telemetry.RecordBusinessEvent("board.ws.leave")
		}()

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		var msgWindowStart time.Time
		var msgCount int

		for {
			msgType, data, err := conn.Read(runCtx)
			if err != nil {
				return
			}
			if msgType != websocket.MessageBinary || len(data) == 0 {
				continue
			}
			if len(data) > boardWSMaxMessageBytes {
				telemetry.RecordBusinessEvent("board.ws.oversized")
				_ = conn.Close(websocket.StatusPolicyViolation, "message too large")
				return
			}
			now := time.Now()
			if msgWindowStart.IsZero() || now.Sub(msgWindowStart) >= time.Second {
				msgWindowStart = now
				msgCount = 0
			}
			msgCount++
			if msgCount > boardWSMaxMsgsPerSec && msgCount > boardWSBurst/2 {
				telemetry.RecordBusinessEvent("board.ws.rate_limited")
				_ = conn.Close(websocket.StatusPolicyViolation, "rate limit exceeded")
				return
			}

			switch data[0] {
			case 0: // sync
				subType := byte(0)
				if len(data) > 1 {
					subType = data[1]
				}
				switch subType {
				case 0:
					writeCtx, cancel := context.WithTimeout(runCtx, 10*time.Second)
					_ = client.Send(writeCtx, yrelay.EncodeEmptySyncStep2())
					replay2, _ := board.GetReplayState(writeCtx, d.Pool, boardID)
					if len(replay2.Snapshot) > 0 {
						_ = client.Send(writeCtx, yrelay.EncodeSyncUpdate(replay2.Snapshot))
					}
					for _, upd := range replay2.Updates {
						_ = client.Send(writeCtx, yrelay.EncodeSyncUpdate(upd))
					}
					cancel()
				case 1, 2:
					// Re-check lock/freeze on each sync write (VC.7).
					liveBoard, gerr := board.Get(runCtx, d.Pool, courseCode, boardID.String())
					if gerr != nil || liveBoard == nil {
						continue
					}
					if err := board.CheckWriteAllowed(liveBoard, caps.CanManage, board.WriteSync, time.Now().UTC()); err != nil {
						telemetry.RecordBusinessEvent("board.ws.write_rejected")
						writeCtx, cancel := context.WithTimeout(runCtx, 2*time.Second)
						_ = conn.Write(writeCtx, websocket.MessageText, []byte(`{"error":"board_locked_or_frozen"}`))
						cancel()
						continue
					}
					rawUpdate := yrelay.ExtractUpdateFromMsg(data)
					if len(rawUpdate) > 0 {
						storeCtx, cancel := context.WithTimeout(runCtx, 5*time.Second)
						_ = board.StoreUpdate(storeCtx, d.Pool, boardID, userID, rawUpdate)
						cancel()
						telemetry.RecordBusinessEvent("board.ws.update_persisted")
					}
					room.Broadcast(runCtx, client.ID, data)
					telemetry.RecordBusinessEvent("board.ws.update_relayed")
				}
			case 1: // awareness — relay only; identity for authz is JWT, not client-claimed userId
				room.Broadcast(runCtx, client.ID, data)
				telemetry.RecordBusinessEvent("board.ws.awareness_relayed")
			}
		}
	}
}
