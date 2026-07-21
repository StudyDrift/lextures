// Package parentassign orchestrates staff assignment of parent/guardian links (PP.1).
package parentassign

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	apw "github.com/lextures/lextures/server/internal/auth/hibp"
	pauth "github.com/lextures/lextures/server/internal/auth"
	mailpkg "github.com/lextures/lextures/server/internal/mail"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	"github.com/lextures/lextures/server/internal/repos/parentlinkinvites"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	platformpeople "github.com/lextures/lextures/server/internal/repos/platformpeople"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

const (
	StatusLinked  = "linked"
	StatusInvited = "invited"
	StatusError   = "error"

	inviteTTL = time.Hour

	EventParentLinkAssign   = "parent_link_assign"
	EventParentLinkInvite   = "parent_link_invite"
	EventParentLinkActivate = "parent_link_activate"
	EventParentLinkResend   = "parent_link_resend"
	EventParentLinkRevoke   = "parent_link_revoke"
)

// GuardianInput is one guardian row from the assign modal.
type GuardianInput struct {
	Name         string
	Email        string
	Relationship string
}

// GuardianResult is the per-email outcome of an assign call.
type GuardianResult struct {
	Email        string  `json:"email"`
	Status       string  `json:"status"`
	LinkID       *string `json:"linkId,omitempty"`
	ParentUserID *string `json:"parentUserId,omitempty"`
	Message      *string `json:"message,omitempty"`
}

// StudentHit is a student search result.
type StudentHit struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName,omitempty"`
	Sid         *string `json:"sid,omitempty"`
}

// Deps are injectable collaborators for the assign service.
type Deps struct {
	Pool   *pgxpool.Pool
	Config config.Config
	HIBP   apw.Checker
}

var (
	ErrInvalidToken   = errors.New("parentassign: invalid or expired invite token")
	ErrInvalidInput   = errors.New("parentassign: invalid input")
	ErrNotFound       = errors.New("parentassign: not found")
	ErrNotPending     = errors.New("parentassign: link is not pending")
	ErrStudentAccount = errors.New("parentassign: email belongs to a student account")
)

// ValidateGuardians checks 1–3 rows with name + valid email.
func ValidateGuardians(rows []GuardianInput) error {
	if len(rows) < 1 || len(rows) > 3 {
		return fmt.Errorf("%w: provide 1 to 3 guardians", ErrInvalidInput)
	}
	seen := map[string]struct{}{}
	for i, g := range rows {
		name := strings.TrimSpace(g.Name)
		email := user.NormalizeEmail(g.Email)
		if name == "" {
			return fmt.Errorf("%w: guardian %d name is required", ErrInvalidInput, i+1)
		}
		if email == "" || !strings.Contains(email, "@") || len(email) > 254 {
			return fmt.Errorf("%w: guardian %d email is invalid", ErrInvalidInput, i+1)
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("%w: guardian %d email is invalid", ErrInvalidInput, i+1)
		}
		rel := normalizeRelationship(g.Relationship)
		if rel == "" {
			return fmt.Errorf("%w: guardian %d relationship is invalid", ErrInvalidInput, i+1)
		}
		if _, ok := seen[email]; ok {
			return fmt.Errorf("%w: duplicate email %s", ErrInvalidInput, email)
		}
		seen[email] = struct{}{}
	}
	return nil
}

func normalizeRelationship(rel string) string {
	r := strings.TrimSpace(strings.ToLower(rel))
	if r == "" {
		return "parent"
	}
	switch r {
	case "parent", "guardian", "other":
		return r
	default:
		return ""
	}
}

