package publicapi

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefeed"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

// EnrollUserInput is the body for POST /api/v1/courses/{id}/enrollments.
type EnrollUserInput struct {
	Email      string `json:"email"`
	CourseRole string `json:"courseRole"`
}

// EnrollUser enrolls an existing org user in a course by email.
func EnrollUser(ctx context.Context, pool *pgxpool.Pool, actorID, courseID uuid.UUID, in EnrollUserInput) (*EnrollmentResource, error) {
	email := strings.TrimSpace(in.Email)
	if email == "" {
		return nil, errors.New("email is required")
	}
	role := strings.ToLower(strings.TrimSpace(in.CourseRole))
	if role == "" {
		role = "student"
	}
	code, err := course.GetCourseCodeByID(ctx, pool, courseID)
	if err != nil || code == nil {
		return nil, err
	}
	ok, err := enrollment.UserHasAccess(ctx, pool, *code, actorID)
	if err != nil || !ok {
		return nil, errors.New("forbidden")
	}
	var orgID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID); err != nil {
		return nil, err
	}
	var targetID uuid.UUID
	err = pool.QueryRow(ctx, `
SELECT id FROM "user".users WHERE org_id = $1 AND lower(trim(email)) = lower(trim($2)) LIMIT 1
`, orgID, email).Scan(&targetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	var enrollmentID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active, invitation_pending)
VALUES ($1, $2, $3, true, false)
ON CONFLICT (course_id, user_id, role) DO UPDATE SET active = true, invitation_pending = false
RETURNING id
`, courseID, targetID, role).Scan(&enrollmentID)
	if err != nil {
		return nil, err
	}
	return &EnrollmentResource{
		ID:     enrollmentID.String(),
		UserID: targetID.String(),
		Role:   role,
		State:  "active",
	}, nil
}

// CreateAnnouncementInput is the body for POST /api/v1/courses/{id}/announcements.
type CreateAnnouncementInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// CreateAnnouncement posts to the course announcements feed channel.
func CreateAnnouncement(ctx context.Context, pool *pgxpool.Pool, actorID, courseID uuid.UUID, in CreateAnnouncementInput) (messageID string, err error) {
	title := strings.TrimSpace(in.Title)
	body := strings.TrimSpace(in.Body)
	if title == "" || body == "" {
		return "", errors.New("title and body are required")
	}
	channels, err := coursefeed.ListChannels(ctx, pool, courseID, actorID)
	if err != nil {
		return "", err
	}
	var channelID uuid.UUID
	for _, ch := range channels {
		if strings.EqualFold(ch.Name, "announcements") {
			channelID = ch.ID
			break
		}
	}
	if channelID == uuid.Nil {
		return "", errors.New("announcements channel not found")
	}
	text := title + "\n\n" + body
	id, err := coursefeed.CreateMessage(ctx, pool, channelID, actorID, text, nil, nil, false)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// PostGradeInput is the body for POST /api/v1/grades.
type PostGradeInput struct {
	CourseID      string  `json:"courseId"`
	StudentUserID string  `json:"studentUserId"`
	ModuleItemID  string  `json:"moduleItemId"`
	PointsEarned  float64 `json:"pointsEarned"`
	Post          bool    `json:"post"`
}

// PostGrade writes a gradebook cell and optionally posts it to learners.
func PostGrade(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID, in PostGradeInput) (*GradeResource, error) {
	courseID, err := uuid.Parse(strings.TrimSpace(in.CourseID))
	if err != nil {
		return nil, errors.New("invalid courseId")
	}
	studentID, err := uuid.Parse(strings.TrimSpace(in.StudentUserID))
	if err != nil {
		return nil, errors.New("invalid studentUserId")
	}
	itemID, err := uuid.Parse(strings.TrimSpace(in.ModuleItemID))
	if err != nil {
		return nil, errors.New("invalid moduleItemId")
	}
	code, err := course.GetCourseCodeByID(ctx, pool, courseID)
	if err != nil || code == nil {
		return nil, err
	}
	ok, err := enrollment.UserHasAccess(ctx, pool, *code, actorID)
	if err != nil || !ok {
		return nil, errors.New("forbidden")
	}
	if err := coursegrades.UpsertCell(ctx, pool, courseID, studentID, itemID, in.PointsEarned, nil, nil, ""); err != nil {
		return nil, err
	}
	if in.Post {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := coursegrades.MarkPosted(ctx, tx, courseID, itemID, time.Now().UTC(), []uuid.UUID{studentID}); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
	}
	return &GradeResource{
		CourseID:      courseID.String(),
		StudentUserID: studentID.String(),
		ModuleItemID:  itemID.String(),
		PointsEarned:  strconv.FormatFloat(in.PointsEarned, 'f', -1, 64),
	}, nil
}
