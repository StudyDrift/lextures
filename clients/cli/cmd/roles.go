package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var rolesCmd = &cobra.Command{
	Use:   "roles",
	Short: "Manage platform RBAC roles and grants",
}

var rolesCreateFlags struct {
	description string
	scope       string
}

var rolesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a platform role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRolesCreate,
}

var rolesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a role by id or name",
	Args:  cobra.ExactArgs(1),
	RunE:  runRolesGet,
}

var rolesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List platform roles and their capabilities",
	RunE:  runRolesList,
}

var rolesUpdateFlags struct {
	name        string
	description string
	scope       string
}

var rolesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a platform role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRolesUpdate,
}

var rolesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a platform role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRolesDelete,
}

var rolesPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage capability grants on a role",
}

var rolesPermissionsListCmd = &cobra.Command{
	Use:   "list <role>",
	Short: "List capabilities granted to a role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRolesPermissionsList,
}

var rolesPermissionsAddCmd = &cobra.Command{
	Use:   "add <role> <capability>",
	Short: "Grant a capability to a role",
	Args:  cobra.ExactArgs(2),
	RunE:  runRolesPermissionsAdd,
}

var rolesPermissionsRemoveCmd = &cobra.Command{
	Use:   "remove <role> <capability>",
	Short: "Revoke a capability from a role",
	Args:  cobra.ExactArgs(2),
	RunE:  runRolesPermissionsRemove,
}

var rolesCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Inspect the capability catalog",
}

var rolesCapabilitiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known capabilities",
	RunE:  runRolesCapabilitiesList,
}

var rolesGrantFlags struct {
	user string
	role string
	org  string
}

var rolesGrantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant a role to a user",
	RunE:  runRolesGrant,
}

var rolesRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke a role from a user",
	RunE:  runRolesRevoke,
}

var rolesExportFlags struct {
	file string
}

var rolesExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the RBAC model to a JSON file",
	RunE:  runRolesExport,
}

var rolesApplyFlags struct {
	file    string
	dryRun  bool
	force   bool
}

var rolesApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a declarative RBAC model from a JSON file",
	RunE:  runRolesApply,
}

func init() {
	rolesCreateCmd.Flags().StringVar(&rolesCreateFlags.description, "description", "", "role description")
	rolesCreateCmd.Flags().StringVar(&rolesCreateFlags.scope, "scope", "global", "role scope: global or course")

	rolesUpdateCmd.Flags().StringVar(&rolesUpdateFlags.name, "name", "", "new role name")
	rolesUpdateCmd.Flags().StringVar(&rolesUpdateFlags.description, "description", "", "role description")
	rolesUpdateCmd.Flags().StringVar(&rolesUpdateFlags.scope, "scope", "", "role scope")

	rolesGrantCmd.Flags().StringVar(&rolesGrantFlags.user, "user", "", "user UUID (required)")
	rolesGrantCmd.Flags().StringVar(&rolesGrantFlags.role, "role", "", "role name or id (required)")
	rolesGrantCmd.Flags().StringVar(&rolesGrantFlags.org, "org", "", "org UUID for org-scoped grants")
	_ = rolesGrantCmd.MarkFlagRequired("user")
	_ = rolesGrantCmd.MarkFlagRequired("role")

	rolesRevokeCmd.Flags().StringVar(&rolesGrantFlags.user, "user", "", "user UUID (required)")
	rolesRevokeCmd.Flags().StringVar(&rolesGrantFlags.role, "role", "", "role name or id (required)")
	rolesRevokeCmd.Flags().StringVar(&rolesGrantFlags.org, "org", "", "org UUID for org-scoped grants")
	_ = rolesRevokeCmd.MarkFlagRequired("user")
	_ = rolesRevokeCmd.MarkFlagRequired("role")

	rolesExportCmd.Flags().StringVar(&rolesExportFlags.file, "file", "roles.json", "output file")

	rolesApplyCmd.Flags().StringVar(&rolesApplyFlags.file, "file", "roles.json", "roles JSON file")
	rolesApplyCmd.Flags().BoolVar(&rolesApplyFlags.dryRun, "dry-run", false, "print diff without applying")
	rolesApplyCmd.Flags().BoolVar(&rolesApplyFlags.force, "force", false, "allow removing your own admin grant")

	rolesPermissionsCmd.AddCommand(rolesPermissionsListCmd, rolesPermissionsAddCmd, rolesPermissionsRemoveCmd)
	rolesCapabilitiesCmd.AddCommand(rolesCapabilitiesListCmd)

	rolesCmd.AddCommand(
		rolesListCmd, rolesGetCmd, rolesCreateCmd, rolesUpdateCmd, rolesDeleteCmd,
		rolesPermissionsCmd, rolesCapabilitiesCmd,
		rolesGrantCmd, rolesRevokeCmd,
		rolesExportCmd, rolesApplyCmd,
	)
	rootCmd.AddCommand(rolesCmd)
}

