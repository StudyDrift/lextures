package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var usersUpdateFlags struct {
	name  string
	role  string
	org   string
	email string
}

var usersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a user account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersUpdate,
}

var usersSuspendFlags struct {
	org string
}

var usersSuspendCmd = &cobra.Command{
	Use:   "suspend <id>",
	Short: "Suspend (deactivate) a user account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersSuspend,
}

var usersReactivateCmd = &cobra.Command{
	Use:   "reactivate <id>",
	Short: "Reactivate a suspended user account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersReactivate,
}

var usersDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a user account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersDelete,
}

var usersImportFlags struct {
	file       string
	dryRun     bool
	org        string
	role       string
	secretsOut string
	enroll     string
}

var usersImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Bulk import users from a CSV file",
	RunE:  runUsersImport,
}

var usersSearchFlags struct {
	query string
	org   string
	role  string
	limit int
}

var usersSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search users in the admin console",
	RunE:  runUsersSearch,
}

func init() {
	usersUpdateCmd.Flags().StringVar(&usersUpdateFlags.name, "name", "", "display name")
	usersUpdateCmd.Flags().StringVar(&usersUpdateFlags.role, "role", "", "role")
	usersUpdateCmd.Flags().StringVar(&usersUpdateFlags.email, "email", "", "email address")
	usersUpdateCmd.Flags().StringVar(&usersUpdateFlags.org, "org", "", "target org UUID (global admin)")

	usersSuspendCmd.Flags().StringVar(&usersSuspendFlags.org, "org", "", "target org UUID (global admin)")
	usersReactivateCmd.Flags().StringVar(&usersSuspendFlags.org, "org", "", "target org UUID (global admin)")

	usersImportCmd.Flags().StringVar(&usersImportFlags.file, "file", "", "CSV file (required)")
	_ = usersImportCmd.MarkFlagRequired("file")
	usersImportCmd.Flags().BoolVar(&usersImportFlags.dryRun, "dry-run", false, "preview import without applying changes")
	usersImportCmd.Flags().StringVar(&usersImportFlags.org, "org", "", "target org UUID (global admin)")
	usersImportCmd.Flags().StringVar(&usersImportFlags.role, "role", "student", "default role when CSV has no role column")
	usersImportCmd.Flags().StringVar(&usersImportFlags.secretsOut, "secrets-out", "", "write temporary passwords to this file (never stdout)")
	usersImportCmd.Flags().StringVar(&usersImportFlags.enroll, "enroll", "", "convenience enroll as course=role after create")

	usersSearchCmd.Flags().StringVar(&usersSearchFlags.query, "query", "", "search query")
	usersSearchCmd.Flags().StringVar(&usersSearchFlags.org, "org", "", "target org UUID (global admin)")
	usersSearchCmd.Flags().StringVar(&usersSearchFlags.role, "role", "", "filter by role")
	usersSearchCmd.Flags().IntVar(&usersSearchFlags.limit, "limit", 50, "maximum results")

	usersCmd.AddCommand(
		usersUpdateCmd,
		usersSuspendCmd,
		usersReactivateCmd,
		usersDeleteCmd,
		usersImportCmd,
		usersSearchCmd,
	)
}

func runUsersUpdate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{}
	if usersUpdateFlags.name != "" {
		payload["name"] = usersUpdateFlags.name
	}
	if usersUpdateFlags.role != "" {
		payload["role"] = usersUpdateFlags.role
	}
	if usersUpdateFlags.email != "" {
		payload["email"] = usersUpdateFlags.email
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one of --name, --role, or --email is required")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	userID, _, err := resolveUserID(c, args[0])
	if err != nil {
		return err
	}
	body, err := patchAdminConsoleUser(c, userID, usersUpdateFlags.org, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(redactSecretsFromJSON(body))
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated user %s\n", userID)
	return nil
}

func runUsersSuspend(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	userID, _, err := resolveUserID(c, args[0])
	if err != nil {
		return err
	}
	body, err := patchAdminConsoleUser(c, userID, usersSuspendFlags.org, map[string]any{"active": false})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(redactSecretsFromJSON(body))
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Suspended user %s\n", userID)
	return nil
}

func runUsersReactivate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	userID, _, err := resolveUserID(c, args[0])
	if err != nil {
		return err
	}
	body, err := patchAdminConsoleUser(c, userID, usersSuspendFlags.org, map[string]any{"active": true})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(redactSecretsFromJSON(body))
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Reactivated user %s\n", userID)
	return nil
}

func runUsersDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	userID, _, err := resolveUserID(c, args[0])
	if err != nil {
		return err
	}
	if err := deleteAdminConsoleUser(c, userID); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": userID})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted user %s\n", userID)
	return nil
}

func runUsersImport(cmd *cobra.Command, args []string) error {
	rows, err := parseUserImportCSV(usersImportFlags.file)
	if err != nil {
		return err
	}
	if usersImportFlags.dryRun {
		summary := userImportSummary{Skipped: len(rows)}
		for _, row := range rows {
			role := row.Role
			if role == "" {
				role = usersImportFlags.role
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] would upsert %s (%s) as %s\n", row.Email, row.Name, role)
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %d rows, no changes made\n", len(rows))
		return nil
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	if len(rows) >= userImportAsyncThreshold {
		out, err := submitImportJob(c, usersImportFlags.org, usersImportFlags.file, false, "upsert", "default")
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}
		jobID, _ := out["jobId"].(string)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submitted async import job %s (%d rows). Poll with: lextures imports status %s --wait\n",
			jobID, len(rows), jobID)
		return nil
	}

	summary := userImportSummary{}
	for _, row := range rows {
		role := row.Role
		if role == "" {
			role = usersImportFlags.role
		}
		name := row.Name
		if name == "" {
			name = row.Email
		}
		usersCreateFlags.email = row.Email
		usersCreateFlags.name = name
		usersCreateFlags.role = role
		if err := runUsersCreate(cmd, nil); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				summary.Skipped++
				continue
			}
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: %v", row.Line, err))
			continue
		}
		summary.Created++
		if usersImportFlags.enroll != "" {
			parts := strings.SplitN(usersImportFlags.enroll, "=", 2)
			if len(parts) == 2 {
				usersEnrollFlags.course = parts[0]
				usersEnrollFlags.user = row.Email
				usersEnrollFlags.role = parts[1]
				_ = runUsersEnroll(cmd, nil)
			}
		}
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Import complete: created=%d skipped=%d failed=%d\n",
		summary.Created, summary.Skipped, summary.Failed)
	return nil
}

func runUsersSearch(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	params := url.Values{}
	if usersSearchFlags.query != "" {
		params.Set("q", usersSearchFlags.query)
	}
	if usersSearchFlags.role != "" {
		params.Set("role", usersSearchFlags.role)
	}
	if usersSearchFlags.limit > 0 {
		params.Set("perPage", fmt.Sprintf("%d", usersSearchFlags.limit))
	}
	path := "/api/v1/admin-console/users"
	if usersSearchFlags.org != "" {
		params.Set("orgId", usersSearchFlags.org)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(redactSecretsFromJSON(body))
		return err
	}
	var result struct {
		Items []adminConsoleUser `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	for _, u := range result.Items {
		state := "active"
		if !u.Active {
			state = "suspended"
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", u.ID, u.Email, u.Role, state)
	}
	return nil
}