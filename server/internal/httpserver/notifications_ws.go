package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const notificationUpdatedJSON = `{"type":"notification_updated"}`

// handleNotificationsWS is GET /api/v1/ws/notifications — first text message: {"authToken":"…"}.
func (d Deps) handleNotificationsWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.JWTSigner == nil {
			http.Error(w, "auth not configured", http.StatusServiceUnavailable)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		readAuthCtx, cancelAuth := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelAuth()

		typ, b, err := conn.Read(readAuthCtx)
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
		uid, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		if d.NotifHub != nil {
			recv, unsub := d.NotifHub.Subscribe(uid)
			defer unsub()
			go func() {
				for {
					select {
					case _, ok := <-recv:
						if !ok {
							return
						}
						_ = conn.Write(runCtx, websocket.MessageText, []byte(notificationUpdatedJSON)) //nolint:errcheck
					case <-runCtx.Done():
						return
					}
				}
			}()
		}

		for {
			if _, _, err := conn.Read(runCtx); err != nil {
				return
			}
		}
	}
}