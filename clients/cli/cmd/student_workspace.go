package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var notebooksCmd = &cobra.Command{Use: "notebooks", Short: "Student notebooks"}

var notebooksListCmd = &cobra.Command{Use: "list", Short: "List notebooks", RunE: runNotebooksList}

var notebooksAddFlags struct {
	course string
	file   string
}
var notebooksAddCmd = &cobra.Command{Use: "add", Short: "Add or replace notebook pages from a file", RunE: runNotebooksAdd}

var notebooksUpdateFlags struct {
	course string
	file   string
}
var notebooksUpdateCmd = &cobra.Command{Use: "update", Short: "Update notebook from JSON file", RunE: runNotebooksUpdate}

var notebooksDeleteFlags struct{ course string }
var notebooksDeleteCmd = &cobra.Command{Use: "delete", Short: "Clear a course notebook", RunE: runNotebooksDelete}

var notebookTasksCmd = &cobra.Command{Use: "notebook-tasks", Short: "Tasks linked to notebook pages"}

var notebookTasksListCmd = &cobra.Command{Use: "list", Short: "List notebook tasks", RunE: runNotebookTasksList}

var notebookTasksAddFlags struct {
	course string
	page   string
	text   string
	file   string
	due    string
}
var notebookTasksAddCmd = &cobra.Command{Use: "add", Short: "Add a notebook task", RunE: runNotebookTasksAdd}

var notebookTasksCompleteCmd = &cobra.Command{
	Use:   "complete <task_id>",
	Short: "Mark a notebook task complete",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotebookTasksComplete,
}

var todoCmd = &cobra.Command{Use: "todo", Short: "Student to-do board"}

var todoListCmd = &cobra.Command{Use: "list", Short: "List todo items", RunE: runTodoList}

var todoAddFlags struct {
	due string
	col string
}
var todoAddCmd = &cobra.Command{
	Use:   "add <text>",
	Short: "Add a todo item",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runTodoAdd,
}

var todoCompleteCmd = &cobra.Command{
	Use:   "complete <item_id>",
	Short: "Move a todo item to done",
	Args:  cobra.ExactArgs(1),
	RunE:  runTodoComplete,
}

var todoRemoveCmd = &cobra.Command{
	Use:   "remove <item_id>",
	Short: "Remove a todo item",
	Args:  cobra.ExactArgs(1),
	RunE:  runTodoRemove,
}

var journalCmd = &cobra.Command{Use: "journal", Short: "Reflection journal"}

var journalListCmd = &cobra.Command{Use: "list", Short: "List journal entries", RunE: runJournalList}

var journalAddFlags struct{ file string }
var journalAddCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a journal entry",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runJournalAdd,
}

var goalsCmd = &cobra.Command{Use: "goals", Short: "Study goals"}

var goalsGetCmd = &cobra.Command{Use: "get", Short: "Get study goal", RunE: runGoalsGet}
var goalsSetFlags struct{ file string }
var goalsSetCmd = &cobra.Command{Use: "set", Short: "Set study goal from JSON", RunE: runGoalsSet}

var remindersCmd = &cobra.Command{Use: "reminders", Short: "Study reminders"}

var remindersGetCmd = &cobra.Command{Use: "get", Short: "Get reminder config", RunE: runRemindersGet}
var remindersSetFlags struct{ file string }
var remindersSetCmd = &cobra.Command{Use: "set", Short: "Set reminder config", RunE: runRemindersSet}

var gamificationCmd = &cobra.Command{Use: "gamification", Short: "Gamification state"}

var gamificationStatusCmd = &cobra.Command{Use: "status", Short: "Show points and streaks", RunE: runGamificationStatus}

var leaderboardCmd = &cobra.Command{Use: "leaderboard", Short: "Course leaderboard"}

var leaderboardFlags struct{ course string }
var leaderboardShowCmd = &cobra.Command{Use: "show", Short: "Show course leaderboard", RunE: runLeaderboardShow}

var coachingTipsCmd = &cobra.Command{Use: "coaching-tips", Short: "Coaching tips"}
var coachingTipsListCmd = &cobra.Command{Use: "list", Short: "List coaching tips", RunE: runCoachingTipsList}

var readingPrefsCmd = &cobra.Command{Use: "reading-preferences", Short: "Reading preferences"}
var readingPrefsGetCmd = &cobra.Command{Use: "get", Short: "Get reading preferences", RunE: runReadingPrefsGet}
var readingPrefsSetFlags struct{ file string }
var readingPrefsSetCmd = &cobra.Command{Use: "set", Short: "Set reading preferences", RunE: runReadingPrefsSet}

