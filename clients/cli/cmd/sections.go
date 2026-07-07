package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var sectionsCmd = &cobra.Command{
	Use:   "sections",
	Short: "Manage course sections",
}

var sectionsCreateFlags struct {
	code     string
	name     string
	capacity int
}

var sectionsCreateCmd = &cobra.Command{
	Use:   "create <course>",
	Short: "Create a section",
	Args:  cobra.ExactArgs(1),
	RunE:  runSectionsCreate,
}

var sectionsUpdateFlags struct {
	code     string
	name     string
	status   string
	capacity int
}

var sectionsUpdateCmd = &cobra.Command{
	Use:   "update <course> <section>",
	Short: "Update a section",
	Args:  cobra.ExactArgs(2),
	RunE:  runSectionsUpdate,
}

var sectionsDeleteCmd = &cobra.Command{
	Use:   "delete <course> <section>",
	Short: "Archive a section",
	Args:  cobra.ExactArgs(2),
	RunE:  runSectionsDelete,
}

var sectionsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List sections for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runSectionsList,
}

var sectionsMoveFlags struct {
	user string
	to   string
	role string
}

var sectionsMoveCmd = &cobra.Command{
	Use:   "move <course>",
	Short: "Move a student enrollment to another section",
	Args:  cobra.ExactArgs(1),
	RunE:  runSectionsMove,
}

var sectionsCrossListFlags struct {
	org     string
	primary string
	name    string
}

var sectionsCrossListCmd = &cobra.Command{
	Use:   "cross-list <course>",
	Short: "Create an org cross-list group with a primary section",
	Args:  cobra.ExactArgs(1),
	RunE:  runSectionsCrossList,
}

func init() {
	sectionsCreateCmd.Flags().StringVar(&sectionsCreateFlags.code, "code", "", "section code (required)")
	_ = sectionsCreateCmd.MarkFlagRequired("code")
	sectionsCreateCmd.Flags().StringVar(&sectionsCreateFlags.name, "name", "", "display name")
	sectionsCreateCmd.Flags().IntVar(&sectionsCreateFlags.capacity, "capacity", 0, "enrollment capacity (0 = unlimited)")

	sectionsUpdateCmd.Flags().StringVar(&sectionsUpdateFlags.code, "code", "", "new section code")
	sectionsUpdateCmd.Flags().StringVar(&sectionsUpdateFlags.name, "name", "", "display name")
	sectionsUpdateCmd.Flags().StringVar(&sectionsUpdateFlags.status, "status", "", "status: active, cancelled, archived")
	sectionsUpdateCmd.Flags().IntVar(&sectionsUpdateFlags.capacity, "capacity", 0, "enrollment capacity")

	sectionsMoveCmd.Flags().StringVar(&sectionsMoveFlags.user, "user", "", "student user UUID or email (required)")
	_ = sectionsMoveCmd.MarkFlagRequired("user")
	sectionsMoveCmd.Flags().StringVar(&sectionsMoveFlags.to, "to", "", "target section UUID or code (required)")
	_ = sectionsMoveCmd.MarkFlagRequired("to")
	sectionsMoveCmd.Flags().StringVar(&sectionsMoveFlags.role, "role", "student", "enrollment role")

	sectionsCrossListCmd.Flags().StringVar(&sectionsCrossListFlags.org, "org", "", "organization UUID (required)")
	_ = sectionsCrossListCmd.MarkFlagRequired("org")
	sectionsCrossListCmd.Flags().StringVar(&sectionsCrossListFlags.primary, "primary", "", "primary section UUID or code (required)")
	_ = sectionsCrossListCmd.MarkFlagRequired("primary")
	sectionsCrossListCmd.Flags().StringVar(&sectionsCrossListFlags.name, "name", "", "cross-list group name")

	sectionsCmd.AddCommand(
		sectionsListCmd,
		sectionsCreateCmd,
		sectionsUpdateCmd,
		sectionsDeleteCmd,
		sectionsMoveCmd,
		sectionsCrossListCmd,
	)
	rootCmd.AddCommand(sectionsCmd)
}

