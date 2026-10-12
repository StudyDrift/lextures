package httpserver

import (
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
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
)

// TestParentHome_HomeschoolParent_Pg covers #772: a standard (non-parent-type) account with a
// managed learner gets a per-child card with due/overdue, completed, grades, and time spent.
func TestParentHome_HomeschoolParent_Pg(t *testing.T) {
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
	em := "ph-par-" + ts + "@e.com"
	dn := "Home Parent"
	parentRow, err := user.InsertUser(ctx, pool, em, ph, &dn)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	emOther := "ph-oth-" + ts + "@e.com"
	otherRow, err := user.InsertUser(ctx, pool, emOther, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}

	dep, err := managedlearners.Create(ctx, pool, managedlearners.CreateParams{ActorID: parentID, DisplayName: "Kid One"})
	if err != nil {
		t.Fatalf("create dependent: %v", err)
	}
	kidID := uuid.MustParse(dep.ID)
	dep2, err := managedlearners.Create(ctx, pool, managedlearners.CreateParams{ActorID: parentID, DisplayName: "Kid Two"})
	if err != nil {
		t.Fatalf("create dependent 2: %v", err)
	}

	var courseID uuid.UUID
	cc := "C-" + strings.ToUpper(uuid.NewString()[:6])
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Home Math', $2) RETURNING id
`, cc, parentID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')`, courseID, kidID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	var modID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published) VALUES ($1, 0, 'module', 'M', NULL, true) RETURNING id`, courseID).Scan(&modID); err != nil {
		t.Fatalf("mod: %v", err)
	}
	now := time.Now().UTC()
	order := 0
	addItem := func(title string, due time.Time, archived bool) uuid.UUID {
		order++
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published, archived, due_at)
VALUES ($1, $6, 'assignment', $2, $3, true, $4, $5) RETURNING id`, courseID, title, modID, archived, due, order).Scan(&id); err != nil {
			t.Fatalf("item %s: %v", title, err)
		}
		return id
	}
	overdueID := addItem("Overdue worksheet", now.Add(-2*time.Hour), false)
	doneID := addItem("Done worksheet", now.Add(48*time.Hour), false)
	addItem("Archived worksheet", now.Add(-2*time.Hour), true)
	if _, err := pool.Exec(ctx, `
INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, submitted_at)
VALUES ($1, $2, $3, $4)`, courseID, doneID, kidID, now.Add(-time.Minute)); err != nil {
		t.Fatalf("submission: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, course_id, event_type, value, occurred_at) VALUES
  ($1, $2, 'heartbeat', NULL, $3), ($1, $2, 'heartbeat', NULL, $3), ($1, $2, 'time_on_task', 60, $3),
  ($1, $2, 'time_on_task', 999, $4)`, kidID, courseID, now.Add(-time.Minute), now.AddDate(0, 0, -14)); err != nil {
		t.Fatalf("events: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})
	get := func(uid, email, path string) *httptest.ResponseRecorder {
		slug, _ := organization.OrgSlugForUser(ctx, pool, uuid.MustParse(uid))
		oid, _ := organization.OrgIDForUser(ctx, pool, uuid.MustParse(uid))
		tok, err := signer.Sign(ctx, uid, email, oid.String(), slug, nil)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
		r.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, r)
		return rr
	}

	if rr := get(otherRow.ID, emOther, "/api/v1/parent/home"); rr.Code != http.StatusForbidden {
		t.Fatalf("unlinked user want 403 got %d %s", rr.Code, rr.Body.String())
	}
	// Existing parent APIs now work for a homeschool parent's managed learners.
	if rr := get(parentRow.ID, em, "/api/v1/parent/children"); rr.Code != http.StatusOK {
		t.Fatalf("children: %d %s", rr.Code, rr.Body.String())
	}

	rr := get(parentRow.ID, em, "/api/v1/parent/home?tz=UTC")
	if rr.Code != http.StatusOK {
		t.Fatalf("home: %d %s", rr.Code, rr.Body.String())
	}
	var out struct {
		Children []parentHomeChild `json:"children"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Children) != 2 {
		t.Fatalf("children = %d", len(out.Children))
	}
	var kid, kid2 *parentHomeChild
	for i := range out.Children {
		switch out.Children[i].StudentUserID {
		case kidID.String():
			kid = &out.Children[i]
		case dep2.ID:
			kid2 = &out.Children[i]
		}
	}
	if kid == nil || kid2 == nil {
		t.Fatalf("missing kids: %+v", out.Children)
	}
	if len(kid2.Courses) != 0 || len(kid2.DueThisWeek) != 0 {
		t.Fatalf("kid2 should be empty: %+v", kid2)
	}
	if kid.DisplayName != "Kid One" || len(kid.Courses) != 1 || kid.Courses[0].CourseCode != cc || kid.Courses[0].EnrollmentID == nil {
		t.Fatalf("courses = %+v", kid.Courses)
	}
	if len(kid.DueThisWeek) != 1 || kid.DueThisWeek[0].ItemID != overdueID.String() || !kid.DueThisWeek[0].Overdue {
		t.Fatalf("due = %+v", kid.DueThisWeek)
	}
	// The completed item is in this week only if "a minute ago" is; at a Monday-midnight boundary skip.
	if ws, _ := parentHomeWeek(now, time.UTC); !now.Add(-time.Minute).Before(ws) {
		if len(kid.CompletedThisWeek) != 1 || kid.CompletedThisWeek[0].ItemID != doneID.String() {
			t.Fatalf("completed = %+v", kid.CompletedThisWeek)
		}
		if kid.TimeSpentSeconds != 120 {
			t.Fatalf("time spent = %d, want 120", kid.TimeSpentSeconds)
		}
	}

	// The linked parent can open the child's progress page.
	pr := get(parentRow.ID, em, "/api/v1/courses/"+cc+"/enrollments/"+*kid.Courses[0].EnrollmentID+"/progress")
	if pr.Code == http.StatusForbidden {
		t.Fatalf("progress forbidden for linked parent: %s", pr.Body.String())
	}
}
