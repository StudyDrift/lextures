package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
)

func TestQuizReports_BuildExportGradebook_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{"pacing": "manual"})
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
	if gameID == "" {
		t.Fatalf("no gameId: %v", started)
	}

	studentTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	joinBody, _ := json.Marshal(map[string]any{"nickname": "Ada"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(joinBody))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("join: %d %s", w.Code, w.Body.String())
	}
	var joined map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &joined)
	playerID, _ := joined["playerId"].(string)

	sess, err := quizgame.GetSession(ctx, pool, gameID)
	if err != nil {
		t.Fatal(err)
	}
	st := sess.EngineState()
	now := time.Now().UTC()
	next, ev, err := engine.Reduce(st, engine.ActionOpen, now)
	if err != nil {
		t.Fatal(err)
	}
	next = engine.ApplyDeadline(next, 20)
	if _, err := quizgame.ApplyHostTransition(ctx, pool, gameID, next, ev, false); err != nil {
		t.Fatalf("open q: %v", err)
	}
	ans, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: playerID, QuestionIndex: 0,
		Answer: json.RawMessage(`{"optionId":"b"}`), ReceivedAt: now.Add(2 * time.Second),
	})
	if err != nil || ans == nil || !ans.Accepted {
		t.Fatalf("answer: %+v err=%v", ans, err)
	}

	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/end", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("end: %d %s", w.Code, w.Body.String())
	}

	// Instructor report
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/report", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("report: %d %s", w.Code, w.Body.String())
	}
	var reportBody map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &reportBody)
	rep, _ := reportBody["report"].(map[string]any)
	if rep == nil {
		t.Fatalf("missing report: %v", reportBody)
	}
	if int(rep["playerCount"].(float64)) < 1 {
		t.Fatalf("playerCount=%v", rep["playerCount"])
	}

	// Student my-results
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/my-results", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("my-results: %d %s", w.Code, w.Body.String())
	}
	var mine map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &mine)
	if mine["nickname"] != "Ada" {
		t.Fatalf("nickname=%v", mine["nickname"])
	}

	// Student cannot open full report
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/report", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("student report want 403 got %d", w.Code)
	}

	// CSV export (instructor)
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/report/export?format=csv", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("csv: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("ct=%s", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "nickname") || !strings.Contains(w.Body.String(), "Ada") {
		t.Fatalf("csv body=%s", w.Body.String())
	}

	// Rebuild matches stored (AC-8)
	stored, err := quizgame.GetGameReport(ctx, pool, gameID)
	if err != nil || stored == nil {
		t.Fatalf("stored report: %v", err)
	}
	rebuilt, err := quizgame.BuildAndStoreReport(ctx, pool, gameID)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if !quizgame.ReportsMatch(stored, rebuilt) {
		t.Fatalf("recompute mismatch stored=%+v rebuilt=%+v", stored, rebuilt)
	}

	// Gradebook push
	gbBody, _ := json.Marshal(map[string]any{
		"mapping": "participation", "pointsPossible": 5, "participationPct": 50,
	})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/gradebook-link", bytes.NewReader(gbBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("gradebook push: %d %s", w.Code, w.Body.String())
	}
	var gb map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &gb)
	link, _ := gb["link"].(map[string]any)
	linkID, _ := link["id"].(string)
	itemID, _ := link["gradebookItemId"].(string)
	if linkID == "" || itemID == "" {
		t.Fatalf("link=%v", gb)
	}

	// Idempotent re-push
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/gradebook-link", bytes.NewReader(gbBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("re-push: %d %s", w.Code, w.Body.String())
	}
	var gb2 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &gb2)
	link2, _ := gb2["link"].(map[string]any)
	if link2["id"] != linkID {
		t.Fatalf("idempotent link id changed: %v vs %v", linkID, link2["id"])
	}

	// Unlink
	req = httptest.NewRequest(http.MethodDelete,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/gradebook-link/"+linkID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("unlink: %d %s", w.Code, w.Body.String())
	}
	// Game still exists
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("game after unlink: %d", w.Code)
	}
	var gone int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.course_structure_items WHERE id = $1::uuid`, itemID).Scan(&gone)
	if gone != 0 {
		t.Fatalf("structure item should be deleted, count=%d", gone)
	}
}