func runSectionsList(cmd *cobra.Command, args []string) error {
	rows, err := fetchSections(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tCODE\tNAME\tSTATUS\tCAPACITY")
	for _, row := range rows {
		name := ""
		if row.Name != nil {
			name = *row.Name
		}
		capacity := ""
		if row.Capacity != nil {
			capacity = fmt.Sprintf("%d", *row.Capacity)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.ID, row.SectionCode, name, row.Status, capacity)
	}
	return w.Flush()
}

func runSectionsCreate(cmd *cobra.Command, args []string) error {
	body := map[string]any{
		"sectionCode": strings.TrimSpace(sectionsCreateFlags.code),
	}
	if sectionsCreateFlags.name != "" {
		body["name"] = sectionsCreateFlags.name
	}
	if sectionsCreateFlags.capacity > 0 {
		body["capacity"] = sectionsCreateFlags.capacity
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/sections"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating section: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return emitRawOrMessage(cmd, respBody, fmt.Sprintf("Created section %s in %s", sectionsCreateFlags.code, args[0]))
}

func runSectionsUpdate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	sections, err := fetchSections(c, args[0])
	if err != nil {
		return err
	}
	sec, err := resolveSectionRef(sections, args[1])
	if err != nil {
		return err
	}
	body := map[string]any{}
	if sectionsUpdateFlags.code != "" {
		body["sectionCode"] = sectionsUpdateFlags.code
	}
	if sectionsUpdateFlags.name != "" {
		body["name"] = sectionsUpdateFlags.name
	}
	if sectionsUpdateFlags.status != "" {
		body["status"] = sectionsUpdateFlags.status
	}
	if sectionsUpdateFlags.capacity > 0 {
		body["capacity"] = sectionsUpdateFlags.capacity
	}
	if len(body) == 0 {
		return fmt.Errorf("provide at least one of --code, --name, --status, or --capacity")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/sections/" + url.PathEscape(sec.ID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating section: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return emitRawOrMessage(cmd, respBody, fmt.Sprintf("Updated section %s", sec.SectionCode))
}

func runSectionsDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	sections, err := fetchSections(c, args[0])
	if err != nil {
		return err
	}
	sec, err := resolveSectionRef(sections, args[1])
	if err != nil {
		return err
	}
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/sections/" + url.PathEscape(sec.ID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("archiving section: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
				"archived": sec.ID,
				"course":   args[0],
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Archived section %s\n", sec.SectionCode)
		return nil
	}
	body, _ := readResponseBody(resp)
	return apiErrorBody(resp.StatusCode, body)
}

func runSectionsMove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	en, err := resolveEnrollmentForUser(c, args[0], sectionsMoveFlags.user, sectionsMoveFlags.role)
	if err != nil {
		return err
	}
	sections, err := fetchSections(c, args[0])
	if err != nil {
		return err
	}
	sec, err := resolveSectionRef(sections, sectionsMoveFlags.to)
	if err != nil {
		return err
	}
	if err := transferEnrollmentSection(c, en.ID, sec.ID); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"enrollmentId": en.ID,
			"sectionId":    sec.ID,
			"sectionCode":  sec.SectionCode,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Moved %s to section %s\n", sectionsMoveFlags.user, sec.SectionCode)
	return nil
}

func runSectionsCrossList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	sections, err := fetchSections(c, args[0])
	if err != nil {
		return err
	}
	primary, err := resolveSectionRef(sections, sectionsCrossListFlags.primary)
	if err != nil {
		return err
	}
	var name *string
	if sectionsCrossListFlags.name != "" {
		name = &sectionsCrossListFlags.name
	}
	body, err := createCrossListGroup(c, sectionsCrossListFlags.org, args[0], primary.ID, name)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Created cross-list group for %s (primary %s)", args[0], primary.SectionCode))
}