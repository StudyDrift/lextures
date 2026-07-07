package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage organization service API tokens",
}

var tokensListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organization service tokens",
	RunE:  runTokensList,
}

var tokensCreateFlags struct {
	file      string
	secretOut string
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a service API token",
	RunE:  runTokensCreate,
}

var tokensRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a service API token",
	Args:  cobra.ExactArgs(1),
	RunE:  runTokensRevoke,
}

func init() {
	tokensCreateCmd.Flags().StringVar(&tokensCreateFlags.file, "file", "", "token JSON (label, serviceAccountName, scopes, expiresAt)")
	_ = tokensCreateCmd.MarkFlagRequired("file")
	tokensCreateCmd.Flags().StringVar(&tokensCreateFlags.secretOut, "secret-out", "", "write one-time token to a file")

	tokensCmd.AddCommand(tokensListCmd, tokensCreateCmd, tokensRevokeCmd)
	rootCmd.AddCommand(tokensCmd)
}

func runTokensList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	tokens, raw, err := fetchAdminTokens(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tLABEL\tSERVICE\tMASK\tSCOPES")
	for _, t := range tokens {
		svc := ""
		if t.ServiceAccountName != nil {
			svc = *t.ServiceAccountName
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\n", t.ID, t.Label, svc, t.TokenMask, t.Scopes)
	}
	return w.Flush()
}

func runTokensCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(tokensCreateFlags.file)
	if err != nil {
		return err
	}
	out, _, err := createAdminToken(c, payload)
	if err != nil {
		return err
	}
	secret, _ := out["token"].(string)
	if globalFlags.jsonOut {
		redactTokenSecret(out)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created service token %v\n", out["id"])
	return writeOneTimeSecret(cmd, secret, tokensCreateFlags.secretOut)
}

func runTokensRevoke(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := revokeAdminToken(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Token revoked.")
	return nil
}