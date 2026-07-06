package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

type externalToolSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type externalToolPublic struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	ClientID        string  `json:"clientId"`
	ToolIssuer      string  `json:"toolIssuer"`
	ToolJWKSURL     string  `json:"toolJwksUrl"`
	ToolOidcAuthURL string  `json:"toolOidcAuthUrl"`
	ToolTokenURL    *string `json:"toolTokenUrl"`
	Active          bool    `json:"active"`
	// Never echoed in CLI output; present only for redaction tests.
	ClientSecret string `json:"clientSecret,omitempty"`
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage LTI external tools and course tool links",
}

var toolsListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List active LTI external tools available in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsList,
}

type toolsAddFlagSet struct {
	register     bool
	name         string
	clientID     string
	issuer       string
	jwksURL      string
	oidcAuthURL  string
	tokenURL     string
	module       string
	toolID       string
	title        string
	resourceLink string
}

var toolsAddFlags toolsAddFlagSet

var toolsAddCmd = &cobra.Command{
	Use:   "add <course_code>",
	Short: "Register an LTI tool (admin) or add an LTI link to a module",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsAdd,
}

var toolsRemoveFlags struct {
	deactivate bool
}

var toolsRemoveCmd = &cobra.Command{
	Use:   "remove <tool_id>",
	Short: "Delete or deactivate an LTI external tool (admin)",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsRemove,
}

var toolsLinksCmd = &cobra.Command{
	Use:   "links",
	Short: "List external links and LTI links in a course",
}

var toolsLinksListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List external and LTI links in course modules",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolsLinksList,
}

