package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/lextures/lextures/server/internal/auth"
)

func TestCommunicationWS_UpgradeThroughMiddlewareChain(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTSigner("test-jwt-secret-for-ws-middleware-test")
	srv := httptest.NewServer(NewHandler(Deps{JWTSigner: signer, Comm: nil}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/communication/ws"
	ctx := t.Context()
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
}

func TestNotificationsWS_UpgradeThroughMiddlewareChain(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTSigner("test-jwt-secret-for-ws-middleware-test")
	srv := httptest.NewServer(NewHandler(Deps{JWTSigner: signer, NotifHub: nil}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws/notifications"
	ctx := t.Context()
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
}
