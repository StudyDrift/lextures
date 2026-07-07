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

var peopleCmd = &cobra.Command{
	Use:   "people",
	Short: "Search and manage people (platform admin)",
}

var peopleListFlags struct {
	query string
	limit int
}

var peopleListCmd = &cobra.Command{
	Use:   "list",
	Short: "Search people across the platform",
	RunE:  runPeopleList,
}

func init() {
	peopleListCmd.Flags().StringVar(&peopleListFlags.query, "query", "", "search query (required)")
	_ = peopleListCmd.MarkFlagRequired("query")
	peopleListCmd.Flags().IntVar(&peopleListFlags.limit, "limit", 25, "maximum results")

	peopleCmd.AddCommand(peopleListCmd)
	rootCmd.AddCommand(peopleCmd)
}

func runPeopleList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	params := url.Values{}
	params.Set("q", peopleListFlags.query)
	if peopleListFlags.limit > 0 {
		params.Set("perPage", fmt.Sprintf("%d", peopleListFlags.limit))
	}
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/people?"+params.Encode(), nil)
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
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tEMAIL\tNAME\tORG")
	for _, item := range result.Items {
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", item["id"], item["email"], item["displayName"], item["orgId"])
	}
	return w.Flush()
}