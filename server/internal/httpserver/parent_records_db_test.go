package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
)

func TestParentHomeschoolRecords_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().Format("20060102150405.000000")
	em := "hr-par-" + ts + "@e.com"
	dn := "Records Parent"
	parentRow, err := user.InsertUser(ctx, pool, em, ph, &dn)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	emOther := "hr-oth-" + ts + "@e.com"
	otherRow, err := user.InsertUser(ctx, pool, emOther, ph, nil)
	if err != nil {
		t.Fatalf("other: %v", err)
	}

	dep, err := managedlearners.Create(ctx, pool, managedlearners.CreateParams{ActorID: parentID, DisplayName: "Records Kid"})
	if err != nil {
		t.Fatalf("dependent: %v", err)
	}
	kidID := uuid.MustParse(dep.ID)

	var courseID uuid.UUID
	cc := "C-" + strings.ToUpper(uuid.NewString()[:6])
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Homeschool Math', $2) RETURNING id
`, cc, parentID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')`, courseID, kidID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	var modID, itemID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published) VALUES ($1, 0, 'module', 'M', NULL, true) RETURNING id`, courseID).Scan(&modID); err != nil {
		t.Fatalf("mod: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 1, 'assignment', 'Essay', $2, true) RETURNING id`, courseID, modID).Scan(&itemID); err != nil {
		t.Fatalf("item: %v", err)
	}
	now := time.Now().UTC()
	var subID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, body_text, submitted_at)
