package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var scimCmd = &cobra.Command{
	Use:   "scim",
	Short: "Inspect SCIM provisioning state",
}

var scimCommonFlags struct {
	institution string
	token       string
}

var scimStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show SCIM tokens and recent provisioning events",
	RunE:  runScimStatus,
}

var scimUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List SCIM-provisioned users",
}

var scimUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users from the SCIM API",
	RunE:  runScimUsersList,
}

var scimGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List SCIM-provisioned groups",
}

var scimGroupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List groups from the SCIM API",
	RunE:  runScimGroupsList,
}

func init() {
	scimStatusCmd.Flags().StringVar(&scimCommonFlags.institution, "institution", "", "institution UUID (required)")
	_ = scimStatusCmd.MarkFlagRequired("institution")

	scimUsersListCmd.Flags().StringVar(&scimCommonFlags.token, "token", "", "SCIM bearer token (required)")
	_ = scimUsersListCmd.MarkFlagRequired("token")

	scimGroupsListCmd.Flags().StringVar(&scimCommonFlags.token, "token", "", "SCIM bearer token (required)")
	_ = scimGroupsListCmd.MarkFlagRequired("token")

	scimUsersCmd.AddCommand(scimUsersListCmd)
	scimGroupsCmd.AddCommand(scimGroupsListCmd)
	scimCmd.AddCommand(scimStatusCmd, scimUsersCmd, scimGroupsCmd)
	rootCmd.AddCommand(scimCmd)
}

func runScimStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	tokens, _, err := fetchScimTokens(c, scimCommonFlags.institution)
	if err != nil {
		return err
	}
	events, _, err := fetchScimEvents(c, scimCommonFlags.institution)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"tokens": tokens,
			"events": events,
		})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Tokens:")
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tLABEL\tREVOKED")
	for _, t := range tokens {
		revoked := ""
		if t.RevokedAt != nil {
			revoked = *t.RevokedAt
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", t.ID, t.Label, revoked)
	}
	_ = w.Flush()
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Recent events:")
	w2 := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w2, "OPERATION\tRESOURCE\tEMAIL\tAT")
	for i, e := range events {
		if i >= 10 {
			break
		}
		email := ""
		if e.UserEmail != nil {
			email = *e.UserEmail
		}
		_, _ = fmt.Fprintf(w2, "%s\t%s\t%s\t%s\n", e.Operation, e.ScimResource, email, e.CreatedAt)
	}
	return w2.Flush()
}

func runScimUsersList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchSCIMCollection(c, "Users", scimCommonFlags.token)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Resources []map[string]any `json:"Resources"`
		Total     int              `json:"totalResults"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, _ = cmd.OutOrStdout().Write(body)
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Showing %d users\n", out.Total)
	_, _ = fmt.Fprintln(w, "ID\tUSER_NAME\tACTIVE")
	for _, u := range out.Resources {
		id, _ := u["id"].(string)
		userName, _ := u["userName"].(string)
		active, _ := u["active"].(bool)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\n", id, userName, active)
	}
	return w.Flush()
}

func runScimGroupsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchSCIMCollection(c, "Groups", scimCommonFlags.token)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Resources []map[string]any `json:"Resources"`
		Total     int              `json:"totalResults"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, _ = cmd.OutOrStdout().Write(body)
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Showing %d groups\n", out.Total)
	_, _ = fmt.Fprintln(w, "ID\tDISPLAY_NAME")
	for _, g := range out.Resources {
		id, _ := g["id"].(string)
		name, _ := g["displayName"].(string)
		_, _ = fmt.Fprintf(w, "%s\t%s\n", id, name)
	}
	return w.Flush()
}