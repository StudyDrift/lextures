package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestCourseAttendance_RollCallAndGradebook_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	teacherEmail := fmt.Sprintf("att-t-%d@test.invalid", time.Now().UnixNano())
	studentEmail := fmt.Sprintf("att-s-%d@test.invalid", time.Now().UnixNano())
	ph, err := auth.HashPassword("longpassword0longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	teacherRow, err := user.InsertUser(ctx, pool, teacherEmail, ph, nil)
	if err != nil {
		t.Fatalf("teacher: %v", err)
	}
	studentRow, err := user.InsertUser(ctx, pool, studentEmail, ph, nil)
	if err != nil {
		t.Fatalf("student: %v", err)
	}
	teacherID, _ := uuid.Parse(teacherRow.ID)
	studentID, _ := uuid.Parse(studentRow.ID)

	cc := fmt.Sprintf("C-ATT%05d", time.Now().UnixNano()%100000)
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id, attendance_enabled)
VALUES ($1, 'Attendance Test', $2, true) RETURNING id
`, cc, teacherID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES
  ($1, $2, 'teacher'),
  ($1, $3, 'student')
`, courseID, teacherID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, teacherID, courseID, cc); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("grants teacher: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	teacherTok, err := signer.Sign(ctx, teacherRow.ID, teacherEmail, "", "", nil)
	if err != nil {
		t.Fatalf("sign teacher: %v", err)
	}
	studentTok, err := signer.Sign(ctx, studentRow.ID, studentEmail, "", "", nil)
	if err != nil {
		t.Fatalf("sign student: %v", err)
	}
	d := Deps{Pool: pool, JWTSigner: signer, Config: config.Config{}}
	h := NewHandler(d)

	createBody, _ := json.Marshal(map[string]any{
		"collectionMethod": "roll_call",
		"title":            "Week 1",
		"sessionDate":      time.Now().UTC().Format("2006-01-02"),
		"gradebookEnabled": true,
		"pointsPossible":   10,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/attendance/sessions", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create session: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	sessionID, _ := created["id"].(string)

	saveBody, _ := json.Marshal(map[string]any{
		"records": []map[string]any{
			{"studentUserId": studentID.String(), "status": "present"},
		},
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/courses/"+cc+"/attendance/sessions/"+sessionID+"/records", bytes.NewReader(saveBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("save records: %d %s", w.Code, w.Body.String())
	}

	closeBody, _ := json.Marshal(map[string]any{"finalizeMissingAsAbsent": true})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/attendance/sessions/"+sessionID+"/close", bytes.NewReader(closeBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("close session: %d %s", w.Code, w.Body.String())
	}
	var closed map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &closed); err != nil {
		t.Fatalf("decode close: %v", err)
	}
	structItemID, _ := closed["structureItemId"].(string)
	if structItemID == "" {
		t.Fatal("expected structureItemId after close with gradebook")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/gradebook/grid", nil)
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("gradebook grid: %d %s", w.Code, w.Body.String())
	}
	var grid map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &grid); err != nil {
		t.Fatalf("decode grid: %v", err)
	}
	cols, _ := grid["columns"].([]any)
	foundCol := false
	for _, c := range cols {
		m, _ := c.(map[string]any)
		if m["id"] == structItemID && m["kind"] == "attendance" {
			foundCol = true
		}
	}
	if !foundCol {
		t.Fatalf("attendance column not in gradebook: %+v", cols)
	}
	grades, _ := grid["grades"].(map[string]any)
	stuGrades, _ := grades[studentID.String()].(map[string]any)
	if stuGrades[structItemID] != "10" && stuGrades[structItemID] != "10.0" && stuGrades[structItemID] != "10.00" {
		t.Fatalf("expected 10 points, got %v", stuGrades[structItemID])
	}

	// Feature off returns 404
	if _, err := pool.Exec(ctx, `UPDATE course.courses SET attendance_enabled = false WHERE id = $1`, courseID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/attendance/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+studentTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("feature off: want 404 got %d", w.Code)
	}
}

func TestCourseAttendance_SelfReport_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	teacherEmail := fmt.Sprintf("att2-t-%d@test.invalid", time.Now().UnixNano())
	studentEmail := fmt.Sprintf("att2-s-%d@test.invalid", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	teacherRow, _ := user.InsertUser(ctx, pool, teacherEmail, ph, nil)
	studentRow, _ := user.InsertUser(ctx, pool, studentEmail, ph, nil)
	teacherID, _ := uuid.Parse(teacherRow.ID)
	studentID, _ := uuid.Parse(studentRow.ID)
	cc := fmt.Sprintf("C-AT2%05d", time.Now().UnixNano()%100000)
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id, attendance_enabled)
VALUES ($1, 'Self Report', $2, true) RETURNING id
`, cc, teacherID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1,$2,'teacher'),($1,$3,'student')
`, courseID, teacherID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	tx, _ := pool.Begin(ctx)
	_ = courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, teacherID, courseID, cc)
	_ = tx.Commit(ctx)

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	teacherTok, _ := signer.Sign(ctx, teacherRow.ID, teacherEmail, "", "", nil)
	studentTok, _ := signer.Sign(ctx, studentRow.ID, studentEmail, "", "", nil)
	d := Deps{Pool: pool, JWTSigner: signer, Config: config.Config{}}
	h := NewHandler(d)

	now := time.Now().UTC()
	createBody, _ := json.Marshal(map[string]any{
		"collectionMethod": "self_report",
		"sessionDate":      now.Format("2006-01-02"),
		"opensAt":          now.Format(time.RFC3339),
		"closesAt":         now.Add(10 * time.Minute).Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/attendance/sessions", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+teacherTok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	sessionID, _ := created["id"].(string)

	selfBody, _ := json.Marshal(map[string]any{"status": "present"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/attendance/sessions/"+sessionID+"/self-report", bytes.NewReader(selfBody))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("self report: %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/attendance/sessions/"+sessionID+"/self-report", bytes.NewReader(selfBody))
	req.Header.Set("Authorization", "Bearer "+studentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate self report: want 409 got %d", w.Code)
	}
}
