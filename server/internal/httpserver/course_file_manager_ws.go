package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

var fileEventsHub struct {
	mu   sync.RWMutex
	subs map[string]map[chan struct{}]struct{}
}

func init() {
	fileEventsHub.subs = make(map[string]map[chan struct{}]struct{})
}

func fileHubSubscribe(courseCode string) chan struct{} {
	ch := make(chan struct{}, 1)
	fileEventsHub.mu.Lock()
	if fileEventsHub.subs[courseCode] == nil {
		fileEventsHub.subs[courseCode] = make(map[chan struct{}]struct{})
	}
	fileEventsHub.subs[courseCode][ch] = struct{}{}
	fileEventsHub.mu.Unlock()
	return ch
}

func fileHubUnsubscribe(courseCode string, ch chan struct{}) {
	fileEventsHub.mu.Lock()
	delete(fileEventsHub.subs[courseCode], ch)
	if len(fileEventsHub.subs[courseCode]) == 0 {
		delete(fileEventsHub.subs, courseCode)
	}
	fileEventsHub.mu.Unlock()
}

func broadcastFilesChanged(courseCode string) {
	fileEventsHub.mu.RLock()
	defer fileEventsHub.mu.RUnlock()
	for ch := range fileEventsHub.subs[courseCode] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// handleCourseFilesWS is GET /api/v1/courses/{course_code}/files/ws.
// Sends {"type":"files_changed"} to all connected clients when a file or folder mutation occurs.
func (d Deps) handleCourseFilesWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.JWTSigner == nil || d.Pool == nil {
			http.Error(w, "server misconfiguration", http.StatusServiceUnavailable)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if courseCode == "" {
			http.Error(w, "missing course", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		// Auth: first message must be {"authToken":"..."}.
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
		uid, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, uid)
		if err != nil || !has {
			return
		}

		ch := fileHubSubscribe(courseCode)
		defer fileHubUnsubscribe(courseCode, ch)

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		// Read goroutine detects client disconnect.
		go func() {
			for {
				if _, _, err := conn.Read(runCtx); err != nil {
					stop()
					return
				}
			}
		}()

		msg := []byte(`{"type":"files_changed"}`)
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ch:
				writeCtx, cancel := context.WithTimeout(runCtx, 5*time.Second)
				_ = conn.Write(writeCtx, websocket.MessageText, msg)
				cancel()
			}
		}
	}
}
