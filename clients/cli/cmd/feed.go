package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// feedChannel mirrors the server's feed channel JSON.
type feedChannel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
}

type feedChannelsBody struct {
	Channels []feedChannel `json:"channels"`
}

// feedMessage mirrors the fields of the server's feed message JSON we display.
type feedMessage struct {
	ID                string        `json:"id"`
	ChannelID         string        `json:"channelId"`
	AuthorEmail       string        `json:"authorEmail"`
	AuthorDisplayName *string       `json:"authorDisplayName"`
	Body              string        `json:"body"`
	PinnedAt          *time.Time    `json:"pinnedAt"`
	CreatedAt         time.Time     `json:"createdAt"`
	LikeCount         int64         `json:"likeCount"`
	Replies           []feedMessage `json:"replies"`
}

type feedMessagesBody struct {
	Messages []feedMessage `json:"messages"`
}

// feedFlags holds the shared --course flag for all feed subcommands.
var feedFlags struct {
	course string
}

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Manage course feed channels and messages",
	Long: `Manage course feed channels and post messages from the terminal.

Channels and posts are written through the same API the web app uses, so the
web feed updates in real time as you run these commands.`,
}

func feedCourse() (string, error) {
	cc := strings.TrimSpace(feedFlags.course)
	if cc == "" {
		return "", fmt.Errorf("--course is required")
	}
	return cc, nil
}

func feedBasePath(courseCode string) string {
	return "/api/v1/courses/" + url.PathEscape(courseCode) + "/feed"
}

// resolveChannelID accepts either a channel id or a channel name. A value that
// looks like a UUID is used as-is; otherwise the course's channels are fetched
// and matched by name (case-insensitive).
func resolveChannelID(c *client.Client, courseCode, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("channel is required")
	}
	if looksLikeUUID(ref) {
		return ref, nil
	}

	req, err := c.NewRequest(http.MethodGet, feedBasePath(courseCode)+"/channels", nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", fmt.Errorf("listing channels: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", apiError(resp, 2)
	}
	var body feedChannelsBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	for _, ch := range body.Channels {
		if strings.EqualFold(ch.Name, ref) {
			return ch.ID, nil
		}
	}
	return "", fmt.Errorf("no channel named %q in course %s", ref, courseCode)
}

// --- feed channels ---

var feedChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Manage feed channels",
}

var feedChannelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List channels in a course feed",
	RunE:  runFeedChannelsList,
}

func runFeedChannelsList(cmd *cobra.Command, args []string) error {
	courseCode, err := feedCourse()
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)

	req, err := c.NewRequest(http.MethodGet, feedBasePath(courseCode)+"/channels", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing channels: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	var body feedChannelsBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(body.Channels)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tORDER\tCREATED")
	for _, ch := range body.Channels {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			ch.ID, ch.Name, ch.SortOrder, ch.CreatedAt.Format(time.RFC3339))
	}
	return w.Flush()
}

var feedChannelsCreateFlags struct {
	name string
}

var feedChannelsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new feed channel",
	RunE:  runFeedChannelsCreate,
}

