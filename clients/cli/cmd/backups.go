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

var backupsCmd = &cobra.Command{
	Use:   "backups",
	Short: "Inspect backup operations and record restore drills",
}

var backupsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup module status",
	RunE:  runBackupsStatus,
}

var backupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backup tiers and recent restore drills",
	RunE:  runBackupsList,
}

var backupsCreateFlags struct {
	file string
}

var backupsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Record a restore drill (backup ops surface)",
	RunE:  runBackupsCreate,
}

func init() {
	backupsCreateCmd.Flags().StringVar(&backupsCreateFlags.file, "file", "", "restore drill JSON file (required)")
	_ = backupsCreateCmd.MarkFlagRequired("file")

	backupsCmd.AddCommand(backupsStatusCmd, backupsListCmd, backupsCreateCmd)
	rootCmd.AddCommand(backupsCmd)
}

func fetchBackupStatus(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/internal/ops/backup-status", nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func runBackupsStatus(cmd *cobra.Command, args []string) error {
	body, err := fetchBackupStatus(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runBackupsList(cmd *cobra.Command, args []string) error {
	body, err := fetchBackupStatus(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Tiers []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"tiers"`
		RestoreDrills []struct {
			ID        string `json:"id"`
			DrillDate string `json:"drillDate"`
			Pass      *bool  `json:"pass"`
		} `json:"restoreDrills"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TIER\tSTATUS")
	for _, tier := range out.Tiers {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", tier.Name, tier.Status)
	}
	_, _ = fmt.Fprintln(w, "\nDRILL\tDATE\tPASS")
	for _, drill := range out.RestoreDrills {
		pass := ""
		if drill.Pass != nil {
			pass = fmt.Sprintf("%v", *drill.Pass)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", drill.ID, drill.DrillDate, pass)
	}
	return w.Flush()
}

func runBackupsCreate(cmd *cobra.Command, args []string) error {
	payload, err := loadJSONFile(backupsCreateFlags.file)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/internal/ops/restore-drill", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}