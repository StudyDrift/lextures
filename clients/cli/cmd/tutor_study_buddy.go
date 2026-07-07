package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var tutorCmd = &cobra.Command{Use: "tutor", Short: "AI tutor sessions"}

var tutorAskFlags struct {
	course  string
	session string
}
var tutorAskCmd = &cobra.Command{
	Use:   "ask [prompt]",
	Short: "Ask the AI tutor a question (streams reply to stderr)",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runTutorAsk,
}

var tutorSessionsListFlags struct{ course string }
var tutorSessionsListCmd = &cobra.Command{
	Use:   "sessions list",
	Short: "List tutor conversation for a course",
	RunE:  runTutorSessionsList,
}

var tutorEvalFlags struct {
	file      string
	out       string
	maxTurns  int
	yes       bool
}
var tutorEvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Batch-evaluate tutor prompts from a JSONL file",
	RunE:  runTutorEval,
}

var studyBuddyCmd = &cobra.Command{Use: "study-buddy", Short: "Study buddy assistant"}

var studyBuddyAskFlags struct{ course string }
var studyBuddyAskCmd = &cobra.Command{
	Use:   "ask [prompt]",
	Short: "Ask the study buddy",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runStudyBuddyAsk,
}

var studyBuddySessionsCmd = &cobra.Command{Use: "sessions", Short: "Study buddy sessions"}
var studyBuddySessionsListFlags struct{ course string }
var studyBuddySessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show study buddy memory",
	RunE:  runStudyBuddySessionsList,
}

var diagnosticCmd = &cobra.Command{Use: "diagnostic", Short: "Adaptive diagnostic assessments"}

var diagnosticRunFlags struct {
	course     string
	enrollment string
}
var diagnosticRunCmd = &cobra.Command{Use: "run", Short: "Start a diagnostic attempt", RunE: runDiagnosticRun}

var diagnosticAttemptsFlags struct {
	course     string
	enrollment string
}
var diagnosticAttemptsCmd = &cobra.Command{Use: "attempts", Short: "Show diagnostic status", RunE: runDiagnosticAttempts}

var diagnosticConfigCmd = &cobra.Command{
	Use:   "config <course>",
	Short: "Get diagnostic config for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runDiagnosticConfig,
}

var pathsCmd = &cobra.Command{Use: "paths", Short: "Adaptive learning paths"}

var pathsStatusCmd = &cobra.Command{Use: "status", Short: "List enrolled paths", RunE: runPathsStatus}
var pathsGetCmd = &cobra.Command{
	Use:   "get <path_id>",
	Short: "Get path progress",
	Args:  cobra.ExactArgs(1),
	RunE:  runPathsGet,
}

var conceptsCmd = &cobra.Command{Use: "concepts", Short: "Concept graph"}

var conceptsListFlags struct{ q string }
var conceptsListCmd = &cobra.Command{Use: "list", Short: "List concepts", RunE: runConceptsList}
var conceptsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a concept",
	Args:  cobra.ExactArgs(1),
	RunE:  runConceptsGet,
}

var learnersCmd = &cobra.Command{Use: "learners", Short: "Learner models"}

var learnersGetFlags struct{ user string }
var learnersGetCmd = &cobra.Command{Use: "get", Short: "Get learner concept mastery", RunE: runLearnersGet}

func runTutorAsk(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	prompt := strings.Join(args, " ")
	resp, err := postTutorMessage(c, tutorAskFlags.course, prompt)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := readResponseBody(resp)
		return apiErrorBody(resp.StatusCode, body)
	}
	result, err := cli.CollectTutorSSE(resp.Body, cmd.ErrOrStderr(), globalFlags.jsonOut)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"conversationId": result.ConversationID,
			"reply":          result.Text,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nconversation_id=%s\n", result.ConversationID)
	return nil
}

