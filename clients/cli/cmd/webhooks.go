package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage outbound webhook subscriptions",
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook subscriptions",
	RunE:  runWebhooksList,
}

var webhooksCreateFlags struct {
	file      string
	label     string
	url       string
	events    []string
	secretOut string
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook subscription",
	RunE:  runWebhooksCreate,
}

var webhooksUpdateFlags struct {
	file   string
	label  string
	url    string
	events []string
	active bool
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a webhook subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksUpdate,
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a webhook subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksDelete,
}

var webhooksTestFlags struct {
	event string
}

var webhooksTestCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Send a signed test delivery",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksTest,
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries <id>",
	Short: "List recent delivery attempts",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksDeliveries,
}

var webhooksEventTypesCmd = &cobra.Command{
	Use:   "event-types",
	Short: "List supported webhook event types",
	RunE:  runWebhooksEventTypes,
}

func init() {
	webhooksCreateCmd.Flags().StringVar(&webhooksCreateFlags.file, "file", "", "webhook JSON (label, endpointUrl, eventTypes)")
	webhooksCreateCmd.Flags().StringVar(&webhooksCreateFlags.label, "label", "", "subscription label")
	webhooksCreateCmd.Flags().StringVar(&webhooksCreateFlags.url, "url", "", "endpoint URL")
	webhooksCreateCmd.Flags().StringSliceVar(&webhooksCreateFlags.events, "event", nil, "event type filter (repeatable or comma-separated)")
	webhooksCreateCmd.Flags().StringVar(&webhooksCreateFlags.secretOut, "secret-out", "", "write one-time signing key to a file")

	webhooksUpdateCmd.Flags().StringVar(&webhooksUpdateFlags.file, "file", "", "partial update JSON")
	webhooksUpdateCmd.Flags().StringVar(&webhooksUpdateFlags.label, "label", "", "new label")
	webhooksUpdateCmd.Flags().StringVar(&webhooksUpdateFlags.url, "url", "", "new endpoint URL")
	webhooksUpdateCmd.Flags().StringSliceVar(&webhooksUpdateFlags.events, "event", nil, "replace event types")
	webhooksUpdateCmd.Flags().BoolVar(&webhooksUpdateFlags.active, "active", true, "set active state (use --active=false to pause)")

	webhooksTestCmd.Flags().StringVar(&webhooksTestFlags.event, "event", "", "event type for test payload (default: grade.posted)")

	webhooksCmd.AddCommand(
		webhooksListCmd,
		webhooksCreateCmd,
		webhooksUpdateCmd,
		webhooksDeleteCmd,
		webhooksTestCmd,
		webhooksDeliveriesCmd,
		webhooksEventTypesCmd,
	)
	rootCmd.AddCommand(webhooksCmd)
}

func runWebhooksList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	subs, raw, err := fetchWebhooks(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tLABEL\tURL\tSTATUS\tEVENTS")
	for _, s := range subs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\n", s.ID, s.Label, s.EndpointURL, s.Status, s.EventTypes)
	}
	return w.Flush()
}

func runWebhooksCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	var payload map[string]any
	var err error
	if webhooksCreateFlags.file != "" {
		payload, err = loadJSONFile(webhooksCreateFlags.file)
		if err != nil {
			return err
		}
	} else {
		payload = map[string]any{}
		if webhooksCreateFlags.label != "" {
			payload["label"] = webhooksCreateFlags.label
		}
		if webhooksCreateFlags.url != "" {
			payload["endpointUrl"] = webhooksCreateFlags.url
		}
		if events := parseWebhookEventTypes(webhooksCreateFlags.events); len(events) > 0 {
			payload["eventTypes"] = events
		}
	}
	if _, ok := payload["label"]; !ok {
		return fmt.Errorf("label is required (use --label or --file)")
	}
	if _, ok := payload["endpointUrl"]; !ok {
		return fmt.Errorf("endpoint URL is required (use --url or --file)")
	}
	if _, ok := payload["eventTypes"]; !ok {
		return fmt.Errorf("at least one --event is required (or eventTypes in --file)")
	}
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), ferpaWebhookWarning)
	out, _, err := createWebhook(c, payload)
	if err != nil {
		return err
	}
	secret, _ := out["signingKey"].(string)
	if globalFlags.jsonOut {
		redactWebhookSigningKey(out)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	sub, _ := out["subscription"].(map[string]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created webhook %v\n", sub["id"])
	return writeOneTimeSecret(cmd, secret, webhooksCreateFlags.secretOut)
}

func runWebhooksUpdate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	var payload map[string]any
	var err error
	if webhooksUpdateFlags.file != "" {
		payload, err = loadJSONFile(webhooksUpdateFlags.file)
		if err != nil {
			return err
		}
	} else {
		payload = map[string]any{}
		if webhooksUpdateFlags.label != "" {
			payload["label"] = webhooksUpdateFlags.label
		}
		if webhooksUpdateFlags.url != "" {
			payload["endpointUrl"] = webhooksUpdateFlags.url
		}
		if events := parseWebhookEventTypes(webhooksUpdateFlags.events); len(events) > 0 {
			payload["eventTypes"] = events
		}
		if cmd.Flags().Changed("active") {
			payload["active"] = webhooksUpdateFlags.active
		}
	}
	if len(payload) == 0 {
		return fmt.Errorf("provide --file or at least one update flag")
	}
	body, err := updateWebhook(c, args[0], payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Webhook updated.")
	return nil
}

func runWebhooksDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := deleteWebhook(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Webhook deleted.")
	return nil
}

func runWebhooksTest(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := testWebhook(c, args[0], webhooksTestFlags.event)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Delivery webhookDelivery `json:"delivery"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Test delivery status=%s attempts=%d\n", out.Delivery.Status, out.Delivery.AttemptCount)
	return nil
}

func runWebhooksDeliveries(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := fetchWebhookDeliveries(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tEVENT\tSTATUS\tATTEMPTS\tHTTP")
	for _, d := range rows {
		httpStatus := ""
		if d.LastHTTPStatus != nil {
			httpStatus = fmt.Sprintf("%d", *d.LastHTTPStatus)
		}
		test := ""
		if d.Test {
			test = " (test)"
		}
		_, _ = fmt.Fprintf(w, "%d\t%s%s\t%s\t%d\t%s\n", d.ID, d.EventType, test, d.Status, d.AttemptCount, httpStatus)
	}
	return w.Flush()
}

func runWebhooksEventTypes(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchWebhookEventTypes(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}