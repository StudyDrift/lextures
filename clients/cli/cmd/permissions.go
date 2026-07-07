package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var permissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Audit effective permissions",
}

var permissionsCheckFlags struct {
	user       string
	capability string
	course     string
	org        string
}

var permissionsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check whether a user has a capability",
	RunE:  runPermissionsCheck,
}

func init() {
	permissionsCheckCmd.Flags().StringVar(&permissionsCheckFlags.user, "user", "", "user UUID (defaults to caller)")
	permissionsCheckCmd.Flags().StringVar(&permissionsCheckFlags.capability, "capability", "", "capability string (required)")
	permissionsCheckCmd.Flags().StringVar(&permissionsCheckFlags.course, "course", "", "course code for course-scoped checks")
	permissionsCheckCmd.Flags().StringVar(&permissionsCheckFlags.org, "org", "", "org UUID for org role grant context")
	_ = permissionsCheckCmd.MarkFlagRequired("capability")

	permissionsCmd.AddCommand(permissionsCheckCmd)
	rootCmd.AddCommand(permissionsCmd)
}

func runPermissionsCheck(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	capability := permissionsCheckFlags.capability
	userID := permissionsCheckFlags.user

	var perms []string
	var err error
	if userID == "" {
		userID, err = fetchMeUserID(c)
		if err != nil {
			return err
		}
		perms, err = fetchMyPermissionStrings(c, permissionsCheckFlags.course)
	} else {
		perms, err = resolveUserPermissionStrings(c, userID)
	}
	if err != nil {
		return err
	}

	allowed := permissionAllowed(perms, capability)
	if permissionsCheckFlags.org != "" && !allowed {
		allowed, err = orgGrantAllowsCapability(c, permissionsCheckFlags.org, userID, capability)
		if err != nil {
			return err
		}
	}

	result := map[string]any{
		"user":       userID,
		"capability": capability,
		"allowed":    allowed,
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}
	state := "denied"
	if allowed {
		state = "allowed"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s for capability %s\n", userID, state, capability)
	return nil
}

func permissionAllowed(perms []string, capability string) bool {
	for _, p := range perms {
		if p == capability {
			return true
		}
	}
	return false
}

func orgGrantAllowsCapability(c *client.Client, orgID, userID, capability string) (bool, error) {
	grants, _, err := fetchOrgRoleGrants(c, orgID)
	if err != nil {
		return false, err
	}
	for _, g := range grants.Grants {
		if g.UserID != userID {
			continue
		}
		role := strings.ToLower(g.Role)
		if strings.Contains(capability, ":manage") && (role == "admin" || role == "org_admin") {
			return true, nil
		}
	}
	return false, nil
}