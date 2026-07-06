package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
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

type quizAttemptSummary struct {
	ID                 string   `json:"id"`
	StudentUserID      *string  `json:"studentUserId"`
	StudentName        *string  `json:"studentName"`
	AttemptNumber      int32    `json:"attemptNumber"`
	SubmittedAt        *string  `json:"submittedAt"`
	ScorePercent       *float64 `json:"scorePercent"`
	PointsEarned       *float64 `json:"pointsEarned"`
	PointsPossible     *float64 `json:"pointsPossible"`
	NeedsManualGrading bool     `json:"needsManualGrading"`
}

type quizAttemptsListBody struct {
	Attempts     []quizAttemptSummary `json:"attempts"`
	RetakePolicy string               `json:"retakePolicy"`
}

type quizGradingQuestion struct {
	QuestionIndex  int32            `json:"questionIndex"`
	QuestionType   string           `json:"questionType"`
	IsCorrect      *bool            `json:"isCorrect"`
	PointsAwarded  *float64         `json:"pointsAwarded"`
	MaxPoints      float64          `json:"maxPoints"`
	NeedsGrading   bool             `json:"needsGrading"`
	ResponseJSON   json.RawMessage  `json:"responseJson"`
}

type quizAttemptGradingBody struct {
	AttemptID          string                `json:"attemptId"`
	StudentUserID      string                `json:"studentUserId"`
	Questions          []quizGradingQuestion `json:"questions"`
	NeedsManualGrading bool                  `json:"needsManualGrading"`
}

type quizCodeRunResult struct {
	Status         string  `json:"status"`
	Passed         bool    `json:"passed"`
	ActualOutput   string  `json:"actualOutput"`
	ExpectedOutput string  `json:"expectedOutput"`
	PointsEarned   float64 `json:"pointsEarned"`
	PointsPossible float64 `json:"pointsPossible"`
}

var quizzesUpdateFlags struct {
	course   string
	title    string
	points   int
	markdown string
	file     string
}

var quizzesDeleteFlags struct {
	course string
}

var quizzesPublishFlags struct {
	course string
}

var quizzesSettingsSetFlags struct {
	course          string
	timeLimit       int
	maxAttempts     int
	unlimited       bool
	shuffleQuestions bool
	shuffleChoices  bool
	availableFrom   string
	availableUntil  string
	policy          string
}

var quizzesQuestionsAddFlags struct {
	course  string
	bank    string
	count   int
	ids     []string
	content string
}

var quizzesQuestionsRemoveFlags struct {
	course string
	id     string
	index  int
}

var quizzesQuestionsListFlags struct {
	course string
}

var quizzesQuestionsReorderFlags struct {
	course string
	order  string
}

var quizzesAttemptsListFlags struct {
	course string
	user   string
	limit  int
	page   int
	yes    bool
}

var quizzesAttemptsGetFlags struct {
	course string
	user   string
}

var quizzesGradeFlags struct {
	course  string
	attempt string
	user    string
	all     bool
}

var quizzesGradeSyncFlags struct {
	course string
	user   string
}

var quizzesCodeRunFlags struct {
	course   string
	attempt  string
	question string
	code     string
	file     string
}

var quizzesUpdateCmd = &cobra.Command{
	Use:   "update <item_id>",
	Short: "Update a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesUpdate,
}

var quizzesDeleteCmd = &cobra.Command{
	Use:   "delete <item_id>",
	Short: "Delete (archive) a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesDelete,
}

var quizzesPublishCmd = &cobra.Command{
	Use:   "publish <item_id>",
	Short: "Publish a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesPublish,
}

var quizzesUnpublishCmd = &cobra.Command{
	Use:   "unpublish <item_id>",
	Short: "Unpublish a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesUnpublish,
}

var quizzesSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage quiz delivery settings",
}

var quizzesSettingsSetCmd = &cobra.Command{
	Use:   "set <item_id>",
	Short: "Set quiz settings (time limit, attempts, shuffle, availability)",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesSettingsSet,
}

var quizzesQuestionsCmd = &cobra.Command{
	Use:   "questions",
	Short: "Add, remove, list, or reorder quiz questions",
}

var quizzesQuestionsAddCmd = &cobra.Command{
	Use:   "add <item_id>",
	Short: "Add questions from a bank or inline JSON",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesQuestionsAdd,
}

var quizzesQuestionsRemoveCmd = &cobra.Command{
	Use:   "remove <item_id>",
	Short: "Remove a question from a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesQuestionsRemove,
}

var quizzesQuestionsListCmd = &cobra.Command{
	Use:   "list <item_id>",
	Short: "List questions on a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesQuestionsList,
}

