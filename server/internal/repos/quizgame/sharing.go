package quizgame

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

const (
	ShareGranteeUser    = "user"
	ShareGranteeCourse  = "course"
	ShareGranteeOrgUnit = "org_unit"
	ShareGranteeOrg     = "org"

	SharePermView = "view"
	SharePermCopy = "copy"
	SharePermEdit = "edit"
)

// KitShare is one quizgame.kit_shares row.
type KitShare struct {
	ID          string    `json:"id"`
	KitID       string    `json:"kitId"`
	GranteeType string    `json:"granteeType"`
	GranteeID   *string   `json:"granteeId"`
	Permission  string    `json:"permission"`
	CreatedBy   *string   `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateShareInput grants access to a kit.
type CreateShareInput struct {
	GranteeType string
	GranteeID   *string
	Permission  string
}

func scanShare(row pgx.Row) (KitShare, error) {
	var s KitShare
	var id, kitID uuid.UUID
	var grantee, createdBy uuid.NullUUID
	if err := row.Scan(
		&id, &kitID, &s.GranteeType, &grantee, &s.Permission, &createdBy, &s.CreatedAt,
	); err != nil {
		return KitShare{}, err
	}
	s.ID = id.String()
	s.KitID = kitID.String()
	if grantee.Valid {
		g := grantee.UUID.String()
		s.GranteeID = &g
	}
	if createdBy.Valid {
		c := createdBy.UUID.String()
		s.CreatedBy = &c
	}
	return s, nil
}

func normalizeSharePermission(p string) (string, error) {
	p = strings.TrimSpace(strings.ToLower(p))
	if p == "" {
		p = SharePermCopy
	}
	switch p {
	case SharePermView, SharePermCopy, SharePermEdit:
		return p, nil
	default:
		return "", fmt.Errorf("quizgame: permission must be view, copy, or edit")
	}
}

func normalizeGranteeType(t string) (string, error) {
	t = strings.TrimSpace(strings.ToLower(t))
	switch t {
	case ShareGranteeUser, ShareGranteeCourse, ShareGranteeOrgUnit, ShareGranteeOrg:
		return t, nil
	default:
		return "", fmt.Errorf("quizgame: granteeType must be user, course, org_unit, or org")
	}
}

// ListShares returns shares for a kit in a course.
func ListShares(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) ([]KitShare, error) {
	k, err := Get(ctx, pool, courseCode, kitID)
	if err != nil || k == nil {
		return nil, err
	}
	kid, err := uuid.Parse(k.ID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT id, kit_id, grantee_type, grantee_id, permission, created_by, created_at
		FROM quizgame.kit_shares
		WHERE kit_id = $1
		ORDER BY created_at DESC
	`, kid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KitShare, 0)
	for rows.Next() {
		s, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CreateShare inserts a share grant.
func CreateShare(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string, createdBy uuid.UUID, in CreateShareInput) (*KitShare, error) {
	k, err := Get(ctx, pool, courseCode, kitID)
	if err != nil || k == nil {
		return nil, err
	}
	gt, err := normalizeGranteeType(in.GranteeType)
	if err != nil {
		return nil, err
	}
	perm, err := normalizeSharePermission(in.Permission)
	if err != nil {
		return nil, err
	}
	kid, err := uuid.Parse(k.ID)
	if err != nil {
		return nil, err
	}

	var grantee any
	switch gt {
	case ShareGranteeOrg:
		grantee = nil
	default:
		if in.GranteeID == nil || strings.TrimSpace(*in.GranteeID) == "" {
			return nil, fmt.Errorf("quizgame: granteeId is required for %s", gt)
		}
		gid, err := uuid.Parse(strings.TrimSpace(*in.GranteeID))
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid granteeId")
		}
		grantee = gid
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO quizgame.kit_shares (kit_id, grantee_type, grantee_id, permission, created_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (kit_id, grantee_type, grantee_id, permission) DO UPDATE
			SET created_by = EXCLUDED.created_by
		RETURNING id, kit_id, grantee_type, grantee_id, permission, created_by, created_at
	`, kid, gt, grantee, perm, createdBy)
	s, err := scanShare(row)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteShare revokes a share. Copies already made are unaffected.
func DeleteShare(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, shareID string) (bool, error) {
	k, err := Get(ctx, pool, courseCode, kitID)
	if err != nil || k == nil {
		return false, err
	}
	sid, err := uuid.Parse(shareID)
	if err != nil {
		return false, nil
	}
	kid, err := uuid.Parse(k.ID)
	if err != nil {
		return false, err
	}
	tag, err := pool.Exec(ctx, `
		DELETE FROM quizgame.kit_shares WHERE id = $1 AND kit_id = $2
	`, sid, kid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// EffectivePermission returns the best share permission the viewer has on a kit, or "".
func EffectivePermission(ctx context.Context, pool *pgxpool.Pool, kitID string, viewer uuid.UUID) (string, error) {
	kid, err := uuid.Parse(kitID)
	if err != nil {
		return "", nil
	}
	orgID, _ := organization.OrgIDForUser(ctx, pool, viewer)

	row := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(CASE permission
			WHEN 'edit' THEN 3
			WHEN 'copy' THEN 2
			WHEN 'view' THEN 1
			ELSE 0
		END), 0)
		FROM quizgame.kit_shares s
		WHERE s.kit_id = $1
		  AND (
			(s.grantee_type = 'user' AND s.grantee_id = $2)
			OR (s.grantee_type = 'course' AND s.grantee_id IN (
				SELECT course_id FROM course.course_enrollments WHERE user_id = $2 AND active
			))
			OR (s.grantee_type = 'org_unit' AND s.grantee_id IN (
				SELECT c.org_unit_id FROM course.course_enrollments e
				INNER JOIN course.courses c ON c.id = e.course_id
				WHERE e.user_id = $2 AND e.active AND c.org_unit_id IS NOT NULL
			))
			OR (s.grantee_type = 'org' AND s.grantee_id IS NULL AND $3::uuid IS NOT NULL
				AND EXISTS (
					SELECT 1 FROM quizgame.kits k
					INNER JOIN course.courses c ON c.id = k.course_id
					WHERE k.id = $1 AND c.org_id = $3
				)
				AND EXISTS (
					SELECT 1 FROM "user".users u WHERE u.id = $2 AND u.org_id = $3
				))
		  )
	`, kid, viewer, nullUUID(orgID))
	var rank int
	if err := row.Scan(&rank); err != nil {
		return "", err
	}
	switch rank {
	case 3:
		return SharePermEdit, nil
	case 2:
		return SharePermCopy, nil
	case 1:
		return SharePermView, nil
	default:
		return "", nil
	}
}

func nullUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// CanAccessKit reports whether viewer can view/copy/edit a kit via ownership, course, share, or catalog.
func CanAccessKit(ctx context.Context, pool *pgxpool.Pool, kit *Kit, viewer uuid.UUID, need string) (bool, error) {
	if kit == nil {
		return false, nil
	}
	need = strings.TrimSpace(strings.ToLower(need))
	if need == "" {
		need = SharePermView
	}
	rank := map[string]int{SharePermView: 1, SharePermCopy: 2, SharePermEdit: 3}
	needRank := rank[need]
	if needRank == 0 {
		return false, fmt.Errorf("quizgame: invalid permission need")
	}

	if kit.CourseID != "" {
		var enrolled bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM course.course_enrollments e
				WHERE e.course_id = $1::uuid AND e.user_id = $2 AND e.active
			)
		`, kit.CourseID, viewer).Scan(&enrolled)
		if err != nil {
			return false, err
		}
		if enrolled {
			return true, nil
		}
	}

	if kit.IsTemplate && kit.TemplateScope != nil && *kit.TemplateScope == "system" {
		return needRank <= rank[SharePermCopy], nil
	}

	if kit.Visibility == "org" && kit.CourseID != "" {
		orgID, err := organization.OrgIDForUser(ctx, pool, viewer)
		if err == nil && orgID != uuid.Nil {
			var sameOrg bool
			_ = pool.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM course.courses c
					WHERE c.id = $1::uuid AND c.org_id = $2
				)
			`, kit.CourseID, orgID).Scan(&sameOrg)
			if sameOrg && needRank <= rank[SharePermCopy] {
				return true, nil
			}
		}
	}

	if kit.CatalogStatus == "listed" && kit.Visibility == "public" && needRank <= rank[SharePermCopy] {
		return true, nil
	}

	perm, err := EffectivePermission(ctx, pool, kit.ID, viewer)
	if err != nil {
		return false, err
	}
	if perm == "" {
		return false, nil
	}
	return rank[perm] >= needRank, nil
}

// CourseCodeForKitID resolves the course_code for a kit, or "" for system templates.
func CourseCodeForKitID(ctx context.Context, pool *pgxpool.Pool, kitID string) (string, error) {
	id, err := uuid.Parse(kitID)
	if err != nil {
		return "", nil
	}
	var code *string
	err = pool.QueryRow(ctx, `
		SELECT c.course_code
		FROM quizgame.kits k
		LEFT JOIN course.courses c ON c.id = k.course_id
		WHERE k.id = $1
	`, id).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if code == nil {
		return "", nil
	}
	return *code, nil
}
