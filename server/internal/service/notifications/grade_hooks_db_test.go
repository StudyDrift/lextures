package notifications_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// A managed learner (undeliverable synthetic email) must still get an in-app grade_posted
// notification even when email and SMS notifications are disabled, without stacking duplicates.
func TestNotifyGradesPostedAfterRelease_InAppForManagedLearner_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	suffix := time.Now().Format("20060102150405.000000")
	creator, err := user.InsertUser(ctx, pool, "gp-creator-"+suffix+"@e.com", "x", nil)
	if err != nil {
		t.Fatalf("creator: %v", err)
	}
	creatorID, _ := uuid.Parse(creator.ID)
	kidID := uuid.New()
	kidName := "QA Kid"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name, org_id)
VALUES ($1, $2, 'x', $3, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1))
`, kidID, user.ManagedEmail(kidID), kidName); err != nil {
		t.Fatalf("managed user: %v", err)
	}

	courseCode := "C-" + fmt.Sprintf("%06X", time.Now().UnixNano()%0xFFFFFF)
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Grade Notify Test', $2, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id
`, creatorID, courseCode).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	var itemID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 0, 'assignment', 'QA Assignment', NULL, TRUE)
RETURNING id
`, courseID).Scan(&itemID); err != nil {
		t.Fatalf("item: %v", err)
	}

	cells := []coursegrades.PostedCell{{StudentUserID: kidID, PointsEarned: 8}}
	cfg := config.Config{} // email + SMS notifications disabled
	notify := func() {
		notifications.NotifyGradesPostedAfterRelease(ctx, pool, cfg, courseID, itemID, cells, nil, nil)
	}

	notify()
	rows, err := notificationsinbox.List(ctx, pool, kidID, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 notification, got %d", len(rows))
	}
	if rows[0].EventType != notifications.EventGradePosted ||
		rows[0].Title != "Grade posted for QA Assignment" ||
		rows[0].ActionURL != "/courses/"+courseCode+"/grades" {
		t.Fatalf("unexpected notification: %+v", rows[0])
	}

	// An identical unread notification is not duplicated by a re-save.
	notify()
	if n, _ := notificationsinbox.UnreadCount(ctx, pool, kidID); n != 1 {
		t.Fatalf("want 1 unread after repeat, got %d", n)
	}

	// Once read, a later grade update notifies again.
	if err := notificationsinbox.MarkRead(ctx, pool, rows[0].ID, kidID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	notify()
	if n, _ := notificationsinbox.UnreadCount(ctx, pool, kidID); n != 1 {
		t.Fatalf("want 1 unread after read+notify, got %d", n)
	}
}
