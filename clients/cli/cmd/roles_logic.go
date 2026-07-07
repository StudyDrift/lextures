package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/clients/cli/internal/client"
)

const rbacManagePermission = "rbac:manage"

type rbacPermission struct {
	ID               string `json:"id"`
	PermissionString string `json:"permissionString"`
	Description      string `json:"description"`
}

type rbacRole struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Scope       string           `json:"scope"`
	Permissions []rbacPermission `json:"permissions"`
}

type rbacUserBrief struct {
	ID string `json:"id"`
}

type rolesExportFile struct {
	Version int                  `json:"version"`
	Roles   []rolesExportEntry   `json:"roles"`
}

type rolesExportEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
}

type roleApplyDiff struct {
	Create []rolesExportEntry         `json:"create,omitempty"`
	Update []roleApplyUpdate          `json:"update,omitempty"`
	Remove []string                   `json:"remove,omitempty"`
	Perms  []rolePermissionDiff       `json:"permissionChanges,omitempty"`
}

type roleApplyUpdate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

type rolePermissionDiff struct {
	Role    string   `json:"role"`
	Add     []string `json:"add,omitempty"`
	Remove  []string `json:"remove,omitempty"`
}

func settingsPath(subpath string) string {
	return "/api/v1/settings" + subpath
}

func fetchRBACPermissions(c *client.Client) ([]rbacPermission, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, settingsPath("/permissions"), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing permissions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, body, fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Permissions []rbacPermission `json:"permissions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Permissions, body, nil
}

func fetchRBACRoles(c *client.Client) ([]rbacRole, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, settingsPath("/roles"), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, body, fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Roles []rbacRole `json:"roles"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Roles, body, nil
}

func fetchRBACRole(c *client.Client, roleID string) (rbacRole, []byte, error) {
	roles, _, err := fetchRBACRoles(c)
	if err != nil {
		return rbacRole{}, nil, err
	}
	for _, role := range roles {
		if role.ID == roleID || strings.EqualFold(role.Name, roleID) {
			return role, nil, nil
		}
	}
	return rbacRole{}, nil, fmt.Errorf("role %q not found", roleID)
}

