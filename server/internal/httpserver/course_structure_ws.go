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

var structureEventsHub struct {
	mu   sync.RWMutex
	subs map[string]map[chan struct{}]struct{}
}

func init() {
	structureEventsHub.subs = make(map[string]map[chan struct{}]struct{})
}

func structureHubSubscribe(courseCode string) chan struct{} {
	ch := make(chan struct{}, 4)
	structureEventsHub.mu.Lock()
	if structureEventsHub.subs[courseCode] == nil {
		structureEventsHub.subs[courseCode] = make(map[chan struct{}]struct{})
	}
	structureEventsHub.subs[courseCode][ch] = struct{}{}
	structureEventsHub.mu.Unlock()
	return ch
}

func structureHubUnsubscribe(courseCode string, ch chan struct{}) {
	structureEventsHub.mu.Lock()
	delete(structureEventsHub.subs[courseCode], ch)
	if len(structureEventsHub.subs[courseCode]) == 0 {
		delete(structureEventsHub.subs, courseCode)
	}
	structureEventsHub.mu.Unlock()
}

func broadcastStructureChanged(courseCode string) {
	if courseCode == "" {
		return
	}
	structureEventsHub.mu.RLock()
	defer structureEventsHub.mu.RUnlock()
	for ch := range structureEventsHub.subs[courseCode] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

const structureChangedJSON = `{"type":"structure_changed"}`

// handleCourseStructureWS is GET /api/v1/courses/{course_code}/structure/ws.
// Notifies clients when course modules/items change (including during Canvas import).
func (d Deps) handleCourseStructureWS() http.HandlerFunc {
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

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
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
		uid, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}
		has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, uid)
		if err != nil || !has {
			return
		}

		ch := structureHubSubscribe(courseCode)
		defer structureHubUnsubscribe(courseCode, ch)

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		go func() {
			for {
				if _, _, err := conn.Read(runCtx); err != nil {
					stop()
					return
				}
			}
		}()

		msg := []byte(structureChangedJSON)
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