VALUES ($1, $2, $3, 'My essay', $4) RETURNING id`, courseID, itemID, kidID, now).Scan(&subID); err != nil {
		t.Fatalf("sub: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_grades (course_id, student_user_id, module_item_id, points_earned, instructor_comment)
VALUES ($1, $2, $3, 9, 'Great work')`, courseID, kidID, itemID); err != nil {
		t.Fatalf("grade: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, course_id, event_type, value, occurred_at) VALUES
  ($1, $2, 'time_on_task', 3600, $3), ($1, $2, 'heartbeat', NULL, $3)`, kidID, courseID, now); err != nil {
		t.Fatalf("events: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFHomeschoolManagedLearners: true}})
	do := func(uid, email, method, path string, body any) *httptest.ResponseRecorder {
		slug, _ := organization.OrgSlugForUser(ctx, pool, uuid.MustParse(uid))
		oid, _ := organization.OrgIDForUser(ctx, pool, uuid.MustParse(uid))
		tok, err := signer.Sign(ctx, uid, email, oid.String(), slug, nil)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		var r *http.Request
		if body != nil {
			b, _ := json.Marshal(body)
			r = httptest.NewRequest(method, path, bytes.NewReader(b)).WithContext(ctx)
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, path, nil).WithContext(ctx)
		}
		r.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, r)
		return rr
	}

	// Unlinked user cannot access.
	if rr := do(otherRow.ID, emOther, http.MethodGet, "/api/v1/parent/students/"+kidID.String()+"/records/hours", nil); rr.Code != http.StatusForbidden {
		t.Fatalf("unlinked hours: %d %s", rr.Code, rr.Body.String())
	}

	// Settings: set family school name.
	rr := do(parentRow.ID, em, http.MethodPut, "/api/v1/parent/records/settings", map[string]any{
		"schoolName": "Willden Homeschool", "schoolYearStartMonth": 8, "schoolYearStartDay": 1,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("settings put: %d %s", rr.Code, rr.Body.String())
	}

	// Add offline hours.
	rr = do(parentRow.ID, em, http.MethodPost, "/api/v1/parent/students/"+kidID.String()+"/records/hours", map[string]any{
		"date": now.Format("2006-01-02"), "subject": "Field Trip", "hours": 2.5, "notes": "Zoo", "source": "offline",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create hours: %d %s", rr.Code, rr.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil || created.ID == "" {
		t.Fatalf("decode created: %v %s", err, rr.Body.String())
	}

	// Hours log includes activity + offline.
	rr = do(parentRow.ID, em, http.MethodGet, "/api/v1/parent/students/"+kidID.String()+"/records/hours", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("hours log: %d %s", rr.Code, rr.Body.String())
	}
	var log struct {
		TotalHours      float64 `json:"totalHours"`
		Entries         []map[string]any `json:"entries"`
		TotalsBySubject []map[string]any `json:"totalsBySubject"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&log); err != nil {
		t.Fatal(err)
	}
	if log.TotalHours < 3.5 { // 1h activity + 30s heartbeat + 2.5 offline ≈ 3.508 → rounded
		t.Fatalf("totalHours = %v entries=%v", log.TotalHours, log.Entries)
	}
	if len(log.TotalsBySubject) < 2 {
		t.Fatalf("totals = %+v", log.TotalsBySubject)
	}

	// Edit + delete offline entry.
	rr = do(parentRow.ID, em, http.MethodPatch, "/api/v1/parent/students/"+kidID.String()+"/records/hours/"+created.ID, map[string]any{
		"hours": 3.0, "notes": "Zoo (longer)",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rr.Code, rr.Body.String())
	}

	// CSV + PDF exports.
	rr = do(parentRow.ID, em, http.MethodGet, "/api/v1/parent/students/"+kidID.String()+"/records/hours/export.csv", nil)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Field Trip") {
		t.Fatalf("csv: %d %s", rr.Code, rr.Body.String())
	}
	rr = do(parentRow.ID, em, http.MethodGet, "/api/v1/parent/students/"+kidID.String()+"/records/hours/export.pdf", nil)
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Body.String(), "%PDF") {
		t.Fatalf("hours pdf: %d", rr.Code)
	}

	// Transcript PDF.
	rr = do(parentRow.ID, em, http.MethodGet, "/api/v1/parent/students/"+kidID.String()+"/records/transcript.pdf", nil)
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Body.String(), "%PDF") {
		t.Fatalf("transcript: %d %s", rr.Code, rr.Body.String())
	}

	// Submissions list + portfolio.
	rr = do(parentRow.ID, em, http.MethodGet, "/api/v1/parent/students/"+kidID.String()+"/records/submissions", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("submissions: %d %s", rr.Code, rr.Body.String())
	}
	var subs struct {
		Submissions []struct {
			SubmissionID      string `json:"submissionId"`
			InstructorComment string `json:"instructorComment"`
		} `json:"submissions"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&subs); err != nil {
		t.Fatal(err)
	}
	if len(subs.Submissions) != 1 || subs.Submissions[0].InstructorComment != "Great work" {
		t.Fatalf("subs = %+v", subs.Submissions)
	}
	rr = do(parentRow.ID, em, http.MethodPost, "/api/v1/parent/students/"+kidID.String()+"/records/portfolio", map[string]any{
		"submissionIds": []string{subID.String()}, "format": "pdf",
	})
	if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Body.String(), "%PDF") {
		t.Fatalf("portfolio pdf: %d", rr.Code)
	}
	rr = do(parentRow.ID, em, http.MethodPost, "/api/v1/parent/students/"+kidID.String()+"/records/portfolio", map[string]any{
		"submissionIds": []string{subID.String()}, "format": "zip",
	})
	if rr.Code != http.StatusOK || rr.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("portfolio zip: %d %s", rr.Code, rr.Header().Get("Content-Type"))
	}

	// Managed learner token cannot reach parent records (guard + requireParentViewer).
	mlTok, _, err := signer.SignManagedLearnerVia(parentID.String(), kidID.String(), kidID.String()+"@managed.lextures.invalid", "", "", auth.ManagedLearnerViaFamilyDevice)
	if err != nil {
		t.Fatalf("sign ml: %v", err)
	}
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/students/"+kidID.String()+"/records/hours", bytes.NewReader([]byte(`{"date":"2026-01-01","subject":"X","hours":1}`))).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+mlTok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("managed learner write want 403 got %d %s", rr.Code, rr.Body.String())
	}

	rr = do(parentRow.ID, em, http.MethodDelete, "/api/v1/parent/students/"+kidID.String()+"/records/hours/"+created.ID, nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rr.Code, rr.Body.String())
	}
}