func runNotebooksList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listNotebooks(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "COURSE\tUPDATED")
	for _, n := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", n.CourseCode, n.UpdatedAt)
	}
	return w.Flush()
}

func runNotebooksAdd(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	var payload []byte
	var err error
	if strings.HasSuffix(notebooksAddFlags.file, ".json") {
		payload, err = cli.ReadFile(notebooksAddFlags.file)
	} else {
		text, err := cli.ReadTextFile(notebooksAddFlags.file)
		if err != nil {
			return err
		}
		payload, err = notebookFromTextFile(text)
	}
	if err != nil {
		return err
	}
	_, err = putNotebook(c, notebooksAddFlags.course, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"courseCode": notebooksAddFlags.course, "status": "saved",
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Notebook saved for %s\n", notebooksAddFlags.course)
	return nil
}

func runNotebooksUpdate(cmd *cobra.Command, _ []string) error {
	notebooksAddFlags.course = notebooksUpdateFlags.course
	notebooksAddFlags.file = notebooksUpdateFlags.file
	return runNotebooksAdd(cmd, nil)
}

func runNotebooksDelete(cmd *cobra.Command, args []string) error {
	course, err := parseNotebookDeleteCourse(args, notebooksDeleteFlags.course)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, _ := notebookFromTextFile(" ")
	_, err = putNotebook(c, course, payload)
	return err
}

func runNotebookTasksList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listNotebookTasks(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tCOURSE\tTEXT\tDONE")
	for _, t := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", t.ID, t.CourseCode, t.TaskText, t.Completed)
	}
	return w.Flush()
}

func runNotebookTasksAdd(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload := map[string]any{
		"id":             uuid.New().String(),
		"courseCode":     notebookTasksAddFlags.course,
		"notebookPageId": notebookTasksAddFlags.page,
		"taskText":       notebookTasksAddFlags.text,
	}
	if notebookTasksAddFlags.file != "" {
		var err error
		payload, err = cli.ReadJSONFile(notebookTasksAddFlags.file)
		if err != nil {
			return err
		}
	}
	if notebookTasksAddFlags.due != "" {
		t, err := cli.ParseRFC3339InTZ(notebookTasksAddFlags.due, cliRuntime.tz)
		if err != nil {
			return err
		}
		s, _ := cli.FormatRFC3339(t, cliRuntime.tz)
		payload["dueAt"] = s
	}
	raw, err := upsertNotebookTask(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runNotebookTasksComplete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	done := true
	raw, err := patchNotebookTask(c, args[0], map[string]any{"completed": done})
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runTodoList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	columns, raw, err := getTodoBoard(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "COLUMN\tITEM")
	for col, items := range columns {
		for _, item := range items {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", col, item)
		}
	}
	return w.Flush()
}

func runTodoAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	columns, _, err := getTodoBoard(c)
	if err != nil {
		return err
	}
	col := todoAddFlags.col
	if col == "" {
		col, err = parseTodoDueColumn(todoAddFlags.due, cliRuntime.tz)
		if err != nil {
			return err
		}
	}
	key := todoItemKey(strings.Join(args, " "))
	if columns == nil {
		columns = map[string][]string{}
	}
	columns[col] = append(columns[col], key)
	if err := putTodoBoard(c, columns); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"id": key})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %s to %s\n", key, col)
	return nil
}

func runTodoComplete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	columns, _, err := getTodoBoard(c)
	if err != nil {
		return err
	}
	key, ok := ensureTodoItemKey(columns, args[0])
	if !ok {
		return fmt.Errorf("todo item %q not found", args[0])
	}
	moveTodoItem(columns, key, "done")
	return putTodoBoard(c, columns)
}

func runTodoRemove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	columns, _, err := getTodoBoard(c)
	if err != nil {
		return err
	}
	key, ok := ensureTodoItemKey(columns, args[0])
	if !ok {
		return fmt.Errorf("todo item %q not found", args[0])
	}
	removeTodoItem(columns, key)
	return putTodoBoard(c, columns)
}

