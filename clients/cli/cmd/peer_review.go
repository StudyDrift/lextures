package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var peerReviewCmd = &cobra.Command{
	Use:   "peer-review",
	Short: "Configure and allocate assignment peer reviews",
}

var peerReviewStatusCmd = &cobra.Command{
	Use:   "status <assignment>",
	Short: "Show peer-review allocation status for an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runPeerReviewStatus,
}

var peerReviewAllocateCmd = &cobra.Command{
	Use:   "allocate <assignment>",
	Short: "Allocate peer reviewers for submitted work",
	Args:  cobra.ExactArgs(1),
	RunE:  runPeerReviewAllocate,
}

var peerReviewListCmd = &cobra.Command{
	Use:   "list <assignment>",
	Short: "List per-submission peer-review progress",
	Args:  cobra.ExactArgs(1),
	RunE:  runPeerReviewList,
}

var peerReviewStatusFlags struct {
	course string
}

var peerReviewAllocateFlags struct {
	course string
	per    int
}

var peerReviewListFlags struct {
	course string
}

func init() {
	peerReviewStatusCmd.Flags().StringVar(&peerReviewStatusFlags.course, "course", "", "course code (required)")
	_ = peerReviewStatusCmd.MarkFlagRequired("course")

	peerReviewAllocateCmd.Flags().StringVar(&peerReviewAllocateFlags.course, "course", "", "course code (required)")
	_ = peerReviewAllocateCmd.MarkFlagRequired("course")
	peerReviewAllocateCmd.Flags().IntVar(&peerReviewAllocateFlags.per, "per", 0, "reviews per reviewer before allocating (1–20)")

	peerReviewListCmd.Flags().StringVar(&peerReviewListFlags.course, "course", "", "course code (required)")
	_ = peerReviewListCmd.MarkFlagRequired("course")

	peerReviewCmd.AddCommand(peerReviewStatusCmd, peerReviewAllocateCmd, peerReviewListCmd)
	rootCmd.AddCommand(peerReviewCmd)
}

func runPeerReviewStatus(cmd *cobra.Command, args []string) error {
	summary, raw, err := fetchPeerReviewSummary(client.New(Cfg.Server, Cfg.APIKey), peerReviewStatusFlags.course, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Total allocations: %v\n", summary["totalAllocations"])
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Completed reviews: %v\n", summary["completedReviews"])
	incomplete, _ := summary["incompleteReviewers"].([]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Incomplete reviewers: %d\n", len(incomplete))
	outliers, _ := summary["outlierReviewers"].([]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Outlier reviewers: %d\n", len(outliers))
	return nil
}

func runPeerReviewAllocate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := peerReviewAllocateFlags.course
	itemID := args[0]

	if peerReviewAllocateFlags.per > 0 {
		if err := peerReviewPerReviewer(peerReviewAllocateFlags.per); err != nil {
			return err
		}
		if err := upsertPeerReviewConfig(c, course, itemID, peerReviewAllocateFlags.per); err != nil {
			return err
		}
	}

	path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/peer-review/allocate",
		url.PathEscape(course), url.PathEscape(itemID))
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("allocating peer reviews: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		AllocationsCreated int `json:"allocationsCreated"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %d peer-review allocation(s).\n", out.AllocationsCreated)
	return nil
}

func runPeerReviewList(cmd *cobra.Command, args []string) error {
	summary, raw, err := fetchPeerReviewSummary(client.New(Cfg.Server, Cfg.APIKey), peerReviewListFlags.course, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	submissions, _ := summary["submissions"].([]any)
	if len(submissions) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No submissions with peer-review data.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SUBMISSION_ID\tSTUDENT_USER_ID\tREVIEW_COUNT\tPEER_AGGREGATE")
	for _, row := range submissions {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		aggregate := ""
		if v, ok := m["peerAggregate"].(float64); ok {
			aggregate = fmt.Sprintf("%.2f", v)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\t%s\n",
			stringField(m, "submissionId"),
			stringField(m, "studentUserId"),
			m["reviewCount"],
			aggregate,
		)
	}
	return w.Flush()
}

func fetchPeerReviewSummary(c *client.Client, course, itemID string) (map[string]any, []byte, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/peer-review/summary",
		url.PathEscape(course), url.PathEscape(itemID))
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("loading peer-review summary: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, apiErrorBody(resp.StatusCode, body)
	}
	var summary map[string]any
	if err := json.Unmarshal(body, &summary); err != nil {
		return nil, nil, fmt.Errorf("decoding response: %w", err)
	}
	return summary, body, nil
}

func upsertPeerReviewConfig(c *client.Client, course, itemID string, per int) error {
	cfg := map[string]any{
		"reviewsPerReviewer": per,
		"anonymity":          "double_blind",
		"gradeMode":          "none",
		"blendWeight":        0,
		"aggregation":        "mean",
		"excludeSameGroup":   true,
	}

	if summary, _, err := fetchPeerReviewSummary(c, course, itemID); err == nil {
		if existing, ok := summary["config"].(map[string]any); ok && existing != nil {
			for _, key := range []string{"anonymity", "gradeMode", "blendWeight", "aggregation", "excludeSameGroup", "opensAt", "closesAt"} {
				if v, ok := existing[key]; ok {
					cfg[key] = v
				}
			}
			cfg["reviewsPerReviewer"] = per
		}
	}

	payload, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/peer-review",
		url.PathEscape(course), url.PathEscape(itemID))
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating peer-review config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}