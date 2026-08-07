package httpserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/lextures/lextures/server/internal/auth"
)

// dialWS opens a websocket against a test server path.
func dialWS(t *testing.T, srvURL, path string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(t.Context(), "ws"+strings.TrimPrefix(srvURL, "http")+path, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", path, err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

// An expired or invalid token must produce a distinguishable close code so
// clients refresh instead of reconnecting with the same dead token in a hot loop.
func TestAppWS_RejectedAuthClosesWithPolicyViolation(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"/api/v1/ws/notifications", "/api/v1/communication/ws"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			signer := auth.NewJWTSigner("test-jwt-secret-for-ws-auth-close")
			srv := httptest.NewServer(NewHandler(Deps{JWTSigner: signer}))
			t.Cleanup(srv.Close)

			conn := dialWS(t, srv.URL, path)
			frame, err := json.Marshal(map[string]string{"authToken": "not-a-valid-jwt"})
			if err != nil {
				t.Fatalf("marshal auth frame: %v", err)
			}
			if err := conn.Write(t.Context(), websocket.MessageText, frame); err != nil {
				t.Fatalf("write auth frame: %v", err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			_, _, err = conn.Read(ctx)
			if got := websocket.CloseStatus(err); got != wsStatusAuthFailed {
				t.Fatalf("close status = %v (err %v), want %v", got, err, wsStatusAuthFailed)
			}
		})
	}
}

// countingListener counts bytes the server writes to accepted connections, so a
// test can tell an idle socket from one carrying keepalive frames.
type countingListener struct {
	net.Listener
	n *atomic.Int64
}

func (l countingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return countingConn{Conn: c, n: l.n}, nil
}

type countingConn struct {
	net.Conn
	n *atomic.Int64
}

func (c countingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.n.Add(int64(n))
	return n, err
}

// Idle sockets carry no application traffic, so without keepalive frames a proxy
// idle timeout (ALB: 60s) reaps them and every client reconnects on a loop.
func TestKeepAliveWS_PingsIdlePeer(t *testing.T) {
	t.Parallel()
	var written atomic.Int64

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()
		go keepAliveWS(runCtx, conn, stop, 10*time.Millisecond)

		// Read dispatches incoming pongs and blocks until the peer goes away.
		for {
			if _, _, err := conn.Read(runCtx); err != nil {
				return
			}
		}
	}))
	srv.Listener = countingListener{Listener: srv.Listener, n: &written}
	srv.Start()
	t.Cleanup(srv.Close)

	conn := dialWS(t, srv.URL, "/")

	// Reading answers the pings (coder/websocket pongs automatically).
	readCtx, cancelRead := context.WithCancel(t.Context())
	defer cancelRead()
	go func() {
		for {
			if _, _, err := conn.Read(readCtx); err != nil {
				return
			}
		}
	}()

	// Baseline after the handshake response has been written.
	time.Sleep(50 * time.Millisecond)
	baseline := written.Load()

	deadline := time.Now().Add(5 * time.Second)
	for written.Load() <= baseline {
		if time.Now().After(deadline) {
			t.Fatal("server sent no keepalive traffic on an idle socket")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A peer that stops answering pings must not leave the handler goroutine parked
// on a socket that is already gone.
func TestKeepAliveWS_CancelsWhenPeerStopsAnswering(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()
		go keepAliveWS(runCtx, conn, stop, 10*time.Millisecond)

		<-runCtx.Done()
		close(done)
	}))
	t.Cleanup(srv.Close)

	conn := dialWS(t, srv.URL, "/")
	// Never read, so pings are never ponged, then drop the connection outright.
	_ = conn.CloseNow()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("keepAliveWS did not cancel the run context after the peer stopped answering")
	}
}
