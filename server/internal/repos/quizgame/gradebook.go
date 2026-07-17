package quizgame

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/gradeauditevents"
)

var (
	ErrGradebookLinkNotFound = errors.New("quizgame: gradebook link not found")
	ErrInvalidMapping        = errors.New("quizgame: invalid gradebook mapping")
	ErrGuestNoGradebook      = errors.New("quizgame: guests cannot receive gradebook rows")
)

// GradebookMapping controls how game scores become gradebook points (FR-4).
type GradebookMapping string

const (
	MappingRawPoints      GradebookMapping = "raw_points"
	MappingPercentCorrect GradebookMapping = "percent_correct"
	MappingParticipation  GradebookMapping = "participation"
)

// NormalizeMapping returns a valid mapping or ErrInvalidMapping.
func NormalizeMapping(s string) (GradebookMapping, error) {
	switch GradebookMapping(strings.TrimSpace(s)) {
	case MappingRawPoints, MappingPercentCorrect, MappingParticipation:
		return GradebookMapping(strings.TrimSpace(s)), nil
	case "":
		return MappingParticipation, nil
	default:
		return "", ErrInvalidMapping
	}
}

// GradebookLink is one quizgame.gradebook_links row.
type GradebookLink struct {
	ID               string
	SessionID        *string
	AssignmentID     *string
	CourseID         string
	GradebookItemID  string
	Mapping          string
	PointsPossible   *float64
	ParticipationPct float64
	CreatedBy        *string
}

// GradePreview is one student's projected gradebook points.
type GradePreview struct {
	UserID         string  `json:"userId"`
	Nickname       string  `json:"nickname"`
	PointsEarned   float64 `json:"pointsEarned"`
	PointsPossible float64 `json:"pointsPossible"`
	SkippedGuest   bool    `json:"skippedGuest,omitempty"`
}

// MapPlayerGrade computes gradebook points for one enrolled player (pure; unit-tested).
func MapPlayerGrade(
	mapping GradebookMapping,
	pointsPossible float64,
	participationPct float64,
	totalScore int,
	maxScoreAcrossPlayers int,
	answered int,
	questionCount int,
	correct int,
) float64 {
	if pointsPossible < 0 {
		pointsPossible = 0
	}
	switch mapping {
	case MappingRawPoints:
		if maxScoreAcrossPlayers <= 0 {
			if totalScore <= 0 {
				return 0
			}
			return math.Min(pointsPossible, float64(totalScore))
		}
		return math.Round(float64(totalScore)/float64(maxScoreAcrossPlayers)*pointsPossible*100) / 100
	case MappingPercentCorrect:
		if questionCount <= 0 {
			return 0
		}
		pct := float64(correct) / float64(questionCount)
		return math.Round(pct*pointsPossible*100) / 100
	default: // participation
		threshold := participationPct
		if threshold <= 0 {
			threshold = 50
		}
		if questionCount <= 0 {
			return 0
		}
		pct := float64(answered) / float64(questionCount) * 100
		if pct+1e-9 >= threshold {
			return pointsPossible
		}
		return 0
	}
}

// CreateGradebookLinkInput creates/updates the link and pushes grades.
type CreateGradebookLinkInput struct {
	CourseCode       string
	SessionID        *string
	AssignmentID     *string
	Mapping          string
	PointsPossible   float64
	ParticipationPct float64 // answered ≥ X% for participation mapping
	Title            string
	CreatedBy        uuid.UUID
}

