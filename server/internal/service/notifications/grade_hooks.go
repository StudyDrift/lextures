package notifications

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
)

// GradePostedInAppContent builds the in-app notification title, body, and action URL for a posted grade.
func GradePostedInAppContent(courseTitle, assignmentTitle, courseCode string) (title, body, actionURL string) {
	title = "Grade posted for " + assignmentTitle
	body = "Your grade or feedback for " + assignmentTitle + " in " + courseTitle + " is available."
	actionURL = "/courses/" + courseCode + "/grades"
	return title, body, actionURL
}

// canEmailRecipient reports whether email can be delivered to the address
// (managed learners only have a synthetic, undeliverable address).
func canEmailRecipient(email string) bool {
	e := strings.TrimSpace(email)
	return e != "" && !user.IsManagedEmail(e)
}

// NotifyGradesPostedAfterRelease always creates an in-app (bell) notification for students whose
// cells were just posted, then also emails and SMS-notifies them when those channels are enabled.
// Email is skipped for managed learners, whose synthetic address cannot receive mail.
// hub may be nil (no live bell update; the notification is still stored).
func NotifyGradesPostedAfterRelease(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, moduleItemID uuid.UUID, cells []coursegrades.PostedCell, smsQueue *smsnotificationqueue.Bus, hub *notifevents.Hub) {
	if len(cells) == 0 || pool == nil {
		return
	}
	code, err := course.GetCourseCodeByID(ctx, pool, courseID)
	if err != nil || code == nil {
		return
	}
	var courseTitle, assignmentTitle string
	var orgID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT c.title, COALESCE(csi.title, 'Assignment'), c.org_id
FROM course.courses c
JOIN course.course_structure_items csi ON csi.course_id = c.id AND csi.id = $2
WHERE c.id = $1
`, courseID, moduleItemID).Scan(&courseTitle, &assignmentTitle, &orgID); err != nil {
		return
	}
	orgPtr := &orgID

	push := &PushService{Pool: pool, Config: cfg, SSEHub: hub}
	ns := &Service{Pool: pool, Config: cfg}
	smsSvc := &SmsService{Pool: pool, Config: cfg, Queue: smsQueue}
	title, body, actionURL := GradePostedInAppContent(courseTitle, assignmentTitle, *code)
	for _, cell := range cells {
		notifyGradePostedInApp(ctx, pool, push, cell.StudentUserID, title, body, actionURL)
		if cfg.EmailNotificationsEnabled && studentCanReceiveEmail(ctx, pool, cell.StudentUserID) {
			ns.NotifyGradePosted(ctx, cell.StudentUserID, courseTitle, assignmentTitle, *code, orgPtr)
		}
		if cfg.SmsNotificationsEnabled {
			smsSvc.NotifyGradePosted(ctx, cell.StudentUserID, courseTitle, assignmentTitle, *code)
		}
	}
	slog.Info("notifications.grade_posted", "course_id", courseID, "count", len(cells))
}

// notifyGradePostedInApp inserts the bell notification unless an identical one is still unread
// (re-saving a grade must not stack duplicates).
func notifyGradePostedInApp(ctx context.Context, pool *pgxpool.Pool, push *PushService, userID uuid.UUID, title, body, actionURL string) {
	dup, err := notificationsinbox.HasUnread(ctx, pool, userID, EventGradePosted, title, actionURL)
	if err == nil && dup {
		return
	}
	if err := push.Enqueue(ctx, userID, EventGradePosted, title, body, actionURL); err != nil {
		slog.Warn("notifications.grade_posted_in_app", "err", err, "user_id", userID)
	}
}

func studentCanReceiveEmail(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) bool {
	row, err := user.FindByID(ctx, pool, userID)
	if err != nil || row == nil {
		return false
	}
	return canEmailRecipient(row.Email)
}