func fetchRoleUsers(c *client.Client, roleID string) ([]rbacUserBrief, error) {
	req, err := c.NewRequest(http.MethodGet, settingsPath("/roles/"+url.PathEscape(roleID)+"/users"), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing role users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Users []rbacUserBrief `json:"users"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out.Users, nil
}

func fetchMeUserID(c *client.Client) (string, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", fmt.Errorf("getting current user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	return out.ID, nil
}

func fetchMyPermissionStrings(c *client.Client, courseCode string) ([]string, error) {
	path := "/api/v1/me/permissions"
	if courseCode != "" {
		path += "?courseCode=" + url.QueryEscape(courseCode)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting permissions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		PermissionStrings []string `json:"permissionStrings"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out.PermissionStrings, nil
}

func resolveUserPermissionStrings(c *client.Client, userID string) ([]string, error) {
	roles, _, err := fetchRBACRoles(c)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var perms []string
	for _, role := range roles {
		users, err := fetchRoleUsers(c, role.ID)
		if err != nil {
			return nil, err
		}
		member := false
		for _, u := range users {
			if u.ID == userID {
				member = true
				break
			}
		}
		if !member {
			continue
		}
		for _, p := range role.Permissions {
			if _, ok := seen[p.PermissionString]; ok {
				continue
			}
			seen[p.PermissionString] = struct{}{}
			perms = append(perms, p.PermissionString)
		}
	}
	sort.Strings(perms)
	return perms, nil
}

func findRoleByName(roles []rbacRole, name string) (rbacRole, bool) {
	for _, role := range roles {
		if strings.EqualFold(role.Name, name) || role.ID == name {
			return role, true
		}
	}
	return rbacRole{}, false
}

func permissionStringsForRole(role rbacRole) []string {
	out := make([]string, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		out = append(out, p.PermissionString)
	}
	sort.Strings(out)
	return out
}

func rolesToExportFile(roles []rbacRole) rolesExportFile {
	entries := make([]rolesExportEntry, 0, len(roles))
	for _, role := range roles {
		entries = append(entries, rolesExportEntry{
			Name:        role.Name,
			Description: role.Description,
			Scope:       role.Scope,
			Permissions: permissionStringsForRole(role),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return rolesExportFile{Version: 1, Roles: entries}
}

func loadRolesExportFile(path string) (rolesExportFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return rolesExportFile{}, fmt.Errorf("reading file: %w", err)
	}
	var file rolesExportFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return rolesExportFile{}, fmt.Errorf("parsing roles file: %w", err)
	}
	if file.Version == 0 {
		file.Version = 1
	}
	return file, nil
}

func computeRoleApplyDiff(current []rbacRole, desired rolesExportFile) roleApplyDiff {
	currentByName := map[string]rbacRole{}
	for _, role := range current {
		currentByName[strings.ToLower(role.Name)] = role
	}
	desiredNames := map[string]struct{}{}
	var diff roleApplyDiff

	for _, want := range desired.Roles {
		desiredNames[strings.ToLower(want.Name)] = struct{}{}
		cur, ok := currentByName[strings.ToLower(want.Name)]
		if !ok {
			diff.Create = append(diff.Create, want)
			continue
		}
		if want.Description != cur.Description || want.Scope != cur.Scope {
			diff.Update = append(diff.Update, roleApplyUpdate{
				Name:        want.Name,
				Description: want.Description,
				Scope:       want.Scope,
			})
		}
		wantPerms := sortedCopy(want.Permissions)
		havePerms := permissionStringsForRole(cur)
		add, remove := diffStringSets(havePerms, wantPerms)
		if len(add) > 0 || len(remove) > 0 {
			diff.Perms = append(diff.Perms, rolePermissionDiff{
				Role:   want.Name,
				Add:    add,
				Remove: remove,
			})
		}
	}
	for _, role := range current {
		if _, ok := desiredNames[strings.ToLower(role.Name)]; !ok {
			diff.Remove = append(diff.Remove, role.Name)
		}
	}
	sort.Strings(diff.Remove)
	return diff
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func diffStringSets(have, want []string) (add, remove []string) {
	haveSet := map[string]struct{}{}
	for _, s := range have {
		haveSet[s] = struct{}{}
	}
	wantSet := map[string]struct{}{}
	for _, s := range want {
		wantSet[s] = struct{}{}
		if _, ok := haveSet[s]; !ok {
			add = append(add, s)
		}
	}
	for _, s := range have {
		if _, ok := wantSet[s]; !ok {
			remove = append(remove, s)
		}
	}
	sort.Strings(add)
	sort.Strings(remove)
	return add, remove
}

func roleHasPermission(role rbacRole, perm string) bool {
	for _, p := range role.Permissions {
		if p.PermissionString == perm {
			return true
		}
	}
	return false
}

func callerWouldLockOut(current []rbacRole, diff roleApplyDiff, callerID string, roleUsers map[string][]rbacUserBrief) bool {
	callerHasManage := false
	for _, role := range current {
		if !userInRoleUsers(roleUsers[role.ID], callerID) {
			continue
		}
		if roleHasPermission(role, rbacManagePermission) {
			callerHasManage = true
			break
		}
	}
	if !callerHasManage {
		return false
	}
	for _, rm := range diff.Remove {
		role, ok := findRoleByName(current, rm)
		if !ok {
			continue
		}
		if userInRoleUsers(roleUsers[role.ID], callerID) && roleHasPermission(role, rbacManagePermission) {
			return true
		}
	}
	for _, ch := range diff.Perms {
		role, ok := findRoleByName(current, ch.Role)
		if !ok || !userInRoleUsers(roleUsers[role.ID], callerID) {
			continue
		}
		for _, rm := range ch.Remove {
			if rm == rbacManagePermission {
				return true
			}
		}
	}
	return false
}

func userInRoleUsers(users []rbacUserBrief, userID string) bool {
	for _, u := range users {
		if u.ID == userID {
			return true
		}
	}
	return false
}

func permissionIDsForStrings(catalog []rbacPermission, want []string) ([]uuid.UUID, error) {
	byString := map[string]uuid.UUID{}
	for _, p := range catalog {
		id, err := uuid.Parse(p.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid permission id %q", p.ID)
		}
		byString[p.PermissionString] = id
	}
	ids := make([]uuid.UUID, 0, len(want))
	for _, s := range want {
		id, ok := byString[s]
		if !ok {
			return nil, fmt.Errorf("unknown capability %q (use roles capabilities list)", s)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func postRBACRole(c *client.Client, body map[string]any) (rbacRole, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return rbacRole{}, err
	}
	req, err := c.NewRequest(http.MethodPost, settingsPath("/roles"), bytes.NewReader(raw))
	if err != nil {
		return rbacRole{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return rbacRole{}, fmt.Errorf("creating role: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return rbacRole{}, err
	}
	if resp.StatusCode == http.StatusForbidden {
		return rbacRole{}, fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusOK {
		return rbacRole{}, apiErrorBody(resp.StatusCode, respBody)
	}
	var out rbacRole
	if err := json.Unmarshal(respBody, &out); err != nil {
		return rbacRole{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func patchRBACRole(c *client.Client, roleID string, body map[string]any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPatch, settingsPath("/roles/"+url.PathEscape(roleID)), bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating role: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return nil
}

func deleteRBACRole(c *client.Client, roleID string) error {
	req, err := c.NewRequest(http.MethodDelete, settingsPath("/roles/"+url.PathEscape(roleID)), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting role: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func putRolePermissions(c *client.Client, roleID string, permissionIDs []uuid.UUID) error {
	raw, err := json.Marshal(map[string]any{"permissionIds": permissionIDs})
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPut, settingsPath("/roles/"+url.PathEscape(roleID)+"/permissions"), bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting role permissions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func addUserToRBACRole(c *client.Client, roleID, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	raw, err := json.Marshal(map[string]any{"userId": uid})
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPost, settingsPath("/roles/"+url.PathEscape(roleID)+"/users"), bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("granting role: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func removeUserFromRBACRole(c *client.Client, roleID, userID string) error {
	req, err := c.NewRequest(http.MethodDelete, settingsPath("/roles/"+url.PathEscape(roleID)+"/users/"+url.PathEscape(userID)), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("revoking role: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied: rbac:manage scope required")
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func applyRolesFile(c *client.Client, file rolesExportFile, dryRun, force bool) (roleApplyDiff, error) {
	current, _, err := fetchRBACRoles(c)
	if err != nil {
		return roleApplyDiff{}, err
	}
	catalog, _, err := fetchRBACPermissions(c)
	if err != nil {
		return roleApplyDiff{}, err
	}
	diff := computeRoleApplyDiff(current, file)
	if dryRun {
		return diff, nil
	}

	callerID, err := fetchMeUserID(c)
	if err != nil {
		return diff, err
	}
	roleUsers := map[string][]rbacUserBrief{}
	for _, role := range current {
		users, uerr := fetchRoleUsers(c, role.ID)
		if uerr != nil {
			return diff, uerr
		}
		roleUsers[role.ID] = users
	}
	if !force && callerWouldLockOut(current, diff, callerID, roleUsers) {
		return diff, fmt.Errorf("apply would remove your own %s grant; re-run with --force to proceed", rbacManagePermission)
	}

	current = append([]rbacRole(nil), current...)
	for _, want := range diff.Create {
		created, err := postRBACRole(c, map[string]any{
			"name":        want.Name,
			"description": want.Description,
			"scope":       want.Scope,
		})
		if err != nil {
			return diff, err
		}
		current = append(current, created)
		if len(want.Permissions) > 0 {
			ids, err := permissionIDsForStrings(catalog, want.Permissions)
			if err != nil {
				return diff, err
			}
			if err := putRolePermissions(c, created.ID, ids); err != nil {
				return diff, err
			}
		}
	}
	for _, upd := range diff.Update {
		role, ok := findRoleByName(current, upd.Name)
		if !ok {
			continue
		}
		body := map[string]any{"name": upd.Name}
		if upd.Description != "" {
			body["description"] = upd.Description
		}
		if upd.Scope != "" {
			body["scope"] = upd.Scope
		}
		if err := patchRBACRole(c, role.ID, body); err != nil {
			return diff, err
		}
	}
	for _, ch := range diff.Perms {
		role, ok := findRoleByName(current, ch.Role)
		if !ok {
			continue
		}
		want := map[string]struct{}{}
		for _, p := range role.Permissions {
			want[p.PermissionString] = struct{}{}
		}
		for _, add := range ch.Add {
			want[add] = struct{}{}
		}
		for _, rm := range ch.Remove {
			delete(want, rm)
		}
		strs := make([]string, 0, len(want))
		for s := range want {
			strs = append(strs, s)
		}
		ids, err := permissionIDsForStrings(catalog, strs)
		if err != nil {
			return diff, err
		}
		if err := putRolePermissions(c, role.ID, ids); err != nil {
			return diff, err
		}
	}
	for _, name := range diff.Remove {
		role, ok := findRoleByName(current, name)
		if !ok {
			continue
		}
		if err := deleteRBACRole(c, role.ID); err != nil {
			return diff, err
		}
	}
	return diff, nil
}

func formatRoleApplyDiff(diff roleApplyDiff) string {
	var b strings.Builder
	if len(diff.Create) > 0 {
		_, _ = fmt.Fprintf(&b, "create %d role(s)\n", len(diff.Create))
		for _, r := range diff.Create {
			_, _ = fmt.Fprintf(&b, "  + %s (%d capabilities)\n", r.Name, len(r.Permissions))
		}
	}
	if len(diff.Update) > 0 {
		_, _ = fmt.Fprintf(&b, "update %d role(s)\n", len(diff.Update))
		for _, r := range diff.Update {
			_, _ = fmt.Fprintf(&b, "  ~ %s\n", r.Name)
		}
	}
	if len(diff.Remove) > 0 {
		_, _ = fmt.Fprintf(&b, "remove %d role(s)\n", len(diff.Remove))
		for _, name := range diff.Remove {
			_, _ = fmt.Fprintf(&b, "  - %s\n", name)
		}
	}
	for _, ch := range diff.Perms {
		if len(ch.Add) > 0 {
			_, _ = fmt.Fprintf(&b, "  %s add: %s\n", ch.Role, strings.Join(ch.Add, ", "))
		}
		if len(ch.Remove) > 0 {
			_, _ = fmt.Fprintf(&b, "  %s remove: %s\n", ch.Role, strings.Join(ch.Remove, ", "))
		}
	}
	if b.Len() == 0 {
		return "no changes\n"
	}
	return b.String()
}