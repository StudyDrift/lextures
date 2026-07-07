package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var devKeysCmd = &cobra.Command{
	Use:   "dev-keys",
	Short: "Manage developer API keys and OAuth apps",
}

var devKeysListFlags struct {
	oauth bool
}

var devKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List developer keys",
	RunE:  runDevKeysList,
}

var devKeysCreateFlags struct {
	file       string
	oauth      bool
	secretOut  string
}

var devKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a developer API key or OAuth app",
	RunE:  runDevKeysCreate,
}

var devKeysRotateFlags struct {
	grace int
}

var devKeysRotateCmd = &cobra.Command{
	Use:   "rotate <id>",
	Short: "Rotate a developer API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runDevKeysRotate,
}

var devKeysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a developer API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runDevKeysRevoke,
}

func init() {
	devKeysListCmd.Flags().BoolVar(&devKeysListFlags.oauth, "oauth", false, "list OAuth developer apps instead of access keys")

	devKeysCreateCmd.Flags().StringVar(&devKeysCreateFlags.file, "file", "", "key/app JSON file (required)")
	_ = devKeysCreateCmd.MarkFlagRequired("file")
	devKeysCreateCmd.Flags().BoolVar(&devKeysCreateFlags.oauth, "oauth", false, "create an OAuth developer app")
	devKeysCreateCmd.Flags().StringVar(&devKeysCreateFlags.secretOut, "secret-out", "", "write one-time secret to a file")

	devKeysRotateCmd.Flags().IntVar(&devKeysRotateFlags.grace, "grace", 24, "overlap hours for the old key")

	devKeysCmd.AddCommand(devKeysListCmd, devKeysCreateCmd, devKeysRotateCmd, devKeysRevokeCmd)
	rootCmd.AddCommand(devKeysCmd)
}

func runDevKeysList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if devKeysListFlags.oauth {
		apps, raw, err := fetchDeveloperApps(c)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tCLIENT_ID\tSECRET_PREFIX")
		for _, a := range apps {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.ClientID, a.ClientSecretPrefix)
		}
		return w.Flush()
	}
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

func runDevKeysCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(devKeysCreateFlags.file)
	if err != nil {
		return err
	}
	if devKeysCreateFlags.oauth {
		out, _, err := createDeveloperApp(c, payload)
		if err != nil {
			return err
		}
		secret, _ := out["clientSecret"].(string)
		if globalFlags.jsonOut {
			redactDeveloperAppSecret(out)
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created OAuth app %v (clientId=%v)\n", out["name"], out["clientId"])
		return writeOneTimeSecret(cmd, secret, devKeysCreateFlags.secretOut)
	}
	out, _, err := createAccessKey(c, payload)
	if err != nil {
		return err
	}
	secret, _ := out["token"].(string)
	if globalFlags.jsonOut {
		redactDeveloperAppSecret(out)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created access key %v\n", out["id"])
	return writeOneTimeSecret(cmd, secret, devKeysCreateFlags.secretOut)
}

func runDevKeysRotate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	out, _, err := rotateAccessKey(c, args[0], devKeysRotateFlags.grace)
	if err != nil {
		return err
	}
	secret, _ := out["token"].(string)
	if globalFlags.jsonOut {
		redactDeveloperAppSecret(out)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Rotated key %v (overlap %dh)\n", out["id"], devKeysRotateFlags.grace)
	_, _ = fmt.Fprintln(os.Stderr, "Save the new token now — it will not be shown again.")
	if secret != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "token=%s\n", secret)
	}
	return nil
}

func runDevKeysRevoke(cmd *cobra.Command, args []string) error {
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

func writeOneTimeSecret(cmd *cobra.Command, secret, secretOut string) error {
	if secret == "" {
		return nil
	}
	_, _ = fmt.Fprintln(os.Stderr, "Save the secret now — it will not be shown again on re-fetch.")
	if secretOut != "" {
		if err := os.WriteFile(secretOut, []byte(secret+"\n"), 0o600); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret written to %s\n", secretOut)
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "secret=%s\n", secret)
	return nil
}