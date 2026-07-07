package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var emailTemplatesCmd = &cobra.Command{
	Use:   "email-templates",
	Short: "Manage transactional email templates",
}

var emailTemplatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List email template slots",
	RunE:  runEmailTemplatesList,
}

var emailTemplatesGetFlags struct {
	locale string
}

var emailTemplatesGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get an email template",
	Args:  cobra.ExactArgs(1),
	RunE:  runEmailTemplatesGet,
}

var emailTemplatesSetFlags struct {
	file   string
	locale string
}

var emailTemplatesSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set an email template from a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runEmailTemplatesSet,
}

var emailTemplatesPreviewFlags struct {
	file   string
	locale string
}

var emailTemplatesPreviewCmd = &cobra.Command{
	Use:   "preview <key>",
	Short: "Preview rendered template HTML",
	Args:  cobra.ExactArgs(1),
	RunE:  runEmailTemplatesPreview,
}

var emailTemplatesTestSendFlags struct {
	to     string
	locale string
}

var emailTemplatesTestSendCmd = &cobra.Command{
	Use:   "test-send <key>",
	Short: "Send a test email for a template slot",
	Args:  cobra.ExactArgs(1),
	RunE:  runEmailTemplatesTestSend,
}

func init() {
	emailTemplatesGetCmd.Flags().StringVar(&emailTemplatesGetFlags.locale, "locale", "", "locale variant (uses slot key suffix, e.g. welcome.es)")
	emailTemplatesSetCmd.Flags().StringVar(&emailTemplatesSetFlags.file, "file", "", "HTML or text template file (required)")
	emailTemplatesSetCmd.Flags().StringVar(&emailTemplatesSetFlags.locale, "locale", "", "locale variant (uses slot key suffix)")
	_ = emailTemplatesSetCmd.MarkFlagRequired("file")
	emailTemplatesPreviewCmd.Flags().StringVar(&emailTemplatesPreviewFlags.file, "file", "", "optional HTML file to preview before saving")
	emailTemplatesPreviewCmd.Flags().StringVar(&emailTemplatesPreviewFlags.locale, "locale", "", "locale variant")
	emailTemplatesTestSendCmd.Flags().StringVar(&emailTemplatesTestSendFlags.to, "to", "", "recipient email (server sends to your account when omitted)")
	emailTemplatesTestSendCmd.Flags().StringVar(&emailTemplatesTestSendFlags.locale, "locale", "", "locale variant")

	emailTemplatesCmd.AddCommand(
		emailTemplatesListCmd,
		emailTemplatesGetCmd,
		emailTemplatesSetCmd,
		emailTemplatesPreviewCmd,
		emailTemplatesTestSendCmd,
	)
	rootCmd.AddCommand(emailTemplatesCmd)
}

func runEmailTemplatesList(cmd *cobra.Command, _ []string) error {
	slots, raw, err := fetchEmailTemplates(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tDESCRIPTION\tCUSTOM\tUPDATED")
	for _, s := range slots {
		custom := "no"
		if s.HasCustom {
			custom = "yes"
		}
		updated := ""
		if s.UpdatedAt != nil {
			updated = *s.UpdatedAt
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Description, custom, updated)
	}
	return w.Flush()
}

func runEmailTemplatesGet(cmd *cobra.Command, args []string) error {
	slotID := resolveEmailTemplateSlot(args[0], emailTemplatesGetFlags.locale)
	tmpl, raw, err := fetchEmailTemplate(client.New(Cfg.Server, Cfg.APIKey), slotID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s — %s\n", tmpl.ID, tmpl.Description)
	if tmpl.Active != nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), tmpl.Active.HTMLBody)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), tmpl.DefaultHTML)
	}
	return nil
}

func runEmailTemplatesSet(cmd *cobra.Command, args []string) error {
	slotID := resolveEmailTemplateSlot(args[0], emailTemplatesSetFlags.locale)
	html, text, err := readTemplateFile(emailTemplatesSetFlags.file)
	if err != nil {
		return err
	}
	if html == "" && text == nil {
		return fmt.Errorf("template file is empty")
	}
	if html == "" && text != nil {
		html = "<pre>{{text}}</pre>"
	}
	body, err := putEmailTemplate(client.New(Cfg.Server, Cfg.APIKey), slotID, html, text)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Template %q saved.\n", slotID)
	return nil
}

func runEmailTemplatesPreview(cmd *cobra.Command, args []string) error {
	slotID := resolveEmailTemplateSlot(args[0], emailTemplatesPreviewFlags.locale)
	var html string
	var text *string
	if emailTemplatesPreviewFlags.file != "" {
		var err error
		html, text, err = readTemplateFile(emailTemplatesPreviewFlags.file)
		if err != nil {
			return err
		}
	}
	body, err := previewEmailTemplate(client.New(Cfg.Server, Cfg.APIKey), slotID, html, text)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		HTML string `json:"html"`
		Text string `json:"text"`
	}
	if json.Unmarshal(body, &out) == nil {
		_, err = os.Stdout.WriteString(out.HTML)
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runEmailTemplatesTestSend(cmd *cobra.Command, args []string) error {
	if emailTemplatesTestSendFlags.to != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: server sends test email to your account (--to %q ignored)\n", emailTemplatesTestSendFlags.to)
	}
	slotID := resolveEmailTemplateSlot(args[0], emailTemplatesTestSendFlags.locale)
	if err := testSendEmailTemplate(client.New(Cfg.Server, Cfg.APIKey), slotID); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"status": "accepted"})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Test email queued for template %q.\n", slotID)
	return nil
}