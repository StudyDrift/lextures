package httpserver

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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

const canvasAvatarMaxBytes = 512 << 10

type canvasEnrollmentImportStats struct {
	Enrolled        int
	AccountsCreated int
	SkippedNoEmail  int
	AvatarsImported int
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
	q.Add("include[]", "avatar_url")
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
		url.Values{"include[]": []string{"email", "avatar_url"}})
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

func canvasAvatarURLFromMaps(enrollment, canvasUser map[string]any) string {
	for _, m := range []map[string]any{canvasUser, enrollment} {
		if m == nil {
			continue
		}
		if u := strings.TrimSpace(strAt(m, "avatar_url", "")); u != "" {
			return u
		}
	}
	return ""
}

func canvasImageBytesToDataURL(data []byte, contentType string) (string, error) {
	ct := strings.TrimSpace(contentType)
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.ToLower(ct)
	if !strings.HasPrefix(ct, "image/") {
		return "", fmt.Errorf("avatar is not an image")
	}
	out := "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data)
	if len(out) > 2_000_000 {
		return "", fmt.Errorf("avatar data url too large")
	}
	return out, nil
}

func canvasDownloadAvatarImage(
	ctx context.Context,
	client *http.Client,
	downloadURL, accessToken string,
) ([]byte, string, error) {
	if client == nil || strings.TrimSpace(downloadURL) == "" {
		return nil, "", fmt.Errorf("missing download client or url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, canvasAvatarMaxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > canvasAvatarMaxBytes {
		return nil, "", fmt.Errorf("avatar too large")
	}
	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct == "" {
		ct = "image/jpeg"
	}
	return data, ct, nil
}

func canvasImportEnrollmentUserAvatar(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	accessToken string,
	userID uuid.UUID,
	enrollment, canvasUser map[string]any,
	stats *canvasEnrollmentImportStats,
) error {
	if tx == nil || userID == uuid.Nil {
		return nil
	}
	var hasAvatar bool
	if err := tx.QueryRow(ctx, `
SELECT avatar_url IS NOT NULL AND TRIM(avatar_url) <> ''
FROM "user".users
WHERE id = $1
`, userID).Scan(&hasAvatar); err != nil {
		return err
	}
	if hasAvatar {
		return nil
	}
	avatarURL := canvasAvatarURLFromMaps(enrollment, canvasUser)
	if avatarURL == "" {
		return nil
	}
	data, contentType, err := canvasDownloadAvatarImage(ctx, client, avatarURL, accessToken)
	if err != nil || len(data) == 0 {
		return nil
	}
	dataURL, err := canvasImageBytesToDataURL(data, contentType)
	if err != nil {
		return nil
	}
	updated, err := user.SetAvatarURLIfEmptyTx(ctx, tx, userID, dataURL)
	if err != nil {
		return err
	}
	if updated && stats != nil {
		stats.AvatarsImported++
	}
	return nil
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
	client *http.Client,
	accessToken string,
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
			if err := canvasImportEnrollmentUserAvatar(ctx, tx, client, accessToken, userID, e, u, stats); err != nil {
				return err
			}
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