var quizzesQuestionsReorderCmd = &cobra.Command{
	Use:   "reorder <item_id>",
	Short: "Reorder quiz questions",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesQuestionsReorder,
}

var quizzesAttemptsCmd = &cobra.Command{
	Use:   "attempts",
	Short: "List or inspect quiz attempts",
}

var quizzesAttemptsListCmd = &cobra.Command{
	Use:   "list <item_id>",
	Short: "List submitted attempts for a quiz",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesAttemptsList,
}

var quizzesAttemptsGetCmd = &cobra.Command{
	Use:   "get <item_id>",
	Short: "Get one student's attempts",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesAttemptsGet,
}

var quizzesGradeCmd = &cobra.Command{
	Use:   "grade <item_id>",
	Short: "Trigger grading / regrade for submitted attempts",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesGrade,
}

var quizzesGradeSyncCmd = &cobra.Command{
	Use:   "grade-sync <item_id>",
	Short: "Push quiz scores to the gradebook",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesGradeSync,
}

var quizzesCodeRunCmd = &cobra.Command{
	Use:   "code-run <item_id>",
	Short: "Run autograder diagnostics for a code question on an attempt",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuizzesCodeRun,
}

func init() {
	quizzesUpdateCmd.Flags().StringVar(&quizzesUpdateFlags.course, "course", "", "course code (required)")
	quizzesUpdateCmd.Flags().StringVar(&quizzesUpdateFlags.title, "title", "", "quiz title")
	quizzesUpdateCmd.Flags().IntVar(&quizzesUpdateFlags.points, "points", -1, "point value")
	quizzesUpdateCmd.Flags().StringVar(&quizzesUpdateFlags.markdown, "markdown", "", "quiz instructions markdown")
	quizzesUpdateCmd.Flags().StringVar(&quizzesUpdateFlags.file, "file", "", "markdown body file (use - for stdin)")
	_ = quizzesUpdateCmd.MarkFlagRequired("course")

	quizzesDeleteCmd.Flags().StringVar(&quizzesDeleteFlags.course, "course", "", "course code (required)")
	_ = quizzesDeleteCmd.MarkFlagRequired("course")

	quizzesPublishCmd.Flags().StringVar(&quizzesPublishFlags.course, "course", "", "course code (required)")
	_ = quizzesPublishCmd.MarkFlagRequired("course")

	quizzesUnpublishCmd.Flags().StringVar(&quizzesPublishFlags.course, "course", "", "course code (required)")
	_ = quizzesUnpublishCmd.MarkFlagRequired("course")

	quizzesSettingsSetCmd.Flags().StringVar(&quizzesSettingsSetFlags.course, "course", "", "course code (required)")
	quizzesSettingsSetCmd.Flags().IntVar(&quizzesSettingsSetFlags.timeLimit, "time-limit", -1, "time limit in minutes (0 clears)")
	quizzesSettingsSetCmd.Flags().IntVar(&quizzesSettingsSetFlags.maxAttempts, "max-attempts", -1, "maximum attempts")
	quizzesSettingsSetCmd.Flags().BoolVar(&quizzesSettingsSetFlags.unlimited, "unlimited-attempts", false, "allow unlimited attempts")
	quizzesSettingsSetCmd.Flags().BoolVar(&quizzesSettingsSetFlags.shuffleQuestions, "shuffle-questions", false, "shuffle question order")
	quizzesSettingsSetCmd.Flags().BoolVar(&quizzesSettingsSetFlags.shuffleChoices, "shuffle-choices", false, "shuffle answer choices")
	quizzesSettingsSetCmd.Flags().StringVar(&quizzesSettingsSetFlags.availableFrom, "available-from", "", "availability start (RFC3339)")
	quizzesSettingsSetCmd.Flags().StringVar(&quizzesSettingsSetFlags.availableUntil, "available-until", "", "availability end (RFC3339)")
	quizzesSettingsSetCmd.Flags().StringVar(&quizzesSettingsSetFlags.policy, "grade-policy", "", "grade policy: highest, latest, first, average")
	_ = quizzesSettingsSetCmd.MarkFlagRequired("course")

	quizzesQuestionsAddCmd.Flags().StringVar(&quizzesQuestionsAddFlags.course, "course", "", "course code (required)")
	quizzesQuestionsAddCmd.Flags().StringVar(&quizzesQuestionsAddFlags.bank, "bank", "", "question bank ID to sample from")
	quizzesQuestionsAddCmd.Flags().IntVar(&quizzesQuestionsAddFlags.count, "count", 0, "number of random questions from --bank")
	quizzesQuestionsAddCmd.Flags().StringSliceVar(&quizzesQuestionsAddFlags.ids, "id", nil, "specific bank question id (repeatable)")
	quizzesQuestionsAddCmd.Flags().StringVar(&quizzesQuestionsAddFlags.content, "content", "", "inline question JSON or @file")
	_ = quizzesQuestionsAddCmd.MarkFlagRequired("course")

	quizzesQuestionsRemoveCmd.Flags().StringVar(&quizzesQuestionsRemoveFlags.course, "course", "", "course code (required)")
	quizzesQuestionsRemoveCmd.Flags().StringVar(&quizzesQuestionsRemoveFlags.id, "id", "", "editor question id to remove")
	quizzesQuestionsRemoveCmd.Flags().IntVar(&quizzesQuestionsRemoveFlags.index, "index", -1, "zero-based question index to remove")
	_ = quizzesQuestionsRemoveCmd.MarkFlagRequired("course")

	quizzesQuestionsListCmd.Flags().StringVar(&quizzesQuestionsListFlags.course, "course", "", "course code (required)")
	_ = quizzesQuestionsListCmd.MarkFlagRequired("course")

	quizzesQuestionsReorderCmd.Flags().StringVar(&quizzesQuestionsReorderFlags.course, "course", "", "course code (required)")
	quizzesQuestionsReorderCmd.Flags().StringVar(&quizzesQuestionsReorderFlags.order, "order", "", "comma-separated question ids in desired order (required)")
	_ = quizzesQuestionsReorderCmd.MarkFlagRequired("course")
	_ = quizzesQuestionsReorderCmd.MarkFlagRequired("order")

	quizzesAttemptsListCmd.Flags().StringVar(&quizzesAttemptsListFlags.course, "course", "", "course code (required)")
	quizzesAttemptsListCmd.Flags().StringVar(&quizzesAttemptsListFlags.user, "user", "", "filter to one student user UUID")
	quizzesAttemptsListCmd.Flags().IntVar(&quizzesAttemptsListFlags.limit, "limit", 50, "maximum results per page")
	quizzesAttemptsListCmd.Flags().IntVar(&quizzesAttemptsListFlags.page, "page", 1, "page number (1-based)")
	quizzesAttemptsListCmd.Flags().BoolVar(&quizzesAttemptsListFlags.yes, "yes", false, "confirm FERPA-covered bulk attempt export")
	_ = quizzesAttemptsListCmd.MarkFlagRequired("course")

	quizzesAttemptsGetCmd.Flags().StringVar(&quizzesAttemptsGetFlags.course, "course", "", "course code (required)")
	quizzesAttemptsGetCmd.Flags().StringVar(&quizzesAttemptsGetFlags.user, "user", "", "student user UUID (required)")
	_ = quizzesAttemptsGetCmd.MarkFlagRequired("course")
	_ = quizzesAttemptsGetCmd.MarkFlagRequired("user")

	quizzesGradeCmd.Flags().StringVar(&quizzesGradeFlags.course, "course", "", "course code (required)")
	quizzesGradeCmd.Flags().StringVar(&quizzesGradeFlags.attempt, "attempt", "", "grade one attempt UUID")
	quizzesGradeCmd.Flags().StringVar(&quizzesGradeFlags.user, "user", "", "grade all attempts for one student")
	quizzesGradeCmd.Flags().BoolVar(&quizzesGradeFlags.all, "all", false, "grade all submitted attempts")
	_ = quizzesGradeCmd.MarkFlagRequired("course")

	quizzesGradeSyncCmd.Flags().StringVar(&quizzesGradeSyncFlags.course, "course", "", "course code (required)")
	quizzesGradeSyncCmd.Flags().StringVar(&quizzesGradeSyncFlags.user, "user", "", "sync one student only")
	_ = quizzesGradeSyncCmd.MarkFlagRequired("course")

	quizzesCodeRunCmd.Flags().StringVar(&quizzesCodeRunFlags.course, "course", "", "course code (required)")
	quizzesCodeRunCmd.Flags().StringVar(&quizzesCodeRunFlags.attempt, "attempt", "", "attempt UUID (required)")
	quizzesCodeRunCmd.Flags().StringVar(&quizzesCodeRunFlags.question, "question", "", "question id (required)")
	quizzesCodeRunCmd.Flags().StringVar(&quizzesCodeRunFlags.code, "code", "", "source code to run")
	quizzesCodeRunCmd.Flags().StringVar(&quizzesCodeRunFlags.file, "file", "", "source file (use - for stdin)")
	_ = quizzesCodeRunCmd.MarkFlagRequired("course")
	_ = quizzesCodeRunCmd.MarkFlagRequired("attempt")
	_ = quizzesCodeRunCmd.MarkFlagRequired("question")

	quizzesSettingsCmd.AddCommand(quizzesSettingsSetCmd)
	quizzesQuestionsCmd.AddCommand(
		quizzesQuestionsAddCmd,
		quizzesQuestionsRemoveCmd,
		quizzesQuestionsListCmd,
		quizzesQuestionsReorderCmd,
	)
	quizzesAttemptsCmd.AddCommand(quizzesAttemptsListCmd, quizzesAttemptsGetCmd)
	quizzesCmd.AddCommand(
		quizzesUpdateCmd,
		quizzesDeleteCmd,
		quizzesPublishCmd,
		quizzesUnpublishCmd,
		quizzesSettingsCmd,
		quizzesQuestionsCmd,
		quizzesAttemptsCmd,
		quizzesGradeCmd,
		quizzesGradeSyncCmd,
		quizzesCodeRunCmd,
	)
}

