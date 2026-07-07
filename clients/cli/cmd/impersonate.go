package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/lextures/lextures/clients/cli/internal/auth"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var impersonateCmd = &cobra.Command{
	Use:   "impersonate",
	Short: "Start or stop admin impersonation sessions",
}

var impersonateStartFlags struct {
	user string
}

var impersonateStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start impersonating a user (read-only writes blocked server-side)",
	RunE:  runImpersonateStart,
}

var impersonateStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "End the active impersonation session",
	RunE:  runImpersonateStop,
}

var impersonateWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show real and effective identity",
	RunE:  runImpersonateWhoami,
}

func init() {
	impersonateStartCmd.Flags().StringVar(&impersonateStartFlags.user, "user", "", "target user UUID (required)")
	_ = impersonateStartCmd.MarkFlagRequired("user")

	impersonateCmd.AddCommand(impersonateStartCmd, impersonateStopCmd, impersonateWhoamiCmd)
	rootCmd.AddCommand(impersonateCmd)
}

func runImpersonateStart(cmd *cobra.Command, _ []string) error {
	if ImpersonationActive {
		return fmt.Errorf("impersonation session already active — run 'lextures impersonate stop' first")
	}
	realToken := Cfg.APIKey
	if realToken == "" {
		return errNotAuthenticated
	}
	c := client.New(Cfg.Server, realToken)
	token, expiresAt, target, err := startImpersonation(c, impersonateStartFlags.user)
	if err != nil {
		return err
	}
	sess := &auth.ImpersonationSession{
		RealAccessToken:    realToken,
		ImpersonationToken: token,
		TargetUserID:       impersonateStartFlags.user,
		ExpiresAt:          expiresAt,
	}
	if err := auth.NewImpersonationStore().Save(activeProfile(), sess); err != nil {
		return fmt.Errorf("saving impersonation session: %w", err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"active":    true,
			"target":    target,
			"expiresAt": expiresAt,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Impersonation started for user %s (expires %s).\n", impersonateStartFlags.user, expiresAt)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Writes are blocked server-side until you run 'lextures impersonate stop'.")
	return nil
}

func runImpersonateStop(cmd *cobra.Command, _ []string) error {
	store := auth.NewImpersonationStore()
	profile := activeProfile()
	sess, err := store.Load(profile)
	if err != nil {
		return fmt.Errorf("loading impersonation session: %w", err)
	}
	if sess == nil || sess.ImpersonationToken == "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No active impersonation session.")
		return nil
	}
	c := client.New(Cfg.Server, sess.ImpersonationToken)
	if err := stopImpersonation(c); err != nil {
		return err
	}
	if err := store.Delete(profile); err != nil {
		return fmt.Errorf("clearing impersonation session: %w", err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"stopped": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Impersonation session ended.")
	return nil
}

func runImpersonateWhoami(cmd *cobra.Command, _ []string) error {
	me, raw, err := fetchIdentityMe(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		out := map[string]any{
			"effective": map[string]any{
				"id":    me.ID,
				"email": me.Email,
			},
			"impersonating": me.Impersonating != nil,
		}
		if me.Impersonating != nil {
			out["real"] = map[string]any{"id": me.Impersonating.AdminID}
		} else if RealAPIKey != "" {
			realMe, _, realErr := fetchIdentityMe(client.New(Cfg.Server, RealAPIKey))
			if realErr == nil {
				out["real"] = map[string]any{"id": realMe.ID, "email": realMe.Email}
			}
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	if me.Impersonating != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "IMPERSONATION ACTIVE\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  real user:       %s\n", me.Impersonating.AdminID)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  effective user:  %s (%s)\n", me.ID, me.Email)
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  writes are blocked server-side")
		return nil
	}
	if ImpersonationActive {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "warning: impersonation token loaded but /api/v1/me did not report impersonation")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "user: %s (%s)\n", me.ID, me.Email)
	_ = raw
	return nil
}