package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// TestCharacterization pins HTTP status, Content-Type, and JSON key sets for a
// curated high-traffic surface (TD.1 FR-5–FR-7). Values (IDs, timestamps) are not asserted.
//
// Requires DATABASE_URL. Regenerate:
//
//	UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run TestCharacterization -count=1
func TestCharacterization(t *testing.T) {
	if testing.Short() {
		t.Skip("characterization needs DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	ph, err := auth.HashPassword("characterization-fixture-password-0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ts := fmt.Sprintf("%s-%d", time.Now().UTC().Format("20060102150405"), time.Now().UnixNano()%1_000_000_000)
	emTeacher := fmt.Sprintf("char-teacher-%s@example.invalid", ts)
	emStudent := fmt.Sprintf("char-student-%s@example.invalid", ts)

	teacherName := "Char Teacher"
	teacherRow, err := user.InsertUser(ctx, pool, emTeacher, ph, &teacherName)
	if err != nil {
		t.Fatalf("insert teacher: %v", err)
	}
	teacherID := uuid.MustParse(teacherRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, teacherID, "Global Admin"); err != nil {
		t.Fatalf("teacher role: %v", err)
	}
	// Course create permission for teachers who are also global admin is via GA.
	// Ensure course:create path works (global:app:course:create comes with GA).

	studentName := "Char Student"
	studentRow, err := user.InsertUser(ctx, pool, emStudent, ph, &studentName)
	if err != nil {
		t.Fatalf("insert student: %v", err)
	}
	studentID := uuid.MustParse(studentRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, studentID, "Student"); err != nil {
		t.Fatalf("student role: %v", err)
	}

	defOrg := organization.SeedDefaultOrgID
	slugTeacher, err := organization.OrgSlugForUser(ctx, pool, teacherID)
	if err != nil {
		t.Fatal(err)
	}
	slugStudent, err := organization.OrgSlugForUser(ctx, pool, studentID)
	if err != nil {
		t.Fatal(err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	teacherTok, err := signer.Sign(ctx, teacherRow.ID, emTeacher, defOrg.String(), slugTeacher, nil)
	if err != nil {
		t.Fatalf("sign teacher: %v", err)
	}
	studentTok, err := signer.Sign(ctx, studentRow.ID, emStudent, defOrg.String(), slugStudent, nil)
	if err != nil {
		t.Fatalf("sign student: %v", err)
	}

	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})

	// --- Auth (error shape is stable; no credentials in golden) ---
	{
		code, ct, body := charDo(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
			"email":    "nobody@example.invalid",
			"password": "wrong-password-not-a-secret",
		})
		assertCharacterizationSnapshot(t, "auth-login-invalid", code, ct, body)
	}

	// --- Platform features ---
	{
		code, ct, body := charDo(t, h, http.MethodGet, "/api/v1/platform/features", teacherTok, nil)
		assertCharacterizationSnapshot(t, "platform-features", code, ct, body)
	}

	// --- Courses list (empty-ish but stable key set) ---
	{
		code, ct, body := charDo(t, h, http.MethodGet, "/api/v1/courses", teacherTok, nil)
		assertCharacterizationSnapshot(t, "courses-list", code, ct, body)
	}

	// Create course via repo for deterministic seed (HTTP create needs permission; GA has it,
	// but repo seed avoids payload churn).
	c, err := course.CreateCourse(ctx, pool, teacherID, "Characterization Course "+ts, "TD.1 fixture", "traditional", nil, nil, nil)
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	courseCode := c.CourseCode

	// --- Course get ---
	{
		code, ct, body := charDo(t, h, http.MethodGet, "/api/v1/courses/"+courseCode, teacherTok, nil)
		assertCharacterizationSnapshot(t, "course-get", code, ct, body)
	}

	// --- Create module (structure) ---
	var moduleID string
	{
		var resp map[string]any
		code := charDoJSON(t, h, http.MethodPost, "/api/v1/courses/"+courseCode+"/structure/modules", teacherTok, map[string]any{
			"title": "Module 1",
		}, &resp)
		if code != http.StatusOK && code != http.StatusCreated {
			// Capture body for debugging.
			_, _, raw := charDo(t, h, http.MethodPost, "/api/v1/courses/"+courseCode+"/structure/modules", teacherTok, map[string]any{"title": "Module 1"})
			t.Fatalf("create module status=%d body=%s", code, truncateBytes(raw, 400))
		}
		moduleID = extractID(resp)
		if moduleID == "" {
			t.Fatalf("module id missing in %v", resp)
		}
		// Snapshot create response shape.
		raw, _ := json.Marshal(resp)
		assertCharacterizationSnapshot(t, "course-module-create", code, "application/json", raw)
	}

	// --- Course structure ---
	{
		code, ct, body := charDo(t, h, http.MethodGet, "/api/v1/courses/"+courseCode+"/structure", teacherTok, nil)
		assertCharacterizationSnapshot(t, "course-structure", code, ct, body)
	}

	// --- Assignment create + get ---
	var assignmentID string
	{
		var resp map[string]any
		code := charDoJSON(t, h, http.MethodPost,
			"/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID+"/assignments",
			teacherTok, map[string]any{"title": "Assignment 1"}, &resp)
		if code != http.StatusOK && code != http.StatusCreated {
			_, _, raw := charDo(t, h, http.MethodPost,
				"/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID+"/assignments",
				teacherTok, map[string]any{"title": "Assignment 1"})
			t.Fatalf("create assignment status=%d body=%s", code, truncateBytes(raw, 400))
		}
		assignmentID = extractID(resp)
		if assignmentID == "" {
			t.Fatalf("assignment id missing in %v", resp)
		}
		raw, _ := json.Marshal(resp)
		assertCharacterizationSnapshot(t, "assignment-create", code, "application/json", raw)
	}
	{
		code, ct, body := charDo(t, h, http.MethodGet,
			"/api/v1/courses/"+courseCode+"/assignments/"+assignmentID, teacherTok, nil)
		assertCharacterizationSnapshot(t, "assignment-get", code, ct, body)
	}

	// --- Quiz create + get ---
	var quizID string
	{
		var resp map[string]any
		code := charDoJSON(t, h, http.MethodPost,
			"/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID+"/quizzes",
			teacherTok, map[string]any{"title": "Quiz 1"}, &resp)
		if code != http.StatusOK && code != http.StatusCreated {
			_, _, raw := charDo(t, h, http.MethodPost,
				"/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID+"/quizzes",
				teacherTok, map[string]any{"title": "Quiz 1"})
			t.Fatalf("create quiz status=%d body=%s", code, truncateBytes(raw, 400))
		}
		quizID = extractID(resp)
		if quizID == "" {
			t.Fatalf("quiz id missing in %v", resp)
		}
		raw, _ := json.Marshal(resp)
		assertCharacterizationSnapshot(t, "quiz-create", code, "application/json", raw)
	}
	{
		code, ct, body := charDo(t, h, http.MethodGet,
			"/api/v1/courses/"+courseCode+"/quizzes/"+quizID, teacherTok, nil)
		assertCharacterizationSnapshot(t, "quiz-get", code, ct, body)
	}

	// Publish module + children so learners can take the quiz.
	{
		code, _, body := charDo(t, h, http.MethodPatch,
			"/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID, teacherTok, map[string]any{
				"title":     "Module 1",
				"published": true,
			})
		if code != http.StatusOK {
			t.Fatalf("publish module: status=%d body=%s", code, truncateBytes(body, 300))
		}
		for _, itemID := range []string{quizID, assignmentID} {
			code, _, body := charDo(t, h, http.MethodPatch,
				"/api/v1/courses/"+courseCode+"/structure/items/"+itemID, teacherTok, map[string]any{
					"published": true,
				})
			if code != http.StatusOK {
				t.Fatalf("publish item %s: status=%d body=%s", itemID, code, truncateBytes(body, 300))
			}
		}
	}

	// --- Enroll student + list enrollments ---
	{
		code, ct, body := charDo(t, h, http.MethodPost,
			"/api/v1/courses/"+courseCode+"/enrollments", teacherTok, map[string]any{
				"emails":     emStudent,
				"courseRole": "student",
			})
		if code != http.StatusOK && code != http.StatusCreated {
			t.Fatalf("enroll student: status=%d body=%s", code, truncateBytes(body, 400))
		}
		assertCharacterizationSnapshot(t, "enrollment-create", code, ct, body)
		// Staff-added student enrollments are invitation_pending/inactive until accepted.
		// Activate for deterministic take/submit characterization (synthetic fixture only).
		if _, err := pool.Exec(ctx, `
UPDATE course.course_enrollments ce
SET active = true, invitation_pending = false
FROM course.courses c
WHERE ce.course_id = c.id AND c.course_code = $1 AND ce.user_id = $2
`, courseCode, studentID); err != nil {
			t.Fatalf("activate enrollment: %v", err)
		}
	}
	{
		code, ct, body := charDo(t, h, http.MethodGet,
			"/api/v1/courses/"+courseCode+"/enrollments", teacherTok, nil)
		assertCharacterizationSnapshot(t, "enrollments-list", code, ct, body)
	}

	// --- Gradebook grid ---
	{
		code, ct, body := charDo(t, h, http.MethodGet,
			"/api/v1/courses/"+courseCode+"/gradebook/grid", teacherTok, nil)
		assertCharacterizationSnapshot(t, "gradebook-grid", code, ct, body)
	}

	// --- Quiz attempts list (teacher staff view; empty list still has stable keys) ---
	{
		code, ct, body := charDo(t, h, http.MethodGet,
			"/api/v1/courses/"+courseCode+"/quizzes/"+quizID+"/attempts", teacherTok, nil)
		if code != http.StatusOK {
			t.Fatalf("quiz attempts list: status=%d body=%s", code, truncateBytes(body, 300))
		}
		assertCharacterizationSnapshot(t, "quiz-attempts-list", code, ct, body)
	}

	// --- Quiz start (take) as enrolled student ---
	var attemptID string
	{
		code, ct, body := charDo(t, h, http.MethodPost,
			"/api/v1/courses/"+courseCode+"/quizzes/"+quizID+"/start", studentTok, map[string]any{})
		assertCharacterizationSnapshot(t, "quiz-start", code, ct, body)
		if code != http.StatusOK {
			t.Fatalf("quiz start: status=%d body=%s", code, truncateBytes(body, 300))
		}
		var resp map[string]any
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("quiz start json: %v", err)
		}
		attemptID, _ = resp["attemptId"].(string)
		if attemptID == "" {
			t.Fatalf("quiz start missing attemptId: %s", truncateBytes(body, 300))
		}
	}

	// --- Quiz submit (no answers; empty quiz still yields a stable score shape) ---
	{
		code, ct, body := charDo(t, h, http.MethodPost,
			"/api/v1/courses/"+courseCode+"/quizzes/"+quizID+"/submit", studentTok, map[string]any{
				"attemptId": attemptID,
				"responses": []any{},
			})
		assertCharacterizationSnapshot(t, "quiz-submit", code, ct, body)
	}

	// --- Auth posture elevated sample: student cannot list enrollments as staff-only ops vary;
	// platform features is session for both. Admin-only: health/detailed with student vs GA. ---
	{
		code, ct, body := charDo(t, h, http.MethodGet, "/health/detailed", teacherTok, nil)
		assertCharacterizationSnapshot(t, "health-detailed-admin", code, ct, body)
	}

	if elapsed := time.Since(start); elapsed > 60*time.Second {
		t.Fatalf("characterization suite took %s (> 60s budget)", elapsed)
	}
	t.Logf("characterization OK in %s", time.Since(start).Round(time.Millisecond))
}

// extractID pulls a UUID-like id from common response shapes (id, itemId, item.id, module.id).
func extractID(resp map[string]any) string {
	for _, k := range []string{"id", "itemId", "item_id", "moduleId", "module_id"} {
		if v, ok := resp[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	for _, nest := range []string{"item", "module", "assignment", "quiz", "data"} {
		if m, ok := resp[nest].(map[string]any); ok {
			if id := extractID(m); id != "" {
				return id
			}
		}
	}
	return ""
}
