package httpserver

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

const (
	// wsPingPeriod keeps long-lived app sockets under the shortest proxy idle
	// timeout in front of the API. An AWS ALB defaults to 60s, so an idle
	// notifications/mailbox socket is reaped roughly a minute after the last
	// event and every client reconnects — repeatedly, for as long as the user
	// sits idle. Ping/pong traffic resets that timer.
	wsPingPeriod = 25 * time.Second
	// wsPongTimeout is how long a peer has to answer a keepalive ping before we
	// treat the socket as dead.
	wsPongTimeout = 10 * time.Second

	// wsStatusAuthFailed closes a socket whose auth frame was rejected. Clients
	// use this code to refresh their access token before reconnecting instead of
	// retrying the same expired token in a loop.
	wsStatusAuthFailed = websocket.StatusPolicyViolation
)

// keepAliveWS pings conn every period until ctx is done or the peer stops
// answering, then cancels via stop so the caller's read loop unblocks.
//
// Ping blocks until the pong arrives, which the caller's concurrent Read loop is
// responsible for dispatching; coder/websocket allows Ping concurrently with a
// single reader.
func keepAliveWS(ctx context.Context, conn *websocket.Conn, stop context.CancelFunc, period time.Duration) {
	if period <= 0 {
		period = wsPingPeriod
	}
	// Cap pong wait at the gap between pings so a dead peer is noticed instead of
	// leaving pings queued, but keep a floor so short test periods are not so
	// tight that a single scheduler hiccup aborts keepalive.
	pongTimeout := wsPongTimeout
	if period < pongTimeout {
		pongTimeout = period
	}
	const minPongTimeout = 100 * time.Millisecond
	if pongTimeout < minPongTimeout {
		pongTimeout = minPongTimeout
	}

	pingOnce := func() bool {
		pingCtx, cancel := context.WithTimeout(ctx, pongTimeout)
		err := conn.Ping(pingCtx)
		cancel()
		if err != nil {
			stop()
			return false
		}
		return true
	}

	// Ping immediately so idle sockets get traffic before the first ticker fire.
	if !pingOnce() {
		return
	}

	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !pingOnce() {
				return
			}
		}
	}
}
