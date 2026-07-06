package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

type glossaryEntry struct {
	ID           string `json:"id"`
	SourceTerm   string `json:"sourceTerm"`
	TargetTerm   string `json:"targetTerm"`
	SourceLocale string `json:"sourceLocale"`
	TargetLocale string `json:"targetLocale"`
}

type glossaryListBody struct {
	Entries []glossaryEntry `json:"entries"`
}

// glossaryFileEntry is one row in a bulk glossary import file.
type glossaryFileEntry struct {
	SourceTerm   string `json:"sourceTerm"`
	TargetTerm   string `json:"targetTerm"`
	SourceLocale string `json:"sourceLocale"`
	TargetLocale string `json:"targetLocale"`
}

var glossaryCmd = &cobra.Command{
	Use:   "glossary",
	Short: "Manage course translation glossary terms",
}

var glossaryListFlags struct {
	sourceLocale string
	targetLocale string
}

var glossaryListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List glossary terms for a course locale pair",
	Args:  cobra.ExactArgs(1),
	RunE:  runGlossaryList,
}

var glossaryGetFlags struct {
	sourceLocale string
	targetLocale string
	term         string
}

var glossaryGetCmd = &cobra.Command{
	Use:   "get <course_code>",
	Short: "Get a glossary term by source text",
	Args:  cobra.ExactArgs(1),
	RunE:  runGlossaryGet,
}

var glossarySetFlags struct {
	file         string
	sourceLocale string
	targetLocale string
}

var glossarySetCmd = &cobra.Command{
	Use:   "set <course_code>",
	Short: "Bulk-load glossary terms from a CSV or JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runGlossarySet,
}

var glossaryAddFlags struct {
	sourceLocale string
	targetLocale string
	sourceTerm   string
	targetTerm   string
}

var glossaryAddCmd = &cobra.Command{
	Use:   "add <course_code>",
	Short: "Add a single glossary term",
	Args:  cobra.ExactArgs(1),
	RunE:  runGlossaryAdd,
}

func init() {
	glossaryListCmd.Flags().StringVar(&glossaryListFlags.sourceLocale, "source-locale", "en", "source locale")
	glossaryListCmd.Flags().StringVar(&glossaryListFlags.targetLocale, "target-locale", "", "target locale (required)")
	_ = glossaryListCmd.MarkFlagRequired("target-locale")

	glossaryGetCmd.Flags().StringVar(&glossaryGetFlags.sourceLocale, "source-locale", "en", "source locale")
	glossaryGetCmd.Flags().StringVar(&glossaryGetFlags.targetLocale, "target-locale", "", "target locale (required)")
	glossaryGetCmd.Flags().StringVar(&glossaryGetFlags.term, "term", "", "source term to look up (required)")
	_ = glossaryGetCmd.MarkFlagRequired("target-locale")
	_ = glossaryGetCmd.MarkFlagRequired("term")

	glossarySetCmd.Flags().StringVar(&glossarySetFlags.file, "file", "", "CSV or JSON file (use - for stdin)")
	glossarySetCmd.Flags().StringVar(&glossarySetFlags.sourceLocale, "source-locale", "en", "default source locale for CSV rows")
	glossarySetCmd.Flags().StringVar(&glossarySetFlags.targetLocale, "target-locale", "", "default target locale for CSV rows (required)")
	_ = glossarySetCmd.MarkFlagRequired("file")
	_ = glossarySetCmd.MarkFlagRequired("target-locale")

	glossaryAddCmd.Flags().StringVar(&glossaryAddFlags.sourceLocale, "source-locale", "en", "source locale")
	glossaryAddCmd.Flags().StringVar(&glossaryAddFlags.targetLocale, "target-locale", "", "target locale (required)")
	glossaryAddCmd.Flags().StringVar(&glossaryAddFlags.sourceTerm, "source", "", "source term (required)")
	glossaryAddCmd.Flags().StringVar(&glossaryAddFlags.targetTerm, "target", "", "target term (required)")
	_ = glossaryAddCmd.MarkFlagRequired("target-locale")
	_ = glossaryAddCmd.MarkFlagRequired("source")
	_ = glossaryAddCmd.MarkFlagRequired("target")

	glossaryCmd.AddCommand(glossaryListCmd, glossaryGetCmd, glossarySetCmd, glossaryAddCmd)
	rootCmd.AddCommand(glossaryCmd)
}