func decodeQuiz(raw []byte) (quizPublic, error) {
	var q quizPublic
	if err := json.Unmarshal(raw, &q); err != nil {
		return quizPublic{}, fmt.Errorf("decoding quiz: %w", err)
	}
	return q, nil
}

func fetchQuizAttempts(c *client.Client, courseCode, itemID, userID string) (quizAttemptsListBody, []byte, error) {
	path := quizAPIPath(courseCode, "/quizzes/"+url.PathEscape(itemID)+"/attempts")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return quizAttemptsListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	if userID != "" {
		q := req.URL.Query()
		q.Set("userId", userID)
		req.URL.RawQuery = q.Encode()
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return quizAttemptsListBody{}, nil, fmt.Errorf("listing attempts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return quizAttemptsListBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return quizAttemptsListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out quizAttemptsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return quizAttemptsListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func putQuizAttemptGrading(c *client.Client, courseCode, itemID, attemptID string, questions []map[string]any) ([]byte, error) {
	raw, _ := json.Marshal(map[string]any{"questions": questions})
	path := quizAPIPath(courseCode, "/quizzes/"+url.PathEscape(itemID)+"/attempts/"+url.PathEscape(attemptID)+"/grading")
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("grading attempt: %w", err)
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

func getQuizAttemptGrading(c *client.Client, courseCode, itemID, attemptID string) (quizAttemptGradingBody, []byte, error) {
	path := quizAPIPath(courseCode, "/quizzes/"+url.PathEscape(itemID)+"/attempts/"+url.PathEscape(attemptID)+"/grading")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return quizAttemptGradingBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return quizAttemptGradingBody{}, nil, fmt.Errorf("getting grading: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return quizAttemptGradingBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return quizAttemptGradingBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out quizAttemptGradingBody
	if err := json.Unmarshal(body, &out); err != nil {
		return quizAttemptGradingBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func putGradebookGrades(c *client.Client, courseCode string, grades map[string]map[string]string) error {
	raw, _ := json.Marshal(map[string]any{"grades": grades})
	req, err := c.NewRequest(http.MethodPut, quizAPIPath(courseCode, "/gradebook/grades"), bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("syncing gradebook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func bankQuestionToQuizQuestion(q questionPublic) quizQuestion {
	prompt := questionPreview(q.Content)
	if prompt == "" {
		prompt = q.ID
	}
	qType := strings.ReplaceAll(q.Type, "-", "_")
	if qType == "multiple_choice" {
		qType = "multiple_choice"
	}
	out := quizQuestion{
		ID:           q.ID,
		Prompt:       prompt,
		QuestionType: qType,
		Points:       1,
		Required:     true,
	}
	if choices, ok := q.Content["choices"].([]any); ok {
		for _, c := range choices {
			out.Choices = append(out.Choices, fmt.Sprintf("%v", c))
		}
	}
	if idx, ok := q.Content["correctChoiceIndex"]; ok {
		switch v := idx.(type) {
		case float64:
			u := uint(v)
			out.CorrectChoiceIndex = &u
		}
	}
	if pts, ok := q.Content["points"]; ok {
		switch v := pts.(type) {
		case float64:
			out.Points = int32(v)
		}
	}
	return out
}

func fetchBankQuestions(c *client.Client, bankID string, limit int) ([]questionPublic, error) {
	path := "/api/question-banks/" + bankID + "/questions"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	q := req.URL.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("page", "1")
	req.URL.RawQuery = q.Encode()
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing bank questions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp, 2)
	}
	var questions []questionPublic
	if err := json.NewDecoder(resp.Body).Decode(&questions); err != nil {
		return nil, fmt.Errorf("decoding bank questions: %w", err)
	}
	return questions, nil
}

func selectBankQuestions(bank []questionPublic, ids []string, count int) ([]quizQuestion, error) {
	if len(ids) > 0 {
		byID := make(map[string]questionPublic, len(bank))
		for _, q := range bank {
			byID[q.ID] = q
		}
		var out []quizQuestion
		for _, id := range ids {
			q, ok := byID[id]
			if !ok {
				return nil, fmt.Errorf("question %q not found in bank", id)
			}
			out = append(out, bankQuestionToQuizQuestion(q))
		}
		return out, nil
	}
	if count <= 0 {
		return nil, fmt.Errorf("specify --bank with --count or --id, or use --content for inline questions")
	}
	if count > len(bank) {
		return nil, fmt.Errorf("bank has %d questions; cannot add %d", len(bank), count)
	}
	var out []quizQuestion
	for i := 0; i < count; i++ {
		out = append(out, bankQuestionToQuizQuestion(bank[i]))
	}
	return out, nil
}

func gradingPayloadFromBody(body quizAttemptGradingBody) []map[string]any {
	var grades []map[string]any
	for _, q := range body.Questions {
		pts := 0.0
		if q.PointsAwarded != nil {
			pts = *q.PointsAwarded
		} else if q.IsCorrect != nil {
			if *q.IsCorrect {
				pts = q.MaxPoints
			}
		}
		grades = append(grades, map[string]any{
			"questionIndex":  q.QuestionIndex,
			"pointsAwarded": pts,
		})
	}
	return grades
}

func policyPointsFromAttempts(attempts []quizAttemptSummary, policy string) (float64, bool) {
	type scored struct {
		points float64
		number int32
	}
	var ready []scored
	for _, a := range attempts {
		if a.NeedsManualGrading || a.PointsEarned == nil {
			continue
		}
		ready = append(ready, scored{points: *a.PointsEarned, number: a.AttemptNumber})
	}
	if len(ready) == 0 {
		return 0, false
	}
	switch policy {
	case "highest":
		best := ready[0].points
		for _, s := range ready[1:] {
			if s.points > best {
				best = s.points
			}
		}
		return best, true
	case "first":
		return ready[0].points, true
	case "average":
		var sum float64
		for _, s := range ready {
			sum += s.points
		}
		return sum / float64(len(ready)), true
	default:
		return ready[len(ready)-1].points, true
	}
}

func runQuizzesUpdate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := quizzesUpdateFlags.course
	itemID := args[0]

	if quizzesUpdateFlags.title != "" {
		title := quizzesUpdateFlags.title
		if err := patchStructureItem(c, courseCode, itemID, structureItemPatchOpts{title: &title}); err != nil {
			return err
		}
	}

	patch := map[string]any{}
	if quizzesUpdateFlags.points >= 0 {
		patch["pointsWorth"] = quizzesUpdateFlags.points
	}
	markdown := quizzesUpdateFlags.markdown
	if quizzesUpdateFlags.file != "" {
		var data []byte
		var err error
		if quizzesUpdateFlags.file == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(quizzesUpdateFlags.file)
		}
		if err != nil {
			return fmt.Errorf("reading markdown file: %w", err)
		}
		markdown = string(data)
	}
	if markdown != "" {
		patch["markdown"] = markdown
	}
	if len(patch) > 0 {
		if _, err := patchQuiz(c, courseCode, itemID, patch); err != nil {
			return err
		}
	}
	if quizzesUpdateFlags.title == "" && len(patch) == 0 {
		return fmt.Errorf("no fields to update")
	}
	if globalFlags.jsonOut {
		raw, err := fetchQuizRaw(c, courseCode, itemID)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated quiz %s\n", itemID)
	return nil
}

func runQuizzesDelete(cmd *cobra.Command, args []string) error {
	if err := deleteStructureItem(client.New(Cfg.Server, Cfg.APIKey), quizzesDeleteFlags.course, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": args[0]})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted quiz %s\n", args[0])
	return nil
}

func runQuizzesPublish(cmd *cobra.Command, args []string) error {
	pub := true
	if err := patchStructureItem(client.New(Cfg.Server, Cfg.APIKey), quizzesPublishFlags.course, args[0], structureItemPatchOpts{published: &pub}); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"id": args[0], "published": true})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Published quiz %s\n", args[0])
	return nil
}

func runQuizzesUnpublish(cmd *cobra.Command, args []string) error {
	pub := false
	if err := patchStructureItem(client.New(Cfg.Server, Cfg.APIKey), quizzesPublishFlags.course, args[0], structureItemPatchOpts{published: &pub}); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"id": args[0], "published": false})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Unpublished quiz %s\n", args[0])
	return nil
}

func runQuizzesSettingsSet(cmd *cobra.Command, args []string) error {
	patch := map[string]any{}
	if quizzesSettingsSetFlags.timeLimit >= 0 {
		v := int32(quizzesSettingsSetFlags.timeLimit)
		patch["timeLimitMinutes"] = v
	}
	if quizzesSettingsSetFlags.maxAttempts >= 0 {
		patch["maxAttempts"] = int32(quizzesSettingsSetFlags.maxAttempts)
	}
	if quizzesSettingsSetFlags.unlimited {
		patch["unlimitedAttempts"] = true
	}
	if quizzesSettingsSetFlags.shuffleQuestions {
		patch["shuffleQuestions"] = true
	}
	if quizzesSettingsSetFlags.shuffleChoices {
		patch["shuffleChoices"] = true
	}
	if quizzesSettingsSetFlags.policy != "" {
		patch["gradeAttemptPolicy"] = quizzesSettingsSetFlags.policy
	}
	if quizzesSettingsSetFlags.availableFrom != "" {
		t, err := parseOptionalRFC3339("available-from", quizzesSettingsSetFlags.availableFrom)
		if err != nil {
			return err
		}
		if t != nil {
			patch["availableFrom"] = t.Format(time.RFC3339)
		}
	}
	if quizzesSettingsSetFlags.availableUntil != "" {
		t, err := parseOptionalRFC3339("available-until", quizzesSettingsSetFlags.availableUntil)
		if err != nil {
			return err
		}
		if t != nil {
			patch["availableUntil"] = t.Format(time.RFC3339)
		}
	}
	if len(patch) == 0 {
		return fmt.Errorf("no settings to update")
	}
	raw, err := patchQuiz(client.New(Cfg.Server, Cfg.APIKey), quizzesSettingsSetFlags.course, args[0], patch)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated settings for quiz %s\n", args[0])
	return nil
}

func runQuizzesQuestionsAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := quizzesQuestionsAddFlags.course
	itemID := args[0]

	raw, err := fetchQuizRaw(c, courseCode, itemID)
	if err != nil {
		return err
	}
	quiz, err := decodeQuiz(raw)
	if err != nil {
		return err
	}

	var toAdd []quizQuestion
	switch {
	case quizzesQuestionsAddFlags.content != "":
		contentStr, err := resolveContent(quizzesQuestionsAddFlags.content)
		if err != nil {
			return err
		}
		var inline quizQuestion
		if err := json.Unmarshal([]byte(contentStr), &inline); err != nil {
			return fmt.Errorf("parsing inline question JSON: %w", err)
		}
		if inline.ID == "" {
			inline.ID = fmt.Sprintf("q-%d", len(quiz.Questions)+1)
		}
		toAdd = []quizQuestion{inline}
	case quizzesQuestionsAddFlags.bank != "":
		limit := quizzesQuestionsAddFlags.count
		if limit <= 0 {
			limit = len(quizzesQuestionsAddFlags.ids)
		}
		if limit <= 0 {
			limit = 50
		}
		bank, err := fetchBankQuestions(c, quizzesQuestionsAddFlags.bank, limit)
		if err != nil {
			return err
		}
		toAdd, err = selectBankQuestions(bank, quizzesQuestionsAddFlags.ids, quizzesQuestionsAddFlags.count)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("specify --bank, --id, or --content")
	}

	questions := append(quiz.Questions, toAdd...)
	patchRaw, err := patchQuiz(c, courseCode, itemID, map[string]any{"questions": questions})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		var updated quizPublic
		_ = json.Unmarshal(patchRaw, &updated)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"added":     len(toAdd),
			"questions": len(updated.Questions),
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %d question(s); quiz now has %d\n", len(toAdd), len(questions))
	return nil
}

