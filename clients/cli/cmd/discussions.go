package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var discussionsCmd = &cobra.Command{
	Use:   "discussions",
	Short: "List, export, and moderate course discussions",
}

var discussionsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List discussion forums in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runDiscussionsList,
}

var discussionsExportFlags struct {
	yes  bool
	file string
}

var discussionsExportCmd = &cobra.Command{
	Use:   "export <course>",
	Short: "Export forums, threads, and posts for archival",
	Args:  cobra.ExactArgs(1),
	RunE:  runDiscussionsExport,
}

var discussionsPostFlags struct {
	forum  string
	title  string
	body   string
}

var discussionsPostCmd = &cobra.Command{
	Use:   "post <course>",
	Short: "Create a discussion thread in a forum",
	Args:  cobra.ExactArgs(1),
	RunE:  runDiscussionsPost,
}

var discussionsLockFlags struct {
	lock bool
}

var discussionsLockCmd = &cobra.Command{
	Use:   "lock <course> <thread>",
	Short: "Lock or unlock a discussion thread",
	Args:  cobra.ExactArgs(2),
	RunE:  runDiscussionsLock,
}

var discussionsExportInput io.Reader

func init() {
	discussionsExportCmd.Flags().BoolVar(&discussionsExportFlags.yes, "yes", false,
		"confirm export of student-authored discussion content (FERPA)")
	discussionsExportCmd.Flags().StringVar(&discussionsExportFlags.file, "file", "",
		"write export JSON to file (default: stdout)")

	discussionsPostCmd.Flags().StringVar(&discussionsPostFlags.forum, "forum", "", "forum UUID (required)")
	_ = discussionsPostCmd.MarkFlagRequired("forum")
	discussionsPostCmd.Flags().StringVar(&discussionsPostFlags.title, "title", "", "thread title (required)")
	_ = discussionsPostCmd.MarkFlagRequired("title")
	discussionsPostCmd.Flags().StringVar(&discussionsPostFlags.body, "body", "", "opening post body (required)")
	_ = discussionsPostCmd.MarkFlagRequired("body")

	discussionsLockCmd.Flags().BoolVar(&discussionsLockFlags.lock, "lock", true, "lock the thread (use --lock=false to unlock)")

	discussionsCmd.AddCommand(
		discussionsListCmd,
		discussionsExportCmd,
		discussionsPostCmd,
		discussionsLockCmd,
	)
	rootCmd.AddCommand(discussionsCmd)
}

func fetchDiscussionForums(c *client.Client, course string) (map[string]any, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+course+"/forums", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing forums: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func fetchDiscussionThreads(c *client.Client, course, forumID string) (map[string]any, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/forums/%s/threads", course, forumID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func fetchDiscussionPosts(c *client.Client, course, threadID string) (map[string]any, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/discussion-threads/%s/posts", course, threadID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing posts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func confirmDiscussionExport(cmd *cobra.Command) error {
	if discussionsExportFlags.yes {
		return nil
	}
	in := discussionsExportInput
	if in == nil {
		in = os.Stdin
	}
	r := bufio.NewReader(in)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"Discussion export may contain student-authored content protected by FERPA. Export anyway? (y/N) ")
	line, _ := r.ReadString('\n')
	if !strings.EqualFold(strings.TrimSpace(line), "y") {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return fmt.Errorf("export aborted")
	}
	return nil
}

func runDiscussionsList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchDiscussionForums(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	forums, _ := body["forums"].([]any)
	if len(forums) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No forums.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tPOSITION")
	for _, item := range forums {
		m, _ := item.(map[string]any)
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\n", m["id"], m["name"], m["position"])
	}
	return w.Flush()
}

func runDiscussionsExport(cmd *cobra.Command, args []string) error {
	if err := confirmDiscussionExport(cmd); err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	forumsBody, _, err := fetchDiscussionForums(c, course)
	if err != nil {
		return err
	}
	export := map[string]any{
		"courseCode": course,
		"forums":     forumsBody["forums"],
		"threads":    map[string]any{},
		"posts":      map[string]any{},
	}
	threadsMap := export["threads"].(map[string]any)
	postsMap := export["posts"].(map[string]any)

	forums, _ := forumsBody["forums"].([]any)
	for _, item := range forums {
		forum, _ := item.(map[string]any)
		forumID, _ := forum["id"].(string)
		if forumID == "" {
			continue
		}
		threads, err := fetchDiscussionThreads(c, course, forumID)
		if err != nil {
			return fmt.Errorf("forum %s: %w", forumID, err)
		}
		threadsMap[forumID] = threads
		threadItems, _ := threads["threads"].([]any)
		for _, tItem := range threadItems {
			thread, _ := tItem.(map[string]any)
			threadID, _ := thread["id"].(string)
			if threadID == "" {
				continue
			}
			posts, err := fetchDiscussionPosts(c, course, threadID)
			if err != nil {
				return fmt.Errorf("thread %s: %w", threadID, err)
			}
			postsMap[threadID] = posts
		}
	}

	raw, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding export: %w", err)
	}
	if discussionsExportFlags.file != "" {
		if err := os.WriteFile(discussionsExportFlags.file, raw, 0o600); err != nil {
			return fmt.Errorf("writing export file: %w", err)
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"path":   discussionsExportFlags.file,
				"forums": len(forums),
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported discussions to %s\n", discussionsExportFlags.file)
		return nil
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d forums\n", len(forums))
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runDiscussionsPost(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	payload := map[string]any{
		"title": discussionsPostFlags.title,
		"body":  json.RawMessage(mustJSONString(discussionsPostFlags.body)),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/forums/%s/threads", course, discussionsPostFlags.forum)
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating thread: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return apiErrorBody(resp.StatusCode, body)
	}
	return emitRawOrMessage(cmd, body, "Created discussion thread")
}

func runDiscussionsLock(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	threadID := args[1]
	payload := map[string]any{"isLocked": discussionsLockFlags.lock}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/discussion-threads/%s", course, threadID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating thread: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	msg := "Locked thread"
	if !discussionsLockFlags.lock {
		msg = "Unlocked thread"
	}
	return emitRawOrMessage(cmd, body, msg)
}

func mustJSONString(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}