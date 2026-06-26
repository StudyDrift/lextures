package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/repos/notificationprefs"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/pushjobs"
)

// PushService creates in-app notifications and enqueues push delivery jobs (plan 6.3).
type PushService struct {
	Pool   *pgxpool.Pool
	Config config.Config
	SSEHub *notifevents.Hub
}

// Enqueue stores an in-app notification and enqueues a push delivery job if push is enabled.
func (s *PushService) Enqueue(ctx context.Context, userID uuid.UUID, eventType, title, body, actionURL string) error {
	if s.Pool == nil {
		return nil
	}

	notifID, err := notificationsinbox.Insert(ctx, s.Pool, userID, eventType, title, body, actionURL)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	if s.SSEHub != nil {
		s.SSEHub.Notify(userID)
	}

	if !s.Config.PushNotificationsEnabled {
		return nil
	}

	pref, err := notificationprefs.Get(ctx, s.Pool, userID, eventType)
	if err != nil {
		slog.Warn("push_service.prefs", "err", err, "user_id", userID)
		return nil
	}
	if !pref.PushEnabled {
		return nil
	}

	if err := pushjobs.Enqueue(ctx, s.Pool, userID, &notifID, title, body, actionURL); err != nil {
		slog.Warn("push_service.enqueue_job", "err", err, "user_id", userID)
	}
	return nil
}

// EnqueueCanvasCourseImported notifies the importer that a Canvas course was copied into Lextures.
func (s *PushService) EnqueueCanvasCourseImported(ctx context.Context, userID uuid.UUID, courseName, courseCode string) {
	link := fmt.Sprintf("/courses/%s", courseCode)
	title := "Course imported from Canvas"
	body := fmt.Sprintf("%s is ready in Lextures.", courseName)
	if err := s.Enqueue(ctx, userID, EventCanvasCourseImported, title, body, link); err != nil {
		slog.Warn("push.canvas_course_imported", "err", err, "user_id", userID, "course_code", courseCode)
	}
}

// EnqueueCourseCopyImported notifies the importer that a course was copied from another Lextures course.
func (s *PushService) EnqueueCourseCopyImported(ctx context.Context, userID uuid.UUID, courseName, courseCode string) {
	link := fmt.Sprintf("/courses/%s", courseCode)
	title := "Course copied successfully"
	body := fmt.Sprintf("%s is ready in your catalog.", courseName)
	if err := s.Enqueue(ctx, userID, EventCourseCopyImported, title, body, link); err != nil {
		slog.Warn("push.course_copy_imported", "err", err, "user_id", userID, "course_code", courseCode)
	}
}

// EnqueueCourseCopyImportFailed notifies the importer that copying content into a new course failed.
func (s *PushService) EnqueueCourseCopyImportFailed(ctx context.Context, userID uuid.UUID, courseName, courseCode, detail string) {
	link := fmt.Sprintf("/courses/%s", courseCode)
	title := "Course copy failed"
	body := courseName
	if strings.TrimSpace(detail) != "" {
		body = fmt.Sprintf("%s — %s", courseName, strings.TrimSpace(detail))
	}
	if err := s.Enqueue(ctx, userID, EventCourseCopyImportFailed, title, body, link); err != nil {
		slog.Warn("push.course_copy_import_failed", "err", err, "user_id", userID, "course_code", courseCode)
	}
}
