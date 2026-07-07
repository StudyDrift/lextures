package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// --- messages ---

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Inbox and direct messages",
}

var messagesListFlags struct {
	folder string
	q      string
}

var messagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List inbox messages",
	RunE:  runMessagesList,
}

var messagesGetCmd = &cobra.Command{
	Use:   "get <message_id>",
	Short: "Get a message by id",
	Args:  cobra.ExactArgs(1),
	RunE:  runMessagesGet,
}

var messagesSendFlags struct {
	to      string
	subject string
	body    string
	file    string
}

var messagesSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a direct message",
	RunE:  runMessagesSend,
}

var messagesReplyFlags struct {
	subject string
	body    string
	file    string
}

var messagesReplyCmd = &cobra.Command{
	Use:   "reply <message_id>",
	Short: "Reply to a message (sends a new message to the original sender)",
	Args:  cobra.ExactArgs(1),
	RunE:  runMessagesReply,
}

// --- broadcasts ---

var broadcastsCmd = &cobra.Command{
	Use:   "broadcasts",
	Short: "Org-wide broadcasts and announcements",
}

var broadcastsListFlags struct {
	org   string
	limit int
}

var broadcastsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List org broadcasts",
	RunE:  runBroadcastsList,
}

var broadcastsSendFlags struct {
	org            string
	audience       string
	subject        string
	body           string
	file           string
	schedule       string
	typeArg        string
	yes            bool
	idempotencyKey string
}

var broadcastsSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an org-wide broadcast",
	RunE:  runBroadcastsSend,
}

var broadcastsStatusFlags struct {
	org string
}

var broadcastsStatusCmd = &cobra.Command{
	Use:   "status <broadcast_id>",
	Short: "Show broadcast delivery report",
	Args:  cobra.ExactArgs(1),
	RunE:  runBroadcastsStatus,
}

// --- announcements ---

var announcementsCmd = &cobra.Command{
	Use:   "announcements",
	Short: "Course announcements (feed bridge)",
}

var announcementsPostFlags struct {
	course  string
	channel string
	body    string
	file    string
}

var announcementsPostCmd = &cobra.Command{
	Use:   "post",
	Short: "Post a course announcement via the feed",
	RunE:  runAnnouncementsPost,
}

var announcementsListFlags struct {
	course  string
	channel string
	n       int
}

var announcementsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent announcements in a feed channel",
	RunE:  runAnnouncementsList,
}

// --- notifications ---

var notificationsCmd = &cobra.Command{
	Use:   "notifications",
	Short: "In-app notifications",
}

var notificationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notifications",
	RunE:  runNotificationsList,
}

var notificationsReadCmd = &cobra.Command{
	Use:   "read <notification_id>",
	Short: "Mark a notification as read",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotificationsRead,
}

var notificationsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Mark all notifications as read",
	RunE:  runNotificationsClear,
}

// --- notification-prefs ---

var notificationPrefsCmd = &cobra.Command{
	Use:   "notification-prefs",
	Short: "Notification delivery preferences",
}

var notificationPrefsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get notification preferences",
	RunE:  runNotificationPrefsGet,
}

var notificationPrefsSetFlags struct {
	file string
}

var notificationPrefsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update notification preferences from JSON",
	RunE:  runNotificationPrefsSet,
}