func runFeedChannelsCreate(cmd *cobra.Command, args []string) error {
	courseCode, err := feedCourse()
	if err != nil {
		return err
	}
	name := strings.TrimSpace(feedChannelsCreateFlags.name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)

	raw, _ := json.Marshal(map[string]string{"name": name})
	req, err := c.NewRequest(http.MethodPost, feedBasePath(courseCode)+"/channels", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating channel: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return apiError(resp, 2)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	var ch feedChannel
	if err := json.Unmarshal(respBody, &ch); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created channel %s (%s)\n", ch.Name, ch.ID)
	return nil
}

var feedChannelsUpdateFlags struct {
	name string
}

var feedChannelsUpdateCmd = &cobra.Command{
	Use:   "update <channel_id>",
	Short: "Rename a feed channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeedChannelsUpdate,
}

func runFeedChannelsUpdate(cmd *cobra.Command, args []string) error {
	courseCode, err := feedCourse()
	if err != nil {
		return err
	}
	name := strings.TrimSpace(feedChannelsUpdateFlags.name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	channelID, err := resolveChannelID(c, courseCode, args[0])
	if err != nil {
		return err
	}

	raw, _ := json.Marshal(map[string]string{"name": name})
	path := feedBasePath(courseCode) + "/channels/" + url.PathEscape(channelID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	var ch feedChannel
	if err := json.Unmarshal(respBody, &ch); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated channel %s (%s)\n", ch.Name, ch.ID)
	return nil
}

var feedChannelsDeleteFlags struct {
	force bool
}

// feedChannelsDeleteInput is used in tests to inject the confirmation reader.
var feedChannelsDeleteInput io.Reader

var feedChannelsDeleteCmd = &cobra.Command{
	Use:   "delete <channel_id>",
	Short: "Delete a feed channel and all its messages",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeedChannelsDelete,
}

func runFeedChannelsDelete(cmd *cobra.Command, args []string) error {
	courseCode, err := feedCourse()
	if err != nil {
		return err
	}
	channelID := args[0]

	if !feedChannelsDeleteFlags.force {
		in := feedChannelsDeleteInput
		if in == nil {
			in = os.Stdin
		}
		r := bufio.NewReader(in)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Delete channel %q and all its messages? (y/N) ", channelID)
		line, _ := r.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	resolvedID, err := resolveChannelID(c, courseCode, channelID)
	if err != nil {
		return err
	}
	path := feedBasePath(courseCode) + "/channels/" + url.PathEscape(resolvedID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting channel: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiError(resp, 2)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": channelID})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted channel %s\n", channelID)
	return nil
}

// --- feed post ---

var feedPostFlags struct {
	channel string
	body    string
}

var feedPostCmd = &cobra.Command{
	Use:   "post",
	Short: "Post a message to a feed channel",
	RunE:  runFeedPost,
}

func runFeedPost(cmd *cobra.Command, args []string) error {
	courseCode, err := feedCourse()
	if err != nil {
		return err
	}
	channel := strings.TrimSpace(feedPostFlags.channel)
	if channel == "" {
		return fmt.Errorf("--channel is required")
	}
	body := strings.TrimSpace(feedPostFlags.body)
	if body == "" {
		return fmt.Errorf("--body is required")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	channelID, err := resolveChannelID(c, courseCode, channel)
	if err != nil {
		return err
	}

	raw, _ := json.Marshal(map[string]string{"body": body})
	path := feedBasePath(courseCode) + "/channels/" + url.PathEscape(channelID) + "/messages"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("posting message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return apiError(resp, 2)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(respBody, &out)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Posted message %s\n", out.ID)
	return nil
}

// --- feed recent ---

var feedRecentFlags struct {
	channel string
	n       int
}

var feedRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List the most recent messages in a feed channel",
	RunE:  runFeedRecent,
}

func runFeedRecent(cmd *cobra.Command, args []string) error {
	courseCode, err := feedCourse()
	if err != nil {
		return err
	}
	channel := strings.TrimSpace(feedRecentFlags.channel)
	if channel == "" {
		return fmt.Errorf("--channel is required")
	}
	if feedRecentFlags.n <= 0 {
		return fmt.Errorf("-n must be greater than 0")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	channelID, err := resolveChannelID(c, courseCode, channel)
	if err != nil {
		return err
	}

	path := feedBasePath(courseCode) + "/channels/" + url.PathEscape(channelID) +
		"/messages?limit=" + strconv.Itoa(feedRecentFlags.n)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing messages: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}

	var body feedMessagesBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	// The API returns oldest→newest roots; show the newest n.
	msgs := body.Messages
	if len(msgs) > feedRecentFlags.n {
		msgs = msgs[len(msgs)-feedRecentFlags.n:]
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(msgs)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CREATED\tAUTHOR\tLIKES\tBODY")
	for _, m := range msgs {
		author := m.AuthorEmail
		if m.AuthorDisplayName != nil && strings.TrimSpace(*m.AuthorDisplayName) != "" {
			author = *m.AuthorDisplayName
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			m.CreatedAt.Format(time.RFC3339), author, m.LikeCount, oneLine(m.Body))
	}
	return w.Flush()
}

// oneLine collapses newlines so a multi-line body stays on one table row.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func init() {
	feedCmd.PersistentFlags().StringVar(&feedFlags.course, "course", "", "course code (required)")

	feedChannelsCreateCmd.Flags().StringVar(&feedChannelsCreateFlags.name, "name", "", "channel name (required)")
	feedChannelsUpdateCmd.Flags().StringVar(&feedChannelsUpdateFlags.name, "name", "", "new channel name (required)")
	feedChannelsDeleteCmd.Flags().BoolVar(&feedChannelsDeleteFlags.force, "force", false, "skip confirmation prompt")

	feedChannelsCmd.AddCommand(feedChannelsListCmd, feedChannelsCreateCmd, feedChannelsUpdateCmd, feedChannelsDeleteCmd)

	feedPostCmd.Flags().StringVar(&feedPostFlags.channel, "channel", "", "channel id (required)")
	feedPostCmd.Flags().StringVar(&feedPostFlags.body, "body", "", "message body (required)")

	feedRecentCmd.Flags().StringVar(&feedRecentFlags.channel, "channel", "", "channel id (required)")
	feedRecentCmd.Flags().IntVarP(&feedRecentFlags.n, "number", "n", 20, "number of recent messages to show")

	feedCmd.AddCommand(feedChannelsCmd, feedPostCmd, feedRecentCmd)
	rootCmd.AddCommand(feedCmd)
}
