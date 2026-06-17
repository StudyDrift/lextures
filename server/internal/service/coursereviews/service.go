// Package coursereviews implements learner course ratings and reviews for the
// self-learner catalog (plan 15.7).
package coursereviews

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
	"github.com/lextures/lextures/server/internal/service/selfpaced"
)

const (
	MinCompletionPercent = 10
	MaxReviewTextRunes   = 2000
	EditWindowDays       = 30
	ReviewRemovedText    = "[Review removed]"
	DefaultPageSize      = 10
	MaxPageSize          = 50
)

var (
	ErrNotFound              = errors.New("coursereviews: not found")
	ErrNotEnrolled           = errors.New("coursereviews: not enrolled")
	ErrInsufficientProgress  = errors.New("coursereviews: insufficient progress")
	ErrEditWindowExpired     = errors.New("coursereviews: edit window expired")
	ErrForbidden             = errors.New("coursereviews: forbidden")
	ErrInvalidRating         = errors.New("coursereviews: invalid rating")
	ErrReviewTextTooLong     = errors.New("coursereviews: review text too long")
	ErrReviewRemoved         = errors.New("coursereviews: review removed")
	ErrEmptyResponse         = errors.New("coursereviews: empty response")
)

// Review is one learner rating row surfaced to clients.
type Review struct {
	ID                  uuid.UUID
	CourseID            uuid.UUID
	ReviewerID          uuid.UUID
	Rating              int
	ReviewText          *string
	CreatorResponse     *string
	IsFlagged           bool
	IsRemoved           bool
	ReviewerDisplayName string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Summary holds aggregate rating stats for a course.
type Summary struct {
	AverageRating  *float64
	RatingCount    int
	Distribution   map[int]int
}

// ListResult is a paginated public review list.
type ListResult struct {
	Summary    Summary
	Reviews    []Review
	NextCursor string
}

// SubmitInput is the payload for creating or updating a review (idempotent upsert).
type SubmitInput struct {
	CourseID   uuid.UUID
	ReviewerID uuid.UUID
	Rating     int
	ReviewText *string
}

// Eligibility reports whether a learner may submit a review and their progress percent.
type Eligibility struct {
	Eligible        bool
	ProgressPercent int
	HasReview       bool
	ReviewID        *uuid.UUID
	CanEdit         bool
}

// CheckEligibility returns whether the viewer may leave or edit a review.
func CheckEligibility(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (Eligibility, error) {
	out := Eligibility{}
	eid, err := enrollment.GetStudentEnrollmentID(ctx, pool, courseID, userID)
	if err != nil {
		return out, err
	}
	if eid == nil {
		return out, nil
	}
	totals, err := learnerprogress.CourseProgress(ctx, pool, courseID, *eid)
	if err != nil {
		return out, err
	}
	out.ProgressPercent = selfpaced.ProgressPercent(totals.CompletedItems, totals.TotalItems)
	out.Eligible = out.ProgressPercent >= MinCompletionPercent

	existing, err := getReviewByReviewer(ctx, pool, courseID, userID)
	if err != nil {
		return out, err
	}
	if existing != nil && !existing.IsRemoved {
		out.HasReview = true
		out.ReviewID = &existing.ID
		out.CanEdit = withinEditWindow(existing.CreatedAt, time.Now().UTC())
	}
	return out, nil
}

// Submit creates or updates a review for an enrolled learner (idempotent per user/course).
func Submit(ctx context.Context, pool *pgxpool.Pool, in SubmitInput, now time.Time) (*Review, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, ErrInvalidRating
	}
	if in.ReviewText != nil && utf8.RuneCountInString(*in.ReviewText) > MaxReviewTextRunes {
		return nil, ErrReviewTextTooLong
	}

	elig, err := CheckEligibility(ctx, pool, in.CourseID, in.ReviewerID)
	if err != nil {
		return nil, err
	}
	if !elig.Eligible {
		return nil, ErrInsufficientProgress
	}

	existing, err := getReviewByReviewer(ctx, pool, in.CourseID, in.ReviewerID)
	if err != nil {
		return nil, err
	}
	if existing != nil && !existing.IsRemoved && !withinEditWindow(existing.CreatedAt, now) {
		return nil, ErrEditWindowExpired
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var review Review
	if existing == nil {
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_reviews (course_id, reviewer_id, rating, review_text)
VALUES ($1, $2, $3, $4)
RETURNING id, course_id, reviewer_id, rating, review_text, creator_response,
          is_flagged, is_removed, created_at, updated_at
`, in.CourseID, in.ReviewerID, in.Rating, in.ReviewText).Scan(
			&review.ID, &review.CourseID, &review.ReviewerID, &review.Rating, &review.ReviewText,
			&review.CreatorResponse, &review.IsFlagged, &review.IsRemoved, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := adjustCourseAggregate(ctx, tx, in.CourseID, in.Rating, 1); err != nil {
			return nil, err
		}
	} else {
		oldRating := existing.Rating
		err = tx.QueryRow(ctx, `
UPDATE course.course_reviews
   SET rating = $3,
       review_text = $4,
       is_removed = FALSE,
       updated_at = $5
 WHERE id = $1 AND course_id = $2 AND reviewer_id = $6
RETURNING id, course_id, reviewer_id, rating, review_text, creator_response,
          is_flagged, is_removed, created_at, updated_at
`, existing.ID, in.CourseID, in.Rating, in.ReviewText, now, in.ReviewerID).Scan(
			&review.ID, &review.CourseID, &review.ReviewerID, &review.Rating, &review.ReviewText,
			&review.CreatorResponse, &review.IsFlagged, &review.IsRemoved, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		delta := in.Rating - oldRating
		if existing.IsRemoved {
			if err := adjustCourseAggregate(ctx, tx, in.CourseID, in.Rating, 1); err != nil {
				return nil, err
			}
		} else if delta != 0 {
			if err := adjustCourseAggregate(ctx, tx, in.CourseID, delta, 0); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	RecordReviewSubmitted(in.Rating)
	review.ReviewerDisplayName, err = loadDisplayName(ctx, pool, in.ReviewerID)
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// List returns paginated active reviews and aggregate summary for a course.
func List(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, cursor string, limit int) (ListResult, error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	summary, err := LoadSummary(ctx, pool, courseID)
	if err != nil {
		return ListResult{}, err
	}

	var after time.Time
	if cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return ListResult{}, ErrNotFound
		}
		after = t
	}

	args := []any{courseID}
	query := `
SELECT r.id, r.course_id, r.reviewer_id, r.rating, r.review_text, r.creator_response,
       r.is_flagged, r.is_removed, COALESCE(u.display_name, 'Learner'), r.created_at, r.updated_at
  FROM course.course_reviews r
  JOIN "user".users u ON u.id = r.reviewer_id
 WHERE r.course_id = $1 AND NOT r.is_removed`
	if !after.IsZero() {
		args = append(args, after)
		query += " AND r.created_at < $" + itoa(len(args))
	}
	args = append(args, limit+1)
	query += " ORDER BY r.created_at DESC LIMIT $" + itoa(len(args))

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var rv Review
		if err := rows.Scan(
			&rv.ID, &rv.CourseID, &rv.ReviewerID, &rv.Rating, &rv.ReviewText, &rv.CreatorResponse,
			&rv.IsFlagged, &rv.IsRemoved, &rv.ReviewerDisplayName, &rv.CreatedAt, &rv.UpdatedAt,
		); err != nil {
			return ListResult{}, err
		}
		reviews = append(reviews, rv)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	nextCursor := ""
	if len(reviews) > limit {
		last := reviews[limit-1]
		nextCursor = last.CreatedAt.UTC().Format(time.RFC3339Nano)
		reviews = reviews[:limit]
	}

	return ListResult{Summary: summary, Reviews: reviews, NextCursor: nextCursor}, nil
}

// LoadSummary returns aggregate rating data for a course.
func LoadSummary(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (Summary, error) {
	var sum int
	var count int
	var avg *float64
	err := pool.QueryRow(ctx, `
SELECT rating_sum, rating_count,
       CASE WHEN rating_count > 0 THEN ROUND(rating_sum::numeric / rating_count, 2) ELSE NULL END
  FROM course.courses
 WHERE id = $1
`, courseID).Scan(&sum, &count, &avg)
	if err != nil {
		return Summary{}, err
	}

	distRows, err := pool.Query(ctx, `
SELECT rating, COUNT(*)::int
  FROM course.course_reviews
 WHERE course_id = $1 AND NOT is_removed
 GROUP BY rating
`, courseID)
	if err != nil {
		return Summary{}, err
	}
	defer distRows.Close()

	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for distRows.Next() {
		var rating, n int
		if err := distRows.Scan(&rating, &n); err != nil {
			return Summary{}, err
		}
		if rating >= 1 && rating <= 5 {
			dist[rating] = n
		}
	}
	if err := distRows.Err(); err != nil {
		return Summary{}, err
	}

	return Summary{AverageRating: avg, RatingCount: count, Distribution: dist}, nil
}

// Flag marks a review for admin moderation.
func Flag(ctx context.Context, pool *pgxpool.Pool, reviewID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
UPDATE course.course_reviews
   SET is_flagged = TRUE, updated_at = NOW()
 WHERE id = $1 AND NOT is_removed
`, reviewID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AdminRemove soft-deletes a review and recomputes course aggregates.
func AdminRemove(ctx context.Context, pool *pgxpool.Pool, reviewID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var courseID uuid.UUID
	var rating int
	err = tx.QueryRow(ctx, `
UPDATE course.course_reviews
   SET is_removed = TRUE, updated_at = NOW()
 WHERE id = $1 AND NOT is_removed
RETURNING course_id, rating
`, reviewID).Scan(&courseID, &rating)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := adjustCourseAggregate(ctx, tx, courseID, -rating, -1); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SetCreatorResponse attaches a public instructor reply to a review.
func SetCreatorResponse(ctx context.Context, pool *pgxpool.Pool, reviewID uuid.UUID, response string) error {
	response = strings.TrimSpace(response)
	if response == "" {
		return ErrEmptyResponse
	}
	tag, err := pool.Exec(ctx, `
UPDATE course.course_reviews
   SET creator_response = $2, updated_at = NOW()
 WHERE id = $1 AND NOT is_removed
`, reviewID, response)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AnonymizeReviewerReviews replaces review text for a deleted user while keeping ratings (AC-5).
func AnonymizeReviewerReviews(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.course_reviews
   SET review_text = $2,
       updated_at = NOW()
 WHERE reviewer_id = $1
   AND review_text IS NOT NULL
   AND review_text <> $2
`, userID, ReviewRemovedText)
	return err
}

// ListFlagged returns reviews awaiting admin moderation.
func ListFlagged(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Review, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT r.id, r.course_id, r.reviewer_id, r.rating, r.review_text, r.creator_response,
       r.is_flagged, r.is_removed, COALESCE(u.display_name, 'Learner'), r.created_at, r.updated_at
  FROM course.course_reviews r
  JOIN "user".users u ON u.id = r.reviewer_id
 WHERE r.is_flagged AND NOT r.is_removed
 ORDER BY r.created_at DESC
 LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Review
	for rows.Next() {
		var rv Review
		if err := rows.Scan(
			&rv.ID, &rv.CourseID, &rv.ReviewerID, &rv.Rating, &rv.ReviewText, &rv.CreatorResponse,
			&rv.IsFlagged, &rv.IsRemoved, &rv.ReviewerDisplayName, &rv.CreatedAt, &rv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func getReviewByReviewer(ctx context.Context, pool *pgxpool.Pool, courseID, reviewerID uuid.UUID) (*Review, error) {
	var rv Review
	err := pool.QueryRow(ctx, `
SELECT id, course_id, reviewer_id, rating, review_text, creator_response,
       is_flagged, is_removed, created_at, updated_at
  FROM course.course_reviews
 WHERE course_id = $1 AND reviewer_id = $2
`, courseID, reviewerID).Scan(
		&rv.ID, &rv.CourseID, &rv.ReviewerID, &rv.Rating, &rv.ReviewText,
		&rv.CreatorResponse, &rv.IsFlagged, &rv.IsRemoved, &rv.CreatedAt, &rv.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

func adjustCourseAggregate(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, ratingDelta, countDelta int) error {
	_, err := tx.Exec(ctx, `
UPDATE course.courses
   SET rating_sum = GREATEST(0, rating_sum + $2),
       rating_count = GREATEST(0, rating_count + $3),
       average_rating = CASE
           WHEN GREATEST(0, rating_count + $3) > 0
           THEN ROUND((GREATEST(0, rating_sum + $2))::numeric / GREATEST(0, rating_count + $3), 2)
           ELSE NULL
       END
 WHERE id = $1
`, courseID, ratingDelta, countDelta)
	return err
}

func withinEditWindow(createdAt, now time.Time) bool {
	return now.Sub(createdAt.UTC()) <= EditWindowDays*24*time.Hour
}

func loadDisplayName(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var name string
	err := pool.QueryRow(ctx, `SELECT COALESCE(display_name, 'Learner') FROM "user".users WHERE id = $1`, userID).Scan(&name)
	return name, err
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