func runQuizzesQuestionsRemove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := fetchQuizRaw(c, quizzesQuestionsRemoveFlags.course, args[0])
	if err != nil {
		return err
	}
	quiz, err := decodeQuiz(raw)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(quizzesQuestionsRemoveFlags.id)
	idx := quizzesQuestionsRemoveFlags.index
	if id == "" && idx < 0 {
		return fmt.Errorf("specify --id or --index")
	}
	var kept []quizQuestion
	removed := 0
	for i, q := range quiz.Questions {
		if id != "" && q.ID == id {
			removed++
			continue
		}
		if idx >= 0 && i == idx {
			removed++
			continue
		}
		kept = append(kept, q)
	}
	if removed == 0 {
		return fmt.Errorf("question not found")
	}
	if _, err := patchQuiz(c, quizzesQuestionsRemoveFlags.course, args[0], map[string]any{"questions": kept}); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]int{"removed": removed, "questions": len(kept)})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d question(s)\n", removed)
	return nil
}

func runQuizzesQuestionsList(cmd *cobra.Command, args []string) error {
	raw, err := fetchQuizRaw(client.New(Cfg.Server, Cfg.APIKey), quizzesQuestionsListFlags.course, args[0])
	if err != nil {
		return err
	}
	quiz, err := decodeQuiz(raw)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(quiz.Questions)
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "INDEX\tID\tTYPE\tPOINTS\tPREVIEW")
	for i, q := range quiz.Questions {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\n", i, q.ID, q.QuestionType, q.Points, quizQuestionPreview(q))
	}
	return w.Flush()
}

