package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var meCmd = &cobra.Command{Use: "me", Short: "Current account profile and security"}

var meGetCmd = &cobra.Command{Use: "get", Short: "Get current user profile", RunE: runMeGet}

var meUpdateFlags struct{ file string }
var meUpdateCmd = &cobra.Command{Use: "update", Short: "Update profile fields from JSON", RunE: runMeUpdate}

var meProfileFieldsCmd = &cobra.Command{Use: "profile-fields", Short: "Custom profile fields"}
var meProfileFieldsGetCmd = &cobra.Command{Use: "get", Short: "Get profile fields", RunE: runMeProfileFieldsGet}
var meProfileFieldsSetFlags struct{ file string }
var meProfileFieldsSetCmd = &cobra.Command{Use: "set", Short: "Set profile fields", RunE: runMeProfileFieldsSet}

var meSessionsCmd = &cobra.Command{Use: "sessions", Short: "Active sessions"}
var meSessionsListCmd = &cobra.Command{Use: "list", Short: "List sessions", RunE: runMeSessionsList}
var meSessionsRevokeFlags struct {
	all             bool
	includeCurrent  bool
	yes             bool
}
var meSessionsRevokeCmd = &cobra.Command{Use: "revoke", Short: "Revoke sessions", RunE: runMeSessionsRevoke}

var meMfaCmd = &cobra.Command{Use: "mfa", Short: "Multi-factor authentication"}
var meMfaStatusCmd = &cobra.Command{Use: "status", Short: "List MFA factors", RunE: runMeMfaStatus}
var meMfaDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disable an MFA factor",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeMfaDisable,
}

var meOidcCmd = &cobra.Command{Use: "oidc-identities", Short: "Linked OIDC identities"}
var meOidcListCmd = &cobra.Command{Use: "list", Short: "List linked identities", RunE: runMeOidcList}
var meOidcUnlinkCmd = &cobra.Command{
	Use:   "unlink <id>",
	Short: "Unlink an OIDC identity",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeOidcUnlink,
}

var meAccessKeysCmd = &cobra.Command{Use: "access-keys", Short: "Personal API access keys"}
var meAccessKeysListCmd = &cobra.Command{Use: "list", Short: "List access keys", RunE: runMeAccessKeysList}
var meAccessKeysCreateFlags struct {
	file      string
	secretOut string
}
var meAccessKeysCreateCmd = &cobra.Command{Use: "create", Short: "Create an access key", RunE: runMeAccessKeysCreate}
var meAccessKeysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an access key",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeAccessKeysRevoke,
}

var meDemographicsCmd = &cobra.Command{Use: "demographics", Short: "Demographics self-service"}
var meDemographicsGetCmd = &cobra.Command{Use: "get", Short: "Get demographics", RunE: runMeDemographicsGet}
var meDemographicsSetFlags struct{ file string }
var meDemographicsSetCmd = &cobra.Command{Use: "set", Short: "Set demographics", RunE: runMeDemographicsSet}

var meConsentCmd = &cobra.Command{Use: "consent-studies", Short: "Research consent studies"}
var meConsentListCmd = &cobra.Command{Use: "list", Short: "List pending consent studies", RunE: runMeConsentList}
var meConsentOptInCmd = &cobra.Command{
	Use:   "opt-in <id>",
	Short: "Opt in to a consent study",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeConsentOptIn,
}
var meConsentOptOutCmd = &cobra.Command{
	Use:   "opt-out <id>",
	Short: "Opt out of a consent study",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeConsentOptOut,
}

var meOnboardingCmd = &cobra.Command{Use: "onboarding", Short: "Onboarding status"}
var meOnboardingStatusCmd = &cobra.Command{Use: "status", Short: "Get onboarding status", RunE: runMeOnboardingStatus}

var meEntitlementsCmd = &cobra.Command{Use: "entitlements", Short: "Personal entitlements"}
var meEntitlementsGetCmd = &cobra.Command{Use: "get", Short: "List entitlements", RunE: runMeEntitlementsGet}

var parentCmd = &cobra.Command{Use: "parent", Short: "Parent/guardian persona"}

var parentChildrenCmd = &cobra.Command{Use: "children", Short: "Linked children"}
var parentChildrenListCmd = &cobra.Command{Use: "list", Short: "List linked children", RunE: runParentChildrenList}