// SearchStudents finds active non-parent users in the org by name, email, or SID.
func SearchStudents(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, q string, limit int) ([]StudentHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []StudentHit{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	like := "%" + strings.ToLower(q) + "%"
	rows, err := pool.Query(ctx, `
SELECT id::text, email, display_name, sid
FROM "user".users
WHERE org_id = $1
  AND account_type <> 'parent'
  AND deactivated_at IS NULL
  AND login_blocked = false
  AND (
    LOWER(email) LIKE $2
    OR LOWER(COALESCE(display_name, '')) LIKE $2
    OR COALESCE(sid, '') ILIKE $2
  )
ORDER BY display_name NULLS LAST, email
LIMIT $3
`, orgID, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudentHit
	for rows.Next() {
		var h StudentHit
		var dn, sid *string
		if err := rows.Scan(&h.ID, &h.Email, &dn, &sid); err != nil {
			return nil, err
		}
		h.DisplayName = dn
		h.Sid = sid
		out = append(out, h)
	}
	return out, rows.Err()
}

// AssignGuardians processes 1–3 guardian rows for a student.
func (d Deps) AssignGuardians(
	ctx context.Context,
	orgID, studentID, actorID uuid.UUID,
	guardians []GuardianInput,
	actorIP, userAgent *string,
) ([]GuardianResult, error) {
	if err := ValidateGuardians(guardians); err != nil {
		return nil, err
	}
	if err := d.assertAssignableStudent(ctx, orgID, studentID); err != nil {
		return nil, err
	}
	results := make([]GuardianResult, 0, len(guardians))
	for _, g := range guardians {
		results = append(results, d.assignOne(ctx, orgID, studentID, actorID, g, actorIP, userAgent))
	}
	return results, nil
}

func (d Deps) assignOne(
	ctx context.Context,
	orgID, studentID, actorID uuid.UUID,
	g GuardianInput,
	actorIP, userAgent *string,
) GuardianResult {
	email := user.NormalizeEmail(g.Email)
	name := strings.TrimSpace(g.Name)
	rel := normalizeRelationship(g.Relationship)
	res := GuardianResult{Email: email}

	existing, err := user.FindByEmail(ctx, d.Pool, email)
	if err != nil {
		msg := "Failed to look up user."
		res.Status = StatusError
		res.Message = &msg
		return res
	}
	if existing != nil {
		parentID, perr := uuid.Parse(existing.ID)
		if perr != nil {
			msg := "Invalid user id."
			res.Status = StatusError
			res.Message = &msg
			return res
		}
		if parentID == studentID {
			msg := "Cannot link a student as their own parent."
			res.Status = StatusError
			res.Message = &msg
			return res
		}
		orgOfParent, oerr := organization.OrgIDForUser(ctx, d.Pool, parentID)
		if oerr != nil || orgOfParent != orgID {
			msg := "User must belong to this organization."
			res.Status = StatusError
			res.Message = &msg
			return res
		}
		if existing.DeactivatedAt != nil {
			msg := "Cannot link a deactivated account."
			res.Status = StatusError
			res.Message = &msg
			return res
		}
		if existing.AccountType != user.AccountTypeParent {
			isStudent, serr := userHasStudentRole(ctx, d.Pool, parentID)
			if serr != nil {
				msg := "Failed to verify account type."
				res.Status = StatusError
				res.Message = &msg
				return res
			}
			if isStudent {
				msg := "This email belongs to a student account. Use a different email for the parent/guardian."
				res.Status = StatusError
				res.Message = &msg
				d.audit(ctx, orgID, actorID, EventParentLinkAssign, &parentID, actorIP, userAgent, map[string]any{
					"studentId": studentID.String(),
					"email":     email,
					"outcome":   "error",
					"reason":    "student_account",
				})
				return res
			}
		}
		ln, lerr := parentlinks.UpsertActive(ctx, d.Pool, orgID, parentID, studentID, rel, &actorID)
		if lerr != nil {
			msg := "Failed to create link."
			res.Status = StatusError
			res.Message = &msg
			return res
		}
		_ = markParentAccount(ctx, d.Pool, parentID)
		_ = rbac.AssignUserRoleByName(ctx, d.Pool, parentID, "Parent")
		lid := ln.ID.String()
		pid := parentID.String()
		res.Status = StatusLinked
		res.LinkID = &lid
		res.ParentUserID = &pid
		d.audit(ctx, orgID, actorID, EventParentLinkAssign, &parentID, actorIP, userAgent, map[string]any{
			"studentId": studentID.String(),
			"email":     email,
			"linkId":    lid,
			"outcome":   StatusLinked,
		})
		return res
	}

	// Invite path: provision placeholder + pending link + activate email.
	ph, err := authservice.PlaceholderPasswordHash()
	if err != nil {
		msg := "Failed to provision user."
		res.Status = StatusError
		res.Message = &msg
		return res
	}
	dn := name
	uid, err := platformpeople.InsertUser(ctx, d.Pool, orgID, email, ph, dn, nil, nil)
	if err != nil {
		msg := "Failed to create parent account."
		res.Status = StatusError
		res.Message = &msg
		return res
	}
	_ = markParentAccount(ctx, d.Pool, uid)
	_ = rbac.AssignUserRoleByName(ctx, d.Pool, uid, "Parent")

	ln, err := parentlinks.UpsertPending(ctx, d.Pool, orgID, uid, studentID, rel, &actorID)
	if err != nil {
		msg := "Failed to create pending link."
		res.Status = StatusError
		res.Message = &msg
		return res
	}
	if ln.Status == "active" {
		// Race: already active somehow.
		lid := ln.ID.String()
		pid := uid.String()
		res.Status = StatusLinked
		res.LinkID = &lid
		res.ParentUserID = &pid
		return res
	}
	if err := d.sendInvite(ctx, orgID, studentID, uid, ln.ID, email, name, &actorID); err != nil {
		slog.Warn("parentassign.invite_email_failed", "err", err, "org_id", orgID, "student_id", studentID)
		msg := "Parent account created but invite email could not be sent. Use Resend."
		lid := ln.ID.String()
		pid := uid.String()
		res.Status = StatusInvited
		res.LinkID = &lid
		res.ParentUserID = &pid
		res.Message = &msg
		d.audit(ctx, orgID, actorID, EventParentLinkInvite, &uid, actorIP, userAgent, map[string]any{
			"studentId": studentID.String(),
			"email":     email,
			"linkId":    lid,
			"outcome":   "invited_email_failed",
		})
		return res
	}
	lid := ln.ID.String()
	pid := uid.String()
	res.Status = StatusInvited
	res.LinkID = &lid
	res.ParentUserID = &pid
	d.audit(ctx, orgID, actorID, EventParentLinkInvite, &uid, actorIP, userAgent, map[string]any{
		"studentId": studentID.String(),
		"email":     email,
		"linkId":    lid,
		"outcome":   StatusInvited,
	})
	return res
}

// ResendInvite regenerates the activate token for a pending link and emails it.
func (d Deps) ResendInvite(
	ctx context.Context,
	orgID, linkID, actorID uuid.UUID,
	actorIP, userAgent *string,
) error {
	ln, err := parentlinks.GetByID(ctx, d.Pool, orgID, linkID)
	if err != nil {
		return err
	}
	if ln == nil {
		return ErrNotFound
	}
	if ln.Status != "pending" {
		return ErrNotPending
	}
	name := ""
	if ln.ParentDisplay != nil {
		name = *ln.ParentDisplay
	}
	if err := d.sendInvite(ctx, orgID, ln.StudentUserID, ln.ParentUserID, ln.ID, ln.ParentEmail, name, &actorID); err != nil {
		return err
	}
	d.audit(ctx, orgID, actorID, EventParentLinkResend, &ln.ParentUserID, actorIP, userAgent, map[string]any{
		"studentId": ln.StudentUserID.String(),
		"email":     ln.ParentEmail,
		"linkId":    linkID.String(),
		"outcome":   "resent",
	})
	return nil
}

// ConsumeInvite sets the password, activates pending links, and marks the token used.
func (d Deps) ConsumeInvite(ctx context.Context, rawToken, password string) (uuid.UUID, error) {
	tok := strings.TrimSpace(rawToken)
	if tok == "" {
		return uuid.Nil, ErrInvalidToken
	}
	inv, err := parentlinkinvites.FindByTokenHash(ctx, d.Pool, parentlinkinvites.HashToken(tok))
	if err != nil {
		return uuid.Nil, err
	}
	if inv == nil {
		return uuid.Nil, ErrInvalidToken
	}
	if time.Now().UTC().After(inv.ExpiresAt) {
		return uuid.Nil, ErrInvalidToken
	}
	if _, err := authservice.EnforceNewPassword(ctx, d.Pool, nil, password, d.HIBP); err != nil {
		return uuid.Nil, err
	}
	ph, err := pauth.HashPassword(password)
	if err != nil {
		return uuid.Nil, err
	}
	ok, err := parentlinkinvites.MarkConsumed(ctx, d.Pool, inv.ID)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}
	if _, err := d.Pool.Exec(ctx, `
UPDATE "user".users SET password_hash = $2, account_type = 'parent' WHERE id = $1
`, inv.ParentUserID, ph); err != nil {
		return uuid.Nil, err
	}
	_ = rbac.AssignUserRoleByName(ctx, d.Pool, inv.ParentUserID, "Parent")
	if _, err := parentlinks.ActivatePendingForParent(ctx, d.Pool, inv.ParentUserID); err != nil {
		return uuid.Nil, err
	}
	_ = authservice.RevokeAllSessionsForUser(ctx, d.Pool, inv.ParentUserID)
	_ = authservice.InvalidatePasswordJWTs(ctx, d.Pool, inv.ParentUserID)
	d.audit(ctx, inv.OrgID, inv.ParentUserID, EventParentLinkActivate, &inv.ParentUserID, nil, nil, map[string]any{
		"studentId": inv.StudentUserID.String(),
		"email":     inv.Email,
		"linkId":    inv.LinkID.String(),
		"outcome":   "success",
	})
	return inv.ParentUserID, nil
}

func (d Deps) sendInvite(
	ctx context.Context,
	orgID, studentID, parentID, linkID uuid.UUID,
	email, displayName string,
	invitedBy *uuid.UUID,
) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := parentlinkinvites.HashToken(token)
	exp := time.Now().UTC().Add(inviteTTL)
	if _, err := parentlinkinvites.ReplaceForLink(ctx, d.Pool, orgID, studentID, parentID, linkID, email, invitedBy, hash, exp); err != nil {
		return err
	}
	origin := strings.TrimRight(strings.TrimSpace(d.Config.PublicWebOrigin), "/")
	activateURL := fmt.Sprintf("%s/activate-parent?token=%s", origin, token)

	studentName := "your student"
	su, err := user.FindByID(ctx, d.Pool, studentID)
	if err == nil && su != nil {
		if su.DisplayName != nil && strings.TrimSpace(*su.DisplayName) != "" {
			studentName = strings.TrimSpace(*su.DisplayName)
		} else {
			studentName = su.Email
		}
	}
	orgName := "Your school"
	if org, oerr := organization.GetByID(ctx, d.Pool, orgID); oerr == nil && org != nil && strings.TrimSpace(org.Name) != "" {
		orgName = org.Name
	}
	firstName := displayName
	if parts := strings.Fields(displayName); len(parts) > 0 {
		firstName = parts[0]
	}

	opts := &mailpkg.ParentGuardianInviteOpts{
		OrgID:   &orgID,
		Context: ctx,
	}
	if br, berr := orgbranding.Get(ctx, d.Pool, orgID); berr == nil && br != nil {
		opts.PrimaryColor = br.PrimaryColor
		opts.FromDisplayName = br.CustomEmailDisplayName
		opts.LogoURL = br.LogoURL
	}
	return mailpkg.SendParentGuardianInviteEmail(d.Config, email, activateURL, studentName, orgName, firstName, opts)
}

