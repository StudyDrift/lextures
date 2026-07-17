package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func enrollExtraQuizPlayer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, h http.Handler, cc string, courseID uuid.UUID) string {
	t.Helper()
	em := fmt.Sprintf("quizplayer-%d@test.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("student user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	if _, err := pool.Exec(ctx,
		`INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')`,
		courseID, uid,
	); err != nil {
		t.Fatalf("student enroll: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, uid, courseID, cc); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("student grants: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, _ := signer.Sign(ctx, row.ID, em, "", "", nil)
	_ = h
	return tok
}

func seedReadyKit(t *testing.T, h http.Handler, tok, cc string) string {
	t.Helper()
	kitID := createKitViaAPI(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{
		"questionType":     "mc_single",
		"prompt":           "2+2?",
		"timeLimitSeconds": 20,
		"pointsStyle":      "standard",
		"options": []map[string]any{
			{"id": "a", "text": "3", "isCorrect": false},
			{"id": "b", "text": "4", "isCorrect": true},
			{"id": "c", "text": "5", "isCorrect": false},
			{"id": "d", "text": "22", "isCorrect": false},
		},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/questions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed question: %d %s", w.Code, w.Body.String())
	}
	return kitID
}

func TestQuizGames_StartJoinAnswerEnd_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)

	// Kit not ready → 400
	emptyKit := createKitViaAPI(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{"pacing": "manual"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+emptyKit+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty kit start: want 400 got %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start game: %d %s", w.Code, w.Body.String())
	}
	var started map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &started)
	gameID, _ := started["gameId"].(string)
	joinCode, _ := started["joinCode"].(string)
	if gameID == "" || len(joinCode) != 6 {
		t.Fatalf("bad start payload: %v", started)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/live-quizzes/join/"+joinCode, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("join lookup: %d %s", w.Code, w.Body.String())
	}
	var lookup map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &lookup)
	if lookup["courseCode"] != cc {
		t.Fatalf("lookup courseCode: want %s got %v", cc, lookup["courseCode"])
	}
	if lookup["allowsGuests"] != false {
		t.Fatalf("allowsGuests should default false: %v", lookup["allowsGuests"])
	}

	joinBody, _ := json.Marshal(map[string]any{"nickname": "Ada"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(joinBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("join player: %d %s", w.Code, w.Body.String())
	}
	var player map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &player)
	playerID, _ := player["playerId"].(string)
	playerToken, _ := player["playerToken"].(string)
	if playerID == "" || playerToken == "" {
		t.Fatalf("bad player: %v", player)
	}

	// Rejoin same enrolled user rotates token (HTTP 200).
	rejoinBody, _ := json.Marshal(map[string]any{"nickname": "Ada2"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(rejoinBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("rejoin player: %d %s", w.Code, w.Body.String())
	}
	var rejoined map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &rejoined)
	if rejoined["playerId"] != playerID {
		t.Fatalf("rejoin should keep player id: %v", rejoined)
	}
	newToken, _ := rejoined["playerToken"].(string)
	if newToken == "" || newToken == playerToken {
		t.Fatalf("rejoin should rotate token: old=%s new=%s", playerToken, newToken)
	}
	playerToken = newToken

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
		t.Fatal(err)
	}

	ans1, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID:     gameID,
		PlayerID:      playerID,
		QuestionIndex: 0,
		Answer:        json.RawMessage(`{"optionId":"b"}`),
		ReceivedAt:    now.Add(3200 * time.Millisecond),
	})
	if err != nil || !ans1.Accepted || !ans1.IsCorrect {
		t.Fatalf("answer1: %+v err=%v", ans1, err)
	}
	if ans1.ResponseMs < 3000 || ans1.ResponseMs > 3500 {
		t.Fatalf("response_ms=%d", ans1.ResponseMs)
	}

	_, err = quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID:     gameID,
		PlayerID:      playerID,
		QuestionIndex: 0,
		Answer:        json.RawMessage(`{"optionId":"a"}`),
		ReceivedAt:    now.Add(4 * time.Second),
	})
	if err != quizgame.ErrDuplicateAnswer {
		t.Fatalf("want duplicate, got %v", err)
	}

	// Second enrolled student for late-answer coverage (one player row per user).
	studentTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	joinBody2, _ := json.Marshal(map[string]any{"nickname": "Grace"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(joinBody2))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("join student: %d %s", w.Code, w.Body.String())
	}
	var player2 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &player2)
	p2, _ := player2["playerId"].(string)
	if p2 == "" || p2 == playerID {
		t.Fatalf("expected distinct student player, got %v", player2)
	}
	sess, _ = quizgame.GetSession(ctx, pool, gameID)
	lateAt := sess.QuestionDeadlineAt.Add(time.Second)
	_, err = quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID:     gameID,
		PlayerID:      p2,
		QuestionIndex: 0,
		Answer:        json.RawMessage(`{"optionId":"b"}`),
		ReceivedAt:    lateAt,
	})
	if err != quizgame.ErrLateAnswer {
		t.Fatalf("want late, got %v", err)
	}

	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/end", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("end: %d %s", w.Code, w.Body.String())
	}
	var ended map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &ended)
	if ended["status"] != "ended" || ended["phase"] != "ended" {
		t.Fatalf("ended payload: %v", ended)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/live-quizzes/join/"+joinCode, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("ended join lookup want 404 got %d", w.Code)
	}

	p, err := quizgame.GetPlayerByToken(ctx, pool, gameID, playerToken)
	if err != nil || p.ID != playerID {
		t.Fatalf("reconnect token: %+v err=%v", p, err)
	}
}

