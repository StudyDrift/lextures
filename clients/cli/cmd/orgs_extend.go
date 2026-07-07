package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var orgsUpdateFlags struct {
	name   string
	status string
	slug   string
}

var orgsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsUpdate,
}

var orgsArchiveCmd = &cobra.Command{
	Use:   "archive <id>",
	Short: "Archive (soft-delete) an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsArchive,
}

var orgsSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Get or set organization settings",
}

var orgsSettingsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get organization settings",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsSettingsGet,
}

var orgsSettingsSetFlags struct {
	file string
}

var orgsSettingsSetCmd = &cobra.Command{
	Use:   "set <id>",
	Short: "Set organization settings from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsSettingsSet,
}

var orgsBrandingCmd = &cobra.Command{
	Use:   "branding",
	Short: "Get or set organization branding",
}

var orgsBrandingGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get organization branding",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsBrandingGet,
}

var orgsBrandingSetFlags struct {
	file string
}

var orgsBrandingSetCmd = &cobra.Command{
	Use:   "set <id>",
	Short: "Set organization branding from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsBrandingSet,
}

var orgsRoleGrantsCmd = &cobra.Command{
	Use:   "role-grants",
	Short: "Manage delegated org role grants",
}

var orgsRoleGrantsListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List org role grants",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsRoleGrantsList,
}

var orgsRoleGrantsAddFlags struct {
	user string
	role string
}

var orgsRoleGrantsAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Grant an org role to a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsRoleGrantsAdd,
}

var orgsRoleGrantsRemoveCmd = &cobra.Command{
	Use:   "remove <id> <grant>",
	Short: "Remove an org role grant",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgsRoleGrantsRemove,
}

var orgsParentLinksCmd = &cobra.Command{
	Use:   "parent-links",
	Short: "Manage parent/guardian links",
}

var orgsParentLinksListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List parent links for an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsParentLinksList,
}

var orgsParentLinksAddFlags struct {
	parent    string
	student   string
	relation  string
}

var orgsParentLinksAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Create a parent/guardian link",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsParentLinksAdd,
}

var orgsParentLinksRemoveCmd = &cobra.Command{
	Use:   "remove <id> <link>",
	Short: "Remove a parent link",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgsParentLinksRemove,
}

func init() {
	orgsUpdateCmd.Flags().StringVar(&orgsUpdateFlags.name, "name", "", "organization name")
	orgsUpdateCmd.Flags().StringVar(&orgsUpdateFlags.status, "status", "", "organization status")
	orgsUpdateCmd.Flags().StringVar(&orgsUpdateFlags.slug, "slug", "", "organization slug")

	orgsSettingsSetCmd.Flags().StringVar(&orgsSettingsSetFlags.file, "file", "", "settings JSON file (required)")
	_ = orgsSettingsSetCmd.MarkFlagRequired("file")

	orgsBrandingSetCmd.Flags().StringVar(&orgsBrandingSetFlags.file, "file", "", "branding JSON file (required)")
	_ = orgsBrandingSetCmd.MarkFlagRequired("file")

	orgsRoleGrantsAddCmd.Flags().StringVar(&orgsRoleGrantsAddFlags.user, "user", "", "user UUID (required)")
	_ = orgsRoleGrantsAddCmd.MarkFlagRequired("user")
	orgsRoleGrantsAddCmd.Flags().StringVar(&orgsRoleGrantsAddFlags.role, "role", "", "org role (required)")
	_ = orgsRoleGrantsAddCmd.MarkFlagRequired("role")

	orgsParentLinksAddCmd.Flags().StringVar(&orgsParentLinksAddFlags.parent, "parent", "", "parent user UUID (required)")
	_ = orgsParentLinksAddCmd.MarkFlagRequired("parent")
	orgsParentLinksAddCmd.Flags().StringVar(&orgsParentLinksAddFlags.student, "student", "", "student user UUID (required)")
	_ = orgsParentLinksAddCmd.MarkFlagRequired("student")
	orgsParentLinksAddCmd.Flags().StringVar(&orgsParentLinksAddFlags.relation, "relationship", "parent", "relationship: parent, guardian, other")

	orgsSettingsCmd.AddCommand(orgsSettingsGetCmd, orgsSettingsSetCmd)
	orgsBrandingCmd.AddCommand(orgsBrandingGetCmd, orgsBrandingSetCmd)
	orgsRoleGrantsCmd.AddCommand(orgsRoleGrantsListCmd, orgsRoleGrantsAddCmd, orgsRoleGrantsRemoveCmd)
	orgsParentLinksCmd.AddCommand(orgsParentLinksListCmd, orgsParentLinksAddCmd, orgsParentLinksRemoveCmd)

	orgsCmd.AddCommand(
		orgsUpdateCmd,
		orgsArchiveCmd,
		orgsSettingsCmd,
		orgsBrandingCmd,
		orgsRoleGrantsCmd,
		orgsParentLinksCmd,
	)
}