func (d Deps) assertAssignableStudent(ctx context.Context, orgID, studentID uuid.UUID) error {
	row, err := user.FindByID(ctx, d.Pool, studentID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNotFound
	}
	if row.AccountType == user.AccountTypeParent {
		return fmt.Errorf("%w: selected user is a parent account", ErrInvalidInput)
	}
	if row.DeactivatedAt != nil {
		return fmt.Errorf("%w: student account is deactivated", ErrInvalidInput)
	}
	oid, err := organization.OrgIDForUser(ctx, d.Pool, studentID)
	if err != nil || oid != orgID {
		return fmt.Errorf("%w: student must belong to this organization", ErrInvalidInput)
	}
	return nil
}

func markParentAccount(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE "user".users SET account_type = 'parent' WHERE id = $1`, parentID)
	return err
}

func userHasStudentRole(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM "user".user_app_roles uar
  INNER JOIN "user".app_roles ar ON ar.id = uar.role_id
  WHERE uar.user_id = $1 AND ar.name = 'Student'
)
`, userID).Scan(&exists)
	return exists, err
}

func (d Deps) audit(
	ctx context.Context,
	orgID, actorID uuid.UUID,
	eventType string,
	targetID *uuid.UUID,
	actorIP, userAgent *string,
	after map[string]any,
) {
	var afterBytes []byte
	if after != nil {
		afterBytes, _ = json.Marshal(after)
	}
	tt := "user"
	_, _ = auditservice.Record(ctx, d.Pool, auditservice.RecordParams{
		OrgID:      &orgID,
		EventType:  eventType,
		ActorID:    actorID,
		ActorIP:    actorIP,
		UserAgent:  userAgent,
		TargetType: &tt,
		TargetID:   targetID,
		AfterValue: afterBytes,
	})
}
