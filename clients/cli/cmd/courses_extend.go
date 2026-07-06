package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// courseFeatures mirrors PATCH /features body and GET course feature fields.
type courseFeatures struct {
	NotebookEnabled               bool  `json:"notebookEnabled"`
	FeedEnabled                   bool  `json:"feedEnabled"`
	CalendarEnabled               bool  `json:"calendarEnabled"`
	QuestionBankEnabled           bool  `json:"questionBankEnabled"`
	LockdownModeEnabled           bool  `json:"lockdownModeEnabled"`
	StandardsAlignmentEnabled     *bool `json:"standardsAlignmentEnabled,omitempty"`
	AdaptivePathsEnabled          *bool `json:"adaptivePathsEnabled,omitempty"`
	SRSEnabled                    *bool `json:"srsEnabled,omitempty"`
	DiagnosticAssessmentsEnabled  *bool `json:"diagnosticAssessmentsEnabled,omitempty"`
	HintScaffoldingEnabled        *bool `json:"hintScaffoldingEnabled,omitempty"`
	MisconceptionDetectionEnabled *bool `json:"misconceptionDetectionEnabled,omitempty"`
	SectionsEnabled               *bool `json:"sectionsEnabled,omitempty"`
	DiscussionsEnabled            bool  `json:"discussionsEnabled"`
	CollabDocsEnabled             *bool `json:"collabDocsEnabled,omitempty"`
	LiveSessionsEnabled           *bool `json:"liveSessionsEnabled,omitempty"`
	GroupSpacesEnabled            *bool `json:"groupSpacesEnabled,omitempty"`
	OfficeHoursEnabled            *bool `json:"officeHoursEnabled,omitempty"`
	AiTutorEnabled                *bool `json:"aiTutorEnabled,omitempty"`
	MultilingualMessagingEnabled  *bool `json:"multilingualMessagingEnabled,omitempty"`
	FilesEnabled                  *bool `json:"filesEnabled,omitempty"`
	AttendanceEnabled             *bool `json:"attendanceEnabled,omitempty"`
	WhiteboardEnabled             *bool `json:"whiteboardEnabled,omitempty"`
	ReportCardsEnabled            *bool `json:"reportCardsEnabled,omitempty"`
}

// courseDetail is the GET /courses/{code} payload used for merge updates.
type courseDetail struct {
	coursePublic
	Description         string     `json:"description"`
	ScheduleMode        string     `json:"scheduleMode"`
	VisibleFrom         *time.Time `json:"visibleFrom"`
	HiddenAt            *time.Time `json:"hiddenAt"`
	RelativeEndAfter    *string    `json:"relativeEndAfter"`
	RelativeHiddenAfter *string    `json:"relativeHiddenAfter"`
	courseFeatures
}

