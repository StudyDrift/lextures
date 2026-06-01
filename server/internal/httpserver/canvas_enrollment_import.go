package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

type canvasEnrollmentImportStats struct {
	Enrolled       int
	AccountsCreated int
	SkippedNoEmail int
}

func canvasUserDisplayName(u map[string]any, email string) string {
	if u != nil {
		for _, k := range []string{"name", "sortable_name", "short_name"} {
			if n := strings.TrimSpace(strAt(u, k, "")); n != "" {
				return n
			}
		}
	}
	if email != "" {
		if i := strings.Index(email, "@"); i > 0 {
			return email[:i]
		}
	}
	return "Canvas learner"
}

func canvasEnrollmentListQuery() url.Values {
	q := url.Values{}
	q.Add("state[]", "active")
	q.Add("state[]", "invited")
	q.Add("state[]", "creation_pending")
	q.Add("include[]", "user")
	return q
}

func canvasRosterEmailsByCanvasUserID(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
) (map[int64]string, error) {
	rows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/users", canvasCourseID),
		url.Values{"include[]": []string{"email"}})
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(rows))
	for _, ru := range rows {
		uid := int64At(ru, "id")
		if uid <= 0 {
			continue
		}
		if eg := normalizedLexturesEmailGuessFromCanvasUserMap(ru); eg != "" {
			out[uid] = eg
		}
	}
	return out, nil
}

// canvasResolveLexturesUserForEnrollment finds or creates a Lextures user for a Canvas roster member.
func canvasResolveLexturesUserForEnrollment(
	ctx context.Context,
	pool *pgxpool.Pool,
	tx pgx.Tx,
	orgID uuid.UUID,
	email string,
	canvasUser map[string]any,
	stats *canvasEnrollmentImportStats,
) (uuid.UUID, error) {
	em := user.NormalizeEmail(email)
	if !strings.Contains(em, "@") {
		if stats != nil {
			stats.SkippedNoEmail++
		}
		return uuid.Nil, nil
	}
	if existing, err := user.FindByEmailCI(ctx, pool, em); err != nil {
		return uuid.Nil, err
	} else if existing != nil {
		uid, err := uuid.Parse(existing.ID)
		if err != nil {
			return uuid.Nil, err
		}
		return uid, nil
	}
	ph, err := authservice.PlaceholderPasswordHash()
	if err != nil {
		return uuid.Nil, errors.New("Failed to provision an account for a Canvas enrollment.")
	}
	dn := canvasUserDisplayName(canvasUser, em)
	row, err := user.InsertUserInOrgTx(ctx, tx, orgID, em, ph, &dn)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			if existing, fe := user.FindByEmailCI(ctx, pool, em); fe == nil && existing != nil {
				return uuid.Parse(existing.ID)
			}
		}
		return uuid.Nil, fmt.Errorf("Failed to create account for %s: %w", em, err)
	}
	if stats != nil {
		stats.AccountsCreated++
	}
	uid, err := uuid.Parse(row.ID)
	if err != nil {
		return uuid.Nil, err
	}
	return uid, nil
}

// canvasFillGradeUserMap ensures every Canvas enrollment with an email maps to a Lextures user id (creating accounts when needed).
func canvasFillGradeUserMap(
	ctx context.Context,
	pool *pgxpool.Pool,
	tx pgx.Tx,
	orgID uuid.UUID,
	enrollmentRows []map[string]any,
	rosterEmailByCanvasUID map[int64]string,
	out map[int64]uuid.UUID,
	stats *canvasEnrollmentImportStats,
) error {
	if out == nil {
		return nil
	}
	for _, e := range enrollmentRows {
		u := objAt(e, "user")
		canvasUID := int64At(u, "id")
		if canvasUID <= 0 {
			continue
		}
		if _, ok := out[canvasUID]; ok {
			continue
		}
		email := rosterEmailByCanvasUID[canvasUID]
		if email == "" {
			email = normalizedLexturesEmailGuessFromCanvasUserMap(u)
		}
		userID, err := canvasResolveLexturesUserForEnrollment(ctx, pool, tx, orgID, email, u, stats)
		if err != nil {
			return err
		}
		if userID != uuid.Nil {
			out[canvasUID] = userID
		}
	}
	return nil
}

func canvasApplyEnrollment(
	ctx context.Context,
	tx pgx.Tx,
	courseID uuid.UUID,
	courseCode string,
	userID uuid.UUID,
	role string,
	stats *canvasEnrollmentImportStats,
) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO course.course_enrollments (course_id, user_id, role)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT 1 FROM course.course_enrollments
			WHERE course_id = $1 AND user_id = $2 AND role = 'owner'
		)
		ON CONFLICT (course_id, user_id, role) DO NOTHING
	`, courseID, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if stats != nil {
			stats.Enrolled++
		}
		_ = courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, userID, courseID, courseCode)
	}
	return nil
}
