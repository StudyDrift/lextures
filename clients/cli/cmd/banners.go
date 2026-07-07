package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var bannersCmd = &cobra.Command{
	Use:   "banners",
	Short: "Manage maintenance and announcement banners",
}

var bannersListFlags struct {
	scope string
}

var bannersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List banners",
	RunE:  runBannersList,
}

var bannersWriteFlags struct {
	message  string
	severity string
	from     string
	until    string
	audience string
	scope    string
}

var bannersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a banner",
	RunE:  runBannersCreate,
}

var bannersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a banner",
	Args:  cobra.ExactArgs(1),
	RunE:  runBannersUpdate,
}

var bannersDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a banner",
	Args:  cobra.ExactArgs(1),
	RunE:  runBannersDelete,
}

var bannersPublishCmd = &cobra.Command{
	Use:   "publish <id>",
	Short: "Activate a banner now",
	Args:  cobra.ExactArgs(1),
	RunE:  runBannersPublish,
}

var bannersExpireCmd = &cobra.Command{
	Use:   "expire <id>",
	Short: "Deactivate a banner immediately",
	Args:  cobra.ExactArgs(1),
	RunE:  runBannersExpire,
}

func init() {
	bindBannerWriteFlags(bannersCreateCmd)
	bindBannerWriteFlags(bannersUpdateCmd)
	_ = bannersCreateCmd.MarkFlagRequired("message")
	bannersListCmd.Flags().StringVar(&bannersListFlags.scope, "scope", "", "filter global banners (global)")

	bannersCmd.AddCommand(
		bannersListCmd,
		bannersCreateCmd,
		bannersUpdateCmd,
		bannersDeleteCmd,
		bannersPublishCmd,
		bannersExpireCmd,
	)
	rootCmd.AddCommand(bannersCmd)
}

func bindBannerWriteFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&bannersWriteFlags.message, "message", "", "banner message")
	cmd.Flags().StringVar(&bannersWriteFlags.severity, "severity", "info", "severity: info, warning, critical")
	cmd.Flags().StringVar(&bannersWriteFlags.from, "from", "", "start time (RFC3339)")
	cmd.Flags().StringVar(&bannersWriteFlags.until, "until", "", "end time (RFC3339)")
	cmd.Flags().StringVar(&bannersWriteFlags.audience, "audience", "", "audience scope: global")
	cmd.Flags().StringVar(&bannersWriteFlags.scope, "scope", "org", "banner scope: org or global")
}

func runBannersList(cmd *cobra.Command, _ []string) error {
	rows, raw, err := fetchBanners(client.New(Cfg.Server, Cfg.APIKey), bannersListFlags.scope)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSCOPE\tSEVERITY\tACTIVE\tMESSAGE")
	for _, b := range rows {
		active := "no"
		if b.IsActive {
			active = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", b.ID, b.Scope, b.Severity, active, b.Message)
	}
	return w.Flush()
}

func runBannersCreate(cmd *cobra.Command, _ []string) error {
	if err := validateBannerWindow(bannersWriteFlags.from, bannersWriteFlags.until); err != nil {
		return err
	}
	in := bannerWriteInput{
		Scope:    bannersWriteFlags.scope,
		Message:  bannersWriteFlags.message,
		Severity: bannersWriteFlags.severity,
		From:     bannersWriteFlags.from,
		Until:    bannersWriteFlags.until,
		Audience: bannersWriteFlags.audience,
	}
	row, raw, err := createBanner(client.New(Cfg.Server, Cfg.APIKey), in)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created banner %s (%s).\n", row.ID, row.Severity)
	return nil
}

func runBannersUpdate(cmd *cobra.Command, args []string) error {
	if err := validateBannerWindow(bannersWriteFlags.from, bannersWriteFlags.until); err != nil {
		return err
	}
	in := bannerWriteInput{
		Scope:    bannersWriteFlags.scope,
		Message:  bannersWriteFlags.message,
		Severity: bannersWriteFlags.severity,
		From:     bannersWriteFlags.from,
		Until:    bannersWriteFlags.until,
		Audience: bannersWriteFlags.audience,
	}
	row, raw, err := updateBanner(client.New(Cfg.Server, Cfg.APIKey), args[0], in)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated banner %s.\n", row.ID)
	return nil
}

func runBannersDelete(cmd *cobra.Command, args []string) error {
	if err := deleteBanner(client.New(Cfg.Server, Cfg.APIKey), args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": args[0]})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted banner %s.\n", args[0])
	return nil
}

func runBannersPublish(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	existing, err := findBanner(c, args[0])
	if err != nil {
		return err
	}
	active := true
	now := time.Now().UTC().Format(time.RFC3339)
	in := bannerWriteInput{
		Scope:    existing.Scope,
		Message:  existing.Message,
		Severity: existing.Severity,
		From:     now,
		IsActive: &active,
	}
	row, raw, err := updateBanner(c, args[0], in)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Published banner %s.\n", row.ID)
	return nil
}

func runBannersExpire(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	existing, err := findBanner(c, args[0])
	if err != nil {
		return err
	}
	active := false
	now := time.Now().UTC().Format(time.RFC3339)
	in := bannerWriteInput{
		Scope:    existing.Scope,
		Message:  existing.Message,
		Severity: existing.Severity,
		Until:    now,
		IsActive: &active,
	}
	row, raw, err := updateBanner(c, args[0], in)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Expired banner %s.\n", row.ID)
	return nil
}