func init() {
	messagesListCmd.Flags().StringVar(&messagesListFlags.folder, "folder", "inbox", "folder: inbox, sent, starred, drafts, trash")
	messagesListCmd.Flags().StringVar(&messagesListFlags.q, "q", "", "search query")

	messagesSendCmd.Flags().StringVar(&messagesSendFlags.to, "to", "", "recipient email (required)")
	messagesSendCmd.Flags().StringVar(&messagesSendFlags.subject, "subject", "", "subject (required)")
	messagesSendCmd.Flags().StringVar(&messagesSendFlags.body, "body", "", "message body")
	messagesSendCmd.Flags().StringVar(&messagesSendFlags.file, "file", "", "message body file")

	messagesReplyCmd.Flags().StringVar(&messagesReplyFlags.subject, "subject", "", "reply subject")
	messagesReplyCmd.Flags().StringVar(&messagesReplyFlags.body, "body", "", "reply body")
	messagesReplyCmd.Flags().StringVar(&messagesReplyFlags.file, "file", "", "reply body file")

	broadcastsListCmd.Flags().StringVar(&broadcastsListFlags.org, "org", "", "org id (required)")
	broadcastsListCmd.Flags().IntVar(&broadcastsListFlags.limit, "limit", 100, "max rows")

	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.org, "org", "", "org id (required)")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.audience, "audience", "all", "audience: all, students, staff, or segment name")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.subject, "subject", "", "subject (required)")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.body, "body", "", "body text")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.file, "file", "", "body markdown file")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.schedule, "schedule", "", "RFC3339 schedule time")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.typeArg, "type", "announcement", "announcement or emergency")
	broadcastsSendCmd.Flags().BoolVar(&broadcastsSendFlags.yes, "yes", false, "confirm mass broadcast")
	broadcastsSendCmd.Flags().StringVar(&broadcastsSendFlags.idempotencyKey, "idempotency-key", "", "idempotency key to prevent duplicate sends")

	broadcastsStatusCmd.Flags().StringVar(&broadcastsStatusFlags.org, "org", "", "org id (required)")

	announcementsPostCmd.Flags().StringVar(&announcementsPostFlags.course, "course", "", "course code (required)")
	announcementsPostCmd.Flags().StringVar(&announcementsPostFlags.channel, "channel", "announcements", "feed channel name or id")
	announcementsPostCmd.Flags().StringVar(&announcementsPostFlags.body, "body", "", "announcement body")
	announcementsPostCmd.Flags().StringVar(&announcementsPostFlags.file, "file", "", "announcement body file")

	announcementsListCmd.Flags().StringVar(&announcementsListFlags.course, "course", "", "course code (required)")
	announcementsListCmd.Flags().StringVar(&announcementsListFlags.channel, "channel", "announcements", "feed channel")
	announcementsListCmd.Flags().IntVarP(&announcementsListFlags.n, "number", "n", 20, "recent count")

	notificationPrefsSetCmd.Flags().StringVar(&notificationPrefsSetFlags.file, "file", "", "preferences JSON file (required)")

	messagesCmd.AddCommand(messagesListCmd, messagesGetCmd, messagesSendCmd, messagesReplyCmd)
	broadcastsCmd.AddCommand(broadcastsListCmd, broadcastsSendCmd, broadcastsStatusCmd)
	announcementsCmd.AddCommand(announcementsPostCmd, announcementsListCmd)
	notificationsCmd.AddCommand(notificationsListCmd, notificationsReadCmd, notificationsClearCmd)
	notificationPrefsCmd.AddCommand(notificationPrefsGetCmd, notificationPrefsSetCmd)

	rootCmd.AddCommand(messagesCmd, broadcastsCmd, announcementsCmd, notificationsCmd, notificationPrefsCmd)
}

func runMessagesList(cmd *cobra.Command, _ []string) error {
	items, raw, err := listMailboxMessages(client.New(Cfg.Server, Cfg.APIKey), messagesListFlags.folder, messagesListFlags.q)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"messages": items})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tFROM\tSUBJECT\tCREATED")
	for _, m := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.ID, m.FromEmail, m.Subject, m.CreatedAt)
	}
	return w.Flush()
}

func runMessagesGet(cmd *cobra.Command, args []string) error {
	body, err := getMailboxMessage(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runMessagesSend(cmd *cobra.Command, _ []string) error {
	to := strings.TrimSpace(messagesSendFlags.to)
	subject := strings.TrimSpace(messagesSendFlags.subject)
	body := strings.TrimSpace(messagesSendFlags.body)
	if body == "" && messagesSendFlags.file != "" {
		text, err := readTextFile(messagesSendFlags.file)
		if err != nil {
			return err
		}
		body = text
	}
	if to == "" || subject == "" || body == "" {
		return fmt.Errorf("--to, --subject, and --body (or --file) are required")
	}
	resp, err := sendMailboxMessage(client.New(Cfg.Server, Cfg.APIKey), to, subject, body)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(resp)
	return err
}

func runMessagesReply(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	origRaw, err := getMailboxMessage(c, args[0])
	if err != nil {
		return err
	}
	var orig struct {
		FromEmail string `json:"fromEmail"`
		Subject   string `json:"subject"`
	}
	if err := json.Unmarshal(origRaw, &orig); err != nil {
		return err
	}
	subject := messagesReplyFlags.subject
	if subject == "" {
		subject = "Re: " + strings.TrimSpace(orig.Subject)
	}
	body := strings.TrimSpace(messagesReplyFlags.body)
	if body == "" && messagesReplyFlags.file != "" {
		body, err = readTextFile(messagesReplyFlags.file)
		if err != nil {
			return err
		}
	}
	if body == "" {
		return fmt.Errorf("--body or --file is required")
	}
	resp, err := sendMailboxMessage(c, orig.FromEmail, subject, body)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(resp)
	return err
}

func runBroadcastsList(cmd *cobra.Command, _ []string) error {
	org := strings.TrimSpace(broadcastsListFlags.org)
	if org == "" {
		return fmt.Errorf("--org is required")
	}
	items, raw, err := listOrgBroadcasts(client.New(Cfg.Server, Cfg.APIKey), org, broadcastsListFlags.limit)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"broadcasts": items})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tSUBJECT\tCREATED")
	for _, b := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", b.ID, b.Type, b.Status, b.Subject, b.CreatedAt)
	}
	return w.Flush()
}

