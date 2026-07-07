package cmd

import (
	"encoding/json"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var recommendationsCmd = &cobra.Command{
	Use:   "recommendations",
	Short: "Record recommendation surface events",
}

var recommendationsEventFlags struct {
	file string
}

var recommendationsEventCmd = &cobra.Command{
	Use:   "event",
	Short: "Record a recommendation impression/click/dismiss event",
	RunE:  runRecommendationsEvent,
}

func init() {
	recommendationsEventCmd.Flags().StringVar(&recommendationsEventFlags.file, "file", "", "event JSON (courseId, eventType, surface, itemId, rank)")
	_ = recommendationsEventCmd.MarkFlagRequired("file")

	recommendationsCmd.AddCommand(recommendationsEventCmd)
	rootCmd.AddCommand(recommendationsCmd)
}

func runRecommendationsEvent(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(recommendationsEventFlags.file)
	if err != nil {
		return err
	}
	body, err := postRecommendationEvent(c, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
}