func glossaryPath(courseCode, sourceLocale, targetLocale string) string {
	return fmt.Sprintf("/api/v1/courses/%s/glossary?source_locale=%s&target_locale=%s",
		courseCode, sourceLocale, targetLocale)
}

func fetchGlossary(c *client.Client, courseCode, sourceLocale, targetLocale string) (glossaryListBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, glossaryPath(courseCode, sourceLocale, targetLocale), nil)
	if err != nil {
		return glossaryListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return glossaryListBody{}, nil, fmt.Errorf("listing glossary: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return glossaryListBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return glossaryListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out glossaryListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return glossaryListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func postGlossaryEntry(c *client.Client, courseCode string, entry glossaryFileEntry) (glossaryEntry, error) {
	payload := map[string]string{
		"sourceLocale": entry.SourceLocale,
		"targetLocale": entry.TargetLocale,
		"sourceTerm":   entry.SourceTerm,
		"targetTerm":   entry.TargetTerm,
	}
	raw, _ := json.Marshal(payload)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/glossary", bytes.NewReader(raw))
	if err != nil {
		return glossaryEntry{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return glossaryEntry{}, fmt.Errorf("adding glossary term: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return glossaryEntry{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return glossaryEntry{}, apiErrorBody(resp.StatusCode, respBody)
	}
	var out glossaryEntry
	if err := json.Unmarshal(respBody, &out); err != nil {
		return glossaryEntry{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func parseGlossaryFile(data []byte, defaultSource, defaultTarget string) ([]glossaryFileEntry, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("glossary file is empty")
	}
	if trimmed[0] == '[' || trimmed[0] == '{' {
		return parseGlossaryJSON(data, defaultSource, defaultTarget)
	}
	return parseGlossaryCSV(data, defaultSource, defaultTarget)
}

func parseGlossaryJSON(data []byte, defaultSource, defaultTarget string) ([]glossaryFileEntry, error) {
	var entries []glossaryFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		var wrapped struct {
			Entries []glossaryFileEntry `json:"entries"`
		}
		if err2 := json.Unmarshal(data, &wrapped); err2 != nil {
			return nil, fmt.Errorf("parsing glossary JSON: %w", err)
		}
		entries = wrapped.Entries
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("glossary file contains no entries")
	}
	for i := range entries {
		if strings.TrimSpace(entries[i].SourceLocale) == "" {
			entries[i].SourceLocale = defaultSource
		}
		if strings.TrimSpace(entries[i].TargetLocale) == "" {
			entries[i].TargetLocale = defaultTarget
		}
		if strings.TrimSpace(entries[i].SourceTerm) == "" || strings.TrimSpace(entries[i].TargetTerm) == "" {
			return nil, fmt.Errorf("entry %d: sourceTerm and targetTerm are required", i+1)
		}
	}
	return entries, nil
}

func parseGlossaryCSV(data []byte, defaultSource, defaultTarget string) ([]glossaryFileEntry, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing glossary CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("glossary CSV is empty")
	}

	start := 0
	colSource, colTarget := 0, 1
	colSourceLocale, colTargetLocale := -1, -1
	if len(records[0]) >= 2 {
		h0 := strings.ToLower(strings.TrimSpace(records[0][0]))
		h1 := strings.ToLower(strings.TrimSpace(records[0][1]))
		if h0 == "sourceterm" || h0 == "source_term" || h0 == "source" {
			start = 1
			for i, h := range records[0] {
				switch strings.ToLower(strings.TrimSpace(h)) {
				case "sourceterm", "source_term", "source":
					colSource = i
				case "targetterm", "target_term", "target":
					colTarget = i
				case "sourcelocale", "source_locale":
					colSourceLocale = i
				case "targetlocale", "target_locale":
					colTargetLocale = i
				}
			}
		} else if h0 == "source" && h1 == "target" {
			start = 1
		}
	}

	var entries []glossaryFileEntry
	for i := start; i < len(records); i++ {
		row := records[i]
		if len(row) == 0 || strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		if colSource >= len(row) || colTarget >= len(row) {
			return nil, fmt.Errorf("row %d: expected at least source and target columns", i+1)
		}
		entry := glossaryFileEntry{
			SourceTerm:   strings.TrimSpace(row[colSource]),
			TargetTerm:   strings.TrimSpace(row[colTarget]),
			SourceLocale: defaultSource,
			TargetLocale: defaultTarget,
		}
		if colSourceLocale >= 0 && colSourceLocale < len(row) && strings.TrimSpace(row[colSourceLocale]) != "" {
			entry.SourceLocale = strings.TrimSpace(row[colSourceLocale])
		}
		if colTargetLocale >= 0 && colTargetLocale < len(row) && strings.TrimSpace(row[colTargetLocale]) != "" {
			entry.TargetLocale = strings.TrimSpace(row[colTargetLocale])
		}
		if entry.SourceTerm == "" || entry.TargetTerm == "" {
			return nil, fmt.Errorf("row %d: source and target terms are required", i+1)
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("glossary CSV contains no data rows")
	}
	return entries, nil
}

func runGlossaryList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchGlossary(client.New(Cfg.Server, Cfg.APIKey), args[0],
		glossaryListFlags.sourceLocale, glossaryListFlags.targetLocale)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Entries) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No glossary terms.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SOURCE\tTARGET")
	for _, e := range body.Entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", e.SourceTerm, e.TargetTerm)
	}
	return w.Flush()
}

func runGlossaryGet(cmd *cobra.Command, args []string) error {
	body, _, err := fetchGlossary(client.New(Cfg.Server, Cfg.APIKey), args[0],
		glossaryGetFlags.sourceLocale, glossaryGetFlags.targetLocale)
	if err != nil {
		return err
	}
	term := strings.TrimSpace(glossaryGetFlags.term)
	for _, e := range body.Entries {
		if e.SourceTerm == term {
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(e)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Source: %s\nTarget: %s\n", e.SourceTerm, e.TargetTerm)
			return nil
		}
	}
	return fmt.Errorf("glossary term %q not found", term)
}

func runGlossarySet(cmd *cobra.Command, args []string) error {
	data, err := readInputFile(glossarySetFlags.file)
	if err != nil {
		return fmt.Errorf("reading glossary file: %w", err)
	}
	entries, err := parseGlossaryFile(data, glossarySetFlags.sourceLocale, glossarySetFlags.targetLocale)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	var created []glossaryEntry
	for _, entry := range entries {
		out, err := postGlossaryEntry(c, args[0], entry)
		if err != nil {
			return err
		}
		created = append(created, out)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"created": len(created),
			"entries": created,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d glossary terms\n", len(created))
	return nil
}

func runGlossaryAdd(cmd *cobra.Command, args []string) error {
	entry, err := postGlossaryEntry(client.New(Cfg.Server, Cfg.APIKey), args[0], glossaryFileEntry{
		SourceLocale: glossaryAddFlags.sourceLocale,
		TargetLocale: glossaryAddFlags.targetLocale,
		SourceTerm:   glossaryAddFlags.sourceTerm,
		TargetTerm:   glossaryAddFlags.targetTerm,
	})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(entry)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added glossary term %q → %q\n", entry.SourceTerm, entry.TargetTerm)
	return nil
}