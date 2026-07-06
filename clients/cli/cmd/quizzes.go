package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// quizQuestion mirrors coursemodulequiz.QuizQuestion for CLI patch payloads.
type quizQuestion struct {
	ID                 string          `json:"id"`
	Prompt             string          `json:"prompt"`
	QuestionType       string          `json:"questionType"`
	Choices            []string        `json:"choices,omitempty"`
	ChoiceIDs          []string        `json:"choiceIds,omitempty"`
	TypeConfig         json.RawMessage `json:"typeConfig,omitempty"`
	CorrectChoiceIndex *uint           `json:"correctChoiceIndex,omitempty"`
	MultipleAnswer     bool            `json:"multipleAnswer,omitempty"`
	Required           bool            `json:"required,omitempty"`
	Points             int32           `json:"points"`
}

// quizPublic is the GET/PATCH /quizzes/{id} response envelope.
type quizPublic struct {
	ItemID              string         `json:"itemId"`
	Title               string         `json:"title"`
	Markdown            string         `json:"markdown"`
	DueAt               *time.Time     `json:"dueAt"`
	AvailableFrom       *time.Time     `json:"availableFrom"`
	AvailableUntil      *time.Time     `json:"availableUntil"`
	UnlimitedAttempts   bool           `json:"unlimitedAttempts"`
	MaxAttempts         int32          `json:"maxAttempts"`
	GradeAttemptPolicy  string         `json:"gradeAttemptPolicy"`
	PointsWorth         *int32         `json:"pointsWorth"`
	TimeLimitMinutes    *int32         `json:"timeLimitMinutes"`
	ShuffleQuestions    bool           `json:"shuffleQuestions"`
	ShuffleChoices      bool           `json:"shuffleChoices"`
	Questions           []quizQuestion `json:"questions"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

var quizzesCmd = &cobra.Command{
	Use:   "quizzes",
	Short: "Manage course quizzes",
}

var quizzesListFlags struct {
	course string
	limit  int
	page   int
}

var quizzesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quizzes in a course",
	RunE:  runQuizzesList,
}

var quizzesGetFlags struct {
	course string
}

var quizzesGetCmd = &cobra.Command{
	Use:   "get <item_id>",
	Short: "Get quiz details",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesGet,
}

var quizzesCreateFlags struct {
	course string
	module string
	title  string
	points int
}

var quizzesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a quiz under a module",
	RunE:  runQuizzesCreate,
}

func init() {
	quizzesListCmd.Flags().StringVar(&quizzesListFlags.course, "course", "", "course code (required)")
	_ = quizzesListCmd.MarkFlagRequired("course")
	quizzesListCmd.Flags().IntVar(&quizzesListFlags.limit, "limit", 50, "maximum results per page")
	quizzesListCmd.Flags().IntVar(&quizzesListFlags.page, "page", 1, "page number (1-based)")

	quizzesGetCmd.Flags().StringVar(&quizzesGetFlags.course, "course", "", "course code (required)")
	_ = quizzesGetCmd.MarkFlagRequired("course")

	quizzesCreateCmd.Flags().StringVar(&quizzesCreateFlags.course, "course", "", "course code (required)")
	quizzesCreateCmd.Flags().StringVar(&quizzesCreateFlags.module, "module", "", "module UUID (required)")
	quizzesCreateCmd.Flags().StringVar(&quizzesCreateFlags.title, "title", "", "quiz title (required)")
	quizzesCreateCmd.Flags().IntVar(&quizzesCreateFlags.points, "points", -1, "point value")
	_ = quizzesCreateCmd.MarkFlagRequired("course")
	_ = quizzesCreateCmd.MarkFlagRequired("module")
	_ = quizzesCreateCmd.MarkFlagRequired("title")

	quizzesCmd.AddCommand(quizzesListCmd, quizzesGetCmd, quizzesCreateCmd)
	rootCmd.AddCommand(quizzesCmd)
}

func quizAPIPath(courseCode, suffix string) string {
	return "/api/v1/courses/" + courseCode + suffix
}

func filterQuizzes(items []structureItemPublic) []structureItemPublic {
	var out []structureItemPublic
	for _, it := range items {
		if it.Kind == "quiz" {
			out = append(out, it)
		}
	}
	return out
}

func fetchQuizRaw(c *client.Client, courseCode, itemID string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, quizAPIPath(courseCode, "/quizzes/"+itemID), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting quiz: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func patchQuiz(c *client.Client, courseCode, itemID string, patch map[string]any) ([]byte, error) {
	raw, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding patch: %w", err)
	}
	req, err := c.NewRequest(http.MethodPatch, quizAPIPath(courseCode, "/quizzes/"+itemID), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating quiz: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func runQuizzesList(cmd *cobra.Command, _ []string) error {
	body, _, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), quizzesListFlags.course)
	if err != nil {
		return err
	}
	quizzes := filterQuizzes(body.Items)

	limit := quizzesListFlags.limit
	page := quizzesListFlags.page
	if limit > 0 && page > 0 {
		start := (page - 1) * limit
		if start >= len(quizzes) {
			quizzes = []structureItemPublic{}
		} else {
			end := start + limit
			if end > len(quizzes) {
				end = len(quizzes)
			}
			quizzes = quizzes[start:end]
		}
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(quizzes)
	}

	if len(quizzes) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No quizzes.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tPOINTS\tPUBLISHED\tDUE")
	for _, q := range quizzes {
		points := "-"
		if q.PointsWorth != nil {
			points = fmt.Sprintf("%d", *q.PointsWorth)
		}
		due := "-"
		if q.DueAt != nil {
			due = q.DueAt.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", q.ID, q.Title, points, q.Published, due)
	}
	return w.Flush()
}

func runQuizzesGet(cmd *cobra.Command, args []string) error {
	raw, err := fetchQuizRaw(client.New(Cfg.Server, Cfg.APIKey), quizzesGetFlags.course, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	var q quizPublic
	if err := json.Unmarshal(raw, &q); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "ID:       %s\n", q.ItemID)
	_, _ = fmt.Fprintf(out, "Title:    %s\n", q.Title)
	if q.PointsWorth != nil {
		_, _ = fmt.Fprintf(out, "Points:   %d\n", *q.PointsWorth)
	}
	if q.TimeLimitMinutes != nil {
		_, _ = fmt.Fprintf(out, "Time limit (min): %d\n", *q.TimeLimitMinutes)
	}
	_, _ = fmt.Fprintf(out, "Attempts: max=%d unlimited=%v policy=%s\n", q.MaxAttempts, q.UnlimitedAttempts, q.GradeAttemptPolicy)
	_, _ = fmt.Fprintf(out, "Shuffle:  questions=%v choices=%v\n", q.ShuffleQuestions, q.ShuffleChoices)
	_, _ = fmt.Fprintf(out, "Questions: %d\n", len(q.Questions))
	if q.DueAt != nil {
		_, _ = fmt.Fprintf(out, "Due:      %s\n", q.DueAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintf(out, "Updated:  %s\n", q.UpdatedAt.Format(time.RFC3339))
	return nil
}

func runQuizzesCreate(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	item, err := addItemToModule(c, quizzesCreateFlags.course, quizzesCreateFlags.module, "quiz", quizzesCreateFlags.title, "")
	if err != nil {
		return err
	}
	if quizzesCreateFlags.points >= 0 {
		patch := map[string]any{"pointsWorth": quizzesCreateFlags.points}
		if _, err := patchQuiz(c, quizzesCreateFlags.course, item.ID, patch); err != nil {
			return err
		}
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"id":    item.ID,
			"title": item.Title,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created quiz %s\n", item.ID)
	return nil
}

func quizQuestionPreview(q quizQuestion) string {
	prompt := strings.TrimSpace(q.Prompt)
	if prompt == "" {
		prompt = q.ID
	}
	if len(prompt) > 60 {
		return prompt[:60] + "..."
	}
	return prompt
}