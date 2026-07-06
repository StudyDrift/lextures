package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var linksCmd = &cobra.Command{
	Use:   "links",
	Short: "Manage external links in course modules",
}

var linksListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List external links in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runLinksList,
}

var linksAddFlags struct {
	module string
	title  string
	url    string
}

var linksAddCmd = &cobra.Command{
	Use:   "add <course_code>",
	Short: "Add an external link to a module",
	Args:  cobra.ExactArgs(1),
	RunE:  runLinksAdd,
}

func init() {
	linksAddCmd.Flags().StringVar(&linksAddFlags.module, "module", "", "module id (required)")
	linksAddCmd.Flags().StringVar(&linksAddFlags.title, "title", "", "link title (required)")
	linksAddCmd.Flags().StringVar(&linksAddFlags.url, "url", "", "link URL (required)")
	_ = linksAddCmd.MarkFlagRequired("module")
	_ = linksAddCmd.MarkFlagRequired("title")
	_ = linksAddCmd.MarkFlagRequired("url")

	linksCmd.AddCommand(linksListCmd, linksAddCmd)
	rootCmd.AddCommand(linksCmd)
}

func filterExternalLinks(items []structureItemPublic) []structureItemPublic {
	var links []structureItemPublic
	for _, it := range items {
		if it.Kind == "external_link" {
			links = append(links, it)
		}
	}
	return links
}

func runLinksList(cmd *cobra.Command, args []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	links := filterExternalLinks(body.Items)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(links)
	}
	if len(links) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No external links.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tURL\tPUBLISHED")
	for _, l := range links {
		url := "-"
		if l.ExternalURL != nil {
			url = *l.ExternalURL
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", l.ID, l.Title, url, l.Published)
	}
	return w.Flush()
}

func runLinksAdd(cmd *cobra.Command, args []string) error {
	item, err := addItemToModule(client.New(Cfg.Server, Cfg.APIKey), args[0],
		linksAddFlags.module, "link", linksAddFlags.title, linksAddFlags.url)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added link %s (%s)\n", item.Title, item.ID)
	return nil
}