var parentLinkFlags struct {
	org     string
	parent  string
	student string
}
var parentLinkCmd = &cobra.Command{Use: "link", Short: "Link parent to child (admin)", RunE: runParentLink}

var parentUnlinkFlags struct{ org string }
var parentUnlinkCmd = &cobra.Command{
	Use:   "unlink <link_id>",
	Short: "Remove a parent link (admin)",
	Args:  cobra.ExactArgs(1),
	RunE:  runParentUnlink,
}

var parentGradesFlags struct{ child string }
var parentGradesCmd = &cobra.Command{Use: "grades", Short: "View child grades", RunE: runParentGrades}

var parentAttendanceFlags struct{ child string }
var parentAttendanceCmd = &cobra.Command{Use: "attendance", Short: "View child attendance", RunE: runParentAttendance}

func runMeGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := fetchMeProfile(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeUpdate(cmd *cobra.Command, _ []string) error {
	meProfileFieldsSetFlags.file = meUpdateFlags.file
	return runMeProfileFieldsSet(cmd, nil)
}

func runMeProfileFieldsGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getMyProfileFields(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeProfileFieldsSet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(meProfileFieldsSetFlags.file)
	if err != nil {
		return err
	}
	raw, err := patchMeProfileFields(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeSessionsList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listMySessions(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tCURRENT\tCREATED\tAGENT")
	for _, s := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%v\t%s\t%s\n", s.ID, s.Current, s.CreatedAt, s.UserAgent)
	}
	return w.Flush()
}

func runMeSessionsRevoke(cmd *cobra.Command, _ []string) error {
	if !meSessionsRevokeFlags.yes {
		return fmt.Errorf("%s", sessionRevokeConfirmMessage(meSessionsRevokeFlags.all, meSessionsRevokeFlags.includeCurrent))
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	if meSessionsRevokeFlags.all {
		if meSessionsRevokeFlags.includeCurrent {
			sessions, _, err := listMySessions(c)
			if err != nil {
				return err
			}
			for _, id := range revokeAllSessionsExceptCurrent(sessions, true) {
				_ = revokeSession(c, id)
			}
		} else {
			if err := revokeOtherSessions(c); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("specify --all to revoke sessions, or revoke a single session via API")
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Sessions revoked.")
	return nil
}

func runMeMfaStatus(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listMyMFA(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tCREATED")
	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Type, r.CreatedAt)
	}
	return w.Flush()
}

func runMeMfaDisable(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := deleteMyMFA(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "MFA factor disabled.")
	return nil
}

func runMeOidcList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listOIDCIdentities(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tPROVIDER\tEMAIL")
	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Provider, r.Email)
	}
	return w.Flush()
}

func runMeOidcUnlink(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := unlinkOIDCIdentity(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Identity unlinked.")
	return nil
}

func runMeAccessKeysList(cmd *cobra.Command, _ []string) error {
	return runAccessKeysList(cmd, nil)
}

func runMeAccessKeysCreate(cmd *cobra.Command, _ []string) error {
	accessKeysCreateFlags.file = meAccessKeysCreateFlags.file
	accessKeysCreateFlags.secretOut = meAccessKeysCreateFlags.secretOut
	return runAccessKeysCreate(cmd, nil)
}

func runMeAccessKeysRevoke(cmd *cobra.Command, args []string) error {
	return runAccessKeysRevoke(cmd, args)
}

func runMeDemographicsGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getMyDemographics(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeDemographicsSet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(meDemographicsSetFlags.file)
	if err != nil {
		return err
	}
	raw, err := patchMyDemographics(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeConsentList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := listConsentStudies(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeConsentOptIn(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := respondConsentStudy(c, args[0], true)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeConsentOptOut(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := respondConsentStudy(c, args[0], false)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeOnboardingStatus(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getOnboardingStatus(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runMeEntitlementsGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getMyEntitlements(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runParentChildrenList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listParentChildren(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL")
	for _, ch := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", ch.ID, ch.DisplayName, ch.Email)
	}
	return w.Flush()
}

func runParentLink(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := parentLinkChild(c, parentLinkFlags.org, parentLinkFlags.parent, parentLinkFlags.student)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runParentUnlink(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := parentUnlinkChild(c, parentUnlinkFlags.org, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Parent link removed.")
	return nil
}

func runParentGrades(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	children, _, err := listParentChildren(c)
	if err != nil {
		return err
	}
	if err := validateParentChildID(children, parentGradesFlags.child); err != nil {
		return err
	}
	raw, err := getParentStudentGrades(c, parentGradesFlags.child)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runParentAttendance(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	children, _, err := listParentChildren(c)
	if err != nil {
		return err
	}
	if err := validateParentChildID(children, parentAttendanceFlags.child); err != nil {
		return err
	}
	raw, err := getParentStudentAttendance(c, parentAttendanceFlags.child)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func init() {
	meUpdateCmd.Flags().StringVar(&meUpdateFlags.file, "file", "", "profile fields JSON")
	_ = meUpdateCmd.MarkFlagRequired("file")
	meProfileFieldsSetCmd.Flags().StringVar(&meProfileFieldsSetFlags.file, "file", "", "fields JSON")
	_ = meProfileFieldsSetCmd.MarkFlagRequired("file")
	meSessionsRevokeCmd.Flags().BoolVar(&meSessionsRevokeFlags.all, "all", false, "revoke all other sessions")
	meSessionsRevokeCmd.Flags().BoolVar(&meSessionsRevokeFlags.includeCurrent, "include-current", false, "also revoke current session")
	meSessionsRevokeCmd.Flags().BoolVar(&meSessionsRevokeFlags.yes, "yes", false, "confirm revoke")

	meAccessKeysCreateCmd.Flags().StringVar(&meAccessKeysCreateFlags.file, "file", "", "access key JSON")
	_ = meAccessKeysCreateCmd.MarkFlagRequired("file")
	meAccessKeysCreateCmd.Flags().StringVar(&meAccessKeysCreateFlags.secretOut, "secret-out", "", "write one-time token to file")

	meDemographicsSetCmd.Flags().StringVar(&meDemographicsSetFlags.file, "file", "", "demographics JSON")
	_ = meDemographicsSetCmd.MarkFlagRequired("file")

	parentLinkCmd.Flags().StringVar(&parentLinkFlags.org, "org", "", "organization id")
	parentLinkCmd.Flags().StringVar(&parentLinkFlags.parent, "parent", "", "parent email")
	parentLinkCmd.Flags().StringVar(&parentLinkFlags.student, "student", "", "student email")
	_ = parentLinkCmd.MarkFlagRequired("org")
	_ = parentLinkCmd.MarkFlagRequired("parent")
	_ = parentLinkCmd.MarkFlagRequired("student")
	parentUnlinkCmd.Flags().StringVar(&parentUnlinkFlags.org, "org", "", "organization id")
	_ = parentUnlinkCmd.MarkFlagRequired("org")
	parentGradesCmd.Flags().StringVar(&parentGradesFlags.child, "child", "", "child user id")
	_ = parentGradesCmd.MarkFlagRequired("child")
	parentAttendanceCmd.Flags().StringVar(&parentAttendanceFlags.child, "child", "", "child user id")
	_ = parentAttendanceCmd.MarkFlagRequired("child")

	meProfileFieldsCmd.AddCommand(meProfileFieldsGetCmd, meProfileFieldsSetCmd)
	meSessionsCmd.AddCommand(meSessionsListCmd, meSessionsRevokeCmd)
	meMfaCmd.AddCommand(meMfaStatusCmd, meMfaDisableCmd)
	meOidcCmd.AddCommand(meOidcListCmd, meOidcUnlinkCmd)
	meAccessKeysCmd.AddCommand(meAccessKeysListCmd, meAccessKeysCreateCmd, meAccessKeysRevokeCmd)
	meDemographicsCmd.AddCommand(meDemographicsGetCmd, meDemographicsSetCmd)
	meConsentCmd.AddCommand(meConsentListCmd, meConsentOptInCmd, meConsentOptOutCmd)
	meOnboardingCmd.AddCommand(meOnboardingStatusCmd)
	meEntitlementsCmd.AddCommand(meEntitlementsGetCmd)

	meCmd.AddCommand(
		meGetCmd, meUpdateCmd, meProfileFieldsCmd, meSessionsCmd, meMfaCmd, meOidcCmd,
		meAccessKeysCmd, meDemographicsCmd, meConsentCmd, meOnboardingCmd, meEntitlementsCmd,
	)

	parentChildrenCmd.AddCommand(parentChildrenListCmd)
	parentCmd.AddCommand(parentChildrenCmd, parentLinkCmd, parentUnlinkCmd, parentGradesCmd, parentAttendanceCmd)

	rootCmd.AddCommand(meCmd, parentCmd)
}