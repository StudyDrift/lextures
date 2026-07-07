package cmd

import (
	"bytes"
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

var evaluationTemplatesCmd = &cobra.Command{
	Use:   "evaluation-templates",
	Short: "Manage institution-wide course evaluation templates (admin)",
}

var evaluationTemplatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List evaluation templates for your organization",
	RunE:  runEvaluationTemplatesList,
}

var evaluationTemplatesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get one evaluation template",
	Args:  cobra.ExactArgs(1),
	RunE:  runEvaluationTemplatesGet,
}

var evaluationTemplatesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an evaluation template",
	RunE:  runEvaluationTemplatesCreate,
}

var evaluationTemplatesCreateFlags struct {
	name string
	file string
}

func init() {
	evaluationTemplatesCreateCmd.Flags().StringVar(&evaluationTemplatesCreateFlags.name, "name", "", "template name")
	evaluationTemplatesCreateCmd.Flags().StringVar(&evaluationTemplatesCreateFlags.file, "file", "", "JSON body with name and questions")

	evaluationTemplatesCmd.AddCommand(
		evaluationTemplatesListCmd,
		evaluationTemplatesGetCmd,
		evaluationTemplatesCreateCmd,
	)
	rootCmd.AddCommand(evaluationTemplatesCmd)
}

func runEvaluationTemplatesList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/evaluation-templates", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing evaluation templates: %w", err)
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
		Templates []map[string]any `json:"templates"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(out.Templates) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No evaluation templates.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tUPDATED")
	for _, tmpl := range out.Templates {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			stringField(tmpl, "id"),
			stringField(tmpl, "name"),
			stringField(tmpl, "updatedAt"),
		)
	}
	return w.Flush()
}

func runEvaluationTemplatesGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin/evaluation-templates/" + url.PathEscape(args[0])
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("getting evaluation template: %w", err)
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
	var tmpl map[string]any
	if err := json.Unmarshal(body, &tmpl); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:        %s\n", stringField(tmpl, "id"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name:      %s\n", stringField(tmpl, "name"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated:   %s\n", stringField(tmpl, "updatedAt"))
	questions, _ := tmpl["questions"].([]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Questions: %d\n", len(questions))
	return nil
}

func runEvaluationTemplatesCreate(cmd *cobra.Command, args []string) error {
	payload, err := evaluationTemplateCreatePayload()
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/evaluation-templates", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating evaluation template: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var tmpl map[string]any
	if err := json.Unmarshal(body, &tmpl); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created evaluation template %s (%s).\n",
		stringField(tmpl, "id"), stringField(tmpl, "name"))
	return nil
}

func evaluationTemplateCreatePayload() ([]byte, error) {
	if path := strings.TrimSpace(evaluationTemplatesCreateFlags.file); path != "" {
		raw, err := readInputFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading template file: %w", err)
		}
		return raw, nil
	}
	name := strings.TrimSpace(evaluationTemplatesCreateFlags.name)
	if name == "" {
		return nil, fmt.Errorf("specify --name or --file")
	}
	return json.Marshal(map[string]any{
		"name":      name,
		"questions": []any{},
	})
}