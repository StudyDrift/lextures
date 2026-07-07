package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin utilities (search, etc.)",
}

var adminSearchFlags struct {
	typeFilter string
}

var adminSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search users, courses, and content",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminSearch,
}

func init() {
	adminSearchCmd.Flags().StringVar(&adminSearchFlags.typeFilter, "type", "", "limit to user, course, or content")

	adminCmd.AddCommand(adminSearchCmd)
	rootCmd.AddCommand(adminCmd)
}

func runAdminSearch(cmd *cobra.Command, args []string) error {
	results, raw, err := fetchAdminSearch(client.New(Cfg.Server, Cfg.APIKey), args[0], adminSearchFlags.typeFilter)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tTITLE\tSUBTITLE\tPATH")
	for _, r := range results {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Type, r.Title, r.Subtitle, r.Path)
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(w, "(no results)")
	}
	return w.Flush()
}