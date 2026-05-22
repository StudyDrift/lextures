package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestStudentProgress_InstructorAndStudentAccess_Pg(t *testing.T) {
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

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	instEmail := "prog-inst-" + time.Now().Format("20060102150405") + "@e.com"
	stuEmail := "prog-stu-" + time.Now().Format("20060102150405") + "@e.com"
	otherEmail := "prog-other-" + time.Now().Format("20060102150405") + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	inst, _ := user.InsertUser(ctx, pool, instEmail, ph, strPtr("Instructor"))
	stu, _ := user.InsertUser(ctx, pool, stuEmail, ph, strPtr("Student"))
	other, _ := user.InsertUser(ctx, pool, otherEmail, ph, strPtr("Other"))
	instID, _ := uuid.Parse(inst.ID)
	stuID, _ := uuid.Parse(stu.ID)
	otherID, _ := uuid.Parse(other.ID)

	var courseID uuid.UUID
	var courseCode string
	err = pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Progress Test', 'prog' || substr(md5(random()::text), 1, 8), (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id, course_code
`, instID).Scan(&courseID, &courseCode)
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	var stuEnrollID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student') RETURNING id
`, courseID, stuID).Scan(&stuEnrollID)
	if err != nil {
		t.Fatalf("enroll student: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'teacher')
`, courseID, instID)
	if err != nil {
		t.Fatalf("enroll teacher: %v", err)
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, instID, courseID, courseCode); err != nil {
		t.Fatalf("grants: %v", err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')
`, courseID, otherID)
	if err != nil {
		t.Fatalf("enroll other: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	instTok, _ := signer.Sign(ctx, inst.ID, instEmail, "", "", nil)
	stuTok, _ := signer.Sign(ctx, stu.ID, stuEmail, "", "", nil)
	otherTok, _ := signer.Sign(ctx, other.ID, otherEmail, "", "", nil)

	d := Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{StudentProgressEnabled: true},
	}
	h := NewHandler(d)

	// Instructor can view student progress
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/enrollments/"+stuEnrollID.String()+"/progress", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("instructor progress: %d %s", rr.Code, rr.Body.String())
	}

	// Student can view own progress
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/enrollments/"+stuEnrollID.String()+"/progress", nil)
	req2 = req2.WithContext(ctx)
	req2.Header.Set("Authorization", "Bearer "+stuTok)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("student own progress: %d %s", rr2.Code, rr2.Body.String())
	}
	var body map[string]any
	_ = json.NewDecoder(rr2.Body).Decode(&body)
	sum, _ := body["summary"].(map[string]any)
	if notes, ok := body["notes"]; ok && notes != nil {
		if arr, ok := notes.([]any); ok && len(arr) > 0 {
			t.Fatal("student should not see instructor notes")
		}
	}
	if sum != nil {
		if _, ok := sum["canManageNotes"]; ok && sum["canManageNotes"] == true {
			t.Fatal("student should not manage notes")
		}
	}

	// Other student forbidden
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/enrollments/"+stuEnrollID.String()+"/progress", nil)
	req3 = req3.WithContext(ctx)
	req3.Header.Set("Authorization", "Bearer "+otherTok)
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusForbidden {
		t.Fatalf("other student: want 403 got %d", rr3.Code)
	}

	// Instructor note CRUD
	noteBody, _ := json.Marshal(map[string]string{"noteText": "Called parent about late work"})
	rr4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+courseCode+"/enrollments/"+stuEnrollID.String()+"/notes",
		bytes.NewReader(noteBody))
	req4 = req4.WithContext(ctx)
	req4.Header.Set("Authorization", "Bearer "+instTok)
	req4.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusCreated {
		t.Fatalf("create note: %d %s", rr4.Code, rr4.Body.String())
	}

}

func strPtr(s string) *string { return &s }
