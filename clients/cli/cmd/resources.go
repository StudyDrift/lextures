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

var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Manage library and textbook resource links",
}

var resourcesListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List library and textbook resources in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runResourcesList,
}

var resourcesLinkFlags struct {
	module       string
	title        string
	kind         string
	resourceType string
	sourceURL    string
	provider     string
	toolID       string
}

var resourcesLinkCmd = &cobra.Command{
	Use:   "link <course_code>",
	Short: "Add a library or textbook resource link to a module",
	Args:  cobra.ExactArgs(1),
	RunE:  runResourcesLink,
}

func init() {
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.module, "module", "", "module id (required)")
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.title, "title", "", "resource title (required)")
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.kind, "kind", "library",
		"resource kind: library or textbook")
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.resourceType, "resource-type", "",
		"library resource type (e.g. article, ebook)")
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.sourceURL, "source-url", "",
		"source URL for library resources")
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.provider, "provider", "vitalsource",
		"textbook provider: vitalsource or redshelf")
	resourcesLinkCmd.Flags().StringVar(&resourcesLinkFlags.toolID, "tool-id", "",
		"optional external tool id")
	_ = resourcesLinkCmd.MarkFlagRequired("module")
	_ = resourcesLinkCmd.MarkFlagRequired("title")

	resourcesCmd.AddCommand(resourcesListCmd, resourcesLinkCmd)
	rootCmd.AddCommand(resourcesCmd)
}

func filterLibraryResources(items []structureItemPublic) []structureItemPublic {
	var out []structureItemPublic
	for _, it := range items {
		if it.Kind == "library_resource" {
			out = append(out, it)
		}
	}
	return out
}

func filterTextbookResources(items []structureItemPublic) []structureItemPublic {
	var out []structureItemPublic
	for _, it := range items {
		if it.Kind == "textbook_resource" {
			out = append(out, it)
		}
	}
	return out
}

func runResourcesList(cmd *cobra.Command, args []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	resources := append(filterLibraryResources(body.Items), filterTextbookResources(body.Items)...)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(resources)
	}
	if len(resources) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No library or textbook resources.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tKIND\tTITLE\tPUBLISHED")
	for _, r := range resources {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", r.ID, r.Kind, r.Title, r.Published)
	}
	return w.Flush()
}

func runResourcesLink(cmd *cobra.Command, args []string) error {
	kind := strings.ToLower(strings.TrimSpace(resourcesLinkFlags.kind))
	var segment string
	switch kind {
	case "library", "library_resource":
		segment = "library-resources"
	case "textbook", "textbook_resource":
		segment = "textbook-resources"
	default:
		return fmt.Errorf("invalid --kind %q: must be library or textbook", resourcesLinkFlags.kind)
	}

	payload := map[string]any{"title": resourcesLinkFlags.title}
	if resourcesLinkFlags.toolID != "" {
		payload["externalToolId"] = resourcesLinkFlags.toolID
	}
	if segment == "library-resources" {
		if resourcesLinkFlags.resourceType != "" {
			payload["resourceType"] = resourcesLinkFlags.resourceType
		}
		if resourcesLinkFlags.sourceURL != "" {
			payload["sourceUrl"] = resourcesLinkFlags.sourceURL
		}
		payload["metadata"] = map[string]string{}
	} else {
		payload["provider"] = resourcesLinkFlags.provider
		payload["metadata"] = map[string]string{}
	}

	raw, _ := json.Marshal(payload)
	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/%s",
		args[0], resourcesLinkFlags.module, segment)
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return fmt.Errorf("linking resource: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return apiErrorBody(resp.StatusCode, body)
	}
	var item structureItemPublic
	if err := json.Unmarshal(body, &item); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Linked %s resource %s (%s)\n", kind, item.Title, item.ID)
	return nil
}