package cmd

import (
	"fmt"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var plagiarismCmd = &cobra.Command{
	Use:   "plagiarism",
	Short: "Manage course plagiarism settings",
}

var plagiarismSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Get or set per-course plagiarism policy",
}

var plagiarismSettingsGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get plagiarism settings for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlagiarismSettingsGet,
}

var plagiarismSettingsSetFlags struct {
	file string
}

var plagiarismSettingsSetCmd = &cobra.Command{
	Use:   "set <course>",
	Short: "Set plagiarism settings from a JSON policy file or stdin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlagiarismSettingsSet,
}

func init() {
	plagiarismSettingsSetCmd.Flags().StringVar(&plagiarismSettingsSetFlags.file, "file", "", "JSON policy file (use - for stdin)")
	_ = plagiarismSettingsSetCmd.MarkFlagRequired("file")

	plagiarismSettingsCmd.AddCommand(plagiarismSettingsGetCmd, plagiarismSettingsSetCmd)
	plagiarismCmd.AddCommand(plagiarismSettingsCmd)
	rootCmd.AddCommand(plagiarismCmd)
}

func runPlagiarismSettingsGet(cmd *cobra.Command, args []string) error {
	raw, settings, err := fetchPlagiarismSettings(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	provider := "none"
	if settings.PlagiarismProvider != nil && strings.TrimSpace(*settings.PlagiarismProvider) != "" {
		provider = strings.TrimSpace(*settings.PlagiarismProvider)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Checks enabled: %v\n", settings.PlagiarismChecksEnabled)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\n", provider)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Alert threshold: %.1f%%\n", settings.PlagiarismAlertThresholdPct)
	return nil
}

func runPlagiarismSettingsSet(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(plagiarismSettingsSetFlags.file)
	if err != nil {
		return fmt.Errorf("reading policy file: %w", err)
	}
	patch, err := parsePlagiarismPolicyJSON(raw)
	if err != nil {
		return err
	}
	body, err := patchPlagiarismSettings(client.New(Cfg.Server, Cfg.APIKey), args[0], patch)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Updated plagiarism settings for %s", args[0]))
}