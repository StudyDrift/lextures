package learningevents

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// EmitLoginAsync records SessionEvent.LoggedIn after successful authentication.
func EmitLoginAsync(pool *pgxpool.Pool, cfg config.Config, userRow *user.Row) {
	if !cfg.XAPIEmissionEnabled || pool == nil || userRow == nil {
		return
	}
	go func() {
		ctx := context.Background()
		uid, err := uuid.Parse(userRow.ID)
		if err != nil {
			return
		}
		orgID, err := organization.OrgIDForUser(ctx, pool, uid)
		if err != nil {
			return
		}
		name := userRow.Email
		if userRow.DisplayName != nil && *userRow.DisplayName != "" {
			name = *userRow.DisplayName
		}
		Emitter{Pool: pool, Cfg: cfg}.LoggedIn(ctx, orgID, userRow.Email, name)
	}()
}

// EmitEnrollmentAsync records course enrollment for each newly added email.
func EmitEnrollmentAsync(pool *pgxpool.Pool, cfg config.Config, orgID, courseID uuid.UUID, courseCode string, emails []string) {
	if !cfg.XAPIEmissionEnabled || pool == nil || len(emails) == 0 {
		return
	}
	go func() {
		ctx := context.Background()
		em := Emitter{Pool: pool, Cfg: cfg}
		for _, email := range emails {
			u, err := user.FindByEmail(ctx, pool, user.NormalizeEmail(email))
			if err != nil || u == nil {
				continue
			}
			dn := email
			if u.DisplayName != nil {
				dn = *u.DisplayName
			}
			em.CourseEnrollment(ctx, orgID, courseID, courseCode, u.Email, dn)
		}
	}()
}

// EmitQuizGradedAsync records quiz passed/failed after auto-submit.
func EmitQuizGradedAsync(pool *pgxpool.Pool, cfg config.Config, attemptID uuid.UUID) {
	if !cfg.XAPIEmissionEnabled || pool == nil {
		return
	}
	go func() {
		ctx := context.Background()
		var courseID, itemID, studentID uuid.UUID
		var courseCode, quizTitle string
		var score float32
		err := pool.QueryRow(ctx, `
SELECT qa.course_id, c.course_code, qa.structure_item_id, qa.student_user_id, qa.score_percent,
       COALESCE(csi.title, 'Quiz')
FROM course.quiz_attempts qa
INNER JOIN course.courses c ON c.id = qa.course_id
LEFT JOIN course.course_structure_items csi ON csi.id = qa.structure_item_id
WHERE qa.id = $1 AND qa.status = 'submitted'
`, attemptID).Scan(&courseID, &courseCode, &itemID, &studentID, &score, &quizTitle)
		if err != nil {
			slog.Warn("learningevents.quiz_lookup", "err", err)
			return
		}
		u, err := user.FindByID(ctx, pool, studentID)
		if err != nil || u == nil {
			return
		}
		orgID, err := organization.OrgIDForUser(ctx, pool, studentID)
		if err != nil {
			return
		}
		dn := u.Email
		if u.DisplayName != nil {
			dn = *u.DisplayName
		}
		passed := score >= 60
		Emitter{Pool: pool, Cfg: cfg}.QuizAttemptGraded(ctx, orgID, courseID, courseCode, u.Email, dn, itemID.String(), quizTitle, float64(score), passed)
	}()
}
