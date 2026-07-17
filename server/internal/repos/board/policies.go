package board

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrgPolicies are org-scoped board governance settings (VC.10).
type OrgPolicies struct {
	OrgID                string    `json:"orgId"`
	ExternalSharing      bool      `json:"externalSharing"`
	MinorModerationFloor bool      `json:"minorModerationFloor"`
	DefaultAttribution   string    `json:"defaultAttribution"`
	BoardCapPerCourse    *int      `json:"boardCapPerCourse"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// DefaultOrgPolicies returns safe defaults when no row exists.
func DefaultOrgPolicies(orgID uuid.UUID) OrgPolicies {
	return OrgPolicies{
		OrgID:                orgID.String(),
		ExternalSharing:      false,
		MinorModerationFloor: true,
		DefaultAttribution:   AttributionNamed,
		BoardCapPerCourse:    nil,
		UpdatedAt:            time.Time{},
	}
}

// GetOrgPolicies returns stored policies or nil when unset.
func GetOrgPolicies(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*OrgPolicies, error) {
	row := pool.QueryRow(ctx, `
		SELECT org_id, external_sharing, minor_moderation_floor, default_attribution,
		       board_cap_per_course, updated_at
		FROM board.org_policies
		WHERE org_id = $1
	`, orgID)
	p, err := scanOrgPolicies(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ResolveOrgPolicies returns stored policies or safe defaults.
func ResolveOrgPolicies(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (OrgPolicies, error) {
	p, err := GetOrgPolicies(ctx, pool, orgID)
	if err != nil {
		return OrgPolicies{}, err
	}
	if p == nil {
		return DefaultOrgPolicies(orgID), nil
	}
	return *p, nil
}

// OrgIDForCourse returns the organization that owns the course.
func OrgIDForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT org_id FROM course.courses WHERE course_code = $1
	`, courseCode).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("board: course not found")
		}
		return uuid.Nil, err
	}
	return orgID, nil
}

// PatchOrgPoliciesInput is a partial update for org board policies.
type PatchOrgPoliciesInput struct {
	ExternalSharing      *bool
	MinorModerationFloor *bool
	DefaultAttribution   *string
	BoardCapPerCourse    *int // nil pointer = leave unchanged
	ClearBoardCap        bool // when true, set board_cap_per_course to NULL
}

// UpsertOrgPolicies inserts or updates org policies and returns the stored row.
func UpsertOrgPolicies(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, in PatchOrgPoliciesInput) (OrgPolicies, error) {
	cur, err := ResolveOrgPolicies(ctx, pool, orgID)
	if err != nil {
		return OrgPolicies{}, err
	}
	if in.ExternalSharing != nil {
		cur.ExternalSharing = *in.ExternalSharing
	}
	if in.MinorModerationFloor != nil {
		cur.MinorModerationFloor = *in.MinorModerationFloor
	}
	if in.DefaultAttribution != nil {
		norm, nerr := NormalizeAttribution(*in.DefaultAttribution)
		if nerr != nil {
			return OrgPolicies{}, nerr
		}
		cur.DefaultAttribution = norm
	}
	var cap any
	if in.ClearBoardCap {
		cap = nil
		cur.BoardCapPerCourse = nil
	} else if in.BoardCapPerCourse != nil {
		if *in.BoardCapPerCourse < 0 {
			return OrgPolicies{}, fmt.Errorf("board: board_cap_per_course must be >= 0")
		}
		v := *in.BoardCapPerCourse
		cap = v
		cur.BoardCapPerCourse = &v
	} else if cur.BoardCapPerCourse != nil {
		cap = *cur.BoardCapPerCourse
	} else {
		cap = nil
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO board.org_policies (
			org_id, external_sharing, minor_moderation_floor, default_attribution, board_cap_per_course, updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			external_sharing = EXCLUDED.external_sharing,
			minor_moderation_floor = EXCLUDED.minor_moderation_floor,
			default_attribution = EXCLUDED.default_attribution,
			board_cap_per_course = EXCLUDED.board_cap_per_course,
			updated_at = NOW()
		RETURNING org_id, external_sharing, minor_moderation_floor, default_attribution,
		          board_cap_per_course, updated_at
	`, orgID, cur.ExternalSharing, cur.MinorModerationFloor, cur.DefaultAttribution, cap)
	return scanOrgPolicies(row)
}

// CountBoardsInCourse returns the number of boards (including archived) for a course.
func CountBoardsInCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1
	`, courseCode).Scan(&n)
	return n, err
}

// BoardCapExceeded reports whether creating another board would exceed the org cap.
func BoardCapExceeded(ctx context.Context, pool *pgxpool.Pool, courseCode string, pol OrgPolicies) (bool, error) {
	if pol.BoardCapPerCourse == nil {
		return false, nil
	}
	n, err := CountBoardsInCourse(ctx, pool, courseCode)
	if err != nil {
		return false, err
	}
	return n >= *pol.BoardCapPerCourse, nil
}

// ExternalSharingAllowed combines platform flag + org policy (both must allow).
func ExternalSharingAllowed(platformFlag bool, pol OrgPolicies) bool {
	return platformFlag && pol.ExternalSharing
}

func scanOrgPolicies(row pgx.Row) (OrgPolicies, error) {
	var p OrgPolicies
	var orgID uuid.UUID
	var cap *int
	if err := row.Scan(
		&orgID, &p.ExternalSharing, &p.MinorModerationFloor, &p.DefaultAttribution, &cap, &p.UpdatedAt,
	); err != nil {
		return OrgPolicies{}, err
	}
	p.OrgID = orgID.String()
	p.BoardCapPerCourse = cap
	if strings.TrimSpace(p.DefaultAttribution) == "" {
		p.DefaultAttribution = AttributionNamed
	}
	return p, nil
}
