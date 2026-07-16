package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage platform and tenant settings",
}

var settingsGetCmd = &cobra.Command{
	Use:   "get <scope>",
	Short: "Get settings for a scope (platform, locale, timezone, system-prompts)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSettingsGet,
}

var settingsSetFlags struct {
	file string
}

var settingsSetCmd = &cobra.Command{
	Use:   "set <scope> [key]",
	Short: "Set settings from a JSON or prompt file",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runSettingsSet,
}

var settingsPasswordPolicyCmd = &cobra.Command{
	Use:   "password-policy",
	Short: "Manage password policy",
}

var settingsPasswordPolicyGetFlags struct {
	institution string
}

var settingsPasswordPolicyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get password policy",
	RunE:  runSettingsPasswordPolicyGet,
}

var settingsPasswordPolicySetFlags struct {
	file        string
	institution string
}

var settingsPasswordPolicySetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set password policy from a JSON file",
	RunE:  runSettingsPasswordPolicySet,
}

var settingsAIProviderCmd = &cobra.Command{
	Use:   "ai-provider",
	Short: "Manage AI provider settings (BYOK)",
	Long: `Manage organization AI provider settings (bring-your-own-key).

Supported providers (set "provider" in the JSON payload):
  openrouter, anthropic, openai, azure_openai, bedrock, vertex

Configure platform-wide credentials in the web UI under
Settings → Intelligence → Models, or see docs/ai-providers-byok.md.

Deprecated: platform openRouterApiKey on Settings → AI is legacy; prefer
per-provider credentials. Org payloads may still use byokApiKey for the
selected provider.`,
}

var settingsAIProviderGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get AI provider settings (secrets redacted)",
	Long:  "Fetch org AI provider settings. Secrets are redacted. Provider may be openrouter, anthropic, openai, azure_openai, bedrock, or vertex.",
	RunE:  runSettingsAIProviderGet,
}

var settingsAIProviderSetFlags struct {
	file string
}

var settingsAIProviderSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set AI provider settings from a JSON file",
	Long: `Update org AI provider settings from JSON.

Example:
  {"provider":"azure_openai","byokApiKey":"...","providerSettings":{"azure_base_url":"https://....openai.azure.com"}}

Providers: openrouter, anthropic, openai, azure_openai, bedrock, vertex.
Do not commit real API keys. openRouterApiKey is deprecated; use provider + byokApiKey.`,
	RunE: runSettingsAIProviderSet,
}

var settingsAIProviderTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test AI provider connectivity",
	Long:  "POST a connectivity check for the configured org AI provider (any supported backend).",
	RunE:  runSettingsAIProviderTest,
}

var settingsDataResidencyCmd = &cobra.Command{
	Use:   "data-residency",
	Short: "Inspect data residency configuration",
}

var settingsDataResidencyGetFlags struct {
	org string
}

var settingsDataResidencyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get data residency info for an organization",
	RunE:  runSettingsDataResidencyGet,
}

var settingsDataResidencySetFlags struct {
	file string
	yes  bool
}

var settingsDataResidencySetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set data residency (not supported — region is immutable after provisioning)",
	RunE:  runSettingsDataResidencySet,
}

var settingsExportFlags struct {
	file string
	org  string
}

var settingsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export tenant settings to a JSON file",
	RunE:  runSettingsExport,
}

var settingsApplyFlags struct {
	file    string
	dryRun  bool
	yes     bool
	org     string
}

var settingsApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply declarative tenant settings from a JSON file",
	RunE:  runSettingsApply,
}

func init() {
	settingsSetCmd.Flags().StringVar(&settingsSetFlags.file, "file", "", "settings JSON or prompt text file (required)")
	_ = settingsSetCmd.MarkFlagRequired("file")

	settingsPasswordPolicyGetCmd.Flags().StringVar(&settingsPasswordPolicyGetFlags.institution, "institution", "", "institution UUID")
	settingsPasswordPolicySetCmd.Flags().StringVar(&settingsPasswordPolicySetFlags.file, "file", "", "policy JSON file (required)")
	settingsPasswordPolicySetCmd.Flags().StringVar(&settingsPasswordPolicySetFlags.institution, "institution", "", "institution UUID")
	_ = settingsPasswordPolicySetCmd.MarkFlagRequired("file")

	settingsAIProviderSetCmd.Flags().StringVar(&settingsAIProviderSetFlags.file, "file", "", "provider JSON file (required)")
	_ = settingsAIProviderSetCmd.MarkFlagRequired("file")

	settingsDataResidencyGetCmd.Flags().StringVar(&settingsDataResidencyGetFlags.org, "org", "", "organization UUID (required)")
	_ = settingsDataResidencyGetCmd.MarkFlagRequired("org")
	settingsDataResidencySetCmd.Flags().StringVar(&settingsDataResidencySetFlags.file, "file", "", "residency JSON file")
	settingsDataResidencySetCmd.Flags().BoolVar(&settingsDataResidencySetFlags.yes, "yes", false, "confirm compliance-sensitive change")

	settingsExportCmd.Flags().StringVar(&settingsExportFlags.file, "file", "settings.json", "output file")
	settingsExportCmd.Flags().StringVar(&settingsExportFlags.org, "org", "", "include data residency for org UUID")

	settingsApplyCmd.Flags().StringVar(&settingsApplyFlags.file, "file", "settings.json", "settings JSON file")
	settingsApplyCmd.Flags().BoolVar(&settingsApplyFlags.dryRun, "dry-run", false, "print diff without applying")
	settingsApplyCmd.Flags().BoolVar(&settingsApplyFlags.yes, "yes", false, "confirm applying settings")
	settingsApplyCmd.Flags().StringVar(&settingsApplyFlags.org, "org", "", "org UUID for data residency export baseline")

	settingsPasswordPolicyCmd.AddCommand(settingsPasswordPolicyGetCmd, settingsPasswordPolicySetCmd)
	settingsAIProviderCmd.AddCommand(settingsAIProviderGetCmd, settingsAIProviderSetCmd, settingsAIProviderTestCmd)
	settingsDataResidencyCmd.AddCommand(settingsDataResidencyGetCmd, settingsDataResidencySetCmd)

	settingsCmd.AddCommand(
		settingsGetCmd,
		settingsSetCmd,
		settingsPasswordPolicyCmd,
		settingsAIProviderCmd,
		settingsDataResidencyCmd,
		settingsExportCmd,
		settingsApplyCmd,
	)
	rootCmd.AddCommand(settingsCmd)
}

