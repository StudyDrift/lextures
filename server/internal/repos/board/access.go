package board

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/groupspaces"
)

// Visibility scopes (VC.6).
const (
	VisibilityCourse  = "course"
	VisibilitySection = "section"
	VisibilityGroup   = "group"
	VisibilityInvite  = "invite"
	VisibilityLink    = "link"
	VisibilityPublic  = "public"
)

// Attribution modes (VC.6).
const (
	AttributionNamed       = "named"
	AttributionAnonToPeers = "anon_to_peers"
	AttributionAnonymous   = "anonymous"
)

// Member roles for invite boards (VC.6).
const (
	MemberRoleOwner       = "owner"
	MemberRoleEditor      = "editor"
	MemberRoleContributor = "contributor"
	MemberRoleViewer      = "viewer"
)

// Capabilities is the effective permission set for a viewer on a board (FR-7).
type Capabilities struct {
	CanView     bool
	CanPost     bool
	CanInteract bool
	CanArrange  bool
	CanManage   bool
}

// ResolveOpts controls access resolution (authenticated course path vs share link).
type ResolveOpts struct {
	// CourseCode is required for authenticated resolution.
	CourseCode string
	// ExternalSharingAllowed gates link/public visibility and share-link use.
	ExternalSharingAllowed bool
	// ForbidExternalForMinors blocks link/public when the course is age-gated (COPPA).
	ForbidExternalForMinors bool
	// ShareCapability is set when resolving via a valid share link (view|contribute).
	ShareCapability string
	// ViaShareLink is true when access is granted through a share token (not course enrollment).
	ViaShareLink bool
}

// NormalizeVisibility validates and normalizes a visibility value.
func NormalizeVisibility(raw string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case VisibilityCourse, VisibilitySection, VisibilityGroup, VisibilityInvite, VisibilityLink, VisibilityPublic:
		return v, nil
	default:
		return "", fmt.Errorf("board: invalid visibility")
	}
}

// NormalizeAttribution validates and normalizes an attribution value.
func NormalizeAttribution(raw string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case AttributionNamed, AttributionAnonToPeers, AttributionAnonymous:
		return v, nil
	default:
		return "", fmt.Errorf("board: invalid attribution")
	}
}

// NormalizeMemberRole validates a board member role.
func NormalizeMemberRole(raw string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case MemberRoleOwner, MemberRoleEditor, MemberRoleContributor, MemberRoleViewer:
		return v, nil
	default:
		return "", fmt.Errorf("board: invalid member role")
	}
}

// RevealAuthor reports whether the viewer may see post/comment author ids (FR-6).
func RevealAuthor(attribution string, caps Capabilities) bool {
	switch attribution {
	case AttributionNamed:
		return true
	case AttributionAnonToPeers:
		return caps.CanManage
	case AttributionAnonymous:
		return false
	default:
		return true
	}
}

