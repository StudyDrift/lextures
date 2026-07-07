package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var orgsCrossListCmd = &cobra.Command{
	Use:   "cross-list-groups",
	Short: "Manage org cross-list groups for shared rosters",
}

var orgsCrossListListCmd = &cobra.Command{
	Use:   "list <org>",
	Short: "List cross-list groups for an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsCrossListList,
}

var orgsCrossListCreateFlags struct {
	course  string
	primary string
	name    string
}

var orgsCrossListCreateCmd = &cobra.Command{
	Use:   "create <org>",
	Short: "Create a cross-list group",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgsCrossListCreate,
}

var orgsCrossListAddMemberFlags struct {
	section string
}

var orgsCrossListAddMemberCmd = &cobra.Command{
	Use:   "add-member <org> <group>",
	Short: "Add a section to a cross-list group",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgsCrossListAddMember,
}

var orgsCrossListRemoveMemberCmd = &cobra.Command{
	Use:   "remove-member <org> <group> <section>",
	Short: "Remove a section from a cross-list group",
	Args:  cobra.ExactArgs(3),
	RunE:  runOrgsCrossListRemoveMember,
}

func init() {
	orgsCrossListCreateCmd.Flags().StringVar(&orgsCrossListCreateFlags.course, "course", "", "course code (required)")
	_ = orgsCrossListCreateCmd.MarkFlagRequired("course")
	orgsCrossListCreateCmd.Flags().StringVar(&orgsCrossListCreateFlags.primary, "primary", "", "primary section UUID (required)")
	_ = orgsCrossListCreateCmd.MarkFlagRequired("primary")
	orgsCrossListCreateCmd.Flags().StringVar(&orgsCrossListCreateFlags.name, "name", "", "group display name")

	orgsCrossListAddMemberCmd.Flags().StringVar(&orgsCrossListAddMemberFlags.section, "section", "", "section UUID (required)")
	_ = orgsCrossListAddMemberCmd.MarkFlagRequired("section")

	orgsCrossListCmd.AddCommand(
		orgsCrossListListCmd,
		orgsCrossListCreateCmd,
		orgsCrossListAddMemberCmd,
		orgsCrossListRemoveMemberCmd,
	)
	orgsCmd.AddCommand(orgsCrossListCmd)
}

func runOrgsCrossListList(cmd *cobra.Command, args []string) error {
	groups, err := fetchCrossListGroups(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(groups)
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tCOURSE\tPRIMARY_SECTION\tNAME")
	for _, g := range groups {
		name := ""
		if g.Name != nil {
			name = *g.Name
		}
		primary := ""
		if g.PrimarySectionID != nil {
			primary = *g.PrimarySectionID
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", g.ID, g.CourseID, primary, name)
	}
	return w.Flush()
}

func runOrgsCrossListCreate(cmd *cobra.Command, args []string) error {
	var name *string
	if orgsCrossListCreateFlags.name != "" {
		name = &orgsCrossListCreateFlags.name
	}
	body, err := createCrossListGroup(
		client.New(Cfg.Server, Cfg.APIKey),
		args[0],
		orgsCrossListCreateFlags.course,
		orgsCrossListCreateFlags.primary,
		name,
	)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Created cross-list group for %s", orgsCrossListCreateFlags.course))
}

func runOrgsCrossListAddMember(cmd *cobra.Command, args []string) error {
	body, err := addCrossListMember(
		client.New(Cfg.Server, Cfg.APIKey),
		args[0],
		args[1],
		orgsCrossListAddMemberFlags.section,
	)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Added section to cross-list group")
}

func runOrgsCrossListRemoveMember(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/orgs/" + url.PathEscape(args[0]) +
		"/cross-list-groups/" + url.PathEscape(args[1]) +
		"/members/" + url.PathEscape(args[2])
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("removing cross-list member: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Removed section from cross-list group")
	return nil
}