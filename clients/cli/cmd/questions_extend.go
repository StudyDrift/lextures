package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

type questionBankPublic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CourseID    string `json:"courseId,omitempty"`
}

var questionsBanksCmd = &cobra.Command{
	Use:   "banks",
	Short: "List or create question banks",
}

var questionsBanksListFlags struct {
	course string
}

var questionsBanksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List question banks",
	RunE:  runQuestionsBanksList,
}

var questionsBanksCreateFlags struct {
	course      string
	name        string
	description string
}

var questionsBanksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a question bank (pool)",
	RunE:  runQuestionsBanksCreate,
}

var questionsExportFlags struct {
	bank  string
	out   string
	qti   bool
	quiet bool
}

var questionsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export questions from a bank (QTI .zip)",
	RunE:  runQuestionsExport,
}

func init() {
	questionsBanksListCmd.Flags().StringVar(&questionsBanksListFlags.course, "course", "", "course code (required)")
	_ = questionsBanksListCmd.MarkFlagRequired("course")

	questionsBanksCreateCmd.Flags().StringVar(&questionsBanksCreateFlags.course, "course", "", "course code (required)")
	questionsBanksCreateCmd.Flags().StringVar(&questionsBanksCreateFlags.name, "name", "", "bank name (required)")
	questionsBanksCreateCmd.Flags().StringVar(&questionsBanksCreateFlags.description, "description", "", "bank description")
	_ = questionsBanksCreateCmd.MarkFlagRequired("course")
	_ = questionsBanksCreateCmd.MarkFlagRequired("name")

	questionsExportCmd.Flags().StringVar(&questionsExportFlags.bank, "bank", "", "question bank ID (required)")
	questionsExportCmd.Flags().StringVar(&questionsExportFlags.out, "out", "", "output .zip path (required)")
	questionsExportCmd.Flags().BoolVar(&questionsExportFlags.qti, "qti", true, "export as QTI package")
	questionsExportCmd.Flags().BoolVar(&questionsExportFlags.quiet, "quiet", false, "suppress progress output")
	_ = questionsExportCmd.MarkFlagRequired("bank")
	_ = questionsExportCmd.MarkFlagRequired("out")

	questionsBanksCmd.AddCommand(questionsBanksListCmd, questionsBanksCreateCmd)
	questionsCmd.AddCommand(questionsExportCmd, questionsBanksCmd)
}

func questionBankAPIPath(courseCode, suffix string) string {
	return "/api/v1/courses/" + courseCode + suffix
}

func runQuestionsBanksList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, questionBankAPIPath(questionsBanksListFlags.course, "/question-pools"), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing banks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	var banks []questionBankPublic
	if err := json.Unmarshal(body, &banks); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(banks)
	}
	if len(banks) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No question banks.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME")
	for _, b := range banks {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", b.ID, b.Name)
	}
	return w.Flush()
}

func runQuestionsBanksCreate(cmd *cobra.Command, _ []string) error {
	payload := map[string]string{"name": questionsBanksCreateFlags.name}
	if questionsBanksCreateFlags.description != "" {
		payload["description"] = questionsBanksCreateFlags.description
	}
	raw, _ := json.Marshal(payload)
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, questionBankAPIPath(questionsBanksCreateFlags.course, "/question-pools"), bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating bank: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	var bank questionBankPublic
	if err := json.Unmarshal(body, &bank); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(bank)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created question bank %s\n", bank.ID)
	return nil
}

func runQuestionsExport(cmd *cobra.Command, _ []string) error {
	outPath := filepath.Clean(questionsExportFlags.out)
	if strings.Contains(outPath, "..") {
		return fmt.Errorf("invalid output path: %s", questionsExportFlags.out)
	}
	if !strings.HasSuffix(strings.ToLower(outPath), ".zip") {
		outPath += ".zip"
	}

	path := "/api/question-banks/" + questionsExportFlags.bank + "/export"
	if questionsExportFlags.qti {
		path += "?format=qti"
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("exporting: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading export: %w", err)
	}
	if len(data) < 4 || string(data[:2]) != "PK" {
		return fmt.Errorf("export response is not a valid .zip file")
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"file":  outPath,
			"bytes": len(data),
			"qti":   questionsExportFlags.qti,
		})
	}
	if !questionsExportFlags.quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %s (%s)\n", outPath, formatFileSize(int64(len(data))))
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %s\n", outPath)
	}
	return nil
}