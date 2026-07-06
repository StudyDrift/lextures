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

type collabDocPublic struct {
	ID        string  `json:"id"`
	CourseID  string  `json:"courseId"`
	GroupID   *string `json:"groupId"`
	Title     string  `json:"title"`
	DocType   string  `json:"docType"`
	CreatedBy string  `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type collabDocsListBody struct {
	Docs []collabDocPublic `json:"docs"`
}

var collabDocsCmd = &cobra.Command{
	Use:   "collab-docs",
	Short: "List and export collaborative documents (read-only)",
}

var collabDocsListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List collaborative documents in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runCollabDocsList,
}

var collabDocsExportFlags struct {
	yes  bool
	file string
}

var collabDocsExportCmd = &cobra.Command{
	Use:   "export <course_code>",
	Short: "Export all collaborative documents for backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runCollabDocsExport,
}

var collabDocsExportInput io.Reader

func init() {
	collabDocsExportCmd.Flags().BoolVar(&collabDocsExportFlags.yes, "yes", false,
		"confirm export of documents that may contain student content (FERPA)")
	collabDocsExportCmd.Flags().StringVar(&collabDocsExportFlags.file, "file", "",
		"write export JSON to file (default: stdout)")

	collabDocsCmd.AddCommand(collabDocsListCmd, collabDocsExportCmd)
	rootCmd.AddCommand(collabDocsCmd)
}

func fetchCollabDocs(c *client.Client, courseCode string) (collabDocsListBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/collab-docs", nil)
	if err != nil {
		return collabDocsListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return collabDocsListBody{}, nil, fmt.Errorf("listing collab docs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return collabDocsListBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return collabDocsListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out collabDocsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return collabDocsListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func fetchCollabDocSnapshots(c *client.Client, courseCode, docID string) (map[string]any, error) {
	req, err := c.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/collab-docs/"+docID+"/snapshots", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing snapshots: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func confirmBulkExport(cmd *cobra.Command, label string) error {
	if collabDocsExportFlags.yes {
		return nil
	}
	in := collabDocsExportInput
	if in == nil {
		in = os.Stdin
	}
	r := bufio.NewReader(in)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"%s may contain student content protected by FERPA. Export anyway? (y/N) ", label)
	line, _ := r.ReadString('\n')
	if !strings.EqualFold(strings.TrimSpace(line), "y") {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return fmt.Errorf("export aborted")
	}
	return nil
}

func runCollabDocsList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchCollabDocs(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Docs) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No collaborative documents.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tTITLE\tUPDATED")
	for _, d := range body.Docs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.DocType, d.Title, d.UpdatedAt)
	}
	return w.Flush()
}

func runCollabDocsExport(cmd *cobra.Command, args []string) error {
	if err := confirmBulkExport(cmd, "Collaborative documents"); err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := args[0]
	list, _, err := fetchCollabDocs(c, courseCode)
	if err != nil {
		return err
	}
	export := map[string]any{
		"courseCode": courseCode,
		"docs":       list.Docs,
		"snapshots":  map[string]any{},
	}
	snapshots := export["snapshots"].(map[string]any)
	for _, doc := range list.Docs {
		snaps, err := fetchCollabDocSnapshots(c, courseCode, doc.ID)
		if err != nil {
			return fmt.Errorf("exporting doc %s: %w", doc.ID, err)
		}
		snapshots[doc.ID] = snaps
	}
	raw, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding export: %w", err)
	}
	if collabDocsExportFlags.file != "" {
		if err := os.WriteFile(collabDocsExportFlags.file, raw, 0o600); err != nil {
			return fmt.Errorf("writing export file: %w", err)
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"path": collabDocsExportFlags.file,
				"docs": len(list.Docs),
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d documents to %s\n", len(list.Docs), collabDocsExportFlags.file)
		return nil
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d documents\n", len(list.Docs))
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}