type syllabusResponse struct {
	Sections                  []json.RawMessage `json:"sections"`
	UpdatedAt                 string            `json:"updatedAt"`
	RequireSyllabusAcceptance bool              `json:"requireSyllabusAcceptance"`
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// readInputFile reads payload bytes from a path or stdin when path is "-".
func readInputFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func fetchCourseDetail(c *client.Client, code string) (courseDetail, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+code, nil)
	if err != nil {
		return courseDetail{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return courseDetail{}, nil, fmt.Errorf("getting course: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return courseDetail{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return courseDetail{}, body, apiErrorBody(resp.StatusCode, body)
	}
	if resp.StatusCode != http.StatusOK {
		return courseDetail{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var co courseDetail
	if err := json.Unmarshal(body, &co); err != nil {
		return courseDetail{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return co, body, nil
}

func buildPutCourseBody(co courseDetail, overrides coursesUpdateOpts) map[string]any {
	mode := co.ScheduleMode
	if mode == "" {
		mode = "fixed"
	}
	body := map[string]any{
		"title":               co.Title,
		"description":         co.Description,
		"published":           co.Published,
		"scheduleMode":        mode,
		"startsAt":            co.StartsAt,
		"endsAt":              co.EndsAt,
		"visibleFrom":         co.VisibleFrom,
		"hiddenAt":            co.HiddenAt,
		"relativeEndAfter":    co.RelativeEndAfter,
		"relativeHiddenAfter": co.RelativeHiddenAfter,
	}
	if overrides.title != "" {
		body["title"] = overrides.title
	}
	if overrides.description != "" {
		body["description"] = overrides.description
	}
	if overrides.startsAt != "" {
		t, err := time.Parse(time.RFC3339, overrides.startsAt)
		if err != nil {
			return nil
		}
		body["startsAt"] = t
	}
	if overrides.endsAt != "" {
		t, err := time.Parse(time.RFC3339, overrides.endsAt)
		if err != nil {
			return nil
		}
		body["endsAt"] = t
	}
	if overrides.visibleFrom != "" {
		t, err := time.Parse(time.RFC3339, overrides.visibleFrom)
		if err != nil {
			return nil
		}
		body["visibleFrom"] = t
	}
	if overrides.hiddenAt != "" {
		t, err := time.Parse(time.RFC3339, overrides.hiddenAt)
		if err != nil {
			return nil
		}
		body["hiddenAt"] = t
	}
	if overrides.term != "" {
		if overrides.term == "none" {
			body["termId"] = nil
		} else {
			body["termId"] = overrides.term
		}
	}
	return body
}

func putCourse(c *client.Client, code string, body map[string]any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	req, err := c.NewRequest(http.MethodPut, "/api/v1/courses/"+code, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating course: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	if resp.StatusCode != http.StatusOK {
		return respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func emitRawOrMessage(cmd *cobra.Command, raw []byte, human string) error {
	if globalFlags.jsonOut {
		_, err := cmd.OutOrStdout().Write(raw)
		return err
	}
	if human != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), human)
	}
	return nil
}

// --- courses update ---

type coursesUpdateOpts struct {
	title       string
	description string
	startsAt    string
	endsAt      string
	visibleFrom string
	hiddenAt    string
	term        string
}

var coursesUpdateFlags coursesUpdateOpts

var coursesUpdateCmd = &cobra.Command{
	Use:   "update <course_code>",
	Short: "Update course metadata (title, dates, term, visibility)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesUpdate,
}

func init() {
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.title, "title", "", "course title")
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.description, "description", "", "course description")
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.startsAt, "starts-at", "", "start time (RFC3339)")
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.endsAt, "ends-at", "", "end time (RFC3339)")
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.visibleFrom, "visible-from", "", "visibility start (RFC3339)")
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.hiddenAt, "hidden-at", "", "visibility end (RFC3339)")
	coursesUpdateCmd.Flags().StringVar(&coursesUpdateFlags.term, "term", "", "term UUID (use 'none' to clear)")
	coursesUpdateCmd.Flags().Bool("published", false, "set published state (pass --published=true or --published=false)")
}

func runCoursesUpdate(cmd *cobra.Command, args []string) error {
	code := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)

	co, _, err := fetchCourseDetail(c, code)
	if err != nil {
		return err
	}

	body := buildPutCourseBody(co, coursesUpdateFlags)
	if body == nil {
		return fmt.Errorf("invalid date flag: use RFC3339 format")
	}
	if cmd.Flags().Changed("published") {
		pub, _ := cmd.Flags().GetBool("published")
		body["published"] = pub
	}

	respBody, err := putCourse(c, code, body)
	if err != nil {
		return err
	}

	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	var updated coursePublic
	if json.Unmarshal(respBody, &updated) == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated course %s\n", updated.CourseCode)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated course %s\n", code)
	}
	return nil
}

// --- courses publish / unpublish ---

var coursesPublishCmd = &cobra.Command{
	Use:   "publish <course_code>",
	Short: "Publish a course (idempotent)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesPublish,
}

var coursesUnpublishCmd = &cobra.Command{
	Use:   "unpublish <course_code>",
	Short: "Unpublish a course (idempotent)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesUnpublish,
}

func runCoursesSetPublished(cmd *cobra.Command, code string, want bool, verb string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	co, _, err := fetchCourseDetail(c, code)
	if err != nil {
		return err
	}
	if co.Published == want {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"courseCode": code,
				"published":  want,
				"changed":    false,
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Course %s already %s\n", code, verb)
		return nil
	}
	body := buildPutCourseBody(co, coursesUpdateOpts{})
	body["published"] = want
	respBody, err := putCourse(c, code, body)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		var out map[string]any
		if json.Unmarshal(respBody, &out) == nil {
			out["changed"] = true
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s course %s\n", capitalize(verb), code)
	return nil
}

func runCoursesPublish(cmd *cobra.Command, args []string) error {
	return runCoursesSetPublished(cmd, args[0], true, "published")
}

func runCoursesUnpublish(cmd *cobra.Command, args []string) error {
	return runCoursesSetPublished(cmd, args[0], false, "unpublished")
}

// --- courses restore ---

var coursesRestoreCmd = &cobra.Command{
	Use:   "restore <course_code>",
	Short: "Restore an archived course",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesRestore,
}

func runCoursesRestore(cmd *cobra.Command, args []string) error {
	code := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)

	body, _ := json.Marshal(map[string]bool{"archived": false})
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/courses/"+code+"/archived", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("restoring course: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return apiError(resp, 2)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	return emitRawOrMessage(cmd, respBody, fmt.Sprintf("Restored course %s", code))
}

