package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

type h5pPackagePublic struct {
	PackageID     string `json:"packageId"`
	ItemID        string `json:"itemId,omitempty"`
	Title         string `json:"title"`
	ContentType   string `json:"contentType"`
	ExtractStatus string `json:"extractStatus"`
}

var h5pCmd = &cobra.Command{
	Use:   "h5p",
	Short: "Manage H5P packages in courses",
}

var h5pListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List H5P items in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runH5PList,
}

var h5pGetCmd = &cobra.Command{
	Use:   "get <course_code> <item_id>",
	Short: "Get H5P package details for a module item",
	Args:  cobra.ExactArgs(2),
	RunE:  runH5PGet,
}

var h5pImportFlags struct {
	module string
	title  string
	quiet  bool
}

var h5pImportCmd = &cobra.Command{
	Use:   "import <course_code> <package>",
	Short: "Import an H5P package into a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runH5PImport,
}

var h5pDeleteCmd = &cobra.Command{
	Use:   "delete <course_code> <item_id>",
	Short: "Remove an H5P item from the course structure",
	Args:  cobra.ExactArgs(2),
	RunE:  runH5PDelete,
}

var h5pProgressOut io.Writer

func init() {
	h5pImportCmd.Flags().StringVar(&h5pImportFlags.module, "module", "", "module id (required)")
	h5pImportCmd.Flags().StringVar(&h5pImportFlags.title, "title", "", "activity title (defaults to manifest title)")
	h5pImportCmd.Flags().BoolVar(&h5pImportFlags.quiet, "quiet", false, "suppress progress output")
	_ = h5pImportCmd.MarkFlagRequired("module")

	h5pCmd.AddCommand(h5pListCmd, h5pGetCmd, h5pImportCmd, h5pDeleteCmd)
	rootCmd.AddCommand(h5pCmd)
}

func filterH5PItems(items []structureItemPublic) []structureItemPublic {
	var out []structureItemPublic
	for _, it := range items {
		if it.Kind == "h5p" {
			out = append(out, it)
		}
	}
	return out
}

func getH5PPackage(c *client.Client, courseCode, itemID string) (h5pPackagePublic, []byte, error) {
	req, err := c.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/h5p-items/"+itemID, nil)
	if err != nil {
		return h5pPackagePublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return h5pPackagePublic{}, nil, fmt.Errorf("getting H5P package: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return h5pPackagePublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return h5pPackagePublic{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var pkg h5pPackagePublic
	if err := json.Unmarshal(body, &pkg); err != nil {
		return h5pPackagePublic{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return pkg, body, nil
}

func runH5PList(cmd *cobra.Command, args []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	items := filterH5PItems(body.Items)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No H5P items.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tPUBLISHED")
	for _, it := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\n", it.ID, it.Title, it.Published)
	}
	return w.Flush()
}

func runH5PGet(cmd *cobra.Command, args []string) error {
	pkg, raw, err := getH5PPackage(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Item:     %s\n", pkg.ItemID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Package:  %s\n", pkg.PackageID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Title:    %s\n", pkg.Title)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type:     %s\n", pkg.ContentType)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status:   %s\n", pkg.ExtractStatus)
	return nil
}

func runH5PImport(cmd *cobra.Command, args []string) error {
	progress := h5pProgressOut
	if progress == nil {
		progress = cmd.OutOrStdout()
	}
	item, raw, err := uploadModulePackage(client.New(Cfg.Server, Cfg.APIKey), modulePackageUploadOpts{
		courseCode: args[0],
		moduleID:   h5pImportFlags.module,
		segment:    "h5p",
		localPath:  args[1],
		title:      h5pImportFlags.title,
		quiet:      h5pImportFlags.quiet,
		progress:   progress,
	})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Imported H5P %s (%s)\n", item.Title, item.ID)
	return nil
}

func runH5PDelete(cmd *cobra.Command, args []string) error {
	if err := deleteStructureItem(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": args[1]})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted H5P item %s\n", args[1])
	return nil
}