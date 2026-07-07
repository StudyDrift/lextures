package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var xapiCmd = &cobra.Command{
	Use:   "xapi",
	Short: "Post and query xAPI statements",
}

var xapiPostFlags struct {
	file       string
	course     string
	packageID  string
}

var xapiPostCmd = &cobra.Command{
	Use:   "post",
	Short: "Post one or more xAPI statements (JSON or NDJSON)",
	RunE:  runXAPIPost,
}

var xapiQueryFlags struct {
	course   string
	verb     string
	activity string
	actor    string
	since    string
	yes      bool
}

var xapiQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query stored xAPI events for a course",
	RunE:  runXAPIQuery,
}

func init() {
	xapiPostCmd.Flags().StringVar(&xapiPostFlags.file, "file", "", "statement JSON or NDJSON file (required)")
	_ = xapiPostCmd.MarkFlagRequired("file")
	xapiPostCmd.Flags().StringVar(&xapiPostFlags.course, "course", "", "course code (required for post)")
	_ = xapiPostCmd.MarkFlagRequired("course")
	xapiPostCmd.Flags().StringVar(&xapiPostFlags.packageID, "package-id", "", "H5P package UUID (required for post)")
	_ = xapiPostCmd.MarkFlagRequired("package-id")

	xapiQueryCmd.Flags().StringVar(&xapiQueryFlags.course, "course", "", "course code (required)")
	_ = xapiQueryCmd.MarkFlagRequired("course")
	xapiQueryCmd.Flags().StringVar(&xapiQueryFlags.verb, "verb", "", "filter by verb id substring")
	xapiQueryCmd.Flags().StringVar(&xapiQueryFlags.activity, "activity", "", "filter by activity id/title substring")
	xapiQueryCmd.Flags().StringVar(&xapiQueryFlags.actor, "actor", "", "filter by actor substring in statement JSON")
	xapiQueryCmd.Flags().StringVar(&xapiQueryFlags.since, "since", "", "RFC3339 lower bound (default: server 7d window)")
	xapiQueryCmd.Flags().BoolVar(&xapiQueryFlags.yes, "yes", false, "confirm FERPA-covered export")

	xapiCmd.AddCommand(xapiPostCmd, xapiQueryCmd)
	rootCmd.AddCommand(xapiCmd)
}

func runXAPIPost(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	statements, err := readJSONObjectsFromFile(xapiPostFlags.file)
	if err != nil {
		return err
	}
	posted := 0
	var lastID string
	for _, stmt := range statements {
		if err := postXAPIStatement(c, xapiPostFlags.course, xapiPostFlags.packageID, stmt); err != nil {
			return err
		}
		posted++
		if id := statementIDFromJSON(stmt); id != "" {
			lastID = id
		}
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"posted":      posted,
			"statementId": lastID,
		})
	}
	if lastID != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Posted %d statement(s); last id=%s\n", posted, lastID)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Posted %d statement(s).\n", posted)
	}
	return nil
}

func runXAPIQuery(cmd *cobra.Command, args []string) error {
	if !xapiQueryFlags.yes {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), ferpaXAPIQueryWarning)
		return fmt.Errorf("re-run with --yes to export query results")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	events, raw, err := queryCourseXAPIEvents(c, xapiQueryFlags.course, xapiQueryFlags.since)
	if err != nil {
		return err
	}
	events = filterXAPIEvents(events, xapiQueryFlags.verb, xapiQueryFlags.activity, xapiQueryFlags.actor)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"events": events})
	}
	if len(events) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No matching statements.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STATEMENT\tVERB\tOBJECT\tSTORED")
	for _, e := range events {
		title := e.ObjectID
		if e.ObjectTitle != nil && *e.ObjectTitle != "" {
			title = *e.ObjectTitle
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.StatementID, e.Verb, title, e.StoredAt)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_ = raw
	return nil
}