func TestQuizGames_HostingSubFlagOff_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{
		FFInteractiveQuizzes: true,
		FFIqLiveHosting:      false,
	}})
	kitID := createKitViaAPI(t, h, tok, cc)

	body, _ := json.Marshal(map[string]any{"pacing": "manual"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 when iq_live_hosting off, got %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+kitID+"/ws", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("ws want 404 got %d", w.Code)
	}
}

func TestQuizGames_ScoringLeaderboardBreakdown_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{
		"pacing":             "manual",
		"scoringProfile":     "competitive",
		"leaderboardPrivacy": "nicknames",
		"powerUpsEnabled":    false,
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var started map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &started)
	gameID, _ := started["gameId"].(string)
	game, _ := started["game"].(map[string]any)
	if game["scoringProfile"] != "competitive" {
		t.Fatalf("scoringProfile: %v", game["scoringProfile"])
	}
	if game["leaderboardPrivacy"] != "nicknames" {
		t.Fatalf("leaderboardPrivacy: %v", game["leaderboardPrivacy"])
	}

	join := func(authTok, nick string) string {
		t.Helper()
		jb, _ := json.Marshal(map[string]any{"nickname": nick})
		r := httptest.NewRequest(http.MethodPost,
			"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(jb))
		r.Header.Set("Authorization", "Bearer "+authTok)
		r.Header.Set("Content-Type", "application/json")
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, r)
		if rw.Code != http.StatusCreated && rw.Code != http.StatusOK {
			t.Fatalf("join %s: %d %s", nick, rw.Code, rw.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rw.Body.Bytes(), &out)
		id, _ := out["playerId"].(string)
		return id
	}

	pFast := join(tok, "Fast")
	studentTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	pSlow := join(studentTok, "Slow")

	sess, err := quizgame.GetSession(ctx, pool, gameID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	next, ev, err := engine.Reduce(sess.EngineState(), engine.ActionOpen, now)
	if err != nil {
		t.Fatal(err)
	}
	next = engine.ApplyDeadline(next, 10)
	if _, err := quizgame.ApplyHostTransition(ctx, pool, gameID, next, ev, false); err != nil {
		t.Fatal(err)
	}

	fast, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: pFast, QuestionIndex: 0,
		Answer: json.RawMessage(`{"optionId":"b"}`), ReceivedAt: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	slow, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: pSlow, QuestionIndex: 0,
		Answer: json.RawMessage(`{"optionId":"b"}`), ReceivedAt: now.Add(8 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if fast.Points <= slow.Points {
		t.Fatalf("AC-1: fast=%d should beat slow=%d", fast.Points, slow.Points)
	}
	if fast.Points < 1000 || slow.Points < 1000 {
		t.Fatalf("both ≥ base: fast=%d slow=%d", fast.Points, slow.Points)
	}
	if fast.PointsBreakdown.Total != fast.Points {
		t.Fatalf("breakdown total mismatch: %+v", fast.PointsBreakdown)
	}
	sum := fast.PointsBreakdown.Base + fast.PointsBreakdown.SpeedBonus + fast.PointsBreakdown.StreakBonus
	if sum != fast.Points {
		t.Fatalf("AC-6 components %d != total %d", sum, fast.Points)
	}

	// Idempotent reconnect: re-submit is duplicate; score unchanged.
	before, _ := quizgame.GetPlayer(ctx, pool, pFast)
	_, err = quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: pFast, QuestionIndex: 0,
		Answer: json.RawMessage(`{"optionId":"b"}`), ReceivedAt: now.Add(3 * time.Second),
	})
	if err != quizgame.ErrDuplicateAnswer {
		t.Fatalf("want duplicate got %v", err)
	}
	after, _ := quizgame.GetPlayer(ctx, pool, pFast)
	if after.TotalScore != before.TotalScore {
		t.Fatalf("AC-5 double-award: before=%d after=%d", before.TotalScore, after.TotalScore)
	}

	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/leaderboard", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("leaderboard: %d %s", w.Code, w.Body.String())
	}
	var lb map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &lb)
	if lb["privacy"] != "nicknames" {
		t.Fatalf("privacy: %v", lb["privacy"])
	}

	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/responses/"+pFast, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("responses: %d %s", w.Code, w.Body.String())
	}
	var respBody map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &respBody)
	responses, _ := respBody["responses"].([]any)
	if len(responses) != 1 {
		t.Fatalf("want 1 response got %v", respBody)
	}
}