func runOrgsUpdate(cmd *cobra.Command, args []string) error {
	body := map[string]any{}
	if orgsUpdateFlags.name != "" {
		body["name"] = orgsUpdateFlags.name
	}
	if orgsUpdateFlags.status != "" {
		body["status"] = orgsUpdateFlags.status
	}
	if orgsUpdateFlags.slug != "" {
		body["slug"] = orgsUpdateFlags.slug
	}
	if len(body) == 0 {
		return fmt.Errorf("at least one of --name, --status, or --slug is required")
	}
	respBody, err := patchAdminOrg(client.New(Cfg.Server, Cfg.APIKey), args[0], body)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, respBody, "Updated organization")
}

func runOrgsArchive(cmd *cobra.Command, args []string) error {
	respBody, err := deleteAdminOrg(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, respBody, "Archived organization")
}

func runOrgsSettingsGet(cmd *cobra.Command, args []string) error {
	body, err := fetchAdminConsoleSettings(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var settings map[string]any
	if err := json.Unmarshal(body, &settings); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, key := range []string{"name", "timezone", "locale", "primaryColor", "customDomain"} {
		if v, ok := settings[key]; ok && v != nil {
			_, _ = fmt.Fprintf(out, "%s: %v\n", key, v)
		}
	}
	return nil
}

func runOrgsSettingsSet(cmd *cobra.Command, args []string) error {
	payload, err := readJSONSettingsFile(orgsSettingsSetFlags.file)
	if err != nil {
		return err
	}
	body, err := putAdminConsoleSettings(client.New(Cfg.Server, Cfg.APIKey), args[0], payload)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Updated organization settings")
}

func runOrgsBrandingGet(cmd *cobra.Command, args []string) error {
	body, err := fetchOrgBranding(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var branding map[string]any
	if err := json.Unmarshal(body, &branding); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Primary:   %v\n", branding["primaryColor"])
	_, _ = fmt.Fprintf(out, "Secondary: %v\n", branding["secondaryColor"])
	_, _ = fmt.Fprintf(out, "Logo:      %v\n", branding["logoUrl"])
	return nil
}

func runOrgsBrandingSet(cmd *cobra.Command, args []string) error {
	payload, err := readJSONSettingsFile(orgsBrandingSetFlags.file)
	if err != nil {
		return err
	}
	body, err := putOrgBranding(client.New(Cfg.Server, Cfg.APIKey), args[0], payload)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Updated organization branding")
}

func runOrgsRoleGrantsList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchOrgRoleGrants(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Grants) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No role grants.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tUSER\tROLE")
	for _, g := range body.Grants {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", g.ID, g.UserEmail, g.Role)
	}
	return w.Flush()
}

func runOrgsRoleGrantsAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	userID, _, err := resolveUserID(c, orgsRoleGrantsAddFlags.user)
	if err != nil {
		return err
	}
	body, err := postOrgRoleGrant(c, args[0], userID, orgsRoleGrantsAddFlags.role)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Added role grant")
}

func runOrgsRoleGrantsRemove(cmd *cobra.Command, args []string) error {
	if err := deleteOrgRoleGrant(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, "Removed role grant")
}

func runOrgsParentLinksList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, orgPath(args[0])+"/parent-links", nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var parsed struct {
		Links []map[string]any `json:"links"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tPARENT\tSTUDENT\tRELATIONSHIP")
	for _, ln := range parsed.Links {
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", ln["id"], ln["parentUserId"], ln["studentUserId"], ln["relationship"])
	}
	return w.Flush()
}

func runOrgsParentLinksAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload := map[string]string{
		"parentUserId":  orgsParentLinksAddFlags.parent,
		"studentUserId": orgsParentLinksAddFlags.student,
		"relationship":  orgsParentLinksAddFlags.relation,
	}
	raw, _ := json.Marshal(payload)
	req, err := c.NewRequest(http.MethodPost, orgPath(args[0])+"/parent-links", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return apiErrorBody(resp.StatusCode, body)
	}
	return emitRawOrMessage(cmd, body, "Created parent link")
}

func runOrgsParentLinksRemove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := orgPath(args[0]) + "/parent-links/" + args[1]
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return apiErrorBody(resp.StatusCode, body)
	}
	return emitRawOrMessage(cmd, nil, "Removed parent link")
}