func runQuizzesQuestionsReorder(cmd *cobra.Command, args []string) error {
	ids, err := parseOrderFlag(quizzesQuestionsReorderFlags.order)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := fetchQuizRaw(c, quizzesQuestionsReorderFlags.course, args[0])
	if err != nil {
		return err
	}
	quiz, err := decodeQuiz(raw)
	if err != nil {
		return err
	}
	byID := make(map[string]quizQuestion, len(quiz.Questions))
	for _, q := range quiz.Questions {
		byID[q.ID] = q
	}
	var reordered []quizQuestion
	for _, id := range ids {
		q, ok := byID[id]
		if !ok {
			return fmt.Errorf("question id %q not on quiz", id)
		}
		reordered = append(reordered, q)
	}
	if len(reordered) != len(quiz.Questions) {
		return fmt.Errorf("--order must include every question id (%d expected, %d given)", len(quiz.Questions), len(reordered))
	}
	if _, err := patchQuiz(c, quizzesQuestionsReorderFlags.course, args[0], map[string]any{"questions": reordered}); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]int{"questions": len(reordered)})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Reordered %d questions\n", len(reordered))
	return nil
}

func runQuizzesAttemptsList(cmd *cobra.Command, args []string) error {
	if quizzesAttemptsListFlags.user == "" && !quizzesAttemptsListFlags.yes {
		if err := confirmSensitiveExport(false); err != nil {
			return err
		}
	}
	body, raw, err := fetchQuizAttempts(client.New(Cfg.Server, Cfg.APIKey), quizzesAttemptsListFlags.course, args[0], quizzesAttemptsListFlags.user)
	if err != nil {
		return err
	}
	attempts := body.Attempts
	limit := quizzesAttemptsListFlags.limit
	page := quizzesAttemptsListFlags.page
	if limit > 0 && page > 0 {
		start := (page - 1) * limit
		if start >= len(attempts) {
			attempts = []quizAttemptSummary{}
		} else {
			end := start + limit
			if end > len(attempts) {
				end = len(attempts)
			}
			attempts = attempts[start:end]
		}
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(attempts)
	}
	_ = raw
	if len(attempts) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No attempts.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTUDENT\tATTEMPT\tSCORE\tPOINTS\tMANUAL")
	for _, a := range attempts {
		student := "-"
		if a.StudentName != nil {
			student = *a.StudentName
		} else if a.StudentUserID != nil {
			student = *a.StudentUserID
		}
		score := "-"
		if a.ScorePercent != nil {
			score = fmt.Sprintf("%.1f%%", *a.ScorePercent)
		}
		points := "-"
		if a.PointsEarned != nil && a.PointsPossible != nil {
			points = fmt.Sprintf("%.1f/%.1f", *a.PointsEarned, *a.PointsPossible)
		}
		manual := "no"
		if a.NeedsManualGrading {
			manual = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n", a.ID, student, a.AttemptNumber, score, points, manual)
	}
	return w.Flush()
}

func runQuizzesAttemptsGet(cmd *cobra.Command, args []string) error {
	body, _, err := fetchQuizAttempts(client.New(Cfg.Server, Cfg.APIKey), quizzesAttemptsGetFlags.course, args[0], quizzesAttemptsGetFlags.user)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(body)
	}
	if len(body.Attempts) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No attempts for this student.")
		return nil
	}
	for _, a := range body.Attempts {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Attempt %d (%s): ", a.AttemptNumber, a.ID)
		if a.ScorePercent != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%.1f%%", *a.ScorePercent)
		}
		if a.PointsEarned != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), " (%.1f pts)", *a.PointsEarned)
		}
		if a.NeedsManualGrading {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), " [needs manual grading]")
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

