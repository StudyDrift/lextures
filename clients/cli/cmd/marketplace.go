package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse marketplace apps and installed integrations",
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List published marketplace apps",
	RunE:  runMarketplaceList,
}

var marketplaceGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Get a marketplace app by slug",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceGet,
}

var marketplaceInstalledCmd = &cobra.Command{
	Use:   "installed",
	Short: "List installed marketplace apps (admin)",
	RunE:  runMarketplaceInstalled,
}

var marketplaceUnpublishCmd = &cobra.Command{
	Use:   "unpublish <install-id>",
	Short: "Revoke an installed marketplace app",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceUnpublish,
}

func init() {
	marketplaceCmd.AddCommand(marketplaceListCmd, marketplaceGetCmd, marketplaceInstalledCmd, marketplaceUnpublishCmd)
	rootCmd.AddCommand(marketplaceCmd)
}

func runMarketplaceList(cmd *cobra.Command, args []string) error {
	apps, raw, err := fetchMarketplaceApps(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SLUG\tNAME")
	for _, app := range apps {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", app.Slug, app.Name)
	}
	return w.Flush()
}

func runMarketplaceGet(cmd *cobra.Command, args []string) error {
	body, err := fetchMarketplaceApp(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runMarketplaceInstalled(cmd *cobra.Command, args []string) error {
	body, err := fetchAdminInstalledApps(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Installed []map[string]any `json:"installed"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tAPP")
	for _, row := range out.Installed {
		_, _ = fmt.Fprintf(w, "%v\t%v\n", row["id"], row["name"])
	}
	return w.Flush()
}

func runMarketplaceUnpublish(cmd *cobra.Command, args []string) error {
	if err := revokeInstalledApp(client.New(Cfg.Server, Cfg.APIKey), args[0]); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Revoked installed app")
	return nil
}