func runRolesList(cmd *cobra.Command, args []string) error {
	roles, raw, err := fetchRBACRoles(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tSCOPE\tCAPABILITIES")
	for _, role := range roles {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", role.ID, role.Name, role.Scope, len(role.Permissions))
	}
	return w.Flush()
}

func runRolesGet(cmd *cobra.Command, args []string) error {
	role, raw, err := fetchRBACRole(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if raw != nil {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(role)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s (%s)\n", role.Name, role.Scope)
	for _, p := range role.Permissions {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p.PermissionString)
	}
	return nil
}

func runRolesCreate(cmd *cobra.Command, args []string) error {
	scope := rolesCreateFlags.scope
	if scope == "" {
		scope = "global"
	}
	role, err := postRBACRole(client.New(Cfg.Server, Cfg.APIKey), map[string]any{
		"name":        args[0],
		"description": rolesCreateFlags.description,
		"scope":       scope,
	})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(role)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created role %s (%s)\n", role.Name, role.ID)
	return nil
}

func runRolesUpdate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	role, _, err := fetchRBACRole(c, args[0])
	if err != nil {
		return err
	}
	body := map[string]any{"name": role.Name}
	if rolesUpdateFlags.name != "" {
		body["name"] = rolesUpdateFlags.name
	}
	if rolesUpdateFlags.description != "" {
		body["description"] = rolesUpdateFlags.description
	}
	if rolesUpdateFlags.scope != "" {
		body["scope"] = rolesUpdateFlags.scope
	}
	if len(body) == 1 && body["name"] == role.Name && rolesUpdateFlags.description == "" && rolesUpdateFlags.scope == "" {
		return fmt.Errorf("at least one of --name, --description, or --scope is required")
	}
	if err := patchRBACRole(c, role.ID, body); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Updated role")
	return nil
}

func runRolesDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	role, _, err := fetchRBACRole(c, args[0])
	if err != nil {
		return err
	}
	if err := deleteRBACRole(c, role.ID); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Deleted role")
	return nil
}

func runRolesPermissionsList(cmd *cobra.Command, args []string) error {
	role, _, err := fetchRBACRole(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(role.Permissions)
	}
	for _, p := range role.Permissions {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", p.PermissionString, p.Description)
	}
	return nil
}

func runRolesPermissionsAdd(cmd *cobra.Command, args []string) error {
	return mutateRolePermissions(args[0], args[1], true)
}

func runRolesPermissionsRemove(cmd *cobra.Command, args []string) error {
	return mutateRolePermissions(args[0], args[1], false)
}

func mutateRolePermissions(roleRef, capability string, add bool) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	role, _, err := fetchRBACRole(c, roleRef)
	if err != nil {
		return err
	}
	catalog, _, err := fetchRBACPermissions(c)
	if err != nil {
		return err
	}
	have := map[string]struct{}{}
	for _, p := range role.Permissions {
		have[p.PermissionString] = struct{}{}
	}
	if add {
		have[capability] = struct{}{}
	} else {
		delete(have, capability)
	}
	strs := make([]string, 0, len(have))
	for s := range have {
		strs = append(strs, s)
	}
	ids, err := permissionIDsForStrings(catalog, strs)
	if err != nil {
		return err
	}
	return putRolePermissions(c, role.ID, ids)
}

func runRolesCapabilitiesList(cmd *cobra.Command, args []string) error {
	perms, raw, err := fetchRBACPermissions(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CAPABILITY\tDESCRIPTION")
	for _, p := range perms {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", p.PermissionString, p.Description)
	}
	return w.Flush()
}

func runRolesGrant(cmd *cobra.Command, args []string) error {
	if rolesGrantFlags.org != "" {
		_, err := postOrgRoleGrant(client.New(Cfg.Server, Cfg.APIKey), rolesGrantFlags.org, rolesGrantFlags.user, rolesGrantFlags.role)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Granted org role")
		return nil
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	roles, _, err := fetchRBACRoles(c)
	if err != nil {
		return err
	}
	role, ok := findRoleByName(roles, rolesGrantFlags.role)
	if !ok {
		return fmt.Errorf("role %q not found", rolesGrantFlags.role)
	}
	if err := addUserToRBACRole(c, role.ID, rolesGrantFlags.user); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Granted platform role")
	return nil
}

func runRolesRevoke(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if rolesGrantFlags.org != "" {
		grants, _, err := fetchOrgRoleGrants(c, rolesGrantFlags.org)
		if err != nil {
			return err
		}
		for _, g := range grants.Grants {
			if g.UserID == rolesGrantFlags.user && strings.EqualFold(g.Role, rolesGrantFlags.role) {
				if err := deleteOrgRoleGrant(c, rolesGrantFlags.org, g.ID); err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Revoked org role")
				return nil
			}
		}
		return fmt.Errorf("no matching org role grant found")
	}
	roles, _, err := fetchRBACRoles(c)
	if err != nil {
		return err
	}
	role, ok := findRoleByName(roles, rolesGrantFlags.role)
	if !ok {
		return fmt.Errorf("role %q not found", rolesGrantFlags.role)
	}
	if err := removeUserFromRBACRole(c, role.ID, rolesGrantFlags.user); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Revoked platform role")
	return nil
}

func runRolesExport(cmd *cobra.Command, args []string) error {
	roles, _, err := fetchRBACRoles(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	file := rolesToExportFile(roles)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(file)
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(rolesExportFlags.file, raw, 0o600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d role(s) to %s\n", len(file.Roles), rolesExportFlags.file)
	return nil
}

func runRolesApply(cmd *cobra.Command, args []string) error {
	file, err := loadRolesExportFile(rolesApplyFlags.file)
	if err != nil {
		return err
	}
	diff, err := applyRolesFile(client.New(Cfg.Server, Cfg.APIKey), file, rolesApplyFlags.dryRun, rolesApplyFlags.force)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(diff)
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), formatRoleApplyDiff(diff))
	if rolesApplyFlags.dryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(dry-run: no changes applied)")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Applied RBAC model")
	}
	return nil
}