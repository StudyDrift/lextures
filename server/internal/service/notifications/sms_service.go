package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/notificationprefs"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
)

// SmsService enqueues SMS notification jobs when user preferences allow (RabbitMQ).
type SmsService struct {
	Pool   *pgxpool.Pool
	Config config.Config
	Queue  *smsnotificationqueue.Bus
}

func (s *SmsService) enabled() bool {
	return s != nil && s.Config.SmsNotificationsEnabled && s.Queue != nil && s.Pool != nil
}

// Enqueue checks SMS preferences and publishes a delivery job when allowed.
func (s *SmsService) Enqueue(ctx context.Context, userID uuid.UUID, eventType, title, body, actionURL string) error {
	if !s.enabled() {
		return nil
	}
	pref, err := notificationprefs.Get(ctx, s.Pool, userID, eventType)
	if err != nil {
		return err
	}
	if !pref.SmsEnabled {
		return nil
	}
	phone, err := RecipientPhone(ctx, s.Pool, userID)
	if err != nil {
		return err
	}
	if phone == "" {
		return nil
	}
	return s.Queue.Publish(ctx, smsnotificationqueue.QueueMessage{
		UserID:    userID,
		EventType: eventType,
		Phone:     phone,
		Title:     title,
		Body:      body,
		ActionURL: actionURL,
	})
}

// NotifyGradePosted enqueues an SMS when a grade is posted.
func (s *SmsService) NotifyGradePosted(ctx context.Context, studentUserID uuid.UUID, courseName, assignmentName, courseCode string) {
	link := fmt.Sprintf("%s/courses/%s/grades", s.publicWebOrigin(), courseCode)
	title := fmt.Sprintf("%s: Grade posted", courseName)
	body := fmt.Sprintf("Your grade for %s has been posted.", assignmentName)
	if err := s.Enqueue(ctx, studentUserID, EventGradePosted, title, body, link); err != nil {
		slog.Warn("sms.grade_posted", "err", err, "user_id", studentUserID)
	}
}

// NotifyAssignmentCreated enqueues SMS jobs for a new assignment.
func (s *SmsService) NotifyAssignmentCreated(ctx context.Context, studentIDs []uuid.UUID, courseName, assignmentName, courseCode string) {
	link := fmt.Sprintf("%s/courses/%s", s.publicWebOrigin(), courseCode)
	title := fmt.Sprintf("%s: New assignment", courseName)
	body := fmt.Sprintf("A new assignment has been posted: %s", assignmentName)
	for _, sid := range studentIDs {
		if err := s.Enqueue(ctx, sid, EventAssignmentCreated, title, body, link); err != nil {
			slog.Warn("sms.assignment_created", "err", err, "user_id", sid)
		}
	}
}

// NotifyDiscussionReply enqueues SMS jobs for discussion replies.
func (s *SmsService) NotifyDiscussionReply(ctx context.Context, recipientIDs []uuid.UUID, courseName, threadTitle, courseCode, threadID string) {
	link := fmt.Sprintf("%s/courses/%s/discussions/threads/%s", s.publicWebOrigin(), courseCode, threadID)
	title := fmt.Sprintf("%s: New reply", courseName)
	body := fmt.Sprintf("New reply in %s", threadTitle)
	for _, rid := range recipientIDs {
		if err := s.Enqueue(ctx, rid, EventDiscussionReply, title, body, link); err != nil {
			slog.Warn("sms.discussion_reply", "err", err, "user_id", rid)
		}
	}
}

// NotifyInboxMessage enqueues an SMS for a new inbox message.
func (s *SmsService) NotifyInboxMessage(ctx context.Context, recipientID uuid.UUID, title, body string) {
	if err := s.Enqueue(ctx, recipientID, EventInboxMessage, title, body, "/inbox"); err != nil {
		slog.Warn("sms.inbox_message", "err", err, "user_id", recipientID)
	}
}

func (s *SmsService) publicWebOrigin() string {
	svc := &Service{Config: s.Config}
	return svc.publicWebOrigin()
}

// RecipientPhone loads a user's phone number for SMS delivery.
func RecipientPhone(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	row, err := user.FindByID(ctx, pool, userID)
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", fmt.Errorf("user not found")
	}
	if row.PhoneNumber == nil {
		return "", nil
	}
	return strings.TrimSpace(*row.PhoneNumber), nil
}