func runTutorSessionsList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getTutorConversation(c, tutorSessionsListFlags.course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runTutorEval(cmd *cobra.Command, _ []string) error {
	if err := cli.RequireYes(tutorEvalFlags.yes, "tutor eval may incur AI usage costs"); err != nil {
		return err
	}
	data, err := cli.ReadFile(tutorEvalFlags.file)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if tutorEvalFlags.maxTurns > 0 && len(lines) > tutorEvalFlags.maxTurns {
		lines = lines[:tutorEvalFlags.maxTurns]
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	var results []map[string]any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Course  string `json:"course"`
			Prompt  string `json:"prompt"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return err
		}
		course := row.Course
		if course == "" {
			course = tutorAskFlags.course
		}
		resp, err := postTutorMessage(c, course, row.Prompt)
		if err != nil {
			return err
		}
		result, err := cli.CollectTutorSSE(resp.Body, nil, true)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		results = append(results, map[string]any{
			"course": course, "prompt": row.Prompt,
			"reply": result.Text, "conversationId": result.ConversationID,
		})
	}
	outPath := tutorEvalFlags.out
	if outPath == "" || outPath == "-" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		for _, r := range results {
			_ = enc.Encode(r)
		}
		return nil
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, r := range results {
		_ = enc.Encode(r)
	}
	return nil
}

func runStudyBuddyAsk(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := postStudyBuddyMessage(c, studyBuddyAskFlags.course, strings.Join(args, " "))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runStudyBuddySessionsList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getStudyBuddyMemory(c, studyBuddySessionsListFlags.course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runDiagnosticRun(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	enrollment := diagnosticRunFlags.enrollment
	if enrollment == "" {
		var err error
		enrollment, err = resolveEnrollmentForCourse(c, diagnosticRunFlags.course)
		if err != nil {
			return err
		}
	}
	raw, err := startDiagnostic(c, enrollment)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runDiagnosticAttempts(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	enrollment := diagnosticAttemptsFlags.enrollment
	if enrollment == "" {
		var err error
		enrollment, err = resolveEnrollmentForCourse(c, diagnosticAttemptsFlags.course)
		if err != nil {
			return err
		}
	}
	raw, err := getDiagnosticGate(c, enrollment)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runDiagnosticConfig(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getDiagnosticConfig(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runPathsStatus(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := listMyPaths(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runPathsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getPathProgress(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runConceptsList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := listConcepts(c, conceptsListFlags.q)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	var out struct {
		Concepts []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"concepts"`
	}
	_ = json.Unmarshal(raw, &out)
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME")
	for _, c := range out.Concepts {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", c.ID, c.Name)
	}
	return w.Flush()
}

func runConceptsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getConcept(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runLearnersGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	user := learnersGetFlags.user
	if user == "" {
		raw, err := fetchMeProfile(c)
		if err != nil {
			return err
		}
		var me meProfile
		_ = json.Unmarshal(raw, &me)
		user = me.ID
	}
	raw, err := getLearnerConcepts(c, user)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func init() {
	tutorAskCmd.Flags().StringVar(&tutorAskFlags.course, "course", "", "course code")
	_ = tutorAskCmd.MarkFlagRequired("course")
	tutorSessionsListCmd.Flags().StringVar(&tutorSessionsListFlags.course, "course", "", "course code")
	_ = tutorSessionsListCmd.MarkFlagRequired("course")
	tutorEvalCmd.Flags().StringVar(&tutorEvalFlags.file, "file", "", "JSONL prompts file")
	tutorEvalCmd.Flags().StringVar(&tutorEvalFlags.out, "out", "-", "results output")
	tutorEvalCmd.Flags().IntVar(&tutorEvalFlags.maxTurns, "max-turns", 0, "cap number of prompts")
	tutorEvalCmd.Flags().BoolVar(&tutorEvalFlags.yes, "yes", false, "confirm eval run")
	_ = tutorEvalCmd.MarkFlagRequired("file")

	studyBuddyAskCmd.Flags().StringVar(&studyBuddyAskFlags.course, "course", "", "course code")
	_ = studyBuddyAskCmd.MarkFlagRequired("course")
	studyBuddySessionsListCmd.Flags().StringVar(&studyBuddySessionsListFlags.course, "course", "", "course code")
	_ = studyBuddySessionsListCmd.MarkFlagRequired("course")

	diagnosticRunCmd.Flags().StringVar(&diagnosticRunFlags.course, "course", "", "course code")
	diagnosticRunCmd.Flags().StringVar(&diagnosticRunFlags.enrollment, "enrollment", "", "enrollment id")
	diagnosticAttemptsCmd.Flags().StringVar(&diagnosticAttemptsFlags.course, "course", "", "course code")
	diagnosticAttemptsCmd.Flags().StringVar(&diagnosticAttemptsFlags.enrollment, "enrollment", "", "enrollment id")

	learnersGetCmd.Flags().StringVar(&learnersGetFlags.user, "user", "", "user id (defaults to self)")
	conceptsListCmd.Flags().StringVar(&conceptsListFlags.q, "q", "", "search query")

	tutorCmd.AddCommand(tutorAskCmd, tutorSessionsListCmd, tutorEvalCmd)
	studyBuddyCmd.AddCommand(studyBuddyAskCmd, studyBuddySessionsCmd)
	studyBuddySessionsCmd.AddCommand(studyBuddySessionsListCmd)
	diagnosticCmd.AddCommand(diagnosticRunCmd, diagnosticAttemptsCmd, diagnosticConfigCmd)
	pathsCmd.AddCommand(pathsStatusCmd, pathsGetCmd)
	conceptsCmd.AddCommand(conceptsListCmd, conceptsGetCmd)
	learnersCmd.AddCommand(learnersGetCmd)

	rootCmd.AddCommand(tutorCmd, studyBuddyCmd, diagnosticCmd, pathsCmd, conceptsCmd, learnersCmd)
}