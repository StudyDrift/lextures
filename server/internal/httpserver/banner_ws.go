package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	bannersrepo "github.com/lextures/lextures/server/internal/repos/banners"
)

// handleBannerWS is GET /api/v1/status/banner/ws — public WebSocket that fans out
// maintenance-banner create/update/clear events so StatusBanner updates in real time.
// Auth is optional (the banner itself is public); any client text messages are drained
// so close frames keep the connection state current.
func (d Deps) handleBannerWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.effectiveConfig().MaintenanceBannerEnabled {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		// Drain client reads (close frames / optional first auth message).
		go func() {
			for {
				if _, _, err := conn.Read(runCtx); err != nil {
					stop()
					return
				}
			}
		}()

		if d.BannerHub == nil {
			// Keep the socket open so clients do not thrash reconnects when the hub is unset in tests.
			<-runCtx.Done()
			return
		}

		events, unsubscribe := d.BannerHub.Subscribe()
		defer unsubscribe()
		for {
			select {
			case <-runCtx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				payload, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				writeCtx, cancel := context.WithTimeout(runCtx, 10*time.Second)
				err = conn.Write(writeCtx, websocket.MessageText, payload)
				cancel()
				if err != nil {
					return
				}
			}
		}
	}
}

func (d Deps) publishBannerCleared(b *bannersrepo.Banner) {
	if d.BannerHub == nil || b == nil {
		return
	}
	orgID := ""
	if b.OrgID != nil {
		orgID = b.OrgID.String()
	}
	d.BannerHub.Cleared(b.ID.String(), string(b.Scope), orgID)
}

func (d Deps) publishBannerUpserted(b *bannersrepo.Banner) {
	if d.BannerHub == nil || b == nil {
		return
	}
	orgID := ""
	if b.OrgID != nil {
		orgID = b.OrgID.String()
	}
	d.BannerHub.Upserted(b.ID.String(), string(b.Scope), orgID)
}