func init() {
	toolsAddCmd.Flags().BoolVar(&toolsAddFlags.register, "register", false,
		"register a new platform LTI tool (admin) instead of adding a module link")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.name, "name", "", "tool name (required with --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.clientID, "client-id", "", "LTI client id (required with --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.issuer, "issuer", "", "tool issuer URL (required with --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.jwksURL, "jwks-url", "", "tool JWKS URL (required with --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.oidcAuthURL, "oidc-auth-url", "", "tool OIDC auth URL (required with --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.tokenURL, "token-url", "", "tool token URL (optional)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.module, "module", "", "module id (required without --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.toolID, "tool-id", "", "external tool id (required without --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.title, "title", "", "link title (required without --register)")
	toolsAddCmd.Flags().StringVar(&toolsAddFlags.resourceLink, "resource-link-id", "", "LTI resource link id (optional)")

	toolsRemoveCmd.Flags().BoolVar(&toolsRemoveFlags.deactivate, "deactivate", false,
		"deactivate the tool instead of deleting it")

	toolsLinksCmd.AddCommand(toolsLinksListCmd)
	toolsCmd.AddCommand(toolsListCmd, toolsAddCmd, toolsRemoveCmd, toolsLinksCmd)
	rootCmd.AddCommand(toolsCmd)
}

func redactExternalTool(tool externalToolPublic) map[string]any {
	out := map[string]any{
		"id":              tool.ID,
		"name":            tool.Name,
		"clientId":        tool.ClientID,
		"toolIssuer":      tool.ToolIssuer,
		"toolJwksUrl":     tool.ToolJWKSURL,
		"toolOidcAuthUrl": tool.ToolOidcAuthURL,
		"active":          tool.Active,
	}
	if tool.ToolTokenURL != nil {
		out["toolTokenUrl"] = *tool.ToolTokenURL
	}
	return out
}

func fetchCourseExternalTools(c *client.Client, courseCode string) ([]externalToolSummary, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/lti-external-tools", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing tools: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var tools []externalToolSummary
	if err := json.Unmarshal(body, &tools); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return tools, body, nil
}

func registerExternalTool(c *client.Client, flags toolsAddFlagSet) (externalToolPublic, []byte, error) {
	payload := map[string]any{
		"name":            flags.name,
		"clientId":        flags.clientID,
		"toolIssuer":      flags.issuer,
		"toolJwksUrl":     flags.jwksURL,
		"toolOidcAuthUrl": flags.oidcAuthURL,
	}
	if strings.TrimSpace(flags.tokenURL) != "" {
		payload["toolTokenUrl"] = flags.tokenURL
	}
	raw, _ := json.Marshal(payload)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/lti/external-tools", bytes.NewReader(raw))
	if err != nil {
		return externalToolPublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return externalToolPublic{}, nil, fmt.Errorf("registering tool: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return externalToolPublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return externalToolPublic{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var tool externalToolPublic
	if err := json.Unmarshal(body, &tool); err != nil {
		return externalToolPublic{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return tool, body, nil
}

func addLTILinkToModule(c *client.Client, courseCode string, flags toolsAddFlagSet) (structureItemPublic, []byte, error) {
	payload := map[string]string{
		"title":          flags.title,
		"externalToolId": flags.toolID,
	}
	if strings.TrimSpace(flags.resourceLink) != "" {
		payload["resourceLinkId"] = flags.resourceLink
	}
	raw, _ := json.Marshal(payload)
	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/lti-links", courseCode, flags.module)
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return structureItemPublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return structureItemPublic{}, nil, fmt.Errorf("adding LTI link: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return structureItemPublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return structureItemPublic{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var item structureItemPublic
	if err := json.Unmarshal(body, &item); err != nil {
		return structureItemPublic{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return item, body, nil
}

func runToolsList(cmd *cobra.Command, args []string) error {
	tools, raw, err := fetchCourseExternalTools(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(tools) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No external tools.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME")
	for _, t := range tools {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Name)
	}
	return w.Flush()
}

func runToolsAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if toolsAddFlags.register {
		if toolsAddFlags.name == "" || toolsAddFlags.clientID == "" || toolsAddFlags.issuer == "" ||
			toolsAddFlags.jwksURL == "" || toolsAddFlags.oidcAuthURL == "" {
			return fmt.Errorf("--register requires --name, --client-id, --issuer, --jwks-url, and --oidc-auth-url")
		}
		tool, _, err := registerExternalTool(c, toolsAddFlags)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(redactExternalTool(tool))
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Registered tool %s (%s)\n", tool.Name, tool.ID)
		return nil
	}
	if toolsAddFlags.module == "" || toolsAddFlags.toolID == "" || toolsAddFlags.title == "" {
		return fmt.Errorf("adding a module link requires --module, --tool-id, and --title (or use --register)")
	}
	item, raw, err := addLTILinkToModule(c, args[0], toolsAddFlags)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added LTI link %s (%s)\n", item.Title, item.ID)
	return nil
}

func runToolsRemove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	toolID := args[0]
	if toolsRemoveFlags.deactivate {
		payload, _ := json.Marshal(map[string]bool{"active": false})
		req, err := c.NewRequest(http.MethodPut, "/api/v1/admin/lti/external-tools/"+toolID, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		resp, err := doWithRetry(c, req)
		if err != nil {
			return fmt.Errorf("deactivating tool: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusNoContent {
			return apiError(resp, 2)
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"id": toolID, "active": false})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deactivated tool %s\n", toolID)
		return nil
	}
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/lti/external-tools/"+toolID, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("removing tool: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": toolID})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed tool %s\n", toolID)
	return nil
}

func filterLTILinks(items []structureItemPublic) []structureItemPublic {
	var out []structureItemPublic
	for _, it := range items {
		if it.Kind == "lti_link" {
			out = append(out, it)
		}
	}
	return out
}

func runToolsLinksList(cmd *cobra.Command, args []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	links := append(filterExternalLinks(body.Items), filterLTILinks(body.Items)...)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(links)
	}
	if len(links) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No external or LTI links.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tKIND\tTITLE\tPUBLISHED")
	for _, l := range links {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", l.ID, l.Kind, l.Title, l.Published)
	}
	return w.Flush()
}