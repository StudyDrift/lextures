package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var quarantineCmd = &cobra.Command{
	Use:   "quarantine",
	Short: "Triage AV-scan quarantined files",
}

var quarantineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quarantined files (metadata only)",
	RunE:  runQuarantineList,
}

var quarantineGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get quarantined file metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuarantineGet,
}

var quarantineReleaseFlags struct {
	yes bool
}

var quarantineReleaseCmd = &cobra.Command{
	Use:   "release <id>",
	Short: "Release a quarantined file after manual review",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuarantineRelease,
}

var quarantineDeleteFlags struct {
	yes bool
}

var quarantineDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a quarantined file",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuarantineDelete,
}

func init() {
	quarantineReleaseCmd.Flags().BoolVar(&quarantineReleaseFlags.yes, "yes", false, "confirm release")
	quarantineDeleteCmd.Flags().BoolVar(&quarantineDeleteFlags.yes, "yes", false, "confirm delete")

	quarantineCmd.AddCommand(quarantineListCmd, quarantineGetCmd, quarantineReleaseCmd, quarantineDeleteCmd)
	rootCmd.AddCommand(quarantineCmd)
}

type quarantineItem struct {
	ObjectID  string  `json:"object_id"`
	ObjectKey string  `json:"object_key"`
	VirusName *string `json:"virus_name,omitempty"`
	UploadedAt string `json:"uploaded_at"`
}

func fetchQuarantineItems(c *client.Client) ([]quarantineItem, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/quarantine", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Items []quarantineItem `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Items, body, nil
}

func runQuarantineList(cmd *cobra.Command, args []string) error {
	items, raw, err := fetchQuarantineItems(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tKEY\tUPLOADED")
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", item.ObjectID, item.ObjectKey, item.UploadedAt)
	}
	return w.Flush()
}

func runQuarantineGet(cmd *cobra.Command, args []string) error {
	items, _, err := fetchQuarantineItems(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ObjectID == args[0] {
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
			}
			virus := ""
			if item.VirusName != nil {
				virus = *item.VirusName
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s virus=%s uploaded=%s\n", item.ObjectID, item.ObjectKey, virus, item.UploadedAt)
			return nil
		}
	}
	return fmt.Errorf("quarantined object %q not found", args[0])
}

func runQuarantineRelease(cmd *cobra.Command, args []string) error {
	if !quarantineReleaseFlags.yes {
		return fmt.Errorf("quarantine release is destructive; re-run with --yes")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin/quarantine/" + url.PathEscape(args[0]) + "/release"
	req, err := c.NewRequest(http.MethodPost, path, nil)
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
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Released quarantined file")
	return nil
}

func runQuarantineDelete(cmd *cobra.Command, args []string) error {
	if !quarantineDeleteFlags.yes {
		return fmt.Errorf("quarantine delete is destructive; re-run with --yes")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin/quarantine/" + url.PathEscape(args[0])
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Deleted quarantined file")
	return nil
}