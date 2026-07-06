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

// contentPagePublic mirrors GET/PATCH /content-pages/{id} response.
type contentPagePublic struct {
	ItemID    string `json:"itemId"`
	Title     string `json:"title"`
	Markdown  string `json:"markdown"`
	UpdatedAt string `json:"updatedAt"`
}

var pagesCmd = &cobra.Command{
	Use:   "pages",
	Short: "Manage course content pages",
}

var pagesListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List content pages in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runPagesList,
}

var pagesGetCmd = &cobra.Command{
	Use:   "get <course_code> <item_id>",
	Short: "Get a content page",
	Args:  cobra.ExactArgs(2),
	RunE:  runPagesGet,
}

var pagesCreateFlags struct {
	module  string
	title   string
	file    string
	publish bool
}

var pagesCreateCmd = &cobra.Command{
	Use:   "create <course_code>",
	Short: "Create a content page from a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runPagesCreate,
}

var pagesUpdateFlags struct {
	file string
}

var pagesUpdateCmd = &cobra.Command{
	Use:   "update <course_code> <item_id>",
	Short: "Update a content page from a file",
	Args:  cobra.ExactArgs(2),
	RunE:  runPagesUpdate,
}

var pagesPublishCmd = &cobra.Command{
	Use:   "publish <course_code> <item_id>",
	Short: "Publish a content page",
	Args:  cobra.ExactArgs(2),
	RunE:  runPagesPublish,
}

func init() {
	pagesCreateCmd.Flags().StringVar(&pagesCreateFlags.module, "module", "", "module id (required)")
	pagesCreateCmd.Flags().StringVar(&pagesCreateFlags.title, "title", "", "page title (required)")
	pagesCreateCmd.Flags().StringVar(&pagesCreateFlags.file, "file", "", "Markdown or HTML file (use - for stdin)")
	pagesCreateCmd.Flags().BoolVar(&pagesCreateFlags.publish, "publish", false, "publish the page after creation")
	_ = pagesCreateCmd.MarkFlagRequired("module")
	_ = pagesCreateCmd.MarkFlagRequired("title")
	_ = pagesCreateCmd.MarkFlagRequired("file")

	pagesUpdateCmd.Flags().StringVar(&pagesUpdateFlags.file, "file", "", "Markdown or HTML file (use - for stdin)")
	_ = pagesUpdateCmd.MarkFlagRequired("file")

	pagesCmd.AddCommand(pagesListCmd, pagesGetCmd, pagesCreateCmd, pagesUpdateCmd, pagesPublishCmd)
	rootCmd.AddCommand(pagesCmd)
}

func filterContentPages(items []structureItemPublic) []structureItemPublic {
	var pages []structureItemPublic
	for _, it := range items {
		if it.Kind == "content_page" {
			pages = append(pages, it)
		}
	}
	return pages
}

func patchContentPage(c *client.Client, courseCode, itemID, markdown string) (contentPagePublic, []byte, error) {
	body, _ := json.Marshal(map[string]string{"markdown": markdown})
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/content-pages/"+itemID, bytes.NewReader(body))
	if err != nil {
		return contentPagePublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return contentPagePublic{}, nil, fmt.Errorf("updating content page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return contentPagePublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return contentPagePublic{}, respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	var page contentPagePublic
	if err := json.Unmarshal(respBody, &page); err != nil {
		return contentPagePublic{}, respBody, fmt.Errorf("decoding response: %w", err)
	}
	return page, respBody, nil
}

func getContentPage(c *client.Client, courseCode, itemID string) (contentPagePublic, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/content-pages/"+itemID, nil)
	if err != nil {
		return contentPagePublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return contentPagePublic{}, nil, fmt.Errorf("getting content page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return contentPagePublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return contentPagePublic{}, respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	var page contentPagePublic
	if err := json.Unmarshal(respBody, &page); err != nil {
		return contentPagePublic{}, respBody, fmt.Errorf("decoding response: %w", err)
	}
	return page, respBody, nil
}

func publishContentPage(c *client.Client, courseCode, itemID string) error {
	pub := true
	return patchStructureItem(c, courseCode, itemID, structureItemPatchOpts{published: &pub})
}

func runPagesList(cmd *cobra.Command, args []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	pages := filterContentPages(body.Items)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(pages)
	}
	if len(pages) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No content pages.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tPUBLISHED")
	for _, p := range pages {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\n", p.ID, p.Title, p.Published)
	}
	return w.Flush()
}

func runPagesGet(cmd *cobra.Command, args []string) error {
	page, raw, err := getContentPage(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:       %s\n", page.ItemID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Title:    %s\n", page.Title)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated:  %s\n", page.UpdatedAt)
	if page.Markdown != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "---")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), page.Markdown)
	}
	return nil
}

func runPagesCreate(cmd *cobra.Command, args []string) error {
	content, err := readInputFile(pagesCreateFlags.file)
	if err != nil {
		return fmt.Errorf("reading page file: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := args[0]

	item, err := addItemToModule(c, courseCode, pagesCreateFlags.module, "page", pagesCreateFlags.title, "")
	if err != nil {
		return err
	}

	if len(strings.TrimSpace(string(content))) > 0 {
		if _, _, err := patchContentPage(c, courseCode, item.ID, string(content)); err != nil {
			return err
		}
	}

	if pagesCreateFlags.publish {
		if err := publishContentPage(c, courseCode, item.ID); err != nil {
			return err
		}
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"id":        item.ID,
			"title":     item.Title,
			"published": pagesCreateFlags.publish,
		})
	}
	pub := ""
	if pagesCreateFlags.publish {
		pub = " (published)"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created page %s (%s)%s\n", item.Title, item.ID, pub)
	return nil
}

func runPagesUpdate(cmd *cobra.Command, args []string) error {
	content, err := readInputFile(pagesUpdateFlags.file)
	if err != nil {
		return fmt.Errorf("reading page file: %w", err)
	}
	page, raw, err := patchContentPage(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1], string(content))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated page %s\n", page.ItemID)
	return nil
}

func runPagesPublish(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := publishContentPage(c, args[0], args[1]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"id":        args[1],
			"published": true,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Published page %s\n", args[1])
	return nil
}