func runQuizzesGrade(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := quizzesGradeFlags.course
	itemID := args[0]

	var attemptIDs []string
	switch {
	case quizzesGradeFlags.attempt != "":
		attemptIDs = []string{quizzesGradeFlags.attempt}
	default:
		list, _, err := fetchQuizAttempts(c, courseCode, itemID, quizzesGradeFlags.user)
		if err != nil {
			return err
		}
		for _, a := range list.Attempts {
			if !quizzesGradeFlags.all && a.NeedsManualGrading {
				continue
			}
			attemptIDs = append(attemptIDs, a.ID)
		}
		if len(attemptIDs) == 0 {
			return fmt.Errorf("no attempts to grade")
		}
	}

	graded := 0
	failed := 0
	for _, attemptID := range attemptIDs {
		body, _, err := getQuizAttemptGrading(c, courseCode, itemID, attemptID)
		if err != nil {
			failed++
			continue
		}
		payload := gradingPayloadFromBody(body)
		if len(payload) == 0 {
			failed++
			continue
		}
		if _, err := putQuizAttemptGrading(c, courseCode, itemID, attemptID, payload); err != nil {
			failed++
			continue
		}
		graded++
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]int{"graded": graded, "failed": failed})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Graded %d attempt(s)", graded)
	if failed > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), " (%d failed)", failed)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	return nil
}

