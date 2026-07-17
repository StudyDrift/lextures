package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/service/boardfilter"
)

func TestQuizGames_Safety_NicknameKickMuteGuest_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTestWithCfg(t, ctx, "teacher", true, true, func(c *config.Config) {
		c.FFIqGuestJoin = true
	})
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{
		"pacing":         "manual",
		"allowGuests":    true,
		"oneSessionRule": "takeover",
		"maxJoinsPerIp":  5,
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start game: %d %s", w.Code, w.Body.String())
	}
	var started map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &started)
	gameID, _ := started["gameId"].(string)
	joinCode, _ := started["joinCode"].(string)

	// Offensive nickname rejected + audited.
	badNick, _ := json.Marshal(map[string]any{"nickname": boardfilter.DefaultEnglish[0]})
	studentTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(badNick))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("offensive nickname: want 400 got %d %s", w.Code, w.Body.String())
	}

	joinBody, _ := json.Marshal(map[string]any{"nickname": "Ada"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(joinBody))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:12345"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("join: %d %s", w.Code, w.Body.String())
	}
	var player map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &player)
	playerID, _ := player["playerId"].(string)

	// Mute names.
	muteBody, _ := json.Marshal(map[string]any{"namesMuted": true})
	req = httptest.NewRequest(http.MethodPatch,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/safety", bytes.NewReader(muteBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("mute: %d %s", w.Code, w.Body.String())
	}
	sess, err := quizgame.GetSession(ctx, pool, gameID)
	if err != nil || !sess.NamesMuted {
		t.Fatalf("names muted: err=%v sess=%+v", err, sess)
	}

	// Kick/ban prevents rejoin.
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players/"+playerID+"/kick", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("kick: %d %s", w.Code, w.Body.String())
	}
	rejoin, _ := json.Marshal(map[string]any{"nickname": "Ada2"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(rejoin))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("banned rejoin: want 403 got %d %s", w.Code, w.Body.String())
	}

	// Guest join allowed when flag + session allow_guests.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/live-quizzes/join/"+joinCode, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var lookup map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &lookup)
	if lookup["allowsGuests"] != true {
		t.Fatalf("allowsGuests want true got %v", lookup["allowsGuests"])
	}
	guestBody, _ := json.Marshal(map[string]any{"nickname": "GuestOne"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/live-quizzes/join/"+joinCode+"/players", bytes.NewReader(guestBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.50:9999"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("guest join: %d %s", w.Code, w.Body.String())
	}

	// Open-text filter for projector.
	dist := map[string]int{"ok": 1, boardfilter.DefaultEnglish[0]: 2}
	filtered := quizgame.FilterDistributionForProjector(dist)
	if _, ok := filtered[boardfilter.DefaultEnglish[0]]; ok {
		t.Fatal("projector should withhold profane open text")
	}

	events, err := quizgame.ListSafetyEvents(ctx, pool, gameID, 50)
	if err != nil {
		t.Fatal(err)
	}
	foundDenied := false
	foundKick := false
	for _, e := range events {
		if e.Kind == quizgame.SafetyNicknameDenied {
			foundDenied = true
		}
		if e.Kind == quizgame.SafetyKicked {
			foundKick = true
		}
	}
	if !foundDenied || !foundKick {
		t.Fatalf("safety events missing denied=%v kick=%v events=%d", foundDenied, foundKick, len(events))
	}
}
