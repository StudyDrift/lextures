package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var ltiCmd = &cobra.Command{
	Use:   "lti",
	Short: "Manage LTI 1.3 tool registrations and platform config",
}

var ltiToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage LTI external tools",
}

var ltiToolsListFlags struct {
	course string
}

var ltiToolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List LTI tools",
	RunE:  runLTIToolsList,
}

var ltiToolsRegisterFlags struct {
	file       string
	deployment bool
}

var ltiToolsRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an LTI tool or parent platform from JSON",
	RunE:  runLTIToolsRegister,
}

var ltiToolsUpdateFlags struct {
	active     *bool
	deployment bool
}

var ltiToolsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an LTI tool (active flag)",
	Args:  cobra.ExactArgs(1),
	RunE:  runLTIToolsUpdate,
}

var ltiToolsDeleteFlags struct {
	deployment bool
}

var ltiToolsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an LTI tool or parent platform registration",
	Args:  cobra.ExactArgs(1),
	RunE:  runLTIToolsDelete,
}

var ltiToolsTestCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Validate an LTI tool registration exists and is active",
	Args:  cobra.ExactArgs(1),
	RunE:  runLTIToolsTest,
}

var ltiDeploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "Inspect LTI Advantage deployments",
}

var ltiDeploymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployment ids from parent platform registrations",
	RunE:  runLTIDeploymentsList,
}

var ltiDeploymentsCreateFlags struct {
	file string
}

var ltiDeploymentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a parent platform registration with deployment ids",
	RunE:  runLTIDeploymentsCreate,
}

var ltiPlatformConfigCmd = &cobra.Command{
	Use:   "platform-config",
	Short: "Export issuer, JWKS URL, and deployment ids for tool vendors",
	RunE:  runLTIPlatformConfig,
}

func init() {
	ltiToolsListCmd.Flags().StringVar(&ltiToolsListFlags.course, "course", "", "course code for course-scoped tools")

	ltiToolsRegisterCmd.Flags().StringVar(&ltiToolsRegisterFlags.file, "file", "", "tool JSON file (required)")
	_ = ltiToolsRegisterCmd.MarkFlagRequired("file")
	ltiToolsRegisterCmd.Flags().BoolVar(&ltiToolsRegisterFlags.deployment, "deployment", false, "register a parent platform (deployment) instead of an external tool")

	activeDefault := true
	ltiToolsUpdateFlags.active = &activeDefault
	ltiToolsUpdateCmd.Flags().BoolVar(ltiToolsUpdateFlags.active, "active", activeDefault, "set active state")
	ltiToolsUpdateCmd.Flags().BoolVar(&ltiToolsUpdateFlags.deployment, "deployment", false, "update a parent platform registration")

	ltiToolsDeleteCmd.Flags().BoolVar(&ltiToolsDeleteFlags.deployment, "deployment", false, "delete a parent platform registration")

	ltiDeploymentsCreateCmd.Flags().StringVar(&ltiDeploymentsCreateFlags.file, "file", "", "parent platform JSON (required)")
	_ = ltiDeploymentsCreateCmd.MarkFlagRequired("file")

	ltiToolsCmd.AddCommand(ltiToolsListCmd, ltiToolsRegisterCmd, ltiToolsUpdateCmd, ltiToolsDeleteCmd, ltiToolsTestCmd)
	ltiDeploymentsCmd.AddCommand(ltiDeploymentsListCmd, ltiDeploymentsCreateCmd)
	ltiCmd.AddCommand(ltiToolsCmd, ltiDeploymentsCmd, ltiPlatformConfigCmd)
	rootCmd.AddCommand(ltiCmd)
}

func runLTIToolsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if ltiToolsListFlags.course != "" {
		body, err := fetchCourseLTITools(c, ltiToolsListFlags.course)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(body)
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	parents, tools, raw, err := fetchLTIRegistrations(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tID\tNAME\tCLIENT_ID\tACTIVE")
	for _, t := range tools {
		_, _ = fmt.Fprintf(w, "tool\t%s\t%s\t%s\t%v\n", t.ID, t.Name, t.ClientID, t.Active)
	}
	for _, p := range parents {
		_, _ = fmt.Fprintf(w, "parent\t%s\t%s\t%s\t%v\n", p.ID, p.Name, p.ClientID, p.Active)
	}
	return w.Flush()
}

func runLTIToolsRegister(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(ltiToolsRegisterFlags.file)
	if err != nil {
		return err
	}
	var body []byte
	if ltiToolsRegisterFlags.deployment {
		body, err = registerLTIParentPlatform(c, payload)
	} else {
		body, err = registerLTIExternalTool(c, payload)
	}
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "LTI registration created.")
	_, _ = fmt.Fprintln(os.Stderr, "Save any one-time secrets now — they will not be shown again on re-fetch.")
	return nil
}

func runLTIToolsUpdate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	id := args[0]
	if ltiToolsUpdateFlags.deployment {
		return updateLTIParentPlatform(c, id, *ltiToolsUpdateFlags.active)
	}
	return updateLTIExternalTool(c, id, *ltiToolsUpdateFlags.active)
}

func runLTIToolsDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	id := args[0]
	if ltiToolsDeleteFlags.deployment {
		return deleteLTIParentPlatform(c, id)
	}
	return deleteLTIExternalTool(c, id)
}

func runLTIToolsTest(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	id := args[0]
	_, tools, _, err := fetchLTIRegistrations(c)
	if err != nil {
		return err
	}
	for _, t := range tools {
		if t.ID == id {
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"ok": t.Active, "tool": t})
			}
			if !t.Active {
				return fmt.Errorf("tool %s exists but is inactive", id)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Tool %s (%s) is registered and active.\n", t.Name, t.ClientID)
			return nil
		}
	}
	return fmt.Errorf("tool %q not found", id)
}

func runLTIDeploymentsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	parents, _, raw, err := fetchLTIRegistrations(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PLATFORM\tDEPLOYMENT_ID")
	for _, p := range parents {
		for _, d := range p.DeploymentIds {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", p.Name, d)
		}
	}
	return w.Flush()
}

func runLTIDeploymentsCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(ltiDeploymentsCreateFlags.file)
	if err != nil {
		return err
	}
	body, err := registerLTIParentPlatform(c, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Parent platform registration created.")
	return nil
}

func runLTIPlatformConfig(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	out, err := buildLTIPlatformConfig(c, Cfg.Server)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "issuer=%v\njwksUrl=%v\ndeploymentIds=%v\n",
		out["issuer"], out["jwksUrl"], out["deploymentIds"])
	return nil
}