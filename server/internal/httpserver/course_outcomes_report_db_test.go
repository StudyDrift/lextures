package httpserver

import (
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
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	"github.com/lextures/lextures/server/internal/repos/outcomesreport"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestOutcomesReport_InstructorAccessAndPctMet_Pg(t *testing.T) {
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
	instEmail := "out-inst-" + time.Now().Format("20060102150405") + "@e.com"
	stuEmail := "out-stu-" + time.Now().Format("20060102150405") + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	inst, _ := user.InsertUser(ctx, pool, instEmail, ph, strPtr("Instructor"))
	stu, _ := user.InsertUser(ctx, pool, stuEmail, ph, strPtr("Student"))
	instID, _ := uuid.Parse(inst.ID)
	stuID, _ := uuid.Parse(stu.ID)

	courseCode := "C-" + fmt.Sprintf("%06X", time.Now().UnixNano()%0xFFFFFF)
	var courseID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Outcomes Report Test', $2, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id, course_code
`, instID, courseCode).Scan(&courseID, &courseCode)
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	for _, pair := range []struct {
		uid  uuid.UUID
		role string
	}{
		{instID, "teacher"},
		{stuID, "student"},
	} {
		_, err = tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`, courseID, pair.uid, pair.role)
		if err != nil {
			t.Fatalf("enroll: %v", err)
		}
		if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, pair.uid, courseID, courseCode); err != nil {
			t.Fatalf("grants: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var moduleID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, kind, title, sort_order, published, archived)
VALUES ($1, 'module', 'M1', 0, true, false) RETURNING id
`, courseID).Scan(&moduleID)
	if err != nil {
		t.Fatalf("module: %v", err)
	}

	var itemID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, parent_id, kind, title, sort_order, published, archived)
VALUES ($1, $2, 'assignment', 'A1', 0, true, false) RETURNING id
`, courseID, moduleID).Scan(&itemID)
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO course.module_assignments (structure_item_id, points_worth) VALUES ($1, 100)
`, itemID)
	if err != nil {
		t.Fatalf("assignment: %v", err)
	}

	outcome, err := courseoutcomes.InsertOutcome(ctx, pool, courseID, "CLO 1", "")
	if err != nil {
		t.Fatalf("outcome: %v", err)
	}
	_, err = courseoutcomes.InsertLink(ctx, pool, outcome.ID, nil, itemID, "assignment", "", "summative", "medium")
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO course.course_grades (course_id, module_item_id, student_user_id, points_earned)
VALUES ($1, $2, $3, 85)
`, courseID, itemID, stuID)
	if err != nil {
		t.Fatalf("grade: %v", err)
	}
	if err := outcomesreport.RefreshCourseNow(ctx, pool, courseID); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	instTok, _ := signer.Sign(ctx, inst.ID, instEmail, "", "", nil)
	stuTok, _ := signer.Sign(ctx, stu.ID, stuEmail, "", "", nil)

	d := Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{OutcomesReportEnabled: true},
	}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/analytics/outcomes", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET report: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Outcomes []struct {
			Title   string  `json:"title"`
			NAssessed int   `json:"nAssessed"`
			PctMet  float64 `json:"pctMet"`
		} `json:"outcomes"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Outcomes) != 1 || body.Outcomes[0].NAssessed != 1 || body.Outcomes[0].PctMet != 100 {
		t.Fatalf("unexpected body: %+v", body)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/analytics/outcomes", nil)
	req2 = req2.WithContext(ctx)
	req2.Header.Set("Authorization", "Bearer "+stuTok)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("student expected 403, got %d", w2.Code)
	}

	note, err := outcomesreport.GetImprovementNote(ctx, pool, outcome.ID)
	if err != nil || note != "" {
		t.Fatalf("expected empty note initially, got %q err=%v", note, err)
	}
	if err := outcomesreport.UpsertImprovementNote(ctx, pool, outcome.ID, "Improve rubric clarity."); err != nil {
		t.Fatalf("note: %v", err)
	}
	note, err = outcomesreport.GetImprovementNote(ctx, pool, outcome.ID)
	if err != nil || note != "Improve rubric clarity." {
		t.Fatalf("note persist: %q err=%v", note, err)
	}
}
