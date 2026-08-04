package coursechecklist

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestDismissRestoreIdempotentAndConstraints_Pg(t *testing.T) {
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

	ph, _ := auth.HashPassword("longpassword0")
	email := fmt.Sprintf("cc2-repo-%d@e.com", time.Now().UnixNano())
	u, err := user.InsertUser(ctx, pool, email, ph, strPtr("Teacher"))
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(u.ID)
	var courseID uuid.UUID
	code := fmt.Sprintf("C-%06X", time.Now().UnixNano()%0xFFFFFF)
	err = pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Repo Checklist', $2, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id
`, uid, code).Scan(&courseID)
	if err != nil {
		t.Fatalf("course: %v", err)
	}

	st, changed, err := Dismiss(ctx, pool, DismissInput{
		CourseID: courseID, ItemID: "course.dates", ActorID: uid,
		Reason: "not_applicable", Note: "first",
	})
	if err != nil || !changed || st.DismissedAt == nil {
		t.Fatalf("dismiss1: changed=%v err=%v st=%+v", changed, err, st)
	}
	firstAt := *st.DismissedAt
	st2, changed2, err := Dismiss(ctx, pool, DismissInput{
		CourseID: courseID, ItemID: "course.dates", ActorID: uid,
		Reason: "other", Note: "second",
	})
	if err != nil || changed2 {
		t.Fatalf("idempotent dismiss should not change: changed=%v err=%v", changed2, err)
	}
	if st2.DismissedAt == nil || !st2.DismissedAt.Equal(firstAt) {
		t.Fatalf("dismissed_at mutated: %v vs %v", st2.DismissedAt, firstAt)
	}

	_, _, err = Dismiss(ctx, pool, DismissInput{
		CourseID: courseID, ItemID: "course.dates", ActorID: uid,
		Reason: "nope", Note: "x",
	})
	if err != ErrInvalidReason {
		t.Fatalf("want ErrInvalidReason got %v", err)
	}
	long := make([]rune, MaxDismissNoteLen+1)
	for i := range long {
		long[i] = 'a'
	}
	_, _, err = Dismiss(ctx, pool, DismissInput{
		CourseID: courseID, ItemID: "course.dates", ActorID: uid,
		Reason: "other", Note: string(long),
	})
	if err != ErrNoteTooLong {
		t.Fatalf("want ErrNoteTooLong got %v", err)
	}

	_, changed, err = Restore(ctx, pool, courseID, "course.dates", uid)
	if err != nil || !changed {
		t.Fatalf("restore: %v changed=%v", err, changed)
	}
	events, err := ListEvents(ctx, pool, courseID, 10)
	if err != nil || len(events) < 2 {
		t.Fatalf("events: %v len=%d", err, len(events))
	}
}

func strPtr(s string) *string { return &s }
