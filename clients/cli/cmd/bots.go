package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var botsCmd = &cobra.Command{
	Use:   "bots",
	Short: "Manage classroom bot integrations",
}

var botsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bot connections",
	RunE:  runBotsList,
}

var botsRegisterFlags struct {
	file string
}

var botsRegisterCmd = &cobra.Command{
	Use:   "register <platform>",
	Short: "Register a bot (slack oauth URL or discord config)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBotsRegister,
}

var botsLinkCmd = &cobra.Command{
	Use:   "link <platform>",
	Short: "Start user bot link OAuth flow",
	Args:  cobra.ExactArgs(1),
	RunE:  runBotsLink,
}

var botsUnlinkCmd = &cobra.Command{
	Use:   "unlink <platform>",
	Short: "Unlink your bot account",
	Args:  cobra.ExactArgs(1),
	RunE:  runBotsUnlink,
}

var botsDisconnectCmd = &cobra.Command{
	Use:   "disconnect <id>",
	Short: "Disconnect an organization bot",
	Args:  cobra.ExactArgs(1),
	RunE:  runBotsDisconnect,
}

func init() {
	botsRegisterCmd.Flags().StringVar(&botsRegisterFlags.file, "file", "", "discord connect JSON (guildId, guildName, botToken)")

	botsCmd.AddCommand(botsListCmd, botsRegisterCmd, botsLinkCmd, botsUnlinkCmd, botsDisconnectCmd)
	rootCmd.AddCommand(botsCmd)
}

func runBotsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := fetchBots(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tPLATFORM\tSTATUS")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", row.ID, row.Platform, row.Status)
	}
	return w.Flush()
}

func runBotsRegister(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	platform := strings.ToLower(strings.TrimSpace(args[0]))
	switch platform {
	case "slack":
		body, err := registerBotSlack(c)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(body)
			return err
		}
		var out struct {
			AuthorizeURL string `json:"authorizeUrl"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			_, err = cmd.OutOrStdout().Write(body)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open this URL to install the Slack bot:\n%s\n", out.AuthorizeURL)
		return nil
	case "discord":
		if botsRegisterFlags.file == "" {
			return fmt.Errorf("discord registration requires --file with guildId, guildName, botToken")
		}
		payload, err := loadJSONFile(botsRegisterFlags.file)
		if err != nil {
			return err
		}
		body, err := registerBotDiscord(c, payload)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(body)
		return err
	default:
		return fmt.Errorf("unsupported platform %q (use slack or discord)", platform)
	}
}

func runBotsLink(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := startBotLink(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		AuthorizeURL string `json:"authorizeUrl"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open this URL to link your %s account:\n%s\n", args[0], out.AuthorizeURL)
	return nil
}

func runBotsUnlink(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := unlinkBot(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Bot account unlinked.")
	return nil
}

func runBotsDisconnect(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := disconnectBot(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Bot disconnected.")
	return nil
}