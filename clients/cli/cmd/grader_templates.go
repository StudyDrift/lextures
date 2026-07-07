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

var graderTemplatesCmd = &cobra.Command{
	Use:   "grader-templates",
	Short: "Manage course grader-agent workflow templates",
}

var graderTemplatesListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List grader-agent templates for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderTemplatesList,
}

var graderTemplatesGetCmd = &cobra.Command{
	Use:   "get <course> <template>",
	Short: "Get one grader-agent template",
	Args:  cobra.ExactArgs(2),
	RunE:  runGraderTemplatesGet,
}

var graderTemplatesCreateCmd = &cobra.Command{
	Use:   "create <course>",
	Short: "Create a grader-agent template",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderTemplatesCreate,
}

var graderTemplatesCreateFlags struct {
	name string
	file string
}

func init() {
	graderTemplatesCreateCmd.Flags().StringVar(&graderTemplatesCreateFlags.name, "name", "", "template name")
	graderTemplatesCreateCmd.Flags().StringVar(&graderTemplatesCreateFlags.file, "file", "", "JSON body with name and workflowGraph (required)")

	graderTemplatesCmd.AddCommand(graderTemplatesListCmd, graderTemplatesGetCmd, graderTemplatesCreateCmd)
	rootCmd.AddCommand(graderTemplatesCmd)
}

func runGraderTemplatesList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := graderAgentCoursePath(args[0], "/grader-agent-templates")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing grader templates: %w", err)
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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No grader-agent templates.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tBUILTIN\tUPDATED")
	for _, tmpl := range out.Templates {
		builtin := "no"
		if v, ok := tmpl["isBuiltin"].(bool); ok && v {
			builtin = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			stringField(tmpl, "id"),
			stringField(tmpl, "name"),
			builtin,
			stringField(tmpl, "updatedAt"),
		)
	}
	return w.Flush()
}

func runGraderTemplatesGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := graderAgentCoursePath(args[0], "/grader-agent-templates/"+url.PathEscape(args[1]))
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("getting grader template: %w", err)
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
		Template map[string]any `json:"template"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	tmpl := out.Template
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:      %s\n", stringField(tmpl, "id"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name:    %s\n", stringField(tmpl, "name"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated: %s\n", stringField(tmpl, "updatedAt"))
	if _, ok := tmpl["workflowGraph"]; ok {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Workflow: yes")
	}
	return nil
}

func graderTemplateCreatePayload() ([]byte, error) {
	if path := strings.TrimSpace(graderTemplatesCreateFlags.file); path != "" {
		raw, err := readInputFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading template file: %w", err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		if _, ok := doc["workflowGraph"]; !ok {
			return nil, fmt.Errorf("template file must include workflowGraph")
		}
		if name := strings.TrimSpace(graderTemplatesCreateFlags.name); name != "" {
			doc["name"] = name
			raw, _ = json.Marshal(doc)
		}
		if strings.TrimSpace(stringField(doc, "name")) == "" {
			return nil, fmt.Errorf("template name is required (--name or name in --file)")
		}
		return raw, nil
	}
	name := strings.TrimSpace(graderTemplatesCreateFlags.name)
	if name == "" {
		return nil, fmt.Errorf("specify --name and --file with workflowGraph")
	}
	return nil, fmt.Errorf("specify --file with workflowGraph JSON")
}

func runGraderTemplatesCreate(cmd *cobra.Command, args []string) error {
	payload, err := graderTemplateCreatePayload()
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := graderAgentCoursePath(args[0], "/grader-agent-templates")
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating grader template: %w", err)
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
	var out struct {
		Template map[string]any `json:"template"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created grader template %s (%s).\n",
		stringField(out.Template, "id"), stringField(out.Template, "name"))
	return nil
}