package scim

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// GroupMember is a SCIM group member reference.
type GroupMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// GroupResource is the SCIM Group JSON body we accept and emit.
type GroupResource struct {
	Schemas     []string       `json:"schemas,omitempty"`
	ID          string         `json:"id,omitempty"`
	ExternalID  string         `json:"externalId,omitempty"`
	DisplayName string         `json:"displayName,omitempty"`
	Members     []GroupMember  `json:"members,omitempty"`
	Meta        *Meta          `json:"meta,omitempty"`
}

type groupListResponse struct {
	Schemas      []string         `json:"schemas"`
	TotalResults int              `json:"totalResults"`
	StartIndex   int              `json:"startIndex"`
	ItemsPerPage int              `json:"itemsPerPage"`
	Resources    []*GroupResource `json:"Resources"`
}

type groupMapping struct {
	Kind           string
	AppRoleName    string
	OrgRoleKey     string
	CourseCode     string
	EnrollmentRole string
}

func normalizeGroupDisplayName(s string) string {
	return strings.TrimSpace(s)
}

func findGroupID(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, scimGroupID string) (uuid.UUID, error) {
	scimGroupID = strings.TrimSpace(scimGroupID)
	if scimGroupID == "" {
		return uuid.UUID{}, ErrNotFound
	}
	if id, err := uuid.Parse(scimGroupID); err == nil {
		var ok bool
		err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM provisioning.scim_groups WHERE id = $1 AND institution_id = $2
)`, id, institutionID).Scan(&ok)
		if err != nil {
			return uuid.UUID{}, err
		}
		if !ok {
			return uuid.UUID{}, ErrNotFound
		}
		return id, nil
	}
	var gid uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id FROM provisioning.scim_groups
WHERE institution_id = $1 AND external_id = $2
`, institutionID, scimGroupID).Scan(&gid)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, ErrNotFound
	}
	return gid, err
}

func resolveMemberUserID(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, memberValue string) (uuid.UUID, error) {
	return findBoundUserID(ctx, pool, institutionID, memberValue)
}

