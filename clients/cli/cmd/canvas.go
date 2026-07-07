package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var canvasCmd = &cobra.Command{
	Use:   "canvas",
	Short: "Browse Canvas and drive course imports",
}

var canvasCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Browse the Canvas course catalog",
}

var canvasCatalogCommonFlags struct {
	canvasBase string
	tokenFile  string
}

var canvasCatalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Canvas courses available to the token",
	RunE:  runCanvasCatalogList,
}

var canvasCatalogSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the Canvas catalog",
	Args:  cobra.ExactArgs(1),
	RunE:  runCanvasCatalogSearch,
}

var canvasImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Queue Canvas imports and monitor the job queue",
}

var canvasImportCommonFlags struct {
	course     string
	canvasBase string
	tokenFile  string
	wait       bool
	timeout    time.Duration
	mode       string
}

var canvasImportCourseCmd = &cobra.Command{
	Use:   "course <canvas-course-id>",
	Short: "Import a Canvas course into a Lextures course",
	Args:  cobra.ExactArgs(1),
	RunE:  runCanvasImportCourse,
}

var canvasImportEnrollmentsCmd = &cobra.Command{
	Use:   "enrollments",
	Short: "Import enrollments only from Canvas",
	RunE:  runCanvasImportEnrollments,
}

var canvasImportGradesCmd = &cobra.Command{
	Use:   "grades",
	Short: "Import grades from Canvas",
	RunE:  runCanvasImportGrades,
}

var canvasImportSubmissionsCmd = &cobra.Command{
	Use:   "submissions",
	Short: "Import assignment submissions from Canvas",
	RunE:  runCanvasImportSubmissions,
}

var canvasImportAnnouncementsCmd = &cobra.Command{
	Use:   "announcements",
	Short: "Import announcements from Canvas",
	RunE:  runCanvasImportAnnouncements,
}

var canvasImportQueueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show import jobs (requires --course for link status)",
	RunE:  runCanvasImportQueue,
}

var canvasImportStatusCmd = &cobra.Command{
	Use:   "status <job-id>",
	Short: "Get Canvas import job status",
	Args:  cobra.ExactArgs(1),
	RunE:  runCanvasImportStatus,
}

var canvasImportRetryCmd = &cobra.Command{
	Use:   "retry <job-id>",
	Short: "Retry a failed import by re-queuing with a fresh token",
	Args:  cobra.ExactArgs(1),
	RunE:  runCanvasImportRetry,
}

var canvasImportCancelCmd = &cobra.Command{
	Use:   "cancel <job-id>",
	Short: "Cancel is not supported — imports run to completion or fail",
	Args:  cobra.ExactArgs(1),
	RunE:  runCanvasImportCancel,
}

var canvasLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage Canvas course links",
}

var canvasLinkSetFlags struct {
	course     string
	canvasBase string
	canvasID   string
	tokenFile  string
	gradeSync  bool
}

var canvasLinkSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Link a course to Canvas by running a minimal import",
	RunE:  runCanvasLinkSet,
}

var canvasLinkStatusFlags struct {
	course string
}

var canvasLinkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Canvas link status for a course",
	RunE:  runCanvasLinkStatus,
}

