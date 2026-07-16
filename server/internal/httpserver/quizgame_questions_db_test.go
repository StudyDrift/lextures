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
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func createKitViaAPI(t *testing.T, h http.Handler, tok, cc string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"title": "Authoring Kit", "description": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create kit: %d %s", w.Code, w.Body.String())
	}
	var kit map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &kit)
	id, _ := kit["id"].(string)
	if id == "" {
		t.Fatal("missing kit id")
	}
	return id
}

func TestQuizQuestions_CRUD_Reorder_Validate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	kitID := createKitViaAPI(t, h, tok, cc)
	base := "/api/v1/courses/" + cc + "/live-quizzes/kits/" + kitID + "/questions"

	// Create MC question
	createBody, _ := json.Marshal(map[string]any{
		"questionType":     "mc_single",
		"prompt":           "Capital of France?",
		"timeLimitSeconds": 15,
		"pointsStyle":      "standard",
		"options": []map[string]any{
			{"id": "a", "text": "Paris", "isCorrect": true},
			{"id": "b", "text": "London", "isCorrect": false},
			{"id": "c", "text": "Berlin", "isCorrect": false},
			{"id": "d", "text": "Madrid", "isCorrect": false},
		},
	})
	req := httptest.NewRequest(http.MethodPost, base, bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create question: %d %s", w.Code, w.Body.String())
	}
	var q1 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &q1)
	q1ID, _ := q1["id"].(string)
	ver := int(q1["version"].(float64))
	if q1ID == "" || ver != 1 {
		t.Fatalf("bad create response: %v", q1)
	}

	// Create poll (no correct)
	pollBody, _ := json.Marshal(map[string]any{
		"questionType": "poll",
		"prompt":       "Favorite color?",
		"options": []map[string]any{
			{"id": "a", "text": "Blue", "isCorrect": false},
			{"id": "b", "text": "Red", "isCorrect": false},
		},
	})
	req = httptest.NewRequest(http.MethodPost, base, bytes.NewReader(pollBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create poll: %d %s", w.Code, w.Body.String())
	}
	var q2 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &q2)
	q2ID, _ := q2["id"].(string)

	// Add three more for reorder
	ids := []string{q1ID, q2ID}
	for i := 0; i < 3; i++ {
		b, _ := json.Marshal(map[string]any{
			"questionType": "true_false",
			"prompt":       fmt.Sprintf("TF %d", i),
		})
		req = httptest.NewRequest(http.MethodPost, base, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create tf: %d %s", w.Code, w.Body.String())
		}
		var q map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &q)
		ids = append(ids, q["id"].(string))
	}

	// Reorder: move last before second (simulate drag)
	// New order: ids[0], ids[4], ids[1], ids[2], ids[3]
	reorderItems := []map[string]any{
		{"id": ids[0], "position": 0},
		{"id": ids[4], "position": 1},
		{"id": ids[1], "position": 2},
		{"id": ids[2], "position": 3},
		{"id": ids[3], "position": 4},
	}
	rb, _ := json.Marshal(map[string]any{"items": reorderItems})
	req = httptest.NewRequest(http.MethodPost, base+"/reorder", bytes.NewReader(rb))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("reorder: %d %s", w.Code, w.Body.String())
	}
	var listed map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &listed)
	qs := listed["questions"].([]any)
	if len(qs) != 5 {
		t.Fatalf("want 5 got %d", len(qs))
	}
	if qs[1].(map[string]any)["id"] != ids[4] {
		t.Fatalf("reorder not persisted: %+v", qs[1])
	}

	// List reload
	req = httptest.NewRequest(http.MethodGet, base, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}

	// Patch with version
	patchBody, _ := json.Marshal(map[string]any{"prompt": "Capital of France (updated)?"})
	req = httptest.NewRequest(http.MethodPatch, base+"/"+q1ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	var patched map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &patched)
	if int(patched["version"].(float64)) != 2 {
		t.Fatalf("version=%v", patched["version"])
	}

	// Conflict
	req = httptest.NewRequest(http.MethodPatch, base+"/"+q1ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("want 409 got %d %s", w.Code, w.Body.String())
	}

	// Validate — poll ok; MC with correct ok; but TF defaults may need fix
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("validate: %d %s", w.Code, w.Body.String())
	}
	var vr map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &vr)
	if vr["isReady"] != true {
		t.Fatalf("expected ready kit, got %+v", vr)
	}

	// question_count sync
	var count int
	if err := pool.QueryRow(ctx, `SELECT question_count FROM quizgame.kits WHERE id = $1`, kitID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("question_count=%d", count)
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, base+"/"+q2ID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	if err := pool.QueryRow(ctx, `SELECT question_count FROM quizgame.kits WHERE id = $1`, kitID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("after delete question_count=%d", count)
	}
}

func TestQuizQuestions_BankImport_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	if _, err := pool.Exec(ctx, `UPDATE course.courses SET question_bank_enabled = TRUE WHERE id = $1`, courseID); err != nil {
		t.Fatal(err)
	}

	var bankIDs []string
	for i := 0; i < 3; i++ {
		var id uuid.UUID
		opts, _ := json.Marshal([]string{"A", "B", "C", "D"})
		corr, _ := json.Marshal(map[string]any{"correctChoiceIndex": 0})
		err := pool.QueryRow(ctx, `
			INSERT INTO course.questions (course_id, question_type, stem, options, correct_answer, status, is_published)
			VALUES ($1, 'mc_single', $2, $3, $4, 'active', TRUE)
			RETURNING id
		`, courseID, fmt.Sprintf("Bank Q %d", i), opts, corr).Scan(&id)
		if err != nil {
			t.Fatalf("insert bank: %v", err)
		}
		bankIDs = append(bankIDs, id.String())
	}

	kitID := createKitViaAPI(t, h, tok, cc)
	body, _ := json.Marshal(map[string]any{"questionIds": bankIDs})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/questions/import-bank", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("import: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	qs := resp["questions"].([]any)
	if len(qs) != 3 {
		t.Fatalf("imported %d", len(qs))
	}
	for _, raw := range qs {
		q := raw.(map[string]any)
		if q["sourceQuestionId"] == nil {
			t.Fatalf("missing sourceQuestionId: %+v", q)
		}
	}
}

func TestQuizQuestions_StudentForbidden_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tokTeacher, cc, courseID := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	kitID := createKitViaAPI(t, h, tokTeacher, cc)

	em := fmt.Sprintf("quizq-student-%d@test.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	studentUID, _ := uuid.Parse(row.ID)
	if _, err := pool.Exec(ctx,
		`INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')`,
		courseID, studentUID,
	); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, studentUID, courseID, cc); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tokStudent, _ := signer.Sign(ctx, row.ID, em, "", "", nil)

	body, _ := json.Marshal(map[string]any{"questionType": "mc_single", "prompt": "x"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/questions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokStudent)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", w.Code, w.Body.String())
	}
}