func TestQuizGames_FormativeEqualPoints_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{"pacing": "manual", "scoringProfile": "formative"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var started map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &started)
	gameID, _ := started["gameId"].(string)

	joinBody, _ := json.Marshal(map[string]any{"nickname": "A"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(joinBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var p1 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &p1)
	id1, _ := p1["playerId"].(string)

	stTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	joinBody2, _ := json.Marshal(map[string]any{"nickname": "B"})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(joinBody2))
	req.Header.Set("Authorization", "Bearer "+stTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var p2 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &p2)
	id2, _ := p2["playerId"].(string)

	sess, _ := quizgame.GetSession(ctx, pool, gameID)
	now := time.Now().UTC()
	next, ev, _ := engine.Reduce(sess.EngineState(), engine.ActionOpen, now)
	next = engine.ApplyDeadline(next, 10)
	_, _ = quizgame.ApplyHostTransition(ctx, pool, gameID, next, ev, false)

	a, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: id1, QuestionIndex: 0,
		Answer: json.RawMessage(`{"optionId":"b"}`), ReceivedAt: now.Add(1 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: id2, QuestionIndex: 0,
		Answer: json.RawMessage(`{"optionId":"b"}`), ReceivedAt: now.Add(9 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Points != b.Points || a.Points != 1000 {
		t.Fatalf("AC-4 formative equal: a=%d b=%d", a.Points, b.Points)
	}
}

func TestQuizGames_HostDisconnectGrace_Unit(t *testing.T) {
	now := time.Now().UTC()
	s := engine.State{
		Phase:         engine.PhaseQuestionOpen,
		Status:        engine.StatusRunning,
		QuestionIndex: 0,
		QuestionCount: 2,
	}
	paused, _ := engine.ReduceHostDisconnect(s, now)
	disc := now
	_, _, ok := engine.ReduceHostReconnect(paused, &disc, now.Add(30*time.Second), engine.HostGraceDefault)
	if !ok {
		t.Fatal("expected reconnect within grace")
	}
	ended, _, ok := engine.ReduceHostReconnect(paused, &disc, now.Add(2*time.Minute), engine.HostGraceDefault)
	if ok || ended.Status != engine.StatusAbandoned {
		t.Fatalf("expected abandon on grace expiry, ok=%v status=%s", ok, ended.Status)
	}
}
