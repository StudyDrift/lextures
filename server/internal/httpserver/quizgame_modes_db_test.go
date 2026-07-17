package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/repos/studentaccommodations"
)

func TestQuizModes_TeamLeaderboard_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{
		"mode": "team",
		"teamConfig": map[string]any{
			"teamCount":  2,
			"aggregate":  "average",
			"answerRule": "each_member_answers",
		},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create team game: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	gameID, _ := created["gameId"].(string)
	if gameID == "" {
		t.Fatal("missing gameId")
	}
	game, _ := created["game"].(map[string]any)
	if game["mode"] != "team" {
		t.Fatalf("mode=%v", game["mode"])
	}

	// Assign two students to different teams.
	st1 := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	st2 := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)

	join := func(studentTok, nick string) string {
		b, _ := json.Marshal(map[string]any{"nickname": nick})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost,
			"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(b))
		r.Header.Set("Authorization", "Bearer "+studentTok)
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Fatalf("join %s: %d %s", nick, rr.Code, rr.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		return out["playerId"].(string)
	}
	p1 := join(st1, "AlphaOne")
	p2 := join(st2, "BetaOne")

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/teams/assign",
		bytes.NewReader([]byte(`{"autoBalance":true}`)))
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("assign: %d %s", rr.Code, rr.Body.String())
	}

	// Open question via host transition and submit answers.
	sess, err := quizgame.GetSession(ctx, pool, gameID)
	if err != nil {
		t.Fatal(err)
	}
	st := sess.EngineState()
	st, _, err = engine.Reduce(st, engine.ActionOpen, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	tx, _ := pool.Begin(ctx)
	_ = quizgame.PersistStateFull(ctx, tx, gameID, st, false, false)
	_ = tx.Commit(ctx)

	ans, _ := json.Marshal(map[string]any{"selectedOptionIds": []string{"b"}})
	_, err = quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: p1, QuestionIndex: 0, Answer: ans, ReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("p1 answer: %v", err)
	}
	_, err = quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: p2, QuestionIndex: 0, Answer: ans, ReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("p2 answer: %v", err)
	}

	board, err := quizgame.RefreshTeamScores(ctx, pool, gameID)
	if err != nil {
		t.Fatal(err)
	}
	if len(board) < 1 {
		t.Fatal("expected team leaderboard rows")
	}
	// Individual responses still recorded.
	n, _ := quizgame.CountAnswersForQuestion(ctx, pool, gameID, 0)
	if n != 2 {
		t.Fatalf("expected 2 individual responses, got %d", n)
	}
}

func TestQuizModes_StudentPacedIndependent_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	// Add a second question so players can diverge.
	qbody, _ := json.Marshal(map[string]any{
		"questionType": "mc_single", "prompt": "3+3?", "timeLimitSeconds": 20, "pointsStyle": "standard",
		"options": []map[string]any{
			{"id": "a", "text": "5", "isCorrect": false},
			{"id": "b", "text": "6", "isCorrect": true},
			{"id": "c", "text": "7", "isCorrect": false},
			{"id": "d", "text": "9", "isCorrect": false},
		},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/questions", bytes.NewReader(qbody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("q2: %d %s", w.Code, w.Body.String())
	}

	body, _ := json.Marshal(map[string]any{
		"mode": "student_paced",
		"pacedConfig": map[string]any{
			"shuffle": false, "perQuestionTimers": true, "timeBudgetSeconds": 0,
		},
	})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create paced: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	gameID := created["gameId"].(string)

	st1 := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	st2 := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	join := func(studentTok, nick string) string {
		b, _ := json.Marshal(map[string]any{"nickname": nick})
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost,
			"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/players", bytes.NewReader(b))
		r.Header.Set("Authorization", "Bearer "+studentTok)
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, r)
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		return out["playerId"].(string)
	}
	p1 := join(st1, "PaceA")
	_ = join(st2, "PaceB")

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/games/"+gameID+"/paced/start", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("paced start: %d %s", rr.Code, rr.Body.String())
	}

	ans, _ := json.Marshal(map[string]any{"selectedOptionIds": []string{"b"}})
	_, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: gameID, PlayerID: p1, QuestionIndex: 0, Answer: ans, ReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("p1 answer: %v", err)
	}
	pl1, _ := quizgame.GetPlayer(ctx, pool, p1)
	if pl1.CurrentIndex != 1 {
		t.Fatalf("p1 should advance to index 1, got %d", pl1.CurrentIndex)
	}
	buckets, total, _, err := quizgame.PacedHostProgress(ctx, pool, gameID)
	if err != nil || total != 2 {
		t.Fatalf("progress total=%d err=%v", total, err)
	}
	if len(buckets) < 1 || buckets[0].Reached < 2 {
		t.Fatalf("host aggregate expected both on Q0, got %+v", buckets)
	}
}

