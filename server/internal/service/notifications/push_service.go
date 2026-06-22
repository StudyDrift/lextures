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
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/notificationprefs"
	"github.com/lextures/lextures/server/internal/repos/pushjobs"
)

// PushService creates in-app notifications and enqueues push delivery jobs (plan 6.3).
type PushService struct {
	Pool    *pgxpool.Pool
	Config  config.Config
	SSEHub  *notifevents.Hub
}

func (s *PushService) publicWebOrigin() string {
	svc := &Service{Config: s.Config}
	return svc.publicWebOrigin()
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

// EnqueueGradePosted creates an in-app + push notification for a posted grade.
func (s *PushService) EnqueueGradePosted(ctx context.Context, studentUserID uuid.UUID, courseName, assignmentName, courseCode string) {
	link := fmt.Sprintf("%s/courses/%s/grades", s.publicWebOrigin(), courseCode)
	title := fmt.Sprintf("%s: Grade posted", courseName)
	body := fmt.Sprintf("Your grade for %s has been posted.", assignmentName)
	if err := s.Enqueue(ctx, studentUserID, EventGradePosted, title, body, link); err != nil {
		slog.Warn("push.grade_posted", "err", err, "user_id", studentUserID)
	}
}

// EnqueueAssignmentCreated creates an in-app + push notification for a new assignment.
func (s *PushService) EnqueueAssignmentCreated(ctx context.Context, studentIDs []uuid.UUID, courseName, assignmentName, courseCode string) {
	link := fmt.Sprintf("%s/courses/%s", s.publicWebOrigin(), courseCode)
	title := fmt.Sprintf("%s: New assignment", courseName)
	body := fmt.Sprintf("A new assignment has been posted: %s", assignmentName)
	for _, sid := range studentIDs {
		if err := s.Enqueue(ctx, sid, EventAssignmentCreated, title, body, link); err != nil {
			slog.Warn("push.assignment_created", "err", err, "user_id", sid)
		}
	}
}

// EnqueueMeetingReminder sends an in-app + push notification reminding students about an upcoming meeting.
func (s *PushService) EnqueueMeetingReminder(ctx context.Context, studentIDs []uuid.UUID, courseName, meetingTitle, courseCode string) {
	link := fmt.Sprintf("%s/courses/%s/live", s.publicWebOrigin(), courseCode)
	title := fmt.Sprintf("%s: Live session starting soon", courseName)
	body := fmt.Sprintf("Your live session \"%s\" starts in 10 minutes.", meetingTitle)
	for _, sid := range studentIDs {
		if err := s.Enqueue(ctx, sid, EventMeetingReminder, title, body, link); err != nil {
			slog.Warn("push.meeting_reminder", "err", err, "user_id", sid)
		}
	}
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

// EnqueueDiscussionReply creates an in-app + push notification for a discussion reply.
func (s *PushService) EnqueueDiscussionReply(ctx context.Context, recipientIDs []uuid.UUID, courseName, threadTitle, courseCode, threadID string) {
	link := fmt.Sprintf("%s/courses/%s/discussions/threads/%s", s.publicWebOrigin(), courseCode, threadID)
	title := fmt.Sprintf("%s: New reply", courseName)
	body := fmt.Sprintf("New reply in %s", threadTitle)
	for _, rid := range recipientIDs {
		if err := s.Enqueue(ctx, rid, EventDiscussionReply, title, body, link); err != nil {
			slog.Warn("push.discussion_reply", "err", err, "user_id", rid)
		}
	}
}
