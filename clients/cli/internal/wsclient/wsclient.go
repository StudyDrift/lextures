package wsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

// StreamOptions configures a one-shot WebSocket session: dial, send the first
// text frame, then invoke onMessage for each subsequent text frame until the
// connection closes or ctx is cancelled.
type StreamOptions struct {
	URL        string
	FirstFrame []byte
	OnMessage  func([]byte) error
	DialTimeout time.Duration
}

// Stream opens a WebSocket, sends FirstFrame, and streams inbound text frames.
func Stream(ctx context.Context, opts StreamOptions) error {
	if opts.OnMessage == nil {
		return fmt.Errorf("wsclient: OnMessage is required")
	}
	dialTimeout := opts.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 30 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(dialCtx, opts.URL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	if len(opts.FirstFrame) > 0 {
		if err := conn.Write(ctx, websocket.MessageText, opts.FirstFrame); err != nil {
			return fmt.Errorf("websocket write: %w", err)
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Minute)
		typ, payload, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			return fmt.Errorf("websocket read: %w", err)
		}
		if typ != websocket.MessageText {
			continue
		}
		if err := opts.OnMessage(payload); err != nil {
			return err
		}
	}
}