// CourseHasEnrolledMinors returns true when any active enrollment is flagged is_minor (COPPA hook).
func CourseHasEnrolledMinors(ctx context.Context, pool *pgxpool.Pool, courseCode string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM course.course_enrollments e
			INNER JOIN course.courses c ON c.id = e.course_id
			INNER JOIN "user".users u ON u.id = e.user_id
			WHERE c.course_code = $1 AND e.active AND COALESCE(u.is_minor, FALSE)
		)
	`, courseCode).Scan(&ok)
	return ok, err
}

// ExternalSharingBlocked reports whether link/public must be refused for this course.
// externalAllowed should already combine the platform flag and org policy (VC.10).
func ExternalSharingBlocked(ctx context.Context, pool *pgxpool.Pool, courseCode string, externalAllowed, coppaEnabled bool) (blocked bool, reason string, err error) {
	if !externalAllowed {
		return true, "external_sharing_disabled", nil
	}
	if !coppaEnabled {
		return false, "", nil
	}
	hasMinors, err := CourseHasEnrolledMinors(ctx, pool, courseCode)
	if err != nil {
		return false, "", err
	}
	if hasMinors {
		return true, "minors_policy", nil
	}
	return false, "", nil
}

// ResolveAccess computes capabilities for userID on board b within courseCode.
// userID may be uuid.Nil for anonymous share-link visitors (ViaShareLink must be true).
func ResolveAccess(ctx context.Context, pool *pgxpool.Pool, b *Board, userID uuid.UUID, opts ResolveOpts) (Capabilities, error) {
	if b == nil {
		return Capabilities{}, errors.New("board: nil board")
	}
	var caps Capabilities

	if opts.ViaShareLink {
		return resolveShareLinkCaps(b, opts), nil
	}

	courseCode := strings.TrimSpace(opts.CourseCode)
	if courseCode == "" {
		return Capabilities{}, errors.New("board: course code required")
	}
	if userID == uuid.Nil {
		return Capabilities{}, nil
	}

	isManager, err := courseroles.UserHasPermission(ctx, pool, userID, "course:"+courseCode+":item:create")
	if err != nil {
		return Capabilities{}, err
	}
	if isManager {
		return Capabilities{
			CanView: true, CanPost: true, CanInteract: true, CanArrange: true, CanManage: true,
		}, nil
	}

	inScope, memberRole, err := userInBoardScope(ctx, pool, b, courseCode, userID)
	if err != nil {
		return Capabilities{}, err
	}
	if !inScope {
		return Capabilities{}, nil
	}

	// Public boards are read-only for non-managers (FR-5).
	if b.Visibility == VisibilityPublic {
		return Capabilities{CanView: true}, nil
	}

	switch memberRole {
	case MemberRoleOwner, MemberRoleEditor:
		return Capabilities{
			CanView: true, CanPost: true, CanInteract: true, CanArrange: true, CanManage: true,
		}, nil
	case MemberRoleViewer:
		return Capabilities{CanView: true}, nil
	default:
		// contributor or course-scoped member with no explicit role
		caps.CanView = true
		caps.CanPost = b.CanPost
		caps.CanInteract = b.CanInteract
		caps.CanArrange = b.CanArrange
		return caps, nil
	}
}

func resolveShareLinkCaps(b *Board, opts ResolveOpts) Capabilities {
	if !opts.ExternalSharingAllowed || opts.ForbidExternalForMinors {
		return Capabilities{}
	}
	// Share links may exist on course boards for guest access; still honor capability.
	cap := strings.ToLower(strings.TrimSpace(opts.ShareCapability))
	switch cap {
	case ShareCapabilityContribute:
		return Capabilities{CanView: true, CanPost: true}
	case ShareCapabilityView:
		return Capabilities{CanView: true}
	default:
		return Capabilities{}
	}
}

func userInBoardScope(ctx context.Context, pool *pgxpool.Pool, b *Board, courseCode string, userID uuid.UUID) (bool, string, error) {
	hasAccess, err := enrollment.UserHasAccess(ctx, pool, courseCode, userID)
	if err != nil {
		return false, "", err
	}

	switch b.Visibility {
	case VisibilityCourse:
		return hasAccess, "", nil
	case VisibilityPublic:
		// Enrolled members can open via course UI; anonymous uses share/public endpoints.
		return hasAccess, "", nil
	case VisibilityLink:
		// Unlisted: not listed for regular members; managers handled above.
		// Explicit members still allowed if present.
		role, err := GetMemberRole(ctx, pool, b.ID, userID)
		if err != nil {
			return false, "", err
		}
		if role != "" {
			return true, role, nil
		}
		return false, "", nil
	case VisibilityInvite:
		role, err := GetMemberRole(ctx, pool, b.ID, userID)
		if err != nil {
			return false, "", err
		}
		if role == "" {
			return false, "", nil
		}
		return true, role, nil
	case VisibilitySection:
		if !hasAccess {
			return false, "", nil
		}
		if b.VisibilityTarget == nil {
			return false, "", nil
		}
		courseUUID, err := uuid.Parse(b.CourseID)
		if err != nil {
			return false, "", err
		}
		sid, err := enrollment.GetStudentSectionID(ctx, pool, courseUUID, userID)
		if err != nil {
			return false, "", err
		}
		if sid == nil || sid.String() != *b.VisibilityTarget {
			return false, "", nil
		}
		return true, "", nil
	case VisibilityGroup:
		if !hasAccess {
			return false, "", nil
		}
		if b.VisibilityTarget == nil {
			return false, "", nil
		}
		gid, err := uuid.Parse(*b.VisibilityTarget)
		if err != nil {
			return false, "", nil
		}
		ok, err := groupspaces.IsGroupMember(ctx, pool, courseCode, gid, userID)
		if err != nil {
			return false, "", err
		}
		return ok, "", nil
	default:
		return hasAccess, "", nil
	}
}

// ValidateVisibilityTarget ensures section/group targets belong to the course.
func ValidateVisibilityTarget(ctx context.Context, pool *pgxpool.Pool, courseCode, visibility string, target *string) error {
	vis, err := NormalizeVisibility(visibility)
	if err != nil {
		return err
	}
	needsTarget := vis == VisibilitySection || vis == VisibilityGroup
	if !needsTarget {
		return nil
	}
	if target == nil || strings.TrimSpace(*target) == "" {
		return fmt.Errorf("board: visibility_target is required for %s visibility", vis)
	}
	tid, err := uuid.Parse(strings.TrimSpace(*target))
	if err != nil {
		return fmt.Errorf("board: invalid visibility_target")
	}
	switch vis {
	case VisibilitySection:
		var ok bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM course.course_sections s
				INNER JOIN course.courses c ON c.id = s.course_id
				WHERE s.id = $1 AND c.course_code = $2
			)
		`, tid, courseCode).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("board: visibility_target section not found in course")
		}
	case VisibilityGroup:
		g, err := groupspaces.GetGroupByCourseAndID(ctx, pool, courseCode, tid)
		if err != nil {
			return err
		}
		if g == nil {
			return fmt.Errorf("board: visibility_target group not found in course")
		}
	}
	return nil
}

// GetBoardCourseCode returns the course_code for a board id.
func GetBoardCourseCode(ctx context.Context, pool *pgxpool.Pool, boardID string) (string, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return "", nil
	}
	var code string
	err = pool.QueryRow(ctx, `
		SELECT c.course_code
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE b.id = $1
	`, id).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return code, err
}
