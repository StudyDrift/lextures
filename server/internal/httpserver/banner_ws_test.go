package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/bannerevents"
	"github.com/lextures/lextures/server/internal/config"
	bannersrepo "github.com/lextures/lextures/server/internal/repos/banners"
)

func TestBannerWS_UpgradeAndReceiveCleared(t *testing.T) {
	// Not parallel: exercises a live WebSocket through the full middleware chain.
	hub := bannerevents.New()
	srv := httptest.NewServer(NewHandler(Deps{
		BannerHub: hub,
		Config: config.Config{
			MaintenanceBannerEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/status/banner/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial failed: %v (HTTP %d)", err, resp.StatusCode)
		}
		t.Fatalf("websocket dial failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSwitchingProtocols)
	}

	// Wait for the handler to Subscribe, then publish once and read.
	// Avoid short Read timeouts — canceling a client Read can tear down the socket.
	time.Sleep(100 * time.Millisecond)
	hub.Cleared("banner-1", "global", "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ev bannerevents.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(data))
	}
	if ev.Type != "banner_changed" || ev.Action != "cleared" || ev.ID != "banner-1" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBannerWS_DisabledFeatureReturns404(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{
		BannerHub: bannerevents.New(),
		Config: config.Config{
			MaintenanceBannerEnabled: false,
		},
	})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner/ws", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

func TestPublishBannerCleared_NilSafe(t *testing.T) {
	t.Parallel()
	d := Deps{}
	id := uuid.New()
	d.publishBannerCleared(&bannersrepo.Banner{ID: id, Scope: bannersrepo.ScopeGlobal})
	d.publishBannerUpserted(&bannersrepo.Banner{ID: id, Scope: bannersrepo.ScopeOrg})
}
