package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var accessKeysCmd = &cobra.Command{
	Use:   "access-keys",
	Short: "Manage personal API access keys",
}

var accessKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your access keys",
	RunE:  runAccessKeysList,
}

var accessKeysCreateFlags struct {
	file      string
	secretOut string
}

var accessKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a personal access key",
	RunE:  runAccessKeysCreate,
}

var accessKeysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a personal access key",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccessKeysRevoke,
}

func init() {
	accessKeysCreateCmd.Flags().StringVar(&accessKeysCreateFlags.file, "file", "", "access key JSON (label, scopes, courseIds, expiresAt)")
	_ = accessKeysCreateCmd.MarkFlagRequired("file")
	accessKeysCreateCmd.Flags().StringVar(&accessKeysCreateFlags.secretOut, "secret-out", "", "write one-time token to a file")

	accessKeysCmd.AddCommand(accessKeysListCmd, accessKeysCreateCmd, accessKeysRevokeCmd)
	rootCmd.AddCommand(accessKeysCmd)
}

func runAccessKeysList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	tokens, raw, err := fetchAccessKeys(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tLABEL\tMASK\tSCOPES")
	for _, t := range tokens {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", t.ID, t.Label, t.TokenMask, t.Scopes)
	}
	return w.Flush()
}

func runAccessKeysCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(accessKeysCreateFlags.file)
	if err != nil {
		return err
	}
	out, _, err := createAccessKey(c, payload)
	if err != nil {
		return err
	}
	secret, _ := out["token"].(string)
	if globalFlags.jsonOut {
		redactTokenSecret(out)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created access key %v\n", out["id"])
	return writeOneTimeSecret(cmd, secret, accessKeysCreateFlags.secretOut)
}

func runAccessKeysRevoke(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := revokeAccessKey(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Access key revoked.")
	return nil
}