func runQuizzesGradeSync(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := quizzesGradeSyncFlags.course
	itemID := args[0]

	quizRaw, err := fetchQuizRaw(c, courseCode, itemID)
	if err != nil {
		return err
	}
	quiz, err := decodeQuiz(quizRaw)
	if err != nil {
		return err
	}
	policy := quiz.GradeAttemptPolicy
	if policy == "" {
		policy = "latest"
	}

	list, _, err := fetchQuizAttempts(c, courseCode, itemID, quizzesGradeSyncFlags.user)
	if err != nil {
		return err
	}
	if list.RetakePolicy != "" {
		policy = list.RetakePolicy
	}

	byStudent := make(map[string][]quizAttemptSummary)
	for _, a := range list.Attempts {
		if a.StudentUserID == nil {
			continue
		}
		byStudent[*a.StudentUserID] = append(byStudent[*a.StudentUserID], a)
	}

	grades := make(map[string]map[string]string)
	synced := 0
	skipped := 0
	for studentID, attempts := range byStudent {
		pts, ok := policyPointsFromAttempts(attempts, policy)
		if !ok {
			skipped++
			continue
		}
		if grades[studentID] == nil {
			grades[studentID] = make(map[string]string)
		}
		grades[studentID][itemID] = strconv.FormatFloat(pts, 'f', -1, 64)
		synced++
	}
	if len(grades) == 0 {
		return fmt.Errorf("no gradebook scores to sync")
	}
	if err := putGradebookGrades(c, courseCode, grades); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]int{"synced": synced, "skipped": skipped, "failed": 0})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Synced %d student grade(s)", synced)
	if skipped > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), " (%d skipped)", skipped)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	return nil
}

