package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

type whiteboardPublic struct {
	ID         string          `json:"id"`
	CourseID   string          `json:"courseId"`
	Title      string          `json:"title"`
	CanvasData json.RawMessage `json:"canvasData"`
	CreatedBy  *string         `json:"createdBy,omitempty"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
}

type whiteboardsListBody struct {
	Whiteboards []whiteboardPublic `json:"whiteboards"`
}

var whiteboardsCmd = &cobra.Command{
	Use:   "whiteboards",
	Short: "List and export course whiteboards (read-only)",
}

var whiteboardsListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List whiteboards in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runWhiteboardsList,
}

var whiteboardsExportFlags struct {
	yes  bool
	file string
}

var whiteboardsExportCmd = &cobra.Command{
	Use:   "export <course_code>",
	Short: "Export all whiteboards for backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runWhiteboardsExport,
}

var whiteboardsExportInput io.Reader

func init() {
	whiteboardsExportCmd.Flags().BoolVar(&whiteboardsExportFlags.yes, "yes", false,
		"confirm export of whiteboards that may contain student content (FERPA)")
	whiteboardsExportCmd.Flags().StringVar(&whiteboardsExportFlags.file, "file", "",
		"write export JSON to file (default: stdout)")

	whiteboardsCmd.AddCommand(whiteboardsListCmd, whiteboardsExportCmd)
	rootCmd.AddCommand(whiteboardsCmd)
}

func fetchWhiteboards(c *client.Client, courseCode string) (whiteboardsListBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/whiteboards", nil)
	if err != nil {
		return whiteboardsListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return whiteboardsListBody{}, nil, fmt.Errorf("listing whiteboards: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return whiteboardsListBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return whiteboardsListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out whiteboardsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return whiteboardsListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func confirmWhiteboardExport(cmd *cobra.Command) error {
	if whiteboardsExportFlags.yes {
		return nil
	}
	in := whiteboardsExportInput
	if in == nil {
		in = os.Stdin
	}
	r := bufio.NewReader(in)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"Whiteboards may contain student content protected by FERPA. Export anyway? (y/N) ")
	line, _ := r.ReadString('\n')
	if !strings.EqualFold(strings.TrimSpace(line), "y") {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return fmt.Errorf("export aborted")
	}
	return nil
}

func runWhiteboardsList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchWhiteboards(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Whiteboards) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No whiteboards.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tUPDATED")
	for _, b := range body.Whiteboards {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", b.ID, b.Title, b.UpdatedAt)
	}
	return w.Flush()
}

func runWhiteboardsExport(cmd *cobra.Command, args []string) error {
	if err := confirmWhiteboardExport(cmd); err != nil {
		return err
	}
	body, _, err := fetchWhiteboards(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	export := map[string]any{
		"courseCode":  args[0],
		"whiteboards": body.Whiteboards,
	}
	out, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding export: %w", err)
	}
	if whiteboardsExportFlags.file != "" {
		if err := os.WriteFile(whiteboardsExportFlags.file, out, 0o600); err != nil {
			return fmt.Errorf("writing export file: %w", err)
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"path":        whiteboardsExportFlags.file,
				"whiteboards": len(body.Whiteboards),
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d whiteboards to %s\n",
			len(body.Whiteboards), whiteboardsExportFlags.file)
		return nil
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(out)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d whiteboards\n", len(body.Whiteboards))
	_, err = cmd.OutOrStdout().Write(out)
	return err
}