// --- courses clone ---

var coursesCloneFlags struct {
	toTerm string
	name   string
}

var coursesCloneCmd = &cobra.Command{
	Use:   "clone <course_code>",
	Short: "Clone a course into a new course (optionally assign a term)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesClone,
}

func init() {
	coursesCloneCmd.Flags().StringVar(&coursesCloneFlags.toTerm, "to-term", "", "term UUID for the new course")
	coursesCloneCmd.Flags().StringVar(&coursesCloneFlags.name, "name", "", "title for the new course (defaults to source title)")
}

func runCoursesClone(cmd *cobra.Command, args []string) error {
	source := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)

	src, _, err := fetchCourseDetail(c, source)
	if err != nil {
		return err
	}
	title := coursesCloneFlags.name
	if title == "" {
		title = src.Title
	}

	cloneBody := map[string]any{
		"sourceCourseCode": source,
		"title":            title,
		"include":          map[string]bool{},
	}
	raw, err := json.Marshal(cloneBody)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/import/from-course", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("cloning course: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return apiError(resp, 2)
	}

	var created coursePublic
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if coursesCloneFlags.toTerm != "" {
		co := courseDetail{coursePublic: created, Description: src.Description, ScheduleMode: src.ScheduleMode}
		if co.ScheduleMode == "" {
			co.ScheduleMode = "fixed"
		}
		flags := coursesUpdateOpts{term: coursesCloneFlags.toTerm}
		putBody := buildPutCourseBody(co, flags)
		respBody, err = putCourse(c, created.CourseCode, putBody)
		if err != nil {
			return err
		}
		_ = json.Unmarshal(respBody, &created)
	}

	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(respBody)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cloned %s → %s\n", source, created.CourseCode)
	return nil
}

// --- courses syllabus ---

var coursesSyllabusCmd = &cobra.Command{
	Use:   "syllabus",
	Short: "Get or set course syllabus",
}

var coursesSyllabusGetCmd = &cobra.Command{
	Use:   "get <course_code>",
	Short: "Get course syllabus",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesSyllabusGet,
}

var coursesSyllabusSetFlags struct {
	file string
}

var coursesSyllabusSetCmd = &cobra.Command{
	Use:   "set <course_code>",
	Short: "Set course syllabus from a JSON file or stdin",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesSyllabusSet,
}

func init() {
	coursesSyllabusSetCmd.Flags().StringVar(&coursesSyllabusSetFlags.file, "file", "", "JSON file path (use - for stdin)")
	_ = coursesSyllabusSetCmd.MarkFlagRequired("file")
}

func runCoursesSyllabusGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+args[0]+"/syllabus", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("getting syllabus: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var syl syllabusResponse
	if json.Unmarshal(body, &syl) == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Sections: %d  Updated: %s\n", len(syl.Sections), syl.UpdatedAt)
		return nil
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCoursesSyllabusSet(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(coursesSyllabusSetFlags.file)
	if err != nil {
		return fmt.Errorf("reading syllabus file: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/courses/"+args[0]+"/syllabus", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting syllabus: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Updated syllabus for %s", args[0]))
}

// --- courses settings ---

var coursesSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Get or set course settings (features/tools toggles)",
}

var coursesSettingsGetCmd = &cobra.Command{
	Use:   "get <course_code>",
	Short: "Get course feature settings",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesSettingsGet,
}

var coursesSettingsSetFlags struct {
	file string
}

var coursesSettingsSetCmd = &cobra.Command{
	Use:   "set <course_code>",
	Short: "Set course feature settings from a JSON file or stdin",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesSettingsSet,
}

func init() {
	coursesSettingsSetCmd.Flags().StringVar(&coursesSettingsSetFlags.file, "file", "", "JSON file path (use - for stdin)")
	_ = coursesSettingsSetCmd.MarkFlagRequired("file")
}

func runCoursesSettingsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	co, _, err := fetchCourseDetail(c, args[0])
	if err != nil {
		return err
	}
	out := co.courseFeatures
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Notebook: %v  Feed: %v  Calendar: %v  Discussions: %v\n",
		out.NotebookEnabled, out.FeedEnabled, out.CalendarEnabled, out.DiscussionsEnabled)
	return nil
}

