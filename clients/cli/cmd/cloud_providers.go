package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var cloudProvidersCmd = &cobra.Command{
	Use:   "cloud-providers",
	Short: "Configure cloud storage provider integrations",
}

var cloudProvidersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cloud provider settings",
	RunE:  runCloudProvidersList,
}

var cloudProvidersGetCmd = &cobra.Command{
	Use:   "get <provider>",
	Short: "Get a cloud provider configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runCloudProvidersGet,
}

var cloudProvidersSetFlags struct {
	file string
}

var cloudProvidersSetCmd = &cobra.Command{
	Use:   "set <provider>",
	Short: "Update a cloud provider configuration from JSON",
	Args:  cobra.ExactArgs(1),
	RunE:  runCloudProvidersSet,
}

var cloudProvidersTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Validate provider connectivity configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runCloudProvidersTest,
}

func init() {
	cloudProvidersSetCmd.Flags().StringVar(&cloudProvidersSetFlags.file, "file", "", "provider JSON (enabled, clientId, apiKey, appKey)")
	_ = cloudProvidersSetCmd.MarkFlagRequired("file")

	cloudProvidersCmd.AddCommand(cloudProvidersListCmd, cloudProvidersGetCmd, cloudProvidersSetCmd, cloudProvidersTestCmd)
	rootCmd.AddCommand(cloudProvidersCmd)
}

func runCloudProvidersList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, _, err := fetchAdminCloudProviders(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		out, _ := json.Marshal(rows)
		_, err = cmd.OutOrStdout().Write(out)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROVIDER\tENABLED\tCLIENT_ID\tUPDATED")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%v\t%s\t%s\n", row.Provider, row.Enabled, row.ClientID, row.UpdatedAt)
	}
	return w.Flush()
}

func runCloudProvidersGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	row, _, err := fetchCloudProvider(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(row)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "provider=%s enabled=%v clientId=%s updated=%s\n",
		row.Provider, row.Enabled, row.ClientID, row.UpdatedAt)
	return nil
}

func runCloudProvidersSet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(cloudProvidersSetFlags.file)
	if err != nil {
		return err
	}
	if err := setCloudProvider(c, args[0], payload); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"provider": args[0], "ok": true})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cloud provider %s updated.\n", args[0])
	return nil
}

func runCloudProvidersTest(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	row, _, err := fetchCloudProvider(c, args[0])
	if err != nil {
		return err
	}
	if err := testCloudProvider(row); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"provider": row.Provider,
			"ok":       true,
			"message":  "Provider is enabled and credentials appear configured.",
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider %s is configured.\n", row.Provider)
	return nil
}