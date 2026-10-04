package coursestandards

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/sbg"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

// LoadStandardsGradebook returns the instructor matrix for plan 3.7:
// course standards × enrolled students, with cached proficiency labels.
// Slices are always non-nil so an empty course encodes as [] not null.
func LoadStandardsGradebook(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, courseCode string) (sbg.SbgStandardsGradebookResponse, error) {
	out := sbg.SbgStandardsGradebookResponse{
		Standards:     []sbg.SbgStandardPublic{},
		Students:      []sbg.SbgGradebookStudent{},
		Proficiencies: []sbg.SbgGradebookCell{},
	}

	rows, err := pool.Query(ctx, `
		SELECT id, external_id, description, subject, grade_level, "position"
		FROM course.course_standards
		WHERE course_id = $1
		ORDER BY "position" ASC, id ASC
	`, courseID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var s sbg.SbgStandardPublic
		if err := rows.Scan(&s.ID, &s.ExternalID, &s.Description, &s.Subject, &s.GradeLevel, &s.Position); err != nil {
			return out, err
		}
		out.Standards = append(out.Standards, s)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	students, err := enrollment.ListStudentUsersForCourseCode(ctx, pool, courseCode, nil)
	if err != nil {
		return out, err
	}
	for _, stu := range students {
		out.Students = append(out.Students, sbg.SbgGradebookStudent{
			UserID:       stu.UserID,
			DisplayLabel: stu.DisplayName,
		})
	}

	prows, err := pool.Query(ctx, `
		SELECT student_id, standard_id, COALESCE(level_label, '')
		FROM course.student_standard_proficiencies
		WHERE course_id = $1
	`, courseID)
	if err != nil {
		return out, err
	}
	defer prows.Close()
	for prows.Next() {
		var cell sbg.SbgGradebookCell
		if err := prows.Scan(&cell.StudentUserID, &cell.StandardID, &cell.LevelLabel); err != nil {
			return out, err
		}
		out.Proficiencies = append(out.Proficiencies, cell)
	}
	if err := prows.Err(); err != nil {
		return out, err
	}
	return out, nil
}
