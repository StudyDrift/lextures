package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var storageQuotasCmd = &cobra.Command{
	Use:   "storage-quotas",
	Short: "Manage storage quota limits",
}

var storageQuotasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List storage quotas and usage",
	RunE:  runStorageQuotasList,
}

var storageQuotasSetFlags struct {
	scope    string
	scopeID  string
	limit    string
	unlimited bool
}

var storageQuotasSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a storage quota limit",
	RunE:  runStorageQuotasSet,
}

func init() {
	storageQuotasSetCmd.Flags().StringVar(&storageQuotasSetFlags.scope, "scope", "", "scope: tenant, course, or user (required)")
	storageQuotasSetCmd.Flags().StringVar(&storageQuotasSetFlags.scopeID, "scope-id", "", "scope UUID (required)")
	storageQuotasSetCmd.Flags().StringVar(&storageQuotasSetFlags.limit, "limit", "", "limit in bytes")
	storageQuotasSetCmd.Flags().BoolVar(&storageQuotasSetFlags.unlimited, "unlimited", false, "remove the quota limit")
	_ = storageQuotasSetCmd.MarkFlagRequired("scope")
	_ = storageQuotasSetCmd.MarkFlagRequired("scope-id")

	storageQuotasCmd.AddCommand(storageQuotasListCmd, storageQuotasSetCmd)
	rootCmd.AddCommand(storageQuotasCmd)
}

func runStorageQuotasList(cmd *cobra.Command, _ []string) error {
	rows, raw, err := listStorageQuotas(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SCOPE\tSCOPE_ID\tUSED\tLIMIT\tPCT")
	for _, r := range rows {
		limit := "unlimited"
		if r.LimitBytes != nil {
			limit = formatFileBytes(*r.LimitBytes)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.1f%%\n",
			r.Scope, r.ScopeID, formatFileBytes(r.UsedBytes), limit, r.PercentUsed)
	}
	return w.Flush()
}

func runStorageQuotasSet(cmd *cobra.Command, _ []string) error {
	var limit *int64
	if storageQuotasSetFlags.unlimited {
		limit = nil
	} else if storageQuotasSetFlags.limit == "" {
		return fmt.Errorf("provide --limit or --unlimited")
	} else {
		n, err := strconv.ParseInt(storageQuotasSetFlags.limit, 10, 64)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid --limit")
		}
		limit = &n
	}
	if err := setStorageQuota(client.New(Cfg.Server, Cfg.APIKey),
		storageQuotasSetFlags.scope, storageQuotasSetFlags.scopeID, limit); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"status": "updated"})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Storage quota updated.")
	return nil
}