func TestQuizModes_HomeworkWindowsAndBestGrade_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	now := time.Now().UTC()
	opens := now.Add(-time.Hour)
	due := now.Add(time.Hour)
	closes := now.Add(2 * time.Hour)

	body, _ := json.Marshal(map[string]any{
		"title":           "HW Kit",
		"opensAt":         opens,
		"dueAt":           due,
		"closesAt":        closes,
		"attemptsAllowed": 2,
		"gradePolicy":     "best",
		"shuffle":         false,
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/assignments", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create assignment: %d %s", w.Code, w.Body.String())
	}
	var a map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &a)
	aid := a["id"].(string)

	// Not-yet-open assignment
	futureOpens := now.Add(24 * time.Hour)
	body2, _ := json.Marshal(map[string]any{
		"title": "Future", "opensAt": futureOpens, "attemptsAllowed": 1, "gradePolicy": "best",
	})
	req = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/assignments", bytes.NewReader(body2))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	futureID := ""
	if w.Code == http.StatusCreated {
		var fa map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &fa)
		futureID, _ = fa["id"].(string)
	}

	stTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	// Parse student user id from JWT is hard; join via start and use API.
	if futureID != "" {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost,
			"/api/v1/courses/"+cc+"/live-quizzes/assignments/"+futureID+"/start",
			bytes.NewReader([]byte(`{"nickname":"Early"}`)))
		r.Header.Set("Authorization", "Bearer "+stTok)
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("not yet open: expected 403, got %d %s", rr.Code, rr.Body.String())
		}
	}

	// Two attempts; second lower → best wins.
	startAttempt := func() (attemptID, sessionID, playerID string) {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost,
			"/api/v1/courses/"+cc+"/live-quizzes/assignments/"+aid+"/start",
			bytes.NewReader([]byte(`{"nickname":"HW"}`)))
		r.Header.Set("Authorization", "Bearer "+stTok)
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusCreated {
			t.Fatalf("start: %d %s", rr.Code, rr.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		return out["attemptId"].(string), out["sessionId"].(string), out["playerId"].(string)
	}

	at1, sess1, p1 := startAttempt()
	ans, _ := json.Marshal(map[string]any{"selectedOptionIds": []string{"b"}})
	res, err := quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: sess1, PlayerID: p1, QuestionIndex: 0, Answer: ans, ReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("answer1: %v", err)
	}
	score1 := res.TotalScore

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/assignments/"+aid+"/attempts/"+at1+"/submit", nil)
	r.Header.Set("Authorization", "Bearer "+stTok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("submit1: %d %s", rr.Code, rr.Body.String())
	}

	at2, sess2, p2 := startAttempt()
	wrong, _ := json.Marshal(map[string]any{"selectedOptionIds": []string{"a"}})
	_, err = quizgame.SubmitAnswer(ctx, pool, quizgame.SubmitAnswerInput{
		SessionID: sess2, PlayerID: p2, QuestionIndex: 0, Answer: wrong, ReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("answer2: %v", err)
	}
	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/assignments/"+aid+"/attempts/"+at2+"/submit", nil)
	r.Header.Set("Authorization", "Bearer "+stTok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("submit2: %d %s", rr.Code, rr.Body.String())
	}
	var sub map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &sub)
	gb, _ := sub["gradebookScore"].(float64)
	if gb != float64(score1) {
		t.Fatalf("best policy: gradebook=%v want %d (first attempt)", gb, score1)
	}
}

func TestQuizModes_HomeworkAccommodation_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := seedReadyKit(t, h, tok, cc)
	now := time.Now().UTC()
	opens := now.Add(-time.Hour)
	due := now.Add(30 * time.Minute)
	closes := now.Add(40 * time.Minute)

	body, _ := json.Marshal(map[string]any{
		"title": "Accom HW", "opensAt": opens, "dueAt": due, "closesAt": closes,
		"attemptsAllowed": 1, "gradePolicy": "best",
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/assignments", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var a map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &a)
	aid := a["id"].(string)

	stTok := enrollExtraQuizPlayer(t, ctx, pool, h, cc, courseID)
	// Resolve student user id from enrollment
	var studentID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT user_id FROM course.course_enrollments
		WHERE course_id = $1 AND role = 'student' ORDER BY created_at DESC LIMIT 1`, courseID).Scan(&studentID)
	if err != nil {
		t.Fatalf("student id: %v", err)
	}
	// Teacher id from course creator for created_by.
	var teacherID uuid.UUID
	_ = pool.QueryRow(ctx, `SELECT created_by_user_id FROM course.courses WHERE id = $1`, courseID).Scan(&teacherID)
	_, err = studentaccommodations.InsertRow(ctx, pool, studentID, &courseID, studentaccommodations.AccommodationWrite{
		TimeMultiplier: 2,
		ExtraAttempts:  1,
	}, teacherID)
	if err != nil {
		t.Fatalf("insert accommodation: %v", err)
	}

	asg, err := quizgame.GetAssignment(ctx, pool, aid)
	if err != nil {
		t.Fatal(err)
	}
	win, allowed, err := quizgame.AssignmentWindowForUser(ctx, pool, asg, studentID, now)
	if err != nil {
		t.Fatal(err)
	}
	if allowed < 2 {
		t.Fatalf("extra attempts: allowed=%d", allowed)
	}
	if win.DueAt == nil || !win.DueAt.After(due) {
		t.Fatalf("extended due expected after %v, got %v", due, win.DueAt)
	}
	_ = stTok
}
