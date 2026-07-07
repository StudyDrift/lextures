package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var integrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Manage third-party inbound integrations",
}

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected integrations",
	RunE:  runIntegrationsList,
}

var integrationsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get integration sync status",
	Args:  cobra.ExactArgs(1),
	RunE:  runIntegrationsGet,
}

var integrationsEnableCmd = &cobra.Command{
	Use:   "enable <provider>",
	Short: "Start OAuth connect flow for a provider",
	Args:  cobra.ExactArgs(1),
	RunE:  runIntegrationsEnable,
}

var integrationsDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disconnect an integration",
	Args:  cobra.ExactArgs(1),
	RunE:  runIntegrationsDisable,
}

func init() {
	integrationsCmd.AddCommand(integrationsListCmd, integrationsGetCmd, integrationsEnableCmd, integrationsDisableCmd)
	rootCmd.AddCommand(integrationsCmd)
}

func runIntegrationsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := fetchIntegrations(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tPROVIDER\tSTATUS")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", row.ID, row.Provider, row.Status)
	}
	return w.Flush()
}

func runIntegrationsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchIntegrationSyncStatus(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runIntegrationsEnable(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := connectIntegration(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		AuthorizeURL string `json:"authorizeUrl"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open this URL to connect %s:\n%s\n", args[0], out.AuthorizeURL)
	return nil
}

func runIntegrationsDisable(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := disconnectIntegration(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Integration disconnected.")
	return nil
}