func listGroupMappings(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, displayName string) ([]groupMapping, error) {
	rows, err := pool.Query(ctx, `
SELECT mapping_kind, COALESCE(app_role_name, ''), COALESCE(org_role_key, ''), COALESCE(course_code, ''), enrollment_role
FROM provisioning.scim_group_mappings
WHERE institution_id = $1 AND lower(display_name) = lower($2)
`, institutionID, displayName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []groupMapping
	for rows.Next() {
		var m groupMapping
		if err := rows.Scan(&m.Kind, &m.AppRoleName, &m.OrgRoleKey, &m.CourseCode, &m.EnrollmentRole); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func appendProvisioningRoleMap(ctx context.Context, pool *pgxpool.Pool, displayName string, mappings []groupMapping) ([]groupMapping, error) {
	res, err := rbac.LookupProvisioningRole(ctx, pool, "scim", displayName)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return mappings, nil
	}
	for _, m := range mappings {
		if m.Kind == "app_role" && strings.EqualFold(m.AppRoleName, res.AppRoleName) {
			return mappings, nil
		}
	}
	return append(mappings, groupMapping{Kind: "app_role", AppRoleName: res.AppRoleName}), nil
}

func resolveGroupMappings(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, displayName string) ([]groupMapping, error) {
	mappings, err := listGroupMappings(ctx, pool, institutionID, displayName)
	if err != nil {
		return nil, err
	}
	return appendProvisioningRoleMap(ctx, pool, displayName, mappings)
}

func listUserGroupDisplayNames(ctx context.Context, q pgx.Tx, institutionID, userID uuid.UUID, excludeGroupID *uuid.UUID) ([]string, error) {
	var rows pgx.Rows
	var err error
	if excludeGroupID != nil {
		rows, err = q.Query(ctx, `
SELECT g.display_name
FROM provisioning.scim_group_members m
INNER JOIN provisioning.scim_groups g ON g.id = m.group_id
WHERE g.institution_id = $1 AND m.user_id = $2 AND g.id <> $3
`, institutionID, userID, *excludeGroupID)
	} else {
		rows, err = q.Query(ctx, `
SELECT g.display_name
FROM provisioning.scim_group_members m
INNER JOIN provisioning.scim_groups g ON g.id = m.group_id
WHERE g.institution_id = $1 AND m.user_id = $2
`, institutionID, userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var dn string
		if err := rows.Scan(&dn); err != nil {
			return nil, err
		}
		names = append(names, dn)
	}
	return names, rows.Err()
}

func aggregateMappings(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, displayNames []string) ([]groupMapping, error) {
	seen := make(map[string]struct{})
	var out []groupMapping
	for _, dn := range displayNames {
		mappings, err := resolveGroupMappings(ctx, pool, institutionID, dn)
		if err != nil {
			return nil, err
		}
		for _, m := range mappings {
			key := m.Kind + "|" + strings.ToLower(m.AppRoleName) + "|" + m.OrgRoleKey + "|" + strings.ToLower(m.CourseCode) + "|" + m.EnrollmentRole
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, m)
		}
	}
	return out, nil
}

func applyGroupMappingTx(ctx context.Context, tx pgx.Tx, orgID uuid.UUID, userID uuid.UUID, m groupMapping) error {
	switch m.Kind {
	case "app_role":
		if strings.TrimSpace(m.AppRoleName) == "" {
			return nil
		}
		return rbac.AssignUserRoleByNameTx(ctx, tx, userID, m.AppRoleName)
	case "org_role":
		if strings.TrimSpace(m.OrgRoleKey) == "" {
			return nil
		}
		_, err := tx.Exec(ctx, `
INSERT INTO "user".org_role_grants (org_id, user_id, org_unit_id, role)
VALUES ($1, $2, NULL, $3)
ON CONFLICT (org_id, user_id, role, org_unit_id) DO UPDATE SET granted_at = NOW()
`, orgID, userID, m.OrgRoleKey)
		return err
	case "course_enrollment":
		courseCode := strings.TrimSpace(m.CourseCode)
		if courseCode == "" {
			return nil
		}
		role := strings.TrimSpace(m.EnrollmentRole)
		if role == "" {
			role = "student"
		}
		_, err := tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active)
SELECT c.id, $2, $3, TRUE
FROM course.courses c
WHERE c.course_code = $1
ON CONFLICT (course_id, user_id, role) DO UPDATE SET active = TRUE, invitation_pending = FALSE
`, courseCode, userID, role)
		return err
	default:
		return nil
	}
}

func revokeGroupMappingTx(ctx context.Context, tx pgx.Tx, orgID, userID uuid.UUID, m groupMapping) error {
	switch m.Kind {
	case "app_role":
		if strings.TrimSpace(m.AppRoleName) == "" {
			return nil
		}
		var roleID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM "user".app_roles WHERE name = $1`, m.AppRoleName).Scan(&roleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM "user".user_app_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
		return err
	case "org_role":
		if strings.TrimSpace(m.OrgRoleKey) == "" {
			return nil
		}
		_, err := tx.Exec(ctx, `
DELETE FROM "user".org_role_grants
WHERE org_id = $1 AND user_id = $2 AND role = $3 AND org_unit_id IS NULL
`, orgID, userID, m.OrgRoleKey)
		return err
	case "course_enrollment":
		courseCode := strings.TrimSpace(m.CourseCode)
		if courseCode == "" {
			return nil
		}
		role := strings.TrimSpace(m.EnrollmentRole)
		if role == "" {
			role = "student"
		}
		_, err := tx.Exec(ctx, `
UPDATE course.course_enrollments ce
SET active = FALSE
FROM course.courses c
WHERE ce.course_id = c.id AND c.course_code = $1 AND ce.user_id = $2 AND ce.role = $3
`, courseCode, userID, role)
		return err
	default:
		return nil
	}
}

func syncMembershipGrantsTx(ctx context.Context, pool *pgxpool.Pool, tx pgx.Tx, institutionID, orgID, userID uuid.UUID, added, removed []groupMapping) error {
	for _, m := range added {
		if err := applyGroupMappingTx(ctx, tx, orgID, userID, m); err != nil {
			return err
		}
	}
	for _, m := range removed {
		if err := revokeGroupMappingTx(ctx, tx, orgID, userID, m); err != nil {
			return err
		}
	}
	return nil
}

func diffMappings(before, after []groupMapping) (added, removed []groupMapping) {
	afterSet := make(map[string]groupMapping, len(after))
	for _, m := range after {
		afterSet[mappingKey(m)] = m
	}
	beforeSet := make(map[string]groupMapping, len(before))
	for _, m := range before {
		beforeSet[mappingKey(m)] = m
	}
	for k, m := range afterSet {
		if _, ok := beforeSet[k]; !ok {
			added = append(added, m)
		}
	}
	for k, m := range beforeSet {
		if _, ok := afterSet[k]; !ok {
			removed = append(removed, m)
		}
	}
	return added, removed
}

func mappingKey(m groupMapping) string {
	return m.Kind + "|" + strings.ToLower(m.AppRoleName) + "|" + m.OrgRoleKey + "|" + strings.ToLower(m.CourseCode) + "|" + m.EnrollmentRole
}

func applyMemberAddTx(ctx context.Context, pool *pgxpool.Pool, tx pgx.Tx, institutionID, groupID uuid.UUID, displayName string, userID uuid.UUID) error {
	orgID, err := organization.ResolveOrgIDForProvisioning(ctx, pool, institutionID)
	if err != nil {
		return err
	}
	beforeNames, err := listUserGroupDisplayNames(ctx, tx, institutionID, userID, &groupID)
	if err != nil {
		return err
	}
	beforeMappings, err := aggregateMappings(ctx, pool, institutionID, beforeNames)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
INSERT INTO provisioning.scim_group_members (group_id, user_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, groupID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	afterNames := append(append([]string{}, beforeNames...), displayName)
	afterMappings, err := aggregateMappings(ctx, pool, institutionID, afterNames)
	if err != nil {
		return err
	}
	added, removed := diffMappings(beforeMappings, afterMappings)
	return syncMembershipGrantsTx(ctx, pool, tx, institutionID, orgID, userID, added, removed)
}

func applyMemberRemoveTx(ctx context.Context, pool *pgxpool.Pool, tx pgx.Tx, institutionID, groupID uuid.UUID, userID uuid.UUID) error {
	orgID, err := organization.ResolveOrgIDForProvisioning(ctx, pool, institutionID)
	if err != nil {
		return err
	}
	beforeNames, err := listUserGroupDisplayNames(ctx, tx, institutionID, userID, nil)
	if err != nil {
		return err
	}
	beforeMappings, err := aggregateMappings(ctx, pool, institutionID, beforeNames)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM provisioning.scim_group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	afterNames, err := listUserGroupDisplayNames(ctx, tx, institutionID, userID, &groupID)
	if err != nil {
		return err
	}
	afterMappings, err := aggregateMappings(ctx, pool, institutionID, afterNames)
	if err != nil {
		return err
	}
	added, removed := diffMappings(beforeMappings, afterMappings)
	return syncMembershipGrantsTx(ctx, pool, tx, institutionID, orgID, userID, added, removed)
}

func buildGroupResource(ctx context.Context, pool *pgxpool.Pool, groupID, institutionID uuid.UUID, baseURL string) (*GroupResource, error) {
	var displayName string
	var extID sql.NullString
	var createdAt, updatedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT display_name, external_id, created_at, updated_at
FROM provisioning.scim_groups WHERE id = $1 AND institution_id = $2
`, groupID, institutionID).Scan(&displayName, &extID, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT u.id, u.email, u.display_name
FROM provisioning.scim_group_members m
INNER JOIN "user".users u ON u.id = m.user_id
WHERE m.group_id = $1
ORDER BY u.email ASC
`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []GroupMember
	base := strings.TrimRight(baseURL, "/")
	for rows.Next() {
		var uid uuid.UUID
		var email, dn sql.NullString
		if err := rows.Scan(&uid, &email, &dn); err != nil {
			return nil, err
		}
		ref := base + "/scim/v2/Users/" + uid.String()
		mem := GroupMember{
			Value: uid.String(),
			Type:  "User",
			Ref:   ref,
		}
		if dn.Valid && strings.TrimSpace(dn.String) != "" {
			mem.Display = strings.TrimSpace(dn.String)
		} else if email.Valid {
			mem.Display = strings.TrimSpace(email.String)
		}
		members = append(members, mem)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	loc := base + "/scim/v2/Groups/" + groupID.String()
	res := &GroupResource{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		ID:          groupID.String(),
		DisplayName: displayName,
		Members:     members,
		Meta: &Meta{
			ResourceType: "Group",
			Location:     loc,
			Created:      createdAt.UTC().Format(time.RFC3339),
			LastModified: updatedAt.UTC().Format(time.RFC3339),
		},
	}
	if extID.Valid && strings.TrimSpace(extID.String) != "" {
		res.ExternalID = strings.TrimSpace(extID.String)
	}
	return res, nil
}

// CreateGroup provisions a SCIM group for the institution.
func CreateGroup(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, in *GroupResource, baseURL string) (*GroupResource, error) {
	displayName := normalizeGroupDisplayName(in.DisplayName)
	if displayName == "" {
		return nil, ErrInvalidValue
	}
	extID := strings.TrimSpace(in.ExternalID)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if extID != "" {
		var taken bool
		err = tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM provisioning.scim_groups WHERE institution_id = $1 AND external_id = $2
)`, institutionID, extID).Scan(&taken)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrUniqueness
		}
	}
	var nameTaken bool
	err = tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM provisioning.scim_groups WHERE institution_id = $1 AND lower(display_name) = lower($2)
)`, institutionID, displayName).Scan(&nameTaken)
	if err != nil {
		return nil, err
	}
	if nameTaken {
		return nil, ErrUniqueness
	}

	var extSQL any
	if extID == "" {
		extSQL = nil
	} else {
		extSQL = extID
	}
	var gid uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO provisioning.scim_groups (institution_id, external_id, display_name)
VALUES ($1, $2, $3)
RETURNING id
`, institutionID, extSQL, displayName).Scan(&gid)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return nil, ErrUniqueness
		}
		return nil, err
	}

	for _, mem := range in.Members {
		uid, err := resolveMemberUserID(ctx, pool, institutionID, mem.Value)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		if err := applyMemberAddTx(ctx, pool, tx, institutionID, gid, displayName, uid); err != nil {
			return nil, err
		}
		_ = LogEvent(ctx, pool, institutionID, "member_add", "Group", &uid, map[string]any{"groupId": gid.String(), "displayName": displayName})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = LogEvent(ctx, pool, institutionID, "create", "Group", nil, map[string]any{"groupId": gid.String(), "displayName": displayName})
	return buildGroupResource(ctx, pool, gid, institutionID, baseURL)
}

// GetGroup returns a SCIM group by id or externalId within institution.
func GetGroup(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, scimGroupID, baseURL string) (*GroupResource, error) {
	gid, err := findGroupID(ctx, pool, institutionID, scimGroupID)
	if err != nil {
		return nil, err
	}
	return buildGroupResource(ctx, pool, gid, institutionID, baseURL)
}

// ListGroups returns groups for institution; supports filter=displayName eq "name".
func ListGroups(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, filter, baseURL string) (*groupListResponse, error) {
	displayName := ""
	if f := strings.TrimSpace(filter); f != "" {
		const prefix = `displayName eq "`
		if strings.HasPrefix(strings.ToLower(f), strings.ToLower(prefix)) && strings.HasSuffix(f, `"`) {
			displayName = normalizeGroupDisplayName(f[len(prefix) : len(f)-1])
		}
	}

	var rows pgx.Rows
	var err error
	if displayName != "" {
		rows, err = pool.Query(ctx, `
SELECT id FROM provisioning.scim_groups
WHERE institution_id = $1 AND lower(display_name) = lower($2)
ORDER BY created_at ASC
`, institutionID, displayName)
	} else {
		rows, err = pool.Query(ctx, `
SELECT id FROM provisioning.scim_groups
WHERE institution_id = $1
ORDER BY created_at ASC
LIMIT 1000
`, institutionID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	res := make([]*GroupResource, 0, len(ids))
	for _, id := range ids {
		gr, err := buildGroupResource(ctx, pool, id, institutionID, baseURL)
		if err != nil {
			return nil, err
		}
		res = append(res, gr)
	}
	return &groupListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: len(res),
		StartIndex:   1,
		ItemsPerPage: len(res),
		Resources:    res,
	}, nil
}

// ReplaceGroup full PUT.
func ReplaceGroup(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, scimGroupID string, in *GroupResource, baseURL string) (*GroupResource, error) {
	gid, err := findGroupID(ctx, pool, institutionID, scimGroupID)
	if err != nil {
		return nil, err
	}
	displayName := normalizeGroupDisplayName(in.DisplayName)
	if displayName == "" {
		return nil, ErrInvalidValue
	}
	extID := strings.TrimSpace(in.ExternalID)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldDisplayName string
	if err := tx.QueryRow(ctx, `SELECT display_name FROM provisioning.scim_groups WHERE id = $1`, gid).Scan(&oldDisplayName); err != nil {
		return nil, err
	}

	if extID != "" {
		var conflict bool
		err = tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM provisioning.scim_groups WHERE institution_id = $1 AND external_id = $2 AND id <> $3
)`, institutionID, extID, gid).Scan(&conflict)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, ErrUniqueness
		}
	}
	var nameConflict bool
	err = tx.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM provisioning.scim_groups WHERE institution_id = $1 AND lower(display_name) = lower($2) AND id <> $3
)`, institutionID, displayName, gid).Scan(&nameConflict)
	if err != nil {
		return nil, err
	}
	if nameConflict {
		return nil, ErrUniqueness
	}

	var extSQL any
	if extID == "" {
		extSQL = nil
	} else {
		extSQL = extID
	}
	_, err = tx.Exec(ctx, `
UPDATE provisioning.scim_groups SET display_name = $2, external_id = $3, updated_at = NOW() WHERE id = $1
`, gid, displayName, extSQL)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return nil, ErrUniqueness
		}
		return nil, err
	}

	existingRows, err := tx.Query(ctx, `SELECT user_id FROM provisioning.scim_group_members WHERE group_id = $1`, gid)
	if err != nil {
		return nil, err
	}
	var existing []uuid.UUID
	for existingRows.Next() {
		var uid uuid.UUID
		if err := existingRows.Scan(&uid); err != nil {
			existingRows.Close()
			return nil, err
		}
		existing = append(existing, uid)
	}
	existingRows.Close()
	if err := existingRows.Err(); err != nil {
		return nil, err
	}

	desired := make(map[uuid.UUID]struct{})
	for _, mem := range in.Members {
		uid, err := resolveMemberUserID(ctx, pool, institutionID, mem.Value)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		desired[uid] = struct{}{}
	}

	for _, uid := range existing {
		if _, ok := desired[uid]; ok {
			continue
		}
		if err := applyMemberRemoveTx(ctx, pool, tx, institutionID, gid, uid); err != nil {
			return nil, err
		}
		_ = LogEvent(ctx, pool, institutionID, "member_remove", "Group", &uid, map[string]any{"groupId": gid.String()})
	}
	for uid := range desired {
		found := false
		for _, e := range existing {
			if e == uid {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if err := applyMemberAddTx(ctx, pool, tx, institutionID, gid, displayName, uid); err != nil {
			return nil, err
		}
		_ = LogEvent(ctx, pool, institutionID, "member_add", "Group", &uid, map[string]any{"groupId": gid.String(), "displayName": displayName})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = LogEvent(ctx, pool, institutionID, "update", "Group", nil, map[string]any{"groupId": gid.String(), "displayName": displayName, "previousDisplayName": oldDisplayName})
	return buildGroupResource(ctx, pool, gid, institutionID, baseURL)
}

// PatchGroup applies partial updates (SCIM PATCH).
func PatchGroup(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, scimGroupID string, raw []byte, baseURL string) (*GroupResource, error) {
	gid, err := findGroupID(ctx, pool, institutionID, scimGroupID)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Schemas    []string `json:"schemas"`
		Operations []struct {
			OP    string          `json:"op"`
			Path  string          `json:"path"`
			Value json.RawMessage `json:"value"`
		} `json:"Operations"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, ErrInvalidValue
	}
	if len(envelope.Operations) == 0 {
		return buildGroupResource(ctx, pool, gid, institutionID, baseURL)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var displayName string
	if err := tx.QueryRow(ctx, `SELECT display_name FROM provisioning.scim_groups WHERE id = $1`, gid).Scan(&displayName); err != nil {
		return nil, err
	}

	for _, op := range envelope.Operations {
		switch strings.ToLower(strings.TrimSpace(op.OP)) {
		case "replace":
			p := strings.TrimSpace(op.Path)
			switch {
			case strings.EqualFold(p, "displayName"):
				var s string
				if err := json.Unmarshal(op.Value, &s); err != nil {
					return nil, ErrInvalidValue
				}
				s = normalizeGroupDisplayName(s)
				if s == "" {
					return nil, ErrInvalidValue
				}
				_, err := tx.Exec(ctx, `UPDATE provisioning.scim_groups SET display_name = $2, updated_at = NOW() WHERE id = $1`, gid, s)
				if err != nil {
					var pe *pgconn.PgError
					if errors.As(err, &pe) && pe.Code == "23505" {
						return nil, ErrUniqueness
					}
					return nil, err
				}
				displayName = s
			case strings.EqualFold(p, "externalId"):
				var s string
				if err := json.Unmarshal(op.Value, &s); err != nil {
					return nil, ErrInvalidValue
				}
				s = strings.TrimSpace(s)
				var extSQL any
				if s == "" {
					extSQL = nil
				} else {
					extSQL = s
				}
				_, err := tx.Exec(ctx, `UPDATE provisioning.scim_groups SET external_id = $2, updated_at = NOW() WHERE id = $1`, gid, extSQL)
				if err != nil {
					return nil, err
				}
			case p == "":
				var full GroupResource
				if err := json.Unmarshal(op.Value, &full); err != nil {
					return nil, ErrInvalidValue
				}
				if dn := normalizeGroupDisplayName(full.DisplayName); dn != "" {
					_, err := tx.Exec(ctx, `UPDATE provisioning.scim_groups SET display_name = $2, updated_at = NOW() WHERE id = $1`, gid, dn)
					if err != nil {
						return nil, err
					}
					displayName = dn
				}
			default:
				return nil, ErrInvalidValue
			}
		case "add":
			p := strings.ToLower(strings.TrimSpace(op.Path))
			if p != "members" && p != "" {
				return nil, ErrInvalidValue
			}
			var members []GroupMember
			if err := json.Unmarshal(op.Value, &members); err != nil {
				var one GroupMember
				if err2 := json.Unmarshal(op.Value, &one); err2 != nil {
					return nil, ErrInvalidValue
				}
				members = []GroupMember{one}
			}
			for _, mem := range members {
				uid, err := resolveMemberUserID(ctx, pool, institutionID, mem.Value)
				if err != nil {
					if errors.Is(err, ErrNotFound) {
						continue
					}
					return nil, err
				}
				if err := applyMemberAddTx(ctx, pool, tx, institutionID, gid, displayName, uid); err != nil {
					return nil, err
				}
				_ = LogEvent(ctx, pool, institutionID, "member_add", "Group", &uid, map[string]any{"groupId": gid.String()})
			}
		case "remove":
			p := strings.TrimSpace(op.Path)
			if strings.HasPrefix(strings.ToLower(p), `members[value eq "`) && strings.HasSuffix(p, `"]`) {
				val := p[len(`members[value eq "`): len(p)-2]
				uid, err := resolveMemberUserID(ctx, pool, institutionID, val)
				if err != nil {
					if errors.Is(err, ErrNotFound) {
						continue
					}
					return nil, err
				}
				if err := applyMemberRemoveTx(ctx, pool, tx, institutionID, gid, uid); err != nil {
					return nil, err
				}
				_ = LogEvent(ctx, pool, institutionID, "member_remove", "Group", &uid, map[string]any{"groupId": gid.String()})
				continue
			}
			if strings.EqualFold(p, "members") {
				var members []GroupMember
				if err := json.Unmarshal(op.Value, &members); err != nil {
					var one GroupMember
					if err2 := json.Unmarshal(op.Value, &one); err2 != nil {
						return nil, ErrInvalidValue
					}
					members = []GroupMember{one}
				}
				for _, mem := range members {
					uid, err := resolveMemberUserID(ctx, pool, institutionID, mem.Value)
					if err != nil {
						if errors.Is(err, ErrNotFound) {
							continue
						}
						return nil, err
					}
					if err := applyMemberRemoveTx(ctx, pool, tx, institutionID, gid, uid); err != nil {
						return nil, err
					}
					_ = LogEvent(ctx, pool, institutionID, "member_remove", "Group", &uid, map[string]any{"groupId": gid.String()})
				}
				continue
			}
			return nil, ErrInvalidValue
		default:
			return nil, ErrInvalidValue
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = LogEvent(ctx, pool, institutionID, "update", "Group", nil, map[string]any{"groupId": gid.String(), "patch": true})
	return buildGroupResource(ctx, pool, gid, institutionID, baseURL)
}

// DeleteGroup removes a SCIM group and revokes membership-driven grants.
func DeleteGroup(ctx context.Context, pool *pgxpool.Pool, institutionID uuid.UUID, scimGroupID string) error {
	gid, err := findGroupID(ctx, pool, institutionID, scimGroupID)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	memberRows, err := tx.Query(ctx, `SELECT user_id FROM provisioning.scim_group_members WHERE group_id = $1`, gid)
	if err != nil {
		return err
	}
	var members []uuid.UUID
	for memberRows.Next() {
		var uid uuid.UUID
		if err := memberRows.Scan(&uid); err != nil {
			memberRows.Close()
			return err
		}
		members = append(members, uid)
	}
	memberRows.Close()
	if err := memberRows.Err(); err != nil {
		return err
	}

	for _, uid := range members {
		if err := applyMemberRemoveTx(ctx, pool, tx, institutionID, gid, uid); err != nil {
			return err
		}
		_ = LogEvent(ctx, pool, institutionID, "member_remove", "Group", &uid, map[string]any{"groupId": gid.String(), "groupDeleted": true})
	}

	tag, err := tx.Exec(ctx, `DELETE FROM provisioning.scim_groups WHERE id = $1 AND institution_id = $2`, gid, institutionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_ = LogEvent(ctx, pool, institutionID, "delete", "Group", nil, map[string]any{"groupId": gid.String()})
	return nil
}