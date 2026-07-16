package httpserver

// Y.js WebSocket relay for collaborative documents (plan 6.5).
//
// Protocol: minimal y-websocket sync protocol.
//   Binary messages starting with byte 0 = Y.js sync → relay + persist
//   Binary messages starting with byte 1 = Y.js awareness → relay only
//
// Each new client receives all stored sync updates so it can reconstruct
// the current document state. Y.js CRDTs handle duplicates idempotently.
// Shared framing/room helpers live in server/internal/yrelay.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/collabdocs"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/yrelay"
)

var globalCollabRooms = yrelay.NewRegistry()

// handleCollabDocWS is GET /api/v1/courses/{course_code}/collab-docs/{doc_id}/ws.
// First message must be text JSON: {"authToken":"..."}.
func (d Deps) handleCollabDocWS() http.HandlerFunc {
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
		rawDocID := chi.URLParam(r, "doc_id")
		docID, err := uuid.Parse(rawDocID)
		if err != nil {
			http.Error(w, "invalid doc id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		// Auth: read first text message.
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
		ok, err := collabdocs.BelongsToCourse(r.Context(), d.Pool, *cid, docID)
		if err != nil || !ok {
			return
		}

		updates, err := collabdocs.GetAllUpdates(r.Context(), d.Pool, docID)
		if err == nil {
			writeCtx, cancelWrite := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancelWrite()
			for _, upd := range updates {
				_ = conn.Write(writeCtx, websocket.MessageBinary, yrelay.EncodeSyncUpdate(upd))
			}
		}

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
		room := globalCollabRooms.GetOrCreate(docID)
		room.Add(client)
		defer func() {
			room.Remove(client.ID)
			globalCollabRooms.MaybeDelete(docID)
		}()

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		for {
			msgType, data, err := conn.Read(runCtx)
			if err != nil {
				return
			}
			if msgType != websocket.MessageBinary {
				continue
			}
			if len(data) == 0 {
				continue
			}

			msgClass := data[0]
			switch msgClass {
			case 0:
				subType := byte(0)
				if len(data) > 1 {
					subType = data[1]
				}
				switch subType {
				case 0:
					writeCtx, cancel := context.WithTimeout(runCtx, 10*time.Second)
					_ = client.Send(writeCtx, yrelay.EncodeEmptySyncStep2())
					storedUpdates, _ := collabdocs.GetAllUpdates(writeCtx, d.Pool, docID)
					for _, upd := range storedUpdates {
						_ = client.Send(writeCtx, yrelay.EncodeSyncUpdate(upd))
					}
					cancel()

				case 1, 2:
					rawUpdate := yrelay.ExtractUpdateFromMsg(data)
					if len(rawUpdate) > 0 {
						storeCtx, cancel := context.WithTimeout(runCtx, 5*time.Second)
						_ = collabdocs.StoreUpdate(storeCtx, d.Pool, docID, userID, rawUpdate)
						cancel()
					}
					room.Broadcast(runCtx, client.ID, data)
				}

			case 1:
				room.Broadcast(runCtx, client.ID, data)
			}
		}
	}
}