// PushGradebookLink creates the structure item (if needed), writes grades, and returns the link (FR-4..FR-6).
func PushGradebookLink(ctx context.Context, pool *pgxpool.Pool, in CreateGradebookLinkInput) (*GradebookLink, []GradePreview, error) {
	mapping, err := NormalizeMapping(in.Mapping)
	if err != nil {
		return nil, nil, err
	}
	if in.SessionID == nil && in.AssignmentID == nil {
		return nil, nil, fmt.Errorf("quizgame: session or assignment required")
	}
	pointsPossible := in.PointsPossible
	if pointsPossible <= 0 {
		pointsPossible = 10
	}
	partPct := in.ParticipationPct
	if partPct <= 0 {
		partPct = 50
	}

	var courseID uuid.UUID
	var title string
	var existing *GradebookLink

	if in.AssignmentID != nil {
		a, err := GetAssignmentByCourse(ctx, pool, in.CourseCode, *in.AssignmentID)
		if err != nil || a == nil {
			return nil, nil, ErrAssignmentNotFound
		}
		courseID, err = uuid.Parse(a.CourseID)
		if err != nil {
			return nil, nil, err
		}
		title = a.Title
		if in.Title != "" {
			title = in.Title
		}
		if a.PointsPossible != nil && in.PointsPossible <= 0 {
			pointsPossible = *a.PointsPossible
		}
		existing, _ = GetGradebookLinkByAssignment(ctx, pool, a.ID)
	} else {
		sess, err := GetSessionByCourse(ctx, pool, in.CourseCode, *in.SessionID)
		if err != nil || sess == nil {
			return nil, nil, ErrSessionNotFound
		}
		courseID, err = uuid.Parse(sess.CourseID)
		if err != nil {
			return nil, nil, err
		}
		title = sess.KitSnapshot.Title
		if title == "" {
			title = "Live Quiz"
		}
		if in.Title != "" {
			title = in.Title
		}
		existing, _ = GetGradebookLinkBySession(ctx, pool, sess.ID)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	itemID := uuid.Nil
	if existing != nil {
		itemID, err = uuid.Parse(existing.GradebookItemID)
		if err != nil {
			return nil, nil, err
		}
		// Update mapping/points on the link.
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.gradebook_links
			SET mapping = $2, points_possible = $3, participation_pct = $4
			WHERE id = $1::uuid`,
			existing.ID, string(mapping), pointsPossible, partPct)
		if err != nil {
			return nil, nil, err
		}
		_, _ = tx.Exec(ctx, `
			UPDATE course.module_assignments SET points_worth = $2 WHERE structure_item_id = $1`,
			itemID, pointsPossible)
		_, _ = tx.Exec(ctx, `
			UPDATE course.course_structure_items SET title = $2, updated_at = NOW() WHERE id = $1`,
			itemID, title)
	} else {
		itemID, err = createGradebookStructureItem(ctx, tx, courseID, title, pointsPossible)
		if err != nil {
			return nil, nil, err
		}
		var linkID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO quizgame.gradebook_links (
				session_id, assignment_id, course_id, gradebook_item_id,
				mapping, points_possible, participation_pct, created_by
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id`,
			nullableUUID(in.SessionID), nullableUUID(in.AssignmentID), courseID, itemID,
			string(mapping), pointsPossible, partPct, in.CreatedBy,
		).Scan(&linkID)
		if err != nil {
			return nil, nil, err
		}
		existing = &GradebookLink{ID: linkID.String()}
	}

	if in.AssignmentID != nil {
		_, _ = tx.Exec(ctx, `
			UPDATE quizgame.assignments SET gradebook_item_id = $2 WHERE id = $1::uuid`,
			*in.AssignmentID, itemID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	// Write grades after commit so coursegrades helpers see the structure item.
	previews, err := writeGradebookScores(ctx, pool, courseID, itemID, in, mapping, pointsPossible, partPct)
	if err != nil {
		return nil, nil, err
	}

	link, err := GetGradebookLink(ctx, pool, existing.ID)
	if err != nil {
		return nil, nil, err
	}
	return link, previews, nil
}

func createGradebookStructureItem(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, title string, points float64) (uuid.UUID, error) {
	itemID := uuid.New()
	err := tx.QueryRow(ctx, `
WITH mx AS (
    SELECT COALESCE(MAX(sort_order), -1) AS max_ord
    FROM course.course_structure_items
    WHERE course_id = $1 AND parent_id IS NULL
)
INSERT INTO course.course_structure_items (
    id, course_id, sort_order, kind, title, parent_id, published, archived
)
SELECT $2, $1, max_ord + 1, 'assignment', $3, NULL, true, false
FROM mx
RETURNING id
`, courseID, itemID, title).Scan(&itemID)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO course.module_assignments (structure_item_id, markdown, points_worth, posting_policy)
VALUES ($1, '', $2, 'automatic')
`, itemID, points)
	return itemID, err
}

func writeGradebookScores(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, itemID uuid.UUID,
	in CreateGradebookLinkInput,
	mapping GradebookMapping,
	pointsPossible, partPct float64,
) ([]GradePreview, error) {
	changedBy := in.CreatedBy
	reason := "interactive_quiz_gradebook_push"

	auditTx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = auditTx.Rollback(ctx) }()

	var previews []GradePreview
	if in.AssignmentID != nil {
		grades, err := ListAssignmentGrades(ctx, pool, *in.AssignmentID)
		if err != nil {
			return nil, err
		}
		maxScore := 0.0
		for _, g := range grades {
			if g.Score > maxScore {
				maxScore = g.Score
			}
		}
		for _, g := range grades {
			uid, err := uuid.Parse(g.UserID)
			if err != nil {
				continue
			}
			var pts float64
			switch mapping {
			case MappingParticipation:
				if g.Score > 0 {
					pts = pointsPossible
				}
			case MappingPercentCorrect:
				if maxScore > 0 {
					pts = math.Round(g.Score/maxScore*pointsPossible*100) / 100
				}
			default:
				pts = MapPlayerGrade(MappingRawPoints, pointsPossible, partPct, int(math.Round(g.Score)), int(math.Round(maxScore)), 1, 1, 1)
			}
			prev, _ := coursegrades.GetCell(ctx, pool, courseID, uid, itemID)
			var prevPts *float64
			action := "created"
			if prev != nil {
				prevPts = prev.PointsEarned
				action = "updated"
			}
			if err := coursegrades.UpsertCell(ctx, pool, courseID, uid, itemID, pts, nil, nil, "automatic"); err != nil {
				return nil, err
			}
			newPts := pts
			if err := gradeauditevents.Insert(ctx, auditTx, courseID, itemID, uid, &changedBy, action, prevPts, &newPts, nil, nil, &reason); err != nil {
				return nil, err
			}
			previews = append(previews, GradePreview{
				UserID: uid.String(), PointsEarned: pts, PointsPossible: pointsPossible,
			})
		}
		if err := auditTx.Commit(ctx); err != nil {
			return nil, err
		}
		return previews, nil
	}

	sess, err := GetSession(ctx, pool, *in.SessionID)
	if err != nil {
		return nil, err
	}
	players, err := ListPlayers(ctx, pool, sess.ID)
	if err != nil {
		return nil, err
	}
	responses, err := ListAllResponses(ctx, pool, sess.ID)
	if err != nil {
		return nil, err
	}
	ansBy := map[string]int{}
	corBy := map[string]int{}
	for _, r := range responses {
		ansBy[r.PlayerID]++
		if r.IsCorrect {
			corBy[r.PlayerID]++
		}
	}
	maxScore := 0
	for _, p := range players {
		if p.TotalScore > maxScore {
			maxScore = p.TotalScore
		}
	}
	qCount := len(sess.KitSnapshot.Questions)
	for _, p := range players {
		if p.UserID == nil {
			previews = append(previews, GradePreview{
				Nickname: p.Nickname, SkippedGuest: true, PointsPossible: pointsPossible,
			})
			continue
		}
		uid, err := uuid.Parse(*p.UserID)
		if err != nil {
			continue
		}
		pts := MapPlayerGrade(mapping, pointsPossible, partPct, p.TotalScore, maxScore, ansBy[p.ID], qCount, corBy[p.ID])
		prev, _ := coursegrades.GetCell(ctx, pool, courseID, uid, itemID)
		var prevPts *float64
		action := "created"
		if prev != nil {
			prevPts = prev.PointsEarned
			action = "updated"
		}
		if err := coursegrades.UpsertCell(ctx, pool, courseID, uid, itemID, pts, nil, nil, "automatic"); err != nil {
			return nil, err
		}
		newPts := pts
		if err := gradeauditevents.Insert(ctx, auditTx, courseID, itemID, uid, &changedBy, action, prevPts, &newPts, nil, nil, &reason); err != nil {
			return nil, err
		}
		previews = append(previews, GradePreview{
			UserID: uid.String(), Nickname: p.Nickname,
			PointsEarned: pts, PointsPossible: pointsPossible,
		})
	}
	if err := auditTx.Commit(ctx); err != nil {
		return nil, err
	}
	return previews, nil
}

// UnlinkGradebook removes the link and deletes the gradebook item + cells (FR-4 reversible).
func UnlinkGradebook(ctx context.Context, pool *pgxpool.Pool, courseCode, linkID string, changedBy uuid.UUID) error {
	link, err := GetGradebookLink(ctx, pool, linkID)
	if err != nil || link == nil {
		return ErrGradebookLinkNotFound
	}
	var ok bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM course.courses c
			WHERE c.id = $1::uuid AND c.course_code = $2
		)`, link.CourseID, courseCode).Scan(&ok)
	if err != nil || !ok {
		return ErrGradebookLinkNotFound
	}
	itemID, err := uuid.Parse(link.GradebookItemID)
	courseID, err2 := uuid.Parse(link.CourseID)
	if err != nil || err2 != nil {
		return ErrGradebookLinkNotFound
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Audit unlink for each cell before delete.
	rows, err := tx.Query(ctx, `
		SELECT student_user_id, points_earned FROM course.course_grades
		WHERE course_id = $1 AND module_item_id = $2`, courseID, itemID)
	if err != nil {
		return err
	}
	type cell struct {
		sid uuid.UUID
		pts *float64
	}
	var cells []cell
	for rows.Next() {
		var c cell
		if err := rows.Scan(&c.sid, &c.pts); err != nil {
			rows.Close()
			return err
		}
		cells = append(cells, c)
	}
	rows.Close()
	reason := "interactive_quiz_gradebook_unlink"
	for _, c := range cells {
		if err := gradeauditevents.Insert(ctx, tx, courseID, itemID, c.sid, &changedBy, "deleted", c.pts, nil, nil, nil, &reason); err != nil {
			return err
		}
	}
	_, _ = tx.Exec(ctx, `DELETE FROM course.course_grades WHERE course_id = $1 AND module_item_id = $2`, courseID, itemID)
	_, _ = tx.Exec(ctx, `DELETE FROM course.module_assignments WHERE structure_item_id = $1`, itemID)
	_, _ = tx.Exec(ctx, `DELETE FROM course.course_structure_items WHERE id = $1`, itemID)
	_, err = tx.Exec(ctx, `DELETE FROM quizgame.gradebook_links WHERE id = $1::uuid`, link.ID)
	if err != nil {
		return err
	}
	// Clear assignment.gradebook_item_id if set.
	if link.AssignmentID != nil {
		_, _ = tx.Exec(ctx, `
			UPDATE quizgame.assignments SET gradebook_item_id = NULL WHERE id = $1::uuid`, *link.AssignmentID)
	}
	return tx.Commit(ctx)
}

// PreviewGradebook computes projected scores without writing.
func PreviewGradebook(ctx context.Context, pool *pgxpool.Pool, courseCode, sessionID, mappingStr string, pointsPossible, partPct float64) ([]GradePreview, error) {
	mapping, err := NormalizeMapping(mappingStr)
	if err != nil {
		return nil, err
	}
	sess, err := GetSessionByCourse(ctx, pool, courseCode, sessionID)
	if err != nil {
		return nil, err
	}
	if pointsPossible <= 0 {
		pointsPossible = 10
	}
	if partPct <= 0 {
		partPct = 50
	}
	players, err := ListPlayers(ctx, pool, sess.ID)
	if err != nil {
		return nil, err
	}
	responses, err := ListAllResponses(ctx, pool, sess.ID)
	if err != nil {
		return nil, err
	}
	ansBy := map[string]int{}
	corBy := map[string]int{}
	for _, r := range responses {
		ansBy[r.PlayerID]++
		if r.IsCorrect {
			corBy[r.PlayerID]++
		}
	}
	maxScore := 0
	for _, p := range players {
		if p.TotalScore > maxScore {
			maxScore = p.TotalScore
		}
	}
	qCount := len(sess.KitSnapshot.Questions)
	var out []GradePreview
	for _, p := range players {
		if p.UserID == nil {
			out = append(out, GradePreview{Nickname: p.Nickname, SkippedGuest: true, PointsPossible: pointsPossible})
			continue
		}
		pts := MapPlayerGrade(mapping, pointsPossible, partPct, p.TotalScore, maxScore, ansBy[p.ID], qCount, corBy[p.ID])
		out = append(out, GradePreview{
			UserID: *p.UserID, Nickname: p.Nickname,
			PointsEarned: pts, PointsPossible: pointsPossible,
		})
	}
	return out, nil
}

func GetGradebookLink(ctx context.Context, pool *pgxpool.Pool, id string) (*GradebookLink, error) {
	lid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrGradebookLinkNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, assignment_id, course_id, gradebook_item_id,
			mapping, points_possible, participation_pct, created_by
		FROM quizgame.gradebook_links WHERE id = $1`, lid)
	return scanGradebookLink(row)
}

func GetGradebookLinkBySession(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*GradebookLink, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrGradebookLinkNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, assignment_id, course_id, gradebook_item_id,
			mapping, points_possible, participation_pct, created_by
		FROM quizgame.gradebook_links WHERE session_id = $1`, sid)
	link, err := scanGradebookLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return link, err
}

func GetGradebookLinkByAssignment(ctx context.Context, pool *pgxpool.Pool, assignmentID string) (*GradebookLink, error) {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return nil, ErrGradebookLinkNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, assignment_id, course_id, gradebook_item_id,
			mapping, points_possible, participation_pct, created_by
		FROM quizgame.gradebook_links WHERE assignment_id = $1`, aid)
	link, err := scanGradebookLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return link, err
}

func scanGradebookLink(row pgx.Row) (*GradebookLink, error) {
	var l GradebookLink
	var id, courseID, itemID uuid.UUID
	var sessID, assignID, createdBy *uuid.UUID
	var points *float64
	err := row.Scan(&id, &sessID, &assignID, &courseID, &itemID, &l.Mapping, &points, &l.ParticipationPct, &createdBy)
	if err != nil {
		return nil, err
	}
	l.ID = id.String()
	l.CourseID = courseID.String()
	l.GradebookItemID = itemID.String()
	l.PointsPossible = points
	if sessID != nil {
		s := sessID.String()
		l.SessionID = &s
	}
	if assignID != nil {
		s := assignID.String()
		l.AssignmentID = &s
	}
	if createdBy != nil {
		s := createdBy.String()
		l.CreatedBy = &s
	}
	return &l, nil
}

func nullableUUID(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return id
}

// AssignmentGradeRow is one policy-applied homework grade.
type AssignmentGradeRow struct {
	UserID string
	Score  float64
	Policy string
}

// ListAssignmentGrades returns all stored policy grades for an assignment.
func ListAssignmentGrades(ctx context.Context, pool *pgxpool.Pool, assignmentID string) ([]AssignmentGradeRow, error) {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT user_id, score, policy FROM quizgame.assignment_grades
		WHERE assignment_id = $1 ORDER BY user_id`, aid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssignmentGradeRow
	for rows.Next() {
		var r AssignmentGradeRow
		var uid uuid.UUID
		if err := rows.Scan(&uid, &r.Score, &r.Policy); err != nil {
			return nil, err
		}
		r.UserID = uid.String()
		out = append(out, r)
	}
	return out, rows.Err()
}