func runSettingsGet(cmd *cobra.Command, args []string) error {
	body, err := getSettingsScope(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if args[0] == "platform" {
		var m map[string]any
		if json.Unmarshal(body, &m) == nil {
			redactSettingsSecrets(m)
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(m)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(m)
		}
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runSettingsSet(cmd *cobra.Command, args []string) error {
	scope := args[0]
	key := ""
	if len(args) == 2 {
		key = args[1]
	}
	payload, promptKey, err := loadSettingsPayload(scope, settingsSetFlags.file, key)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	if scope == "system-prompts" {
		body, err := putSystemPrompt(c, promptKey, payload["content"].(string))
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(body)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "System prompt %q updated.\n", promptKey)
		return nil
	}
	body, err := putSettingsScope(c, scope, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut && len(body) > 0 {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Settings scope %q updated.\n", scope)
	return nil
}

func runSettingsPasswordPolicyGet(cmd *cobra.Command, _ []string) error {
	body, err := getPasswordPolicy(client.New(Cfg.Server, Cfg.APIKey), settingsPasswordPolicyGetFlags.institution)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runSettingsPasswordPolicySet(cmd *cobra.Command, _ []string) error {
	payload, err := loadJSONFile(settingsPasswordPolicySetFlags.file)
	if err != nil {
		return err
	}
	body, err := putPasswordPolicy(client.New(Cfg.Server, Cfg.APIKey), settingsPasswordPolicySetFlags.institution, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password policy updated.")
	return nil
}

func runSettingsAIProviderGet(cmd *cobra.Command, _ []string) error {
	m, raw, err := getAIProviderSettings(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(m)
	}
	_ = raw
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

func runSettingsAIProviderSet(cmd *cobra.Command, _ []string) error {
	payload, err := loadJSONFile(settingsAIProviderSetFlags.file)
	if err != nil {
		return err
	}
	body, err := putAIProviderSettings(client.New(Cfg.Server, Cfg.APIKey), payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "AI provider settings updated.")
	return nil
}

func runSettingsAIProviderTest(cmd *cobra.Command, _ []string) error {
	body, err := testAIProviderSettings(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runSettingsDataResidencyGet(cmd *cobra.Command, _ []string) error {
	body, err := getDataResidency(client.New(Cfg.Server, Cfg.APIKey), settingsDataResidencyGetFlags.org)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runSettingsDataResidencySet(_ *cobra.Command, _ []string) error {
	if !settingsDataResidencySetFlags.yes {
		return fmt.Errorf("data residency changes require --yes and are audited; region is immutable after org provisioning")
	}
	return fmt.Errorf("data residency set is not supported: organization data region cannot be changed after provisioning")
}

func runSettingsExport(cmd *cobra.Command, _ []string) error {
	out, err := exportTenantSettings(client.New(Cfg.Server, Cfg.APIKey), settingsExportFlags.org)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsExportFlags.file, raw, 0o600); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported settings to %s\n", settingsExportFlags.file)
	return nil
}

func runSettingsApply(cmd *cobra.Command, _ []string) error {
	data, err := os.ReadFile(settingsApplyFlags.file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	var desired settingsExportFile
	if err := json.Unmarshal(data, &desired); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	current, err := exportTenantSettings(c, settingsApplyFlags.org)
	if err != nil {
		return err
	}
	diff := computeSettingsApplyDiff(current, desired)
	if settingsApplyFlags.dryRun {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(diff)
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "dry-run: planned changes:")
		return enc.Encode(diff)
	}
	if !settingsApplyFlags.yes {
		return fmt.Errorf("settings apply requires --yes")
	}
	if err := applyTenantSettings(c, desired, settingsApplyFlags.yes); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"status": "applied"})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Settings applied.")
	return nil
}