package cmd

import (
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var engagementCmd = &cobra.Command{
	Use:   "engagement",
	Short: "Emit engagement analytics events",
}

var engagementEmitFlags struct {
	file string
}

var engagementEmitCmd = &cobra.Command{
	Use:   "emit",
	Short: "Emit engagement events from a JSON file",
	RunE:  runEngagementEmit,
}

func init() {
	engagementEmitCmd.Flags().StringVar(&engagementEmitFlags.file, "file", "", "event JSON array or single object (required)")
	_ = engagementEmitCmd.MarkFlagRequired("file")

	engagementCmd.AddCommand(engagementEmitCmd)
	rootCmd.AddCommand(engagementCmd)
}

func runEngagementEmit(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	events, err := readJSONObjectsFromFile(engagementEmitFlags.file)
	if err != nil {
		return err
	}
	body, err := postEngagementEvents(c, events)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}