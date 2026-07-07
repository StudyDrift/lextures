package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

const ferpaMasteryExportWarning = `WARNING: Mastery and report-card exports contain FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

var standardsCmd = &cobra.Command{
	Use:   "standards",
	Short: "List, import, and inspect academic standards frameworks",
}

var standardsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List standards in a framework",
	RunE:  runStandardsList,
}

var standardsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get one standard by UUID",
	Args:  cobra.ExactArgs(1),
	RunE:  runStandardsGet,
}

var standardsImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import SBG standards for an organization (CSV or native JSON framework)",
	RunE:  runStandardsImport,
}

var standardsListFlags struct {
	framework string
	grade     string
	query     string
	limit     int
}

var standardsImportFlags struct {
	file string
	org  string
}

func init() {
	standardsListCmd.Flags().StringVar(&standardsListFlags.framework, "framework", "", "framework code (required, e.g. ccss-math)")
	_ = standardsListCmd.MarkFlagRequired("framework")
	standardsListCmd.Flags().StringVar(&standardsListFlags.grade, "grade", "", "grade band filter")
	standardsListCmd.Flags().StringVar(&standardsListFlags.query, "q", "", "search query (uses /standards/search when set)")
	standardsListCmd.Flags().IntVar(&standardsListFlags.limit, "limit", 200, "maximum results")

	standardsImportCmd.Flags().StringVar(&standardsImportFlags.file, "file", "", "framework CSV or JSON file (required)")
	_ = standardsImportCmd.MarkFlagRequired("file")
	standardsImportCmd.Flags().StringVar(&standardsImportFlags.org, "org", "", "organization UUID (required)")
	_ = standardsImportCmd.MarkFlagRequired("org")

	standardsCmd.AddCommand(standardsListCmd, standardsGetCmd, standardsImportCmd)
	rootCmd.AddCommand(standardsCmd)
}

func runStandardsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	params := url.Values{}
	params.Set("framework", strings.TrimSpace(standardsListFlags.framework))
	if g := strings.TrimSpace(standardsListFlags.grade); g != "" {
		params.Set("grade", g)
	}
	if standardsListFlags.limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", standardsListFlags.limit))
	}

	path := "/api/v1/standards"
	if q := strings.TrimSpace(standardsListFlags.query); q != "" {
		path = "/api/v1/standards/search"
		params.Set("q", q)
	}
	path += "?" + params.Encode()

	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing standards: %w", err)
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

	var rows []struct {
		ID          string  `json:"id"`
		Code        string  `json:"code"`
		ShortCode   *string `json:"shortCode"`
		Description string  `json:"description"`
		GradeBand   *string `json:"gradeBand"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No standards found.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tCODE\tGRADE\tDESCRIPTION")
	for _, row := range rows {
		grade := ""
		if row.GradeBand != nil {
			grade = *row.GradeBand
		}
		code := row.Code
		if row.ShortCode != nil && *row.ShortCode != "" {
			code = *row.ShortCode
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row.ID, code, grade, row.Description)
	}
	return w.Flush()
}

func runStandardsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/standards/"+url.PathEscape(args[0]), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("getting standard: %w", err)
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
	var row struct {
		ID               string  `json:"id"`
		Code             string  `json:"code"`
		Description      string  `json:"description"`
		FrameworkCode    string  `json:"frameworkCode"`
		FrameworkName    string  `json:"frameworkName"`
		FrameworkVersion string  `json:"frameworkVersion"`
		GradeBand        *string `json:"gradeBand"`
	}
	if err := json.Unmarshal(body, &row); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	grade := ""
	if row.GradeBand != nil {
		grade = *row.GradeBand
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:          %s\n", row.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Code:        %s\n", row.Code)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Framework:   %s %s (%s)\n", row.FrameworkCode, row.FrameworkVersion, row.FrameworkName)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Grade band:  %s\n", grade)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", row.Description)
	return nil
}

func runStandardsImport(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(standardsImportFlags.file)
	if err != nil {
		return fmt.Errorf("reading framework file: %w", err)
	}

	csvBytes := raw
	ext := strings.ToLower(filepath.Ext(standardsImportFlags.file))
	if ext == ".json" || (len(bytes.TrimSpace(raw)) > 0 && raw[0] == '{') {
		csvBytes, err = frameworkImportToCSV(raw)
		if err != nil {
			return err
		}
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin/orgs/" + url.PathEscape(strings.TrimSpace(standardsImportFlags.org)) + "/sbg/standards/import"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(csvBytes))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "text/csv")

	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("importing standards: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}

	var result struct {
		DomainsCreated    int      `json:"domainsCreated"`
		StandardsImported int      `json:"standardsImported"`
		Errors            []string `json:"errors"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Imported %d standard(s); created %d domain(s).\n",
		result.StandardsImported, result.DomainsCreated)
	for _, e := range result.Errors {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  error: %s\n", e)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%d row(s) failed during import", len(result.Errors))
	}
	return nil
}

func confirmFerpaExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaMasteryExportWarning)
}

func writeFileOrStdout(cmd *cobra.Command, path string, data []byte) error {
	if path == "" || path == "-" {
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("creating output directory: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}