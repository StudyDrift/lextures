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

type scormPackagePublic struct {
	PackageID     string `json:"packageId"`
	ItemID        string `json:"itemId,omitempty"`
	Title         string `json:"title"`
	PackageType   string `json:"packageType"`
	ExtractStatus string `json:"extractStatus"`
}

var scormCmd = &cobra.Command{
	Use:   "scorm",
	Short: "Manage SCORM packages in courses",
}

var scormListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List SCORM items in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runScormList,
}

var scormGetCmd = &cobra.Command{
	Use:   "get <course_code> <item_id>",
	Short: "Get SCORM package details for a module item",
	Args:  cobra.ExactArgs(2),
	RunE:  runScormGet,
}

var scormImportFlags struct {
	module string
	title  string
	quiet  bool
}

var scormImportCmd = &cobra.Command{
	Use:   "import <course_code> <zip>",
	Short: "Import a SCORM zip into a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runScormImport,
}

var scormDeleteCmd = &cobra.Command{
	Use:   "delete <course_code> <item_id>",
	Short: "Remove a SCORM item from the course structure",
	Args:  cobra.ExactArgs(2),
	RunE:  runScormDelete,
}

var scormProgressOut io.Writer

func init() {
	scormImportCmd.Flags().StringVar(&scormImportFlags.module, "module", "", "module id (required)")
	scormImportCmd.Flags().StringVar(&scormImportFlags.title, "title", "", "activity title (defaults to manifest title)")
	scormImportCmd.Flags().BoolVar(&scormImportFlags.quiet, "quiet", false, "suppress progress output")
	_ = scormImportCmd.MarkFlagRequired("module")

	scormCmd.AddCommand(scormListCmd, scormGetCmd, scormImportCmd, scormDeleteCmd)
	rootCmd.AddCommand(scormCmd)
}

func filterScormItems(items []structureItemPublic) []structureItemPublic {
	var out []structureItemPublic
	for _, it := range items {
		if it.Kind == "scorm" {
			out = append(out, it)
		}
	}
	return out
}

func getScormPackage(c *client.Client, courseCode, itemID string) (scormPackagePublic, []byte, error) {
	req, err := c.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/scorm-items/"+itemID, nil)
	if err != nil {
		return scormPackagePublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return scormPackagePublic{}, nil, fmt.Errorf("getting SCORM package: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return scormPackagePublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return scormPackagePublic{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var pkg scormPackagePublic
	if err := json.Unmarshal(body, &pkg); err != nil {
		return scormPackagePublic{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return pkg, body, nil
}

func runScormList(cmd *cobra.Command, args []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	items := filterScormItems(body.Items)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No SCORM items.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tPUBLISHED")
	for _, it := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\n", it.ID, it.Title, it.Published)
	}
	return w.Flush()
}

func runScormGet(cmd *cobra.Command, args []string) error {
	pkg, raw, err := getScormPackage(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1])
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
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type:     %s\n", pkg.PackageType)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status:   %s\n", pkg.ExtractStatus)
	return nil
}

func runScormImport(cmd *cobra.Command, args []string) error {
	progress := scormProgressOut
	if progress == nil {
		progress = cmd.OutOrStdout()
	}
	item, raw, err := uploadModulePackage(client.New(Cfg.Server, Cfg.APIKey), modulePackageUploadOpts{
		courseCode: args[0],
		moduleID:   scormImportFlags.module,
		segment:    "scorm",
		localPath:  args[1],
		title:      scormImportFlags.title,
		quiet:      scormImportFlags.quiet,
		progress:   progress,
	})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Imported SCORM %s (%s)\n", item.Title, item.ID)
	return nil
}

func runScormDelete(cmd *cobra.Command, args []string) error {
	if err := deleteStructureItem(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": args[1]})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted SCORM item %s\n", args[1])
	return nil
}