func runBroadcastsSend(cmd *cobra.Command, _ []string) error {
	org := strings.TrimSpace(broadcastsSendFlags.org)
	if org == "" {
		return fmt.Errorf("--org is required")
	}
	if !broadcastsSendFlags.yes {
		return fmt.Errorf("%s\nRe-run with --yes to confirm.", broadcastWarning)
	}
	subject := strings.TrimSpace(broadcastsSendFlags.subject)
	body := strings.TrimSpace(broadcastsSendFlags.body)
	if body == "" && broadcastsSendFlags.file != "" {
		text, err := readTextFile(broadcastsSendFlags.file)
		if err != nil {
			return err
		}
		body = text
	}
	if subject == "" || body == "" {
		return fmt.Errorf("--subject and --body (or --file) are required")
	}
	scheduledAt, err := parseRFC3339Schedule(broadcastsSendFlags.schedule)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"type":     broadcastsSendFlags.typeArg,
		"subject":  subject,
		"body":     body,
		"audience": buildAudienceJSON(broadcastsSendFlags.audience),
	}
	if scheduledAt != nil {
		payload["scheduledAt"] = *scheduledAt
	}
	resp, err := sendOrgBroadcast(client.New(Cfg.Server, Cfg.APIKey), org, payload, broadcastsSendFlags.idempotencyKey)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(resp)
	return err
}

func runBroadcastsStatus(cmd *cobra.Command, args []string) error {
	org := strings.TrimSpace(broadcastsStatusFlags.org)
	if org == "" {
		return fmt.Errorf("--org is required")
	}
	body, err := fetchBroadcastDeliveryReport(client.New(Cfg.Server, Cfg.APIKey), org, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAnnouncementsPost(cmd *cobra.Command, _ []string) error {
	if announcementsPostFlags.course == "" {
		return fmt.Errorf("--course is required")
	}
	body := strings.TrimSpace(announcementsPostFlags.body)
	if body == "" && announcementsPostFlags.file != "" {
		text, err := readTextFile(announcementsPostFlags.file)
		if err != nil {
			return err
		}
		body = text
	}
	if body == "" {
		return fmt.Errorf("--body or --file is required")
	}
	feedFlags.course = announcementsPostFlags.course
	feedPostFlags.channel = announcementsPostFlags.channel
	feedPostFlags.body = body
	return runFeedPost(cmd, nil)
}

func runAnnouncementsList(cmd *cobra.Command, _ []string) error {
	if announcementsListFlags.course == "" {
		return fmt.Errorf("--course is required")
	}
	feedFlags.course = announcementsListFlags.course
	feedRecentFlags.channel = announcementsListFlags.channel
	feedRecentFlags.n = announcementsListFlags.n
	return runFeedRecent(cmd, nil)
}

func runNotificationsList(cmd *cobra.Command, _ []string) error {
	items, unread, raw, err := listNotifications(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"notifications": items,
			"unreadCount":   unread,
		})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Unread: %d\n", unread)
	_, _ = fmt.Fprintln(w, "ID\tEVENT\tTITLE\tREAD")
	for _, n := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", n.ID, n.EventType, n.Title, n.Read)
	}
	return w.Flush()
}

func runNotificationsRead(cmd *cobra.Command, args []string) error {
	if err := markNotificationRead(client.New(Cfg.Server, Cfg.APIKey), args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"read": args[0]})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Marked read.")
	return nil
}

func runNotificationsClear(cmd *cobra.Command, _ []string) error {
	if err := markAllNotificationsRead(client.New(Cfg.Server, Cfg.APIKey)); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"cleared": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "All notifications marked read.")
	return nil
}

func runNotificationPrefsGet(cmd *cobra.Command, _ []string) error {
	body, err := getNotificationPreferences(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runNotificationPrefsSet(cmd *cobra.Command, _ []string) error {
	if notificationPrefsSetFlags.file == "" {
		return fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(notificationPrefsSetFlags.file)
	if err != nil {
		return err
	}
	var payload struct {
		Preferences []map[string]any `json:"preferences"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	body, err := setNotificationPreferences(client.New(Cfg.Server, Cfg.APIKey), payload.Preferences)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}