func init() {
	canvasCatalogListCmd.Flags().StringVar(&canvasCatalogCommonFlags.canvasBase, "canvas-base", "", "Canvas base URL (required)")
	_ = canvasCatalogListCmd.MarkFlagRequired("canvas-base")
	canvasCatalogListCmd.Flags().StringVar(&canvasCatalogCommonFlags.tokenFile, "token-file", "", "Canvas token file (required)")
	_ = canvasCatalogListCmd.MarkFlagRequired("token-file")

	canvasCatalogSearchCmd.Flags().StringVar(&canvasCatalogCommonFlags.canvasBase, "canvas-base", "", "Canvas base URL (required)")
	_ = canvasCatalogSearchCmd.MarkFlagRequired("canvas-base")
	canvasCatalogSearchCmd.Flags().StringVar(&canvasCatalogCommonFlags.tokenFile, "token-file", "", "Canvas token file (required)")
	_ = canvasCatalogSearchCmd.MarkFlagRequired("token-file")

	for _, cmd := range []*cobra.Command{
		canvasImportCourseCmd, canvasImportEnrollmentsCmd, canvasImportGradesCmd,
		canvasImportSubmissionsCmd, canvasImportAnnouncementsCmd,
	} {
		cmd.Flags().StringVar(&canvasImportCommonFlags.course, "course", "", "target Lextures course code (required)")
		_ = cmd.MarkFlagRequired("course")
		cmd.Flags().StringVar(&canvasImportCommonFlags.canvasBase, "canvas-base", "", "Canvas base URL (required)")
		_ = cmd.MarkFlagRequired("canvas-base")
		cmd.Flags().StringVar(&canvasImportCommonFlags.tokenFile, "token-file", "", "Canvas token file (required)")
		_ = cmd.MarkFlagRequired("token-file")
		cmd.Flags().BoolVar(&canvasImportCommonFlags.wait, "wait", false, "poll until the import completes")
		cmd.Flags().DurationVar(&canvasImportCommonFlags.timeout, "timeout", 2*time.Hour, "max wait time")
		cmd.Flags().StringVar(&canvasImportCommonFlags.mode, "mode", "course", "import mode")
	}

	canvasImportQueueCmd.Flags().StringVar(&canvasImportCommonFlags.course, "course", "", "course code to inspect link/import status")

	canvasImportStatusCmd.Flags().BoolVar(&canvasImportCommonFlags.wait, "wait", false, "block until the job completes")
	canvasImportStatusCmd.Flags().DurationVar(&canvasImportCommonFlags.timeout, "timeout", 2*time.Hour, "max wait time")

	canvasImportRetryCmd.Flags().StringVar(&canvasImportCommonFlags.course, "course", "", "target course code (required)")
	_ = canvasImportRetryCmd.MarkFlagRequired("course")
	canvasImportRetryCmd.Flags().StringVar(&canvasImportCommonFlags.canvasBase, "canvas-base", "", "Canvas base URL (required)")
	_ = canvasImportRetryCmd.MarkFlagRequired("canvas-base")
	canvasImportRetryCmd.Flags().StringVar(&canvasImportCommonFlags.tokenFile, "token-file", "", "Canvas token file (required)")
	_ = canvasImportRetryCmd.MarkFlagRequired("token-file")
	canvasImportRetryCmd.Flags().StringVar(&canvasImportCommonFlags.mode, "mode", "course", "import mode")
	canvasImportRetryCmd.Flags().BoolVar(&canvasImportCommonFlags.wait, "wait", false, "wait for retry to finish")
	canvasImportRetryCmd.Flags().DurationVar(&canvasImportCommonFlags.timeout, "timeout", 2*time.Hour, "max wait time")

	canvasLinkSetCmd.Flags().StringVar(&canvasLinkSetFlags.course, "course", "", "Lextures course code (required)")
	_ = canvasLinkSetCmd.MarkFlagRequired("course")
	canvasLinkSetCmd.Flags().StringVar(&canvasLinkSetFlags.canvasBase, "canvas-base", "", "Canvas base URL (required)")
	_ = canvasLinkSetCmd.MarkFlagRequired("canvas-base")
	canvasLinkSetCmd.Flags().StringVar(&canvasLinkSetFlags.canvasID, "canvas-id", "", "Canvas course id (required)")
	_ = canvasLinkSetCmd.MarkFlagRequired("canvas-id")
	canvasLinkSetCmd.Flags().StringVar(&canvasLinkSetFlags.tokenFile, "token-file", "", "Canvas token file (required)")
	_ = canvasLinkSetCmd.MarkFlagRequired("token-file")
	canvasLinkSetCmd.Flags().BoolVar(&canvasLinkSetFlags.gradeSync, "grade-sync", false, "enable grade sync after linking")

	canvasLinkStatusCmd.Flags().StringVar(&canvasLinkStatusFlags.course, "course", "", "Lextures course code (required)")
	_ = canvasLinkStatusCmd.MarkFlagRequired("course")

	canvasCatalogCmd.AddCommand(canvasCatalogListCmd, canvasCatalogSearchCmd)
	canvasImportCmd.AddCommand(
		canvasImportCourseCmd, canvasImportEnrollmentsCmd, canvasImportGradesCmd,
		canvasImportSubmissionsCmd, canvasImportAnnouncementsCmd,
		canvasImportQueueCmd, canvasImportStatusCmd, canvasImportRetryCmd, canvasImportCancelCmd,
	)
	canvasLinkCmd.AddCommand(canvasLinkSetCmd, canvasLinkStatusCmd)
	canvasCmd.AddCommand(canvasCatalogCmd, canvasImportCmd, canvasLinkCmd)
	rootCmd.AddCommand(canvasCmd)
}

func runCanvasCatalogList(cmd *cobra.Command, args []string) error {
	return runCanvasCatalog(cmd, "")
}

func runCanvasCatalogSearch(cmd *cobra.Command, args []string) error {
	return runCanvasCatalog(cmd, args[0])
}

func runCanvasCatalog(cmd *cobra.Command, query string) error {
	token, err := readCanvasToken("", canvasCatalogCommonFlags.tokenFile)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	courses, raw, err := listCanvasCatalog(c, canvasCatalogCommonFlags.canvasBase, token)
	if err != nil {
		return err
	}
	courses = filterCanvasCourses(courses, query)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"courses": courses})
	}
	if query == "" && len(courses) == 0 {
		_, _ = cmd.OutOrStdout().Write(raw)
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tCODE\tSTATE\tTERM")
	for _, course := range courses {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			course.ID, course.Name, course.CourseCode, course.WorkflowState, course.TermName)
	}
	return w.Flush()
}

func runCanvasImportArtifact(cmd *cobra.Command, canvasCourseID, artifact string) error {
	token, err := readCanvasToken("", canvasImportCommonFlags.tokenFile)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	include := includeForArtifact(artifact)
	jobID, raw, err := submitCanvasImport(
		c, canvasImportCommonFlags.course, canvasImportCommonFlags.mode,
		canvasImportCommonFlags.canvasBase, canvasCourseID, token, include, nil,
	)
	if err != nil {
		return err
	}
	return finishCanvasImport(cmd, c, jobID, raw)
}