func runCoursesSettingsSet(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(coursesSettingsSetFlags.file)
	if err != nil {
		return fmt.Errorf("reading settings file: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/courses/"+args[0]+"/features", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting course features: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Updated settings for %s", args[0]))
}

// --- courses hero-image ---

var coursesHeroImageCmd = &cobra.Command{
	Use:   "hero-image",
	Short: "Manage course hero image",
}

var coursesHeroImageSetCmd = &cobra.Command{
	Use:   "set <course_code> <path-or-url>",
	Short: "Set hero image from a URL or JSON file",
	Args:  cobra.ExactArgs(2),
	RunE:  runCoursesHeroImageSet,
}

func runCoursesHeroImageSet(cmd *cobra.Command, args []string) error {
	code, pathOrURL := args[0], args[1]
	var raw []byte
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") ||
		strings.HasPrefix(pathOrURL, "/api/") {
		payload := map[string]string{"imageUrl": pathOrURL}
		var err error
		raw, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
	} else {
		var err error
		raw, err = os.ReadFile(pathOrURL)
		if err != nil {
			return fmt.Errorf("reading hero image file: %w", err)
		}
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPut, "/api/v1/courses/"+code+"/hero-image", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting hero image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Updated hero image for %s", code))
}

// --- courses catalog-listing ---

var coursesCatalogListingCmd = &cobra.Command{
	Use:   "catalog-listing",
	Short: "Manage public catalog listing for a course",
}

var coursesCatalogListingGetCmd = &cobra.Command{
	Use:   "get <course_code>",
	Short: "Get catalog listing settings",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesCatalogListingGet,
}

var coursesCatalogListingSetFlags struct {
	file string
}

var coursesCatalogListingSetCmd = &cobra.Command{
	Use:   "set <course_code>",
	Short: "Set catalog listing from a JSON file or stdin",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesCatalogListingSet,
}

func init() {
	coursesCatalogListingSetCmd.Flags().StringVar(&coursesCatalogListingSetFlags.file, "file", "", "JSON file path (use - for stdin)")
	_ = coursesCatalogListingSetCmd.MarkFlagRequired("file")
}

func runCoursesCatalogListingGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+args[0]+"/catalog-listing", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("getting catalog listing: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return nil
}

func runCoursesCatalogListingSet(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(coursesCatalogListingSetFlags.file)
	if err != nil {
		return fmt.Errorf("reading catalog listing file: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPut, "/api/v1/courses/"+args[0]+"/catalog-listing", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting catalog listing: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Updated catalog listing for %s", args[0]))
}

// --- courses blueprint sync ---

var coursesBlueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Blueprint course operations",
}

var coursesBlueprintSyncCmd = &cobra.Command{
	Use:   "sync <course_code>",
	Short: "Push blueprint changes to all linked child courses",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesBlueprintSync,
}

func runCoursesBlueprintSync(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/"+args[0]+"/blueprint/push", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("blueprint sync: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var result map[string]any
	if json.Unmarshal(body, &result) == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Blueprint sync: %v children OK\n", result["childrenSuccess"])
		return nil
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

// --- courses storage-usage ---

var coursesStorageUsageCmd = &cobra.Command{
	Use:   "storage-usage <course_code>",
	Short: "Show course storage usage",
	Args:  cobra.ExactArgs(1),
	RunE:  runCoursesStorageUsage,
}

func runCoursesStorageUsage(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+args[0]+"/storage-usage", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("getting storage usage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, 2)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var usage map[string]any
	if json.Unmarshal(body, &usage) == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Used: %v bytes  Limit: %v bytes  Percent: %v%%\n",
			usage["used_bytes"], usage["limit_bytes"], usage["percent_used"])
		return nil
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func init() {
	coursesSyllabusCmd.AddCommand(coursesSyllabusGetCmd, coursesSyllabusSetCmd)
	coursesSettingsCmd.AddCommand(coursesSettingsGetCmd, coursesSettingsSetCmd)
	coursesHeroImageCmd.AddCommand(coursesHeroImageSetCmd)
	coursesCatalogListingCmd.AddCommand(coursesCatalogListingGetCmd, coursesCatalogListingSetCmd)
	coursesBlueprintCmd.AddCommand(coursesBlueprintSyncCmd)

	coursesCmd.AddCommand(
		coursesUpdateCmd,
		coursesPublishCmd,
		coursesUnpublishCmd,
		coursesRestoreCmd,
		coursesCloneCmd,
		coursesSyllabusCmd,
		coursesSettingsCmd,
		coursesHeroImageCmd,
		coursesCatalogListingCmd,
		coursesBlueprintCmd,
		coursesStorageUsageCmd,
	)
}