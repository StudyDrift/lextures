package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var sbgCmd = &cobra.Command{
	Use:   "sbg",
	Short: "Standards-based grading rollups",
}

var sbgGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get course SBG standards and mastery heatmap for a grading period",
	Args:  cobra.ExactArgs(1),
	RunE:  runSbgGet,
}

var sbgGetFlags struct {
	period string
}

func init() {
	sbgGetCmd.Flags().StringVar(&sbgGetFlags.period, "period", "", "grading period code (required)")
	_ = sbgGetCmd.MarkFlagRequired("period")

	sbgCmd.AddCommand(sbgGetCmd)
	rootCmd.AddCommand(sbgCmd)
}

func runSbgGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	period := strings.TrimSpace(sbgGetFlags.period)

	standards, err := fetchCourseSbgStandards(c, course)
	if err != nil {
		return err
	}
	heatmap, err := fetchSbgHeatmap(c, course, period)
	if err != nil {
		return err
	}

	out := map[string]any{
		"courseCode": course,
		"period":     period,
		"standards":  standards,
		"cells":      heatmap,
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Course %s — period %s\n", course, period)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d standard(s), %d mastery cell(s)\n", len(standards), len(heatmap))
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STANDARD\tCODE\tDESCRIPTION")
	for _, s := range standards {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", s.ID, s.Code, s.Description)
	}
	_ = w.Flush()
	if len(heatmap) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nHeatmap:")
		w = tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "STUDENT\tSTANDARD\tSCORE")
		for _, cell := range heatmap {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\n", cell.StudentID, cell.StandardID, cell.ScoreValue)
		}
		return w.Flush()
	}
	return nil
}

type sbgStandard struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type sbgHeatmapCell struct {
	StudentID  string `json:"studentId"`
	StandardID string `json:"standardId"`
	ScoreValue int    `json:"scoreValue"`
}

func fetchCourseSbgStandards(c *client.Client, course string) ([]sbgStandard, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+url.PathEscape(course)+"/sbg/standards", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing course standards: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Standards []sbgStandard `json:"standards"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out.Standards, nil
}

func fetchSbgHeatmap(c *client.Client, course, period string) ([]sbgHeatmapCell, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/sbg/heatmap/%s", url.PathEscape(course), url.PathEscape(period))
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("loading heatmap: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Cells []sbgHeatmapCell `json:"cells"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out.Cells, nil
}