func runJournalList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := journalList(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runJournalAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	text := strings.Join(args, " ")
	if journalAddFlags.file != "" {
		var err error
		text, err = cli.ReadTextFile(journalAddFlags.file)
		if err != nil {
			return err
		}
	}
	raw, err := journalAdd(c, text)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runGoalsGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := goalsGet(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runGoalsSet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(goalsSetFlags.file)
	if err != nil {
		return err
	}
	raw, err := goalsSet(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runRemindersGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := remindersGet(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runRemindersSet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(remindersSetFlags.file)
	if err != nil {
		return err
	}
	raw, err := remindersSet(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runGamificationStatus(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getGamification(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runLeaderboardShow(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getLeaderboard(c, leaderboardFlags.course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runCoachingTipsList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getCoachingTips(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runReadingPrefsGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := readingPrefsGet(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runReadingPrefsSet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(readingPrefsSetFlags.file)
	if err != nil {
		return err
	}
	raw, err := readingPrefsSet(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func init() {
	notebooksAddCmd.Flags().StringVar(&notebooksAddFlags.course, "course", "", "course code")
	notebooksAddCmd.Flags().StringVar(&notebooksAddFlags.file, "file", "", "markdown or notebook JSON")
	_ = notebooksAddCmd.MarkFlagRequired("course")
	_ = notebooksAddCmd.MarkFlagRequired("file")
	notebooksUpdateCmd.Flags().StringVar(&notebooksUpdateFlags.course, "course", "", "course code")
	notebooksUpdateCmd.Flags().StringVar(&notebooksUpdateFlags.file, "file", "", "notebook JSON")
	_ = notebooksUpdateCmd.MarkFlagRequired("course")
	_ = notebooksUpdateCmd.MarkFlagRequired("file")
	notebooksDeleteCmd.Flags().StringVar(&notebooksDeleteFlags.course, "course", "", "course code")

	notebookTasksAddCmd.Flags().StringVar(&notebookTasksAddFlags.course, "course", "", "course code")
	notebookTasksAddCmd.Flags().StringVar(&notebookTasksAddFlags.page, "page", "main", "notebook page id")
	notebookTasksAddCmd.Flags().StringVar(&notebookTasksAddFlags.text, "text", "", "task text")
	notebookTasksAddCmd.Flags().StringVar(&notebookTasksAddFlags.file, "file", "", "task JSON")
	notebookTasksAddCmd.Flags().StringVar(&notebookTasksAddFlags.due, "due", "", "due date")
	_ = notebookTasksAddCmd.MarkFlagRequired("course")

	todoAddCmd.Flags().StringVar(&todoAddFlags.due, "due", "", "due day or date")
	todoAddCmd.Flags().StringVar(&todoAddFlags.col, "column", "", "column id (mon..sun, done)")

	journalAddCmd.Flags().StringVar(&journalAddFlags.file, "file", "", "entry text file")
	goalsSetCmd.Flags().StringVar(&goalsSetFlags.file, "file", "", "goal JSON")
	_ = goalsSetCmd.MarkFlagRequired("file")
	remindersSetCmd.Flags().StringVar(&remindersSetFlags.file, "file", "", "reminder JSON")
	_ = remindersSetCmd.MarkFlagRequired("file")
	leaderboardShowCmd.Flags().StringVar(&leaderboardFlags.course, "course", "", "course code")
	_ = leaderboardShowCmd.MarkFlagRequired("course")
	readingPrefsSetCmd.Flags().StringVar(&readingPrefsSetFlags.file, "file", "", "preferences JSON")
	_ = readingPrefsSetCmd.MarkFlagRequired("file")

	notebooksCmd.AddCommand(notebooksListCmd, notebooksAddCmd, notebooksUpdateCmd, notebooksDeleteCmd)
	notebookTasksCmd.AddCommand(notebookTasksListCmd, notebookTasksAddCmd, notebookTasksCompleteCmd)
	todoCmd.AddCommand(todoListCmd, todoAddCmd, todoCompleteCmd, todoRemoveCmd)
	journalCmd.AddCommand(journalListCmd, journalAddCmd)
	goalsCmd.AddCommand(goalsGetCmd, goalsSetCmd)
	remindersCmd.AddCommand(remindersGetCmd, remindersSetCmd)
	gamificationCmd.AddCommand(gamificationStatusCmd)
	leaderboardCmd.AddCommand(leaderboardShowCmd)
	coachingTipsCmd.AddCommand(coachingTipsListCmd)
	readingPrefsCmd.AddCommand(readingPrefsGetCmd, readingPrefsSetCmd)

	rootCmd.AddCommand(notebooksCmd, notebookTasksCmd, todoCmd, journalCmd, goalsCmd, remindersCmd,
		gamificationCmd, leaderboardCmd, coachingTipsCmd, readingPrefsCmd)
}