func runQuizzesCodeRun(cmd *cobra.Command, args []string) error {
	code := quizzesCodeRunFlags.code
	if quizzesCodeRunFlags.file != "" {
		var data []byte
		var err error
		if quizzesCodeRunFlags.file == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(quizzesCodeRunFlags.file)
		}
		if err != nil {
			return fmt.Errorf("reading code file: %w", err)
		}
		code = string(data)
	}
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("provide --code or --file")
	}

	path := quizAPIPath(
		quizzesCodeRunFlags.course,
		"/quizzes/"+url.PathEscape(args[0])+
			"/attempts/"+url.PathEscape(quizzesCodeRunFlags.attempt)+
			"/questions/"+url.PathEscape(quizzesCodeRunFlags.question)+"/run",
	)
	body, _ := json.Marshal(map[string]string{"code": code})
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return fmt.Errorf("running code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	var out struct {
		Results []quizCodeRunResult `json:"results"`
		PointsEarned float64 `json:"pointsEarned"`
		PointsPossible float64 `json:"pointsPossible"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	for i, r := range out.Results {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Test %d: %s passed=%v\n", i+1, r.Status, r.Passed)
		if r.ActualOutput != "" || r.ExpectedOutput != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  expected: %s\n  actual:   %s\n", r.ExpectedOutput, r.ActualOutput)
		}
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Score: %.2f / %.2f\n", out.PointsEarned, math.Max(out.PointsPossible, 0))
	return nil
}