package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/models/gradecomment"
	"github.com/lextures/lextures/server/internal/notifevents"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
)

// SubmissionCommentInAppContent builds the bell title, body, and action URL for new feedback.
func SubmissionCommentInAppContent(assignmentTitle, courseTitle, courseCode, itemKind, itemID, commentID string) (title, body, actionURL string) {
	title = "New comment on " + assignmentTitle
	body = "New feedback on " + assignmentTitle + " in " + courseTitle + "."
	actionURL = submissionFeedbackPath(courseCode, itemKind, itemID, commentID)
	return title, body, actionURL
}

func submissionFeedbackPath(courseCode, itemKind, itemID, commentID string) string {
	var path string
	switch strings.TrimSpace(itemKind) {
	case "quiz":
		path = "/courses/" + courseCode + "/modules/quiz/" + itemID
	case "assignment", "":
		path = "/courses/" + courseCode + "/modules/assignment/" + itemID
	default:
		path = "/courses/" + courseCode + "/my-grades"
	}
	if commentID != "" && (itemKind == "assignment" || itemKind == "quiz" || itemKind == "") {
		path += "?comment=" + commentID
	}
	return path
}

// DeliverSubmissionComment notifies the learner in the bell, and by push, email, and SMS
// when those channels are enabled. A repeat of the same comment id is ignored.
func DeliverSubmissionComment(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.Config,
	hub *notifevents.Hub,
	smsQueue *smsnotificationqueue.Bus,
	studentID uuid.UUID,
	courseTitle, assignmentTitle, courseCode, itemKind, itemID, commentID, excerpt, authorUserID string,
) {
	if pool == nil || studentID == uuid.Nil {
		return
	}
	if authorUserID != "" && authorUserID == studentID.String() {
		return
	}
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		commentID = "latest"
	}
	title, body, actionURL := SubmissionCommentInAppContent(assignmentTitle, courseTitle, courseCode, itemKind, itemID, commentID)
	already, err := notificationsinbox.Exists(ctx, pool, studentID, EventSubmissionComment, actionURL)
	if err != nil || already {
		return
	}
	push := &PushService{Pool: pool, Config: cfg, SSEHub: hub}
	if err := push.Enqueue(ctx, studentID, EventSubmissionComment, title, body, actionURL); err != nil {
		slog.Warn("notifications.submission_comment_in_app", "err", err, "user_id", studentID)
	}
	if cfg.EmailNotificationsEnabled && studentCanReceiveEmail(ctx, pool, studentID) {
		ns := &Service{Pool: pool, Config: cfg}
		ns.NotifySubmissionComment(ctx, studentID, courseTitle, assignmentTitle, courseCode, itemKind, itemID, commentID, excerpt)
	}
	if cfg.SmsNotificationsEnabled {
		sms := &SmsService{Pool: pool, Config: cfg, Queue: smsQueue}
		sms.NotifySubmissionComment(ctx, studentID, courseTitle, assignmentTitle, actionURL)
	}
}

func notifyLatestPostedComment(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.Config,
	hub *notifevents.Hub,
	smsQueue *smsnotificationqueue.Bus,
	courseID, itemID, studentID uuid.UUID,
	courseTitle, assignmentTitle, courseCode, itemKind string,
) {
	var raw []byte
	var legacy *string
	err := pool.QueryRow(ctx, `
SELECT instructor_comments_json, instructor_comment
FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3 AND posted_at IS NOT NULL
`, courseID, studentID, itemID).Scan(&raw, &legacy)
	if err != nil {
		return
	}
	comments := gradecomment.ResolveList(raw, legacy)
	var latest gradecomment.Comment
	found := false
	for i := len(comments) - 1; i >= 0; i-- {
		if strings.TrimSpace(comments[i].Body) == "" {
			continue
		}
		latest = comments[i]
		found = true
		break
	}
	if !found {
		return
	}
	author := ""
	if latest.UserID != nil {
		author = strings.TrimSpace(*latest.UserID)
	}
	DeliverSubmissionComment(ctx, pool, cfg, hub, smsQueue, studentID, courseTitle, assignmentTitle, courseCode, itemKind, itemID.String(), latest.ID, latest.Body, author)
}

// DeliverSubmissionCommentByItem looks up the assignment and notifies the learner.
func DeliverSubmissionCommentByItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.Config,
	hub *notifevents.Hub,
	smsQueue *smsnotificationqueue.Bus,
	courseCode string,
	itemID, studentID uuid.UUID,
	commentID, excerpt, authorUserID string,
) {
	if pool == nil || itemID == uuid.Nil {
		return
	}
	var courseTitle, assignmentTitle, itemKind string
	err := pool.QueryRow(ctx, `
SELECT c.title, COALESCE(csi.title, 'Assignment'), COALESCE(csi.kind, '')
FROM course.courses c
JOIN course.course_structure_items csi ON csi.course_id = c.id AND csi.id = $2
WHERE c.course_code = $1
`, courseCode, itemID).Scan(&courseTitle, &assignmentTitle, &itemKind)
	if err != nil {
		return
	}
	DeliverSubmissionComment(ctx, pool, cfg, hub, smsQueue, studentID, courseTitle, assignmentTitle, courseCode, itemKind, itemID.String(), commentID, excerpt, authorUserID)
}

func excerptComment(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 240 {
		return body[:237] + "..."
	}
	return body
}

// NotifySubmissionComment emails the learner when new feedback is visible.
func (s *Service) NotifySubmissionComment(ctx context.Context, studentUserID uuid.UUID, courseName, assignmentName, courseCode, itemKind, itemID, commentID, excerpt string) {
	if !s.enabled() {
		return
	}
	path := submissionFeedbackPath(courseCode, itemKind, itemID, commentID)
	link := s.publicWebOrigin() + path
	excerpt = excerptComment(excerpt)
	vars := map[string]string{
		"courseName":     courseName,
		"assignmentName": assignmentName,
		"excerpt":        excerpt,
		"link":           link,
		"digestLine":     fmt.Sprintf("New comment on %s in %s", assignmentName, courseName),
		"subject":        fmt.Sprintf("New comment on %s", assignmentName),
	}
	if err := s.EnqueueEmail(ctx, studentUserID, EventSubmissionComment, "submission_comment", vars, nil); err != nil {
		slog.Warn("notifications.submission_comment", "err", err, "user_id", studentUserID)
	}
}