func runCanvasImportCourse(cmd *cobra.Command, args []string) error {
	return runCanvasImportArtifact(cmd, args[0], "course")
}

func runCanvasImportEnrollments(cmd *cobra.Command, args []string) error {
	link, err := fetchCanvasLinkCourseID()
	if err != nil {
		return err
	}
	return runCanvasImportArtifact(cmd, link, "enrollments")
}

func runCanvasImportGrades(cmd *cobra.Command, args []string) error {
	link, err := fetchCanvasLinkCourseID()
	if err != nil {
		return err
	}
	return runCanvasImportArtifact(cmd, link, "grades")
}

func runCanvasImportSubmissions(cmd *cobra.Command, args []string) error {
	link, err := fetchCanvasLinkCourseID()
	if err != nil {
		return err
	}
	return runCanvasImportArtifact(cmd, link, "submissions")
}

func runCanvasImportAnnouncements(cmd *cobra.Command, args []string) error {
	link, err := fetchCanvasLinkCourseID()
	if err != nil {
		return err
	}
	return runCanvasImportArtifact(cmd, link, "announcements")
}

func fetchCanvasLinkCourseID() (string, error) {
	if canvasImportCommonFlags.course == "" {
		return "", fmt.Errorf("--course is required")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	link, _, err := fetchCourseCanvasLink(c, canvasImportCommonFlags.course)
	if err != nil {
		return "", err
	}
	linked, _ := link["linked"].(bool)
	if !linked {
		return "", fmt.Errorf("course %q is not linked to Canvas", canvasImportCommonFlags.course)
	}
	id, _ := link["canvasCourseId"].(string)
	if id == "" {
		return "", fmt.Errorf("course %q has no canvas course id", canvasImportCommonFlags.course)
	}
	return id, nil
}

func finishCanvasImport(cmd *cobra.Command, c *client.Client, jobID string, raw []byte) error {
	if canvasImportCommonFlags.wait {
		status, err := waitForCanvasImportJob(c, Cfg.APIKey, jobID, canvasImportCommonFlags.timeout, func(s canvasImportJobStatus) {
			if !globalFlags.jsonOut && s.Progress != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "progress=%s\n", s.Progress)
			}
		})
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Import %s finished: course=%s\n", jobID, status.CourseCode)
		return nil
	}
	if globalFlags.jsonOut {
		_, err := cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Queued Canvas import %s\n", jobID)
	return nil
}

func runCanvasImportQueue(cmd *cobra.Command, args []string) error {
	if canvasImportCommonFlags.course == "" {
		return fmt.Errorf("--course is required to inspect Canvas link/import status")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	link, raw, err := fetchCourseCanvasLink(c, canvasImportCommonFlags.course)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "course=%s linked=%v canvasCourseId=%v gradeSync=%v\n",
		canvasImportCommonFlags.course, link["linked"], link["canvasCourseId"], link["gradeSyncEnabled"])
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Use canvas import status <job-id> to monitor a queued import.")
	return nil
}

func runCanvasImportStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	jobID := args[0]
	if canvasImportCommonFlags.wait {
		status, err := waitForCanvasImportJob(c, Cfg.APIKey, jobID, canvasImportCommonFlags.timeout, func(s canvasImportJobStatus) {
			if !globalFlags.jsonOut && s.Progress != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "progress=%s\n", s.Progress)
			}
		})
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Job %s: %s\n", jobID, status.Status)
		return nil
	}
	status, err := waitForCanvasImportJob(c, Cfg.APIKey, jobID, 5*time.Second, nil)
	if err == nil || status.Status != "" {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Job %s: %s\n", jobID, status.Status)
		return nil
	}
	return err
}

func runCanvasImportRetry(cmd *cobra.Command, args []string) error {
	_ = args[0]
	link, err := fetchCanvasLinkCourseID()
	if err != nil {
		return err
	}
	return runCanvasImportArtifact(cmd, link, "course")
}

func runCanvasImportCancel(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("cancel is not supported for job %s; imports cannot be cancelled once queued", args[0])
}

func runCanvasLinkSet(cmd *cobra.Command, args []string) error {
	token, err := readCanvasToken("", canvasLinkSetFlags.tokenFile)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	include := canvasImportInclude{Settings: true}
	gradeSync := canvasLinkSetFlags.gradeSync
	jobID, _, err := submitCanvasImport(
		c, canvasLinkSetFlags.course, "course",
		canvasLinkSetFlags.canvasBase, canvasLinkSetFlags.canvasID, token, include, &gradeSync,
	)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"jobId": jobID, "linked": true})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Canvas link import queued: %s\n", jobID)
	return nil
}

func runCanvasLinkStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	link, raw, err := fetchCourseCanvasLink(c, canvasLinkStatusFlags.course)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "linked=%v canvasCourseId=%v gradeSync=%v\n",
		link["linked"], link["canvasCourseId"], link["gradeSyncEnabled"])
	return nil
}