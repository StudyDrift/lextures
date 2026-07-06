package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

const ferpaBulkExportWarning = `WARNING: Bulk submission download exports FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

// assignmentOverrideTarget mirrors one assign-to row from the overrides API.
type assignmentOverrideTarget struct {
	ID             string  `json:"id"`
	TargetType     string  `json:"targetType"`
	TargetID       *string `json:"targetId"`
	DueAt          *string `json:"dueAt"`
	AvailableFrom  *string `json:"availableFrom"`
	AvailableUntil *string `json:"availableUntil"`
}

type assignmentOverridesBody struct {
	Targets  []assignmentOverrideTarget `json:"targets"`
	Orphaned bool                     `json:"orphaned"`
}

type assignmentSubmissionEntry struct {
	ID                    string           `json:"id"`
	SubmittedBy           string           `json:"submittedBy"`
	SubmittedByDisplayName string          `json:"submittedByDisplayName"`
	SubmittedAt           string           `json:"submittedAt"`
	IsGraded              bool             `json:"isGraded"`
	Attachments           []map[string]any `json:"attachments"`
	AttachmentFilename    string           `json:"attachmentFilename"`
	AttachmentContentPath string           `json:"attachmentContentPath"`
	BodyText              string           `json:"bodyText"`
}

type assignmentSubmissionsBody struct {
	Submissions []assignmentSubmissionEntry `json:"submissions"`
}

type assignmentGradeHistoryEvent struct {
	ID             string   `json:"id"`
	Action         string   `json:"action"`
	PreviousScore  *float64 `json:"previousScore"`
	NewScore       *float64 `json:"newScore"`
	PreviousStatus *string  `json:"previousStatus"`
	NewStatus      *string  `json:"newStatus"`
	Reason         *string  `json:"reason"`
	ChangedAt      string   `json:"changedAt"`
	ChangedBy      *string  `json:"changedBy"`
}

type assignmentGradeHistoryBody struct {
	Events []assignmentGradeHistoryEvent `json:"events"`
}

type assignmentDownloadResult struct {
	UserID   string `json:"userId"`
	File     string `json:"file"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Skipped  bool   `json:"skipped,omitempty"`
	Error    string `json:"error,omitempty"`
}

var assignmentsUpdateFlags struct {
	course string
	title  string
	points int
	due    string
	file   string
}

var assignmentsDeleteFlags struct {
	course string
}

var assignmentsPublishFlags struct {
	course string
}

var assignmentsGradeHistoryFlags struct {
	course  string
	student string
}

var assignmentsOverridesListFlags struct {
	course string
}

var assignmentsOverridesSetFlags struct {
	course        string
	section       string
	user          string
	due           string
	availableFrom string
	availableUntil string
}

var assignmentsOverridesDeleteFlags struct {
	course  string
	section string
	user    string
}

var assignmentsSubmissionsListFlags struct {
	course string
	status string
	user   string
	late   bool
}

var assignmentsSubmissionsGetFlags struct {
	course string
	user   string
}

var assignmentsSubmissionsDownloadFlags struct {
	course       string
	out          string
	all          bool
	yes          bool
	user         string
	skipExisting bool
}

var assignmentsSubmissionsAnnotateFlags struct {
	course     string
	submission string
	body       string
	tool       string
	page       int
}

var assignmentsSubmissionsCommentFlags struct {
	course  string
	user    string
	comment string
}

var assignmentsUpdateCmd = &cobra.Command{
	Use:   "update <item_id>",
	Short: "Update an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsUpdate,
}

var assignmentsDeleteCmd = &cobra.Command{
	Use:   "delete <item_id>",
	Short: "Delete (archive) an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsDelete,
}

var assignmentsPublishCmd = &cobra.Command{
	Use:   "publish <item_id>",
	Short: "Publish an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsPublish,
}

var assignmentsUnpublishCmd = &cobra.Command{
	Use:   "unpublish <item_id>",
	Short: "Unpublish an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsUnpublish,
}

var assignmentsGradeHistoryCmd = &cobra.Command{
	Use:   "grade-history <item_id>",
	Short: "Show grade change history for a student on an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsGradeHistory,
}

var assignmentsOverridesCmd = &cobra.Command{
	Use:   "overrides",
	Short: "Manage differentiated assignment overrides",
}

var assignmentsOverridesListCmd = &cobra.Command{
	Use:   "list <item_id>",
	Short: "List assign-to overrides for an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsOverridesList,
}

var assignmentsOverridesSetCmd = &cobra.Command{
	Use:   "set <item_id>",
	Short: "Set or update an assign-to override",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsOverridesSet,
}

var assignmentsOverridesDeleteCmd = &cobra.Command{
	Use:   "delete <item_id>",
	Short: "Remove assign-to overrides",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsOverridesDelete,
}

var assignmentsSubmissionsCmd = &cobra.Command{
	Use:   "submissions",
	Short: "List, download, and annotate assignment submissions",
}

var assignmentsSubmissionsListCmd = &cobra.Command{
	Use:   "list <item_id>",
	Short: "List submissions for an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsSubmissionsList,
}

var assignmentsSubmissionsGetCmd = &cobra.Command{
	Use:   "get <item_id>",
	Short: "Get one student's submission",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsSubmissionsGet,
}

var assignmentsSubmissionsDownloadCmd = &cobra.Command{
	Use:   "download <item_id>",
	Short: "Download submission attachment(s)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsSubmissionsDownload,
}

var assignmentsSubmissionsAnnotateCmd = &cobra.Command{
	Use:   "annotate <item_id>",
	Short: "Add an annotation to a submission",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsSubmissionsAnnotate,
}

var assignmentsSubmissionsCommentCmd = &cobra.Command{
	Use:   "comment <item_id>",
	Short: "Add instructor feedback comment on a student's grade",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsSubmissionsComment,
}

func init() {
	assignmentsUpdateCmd.Flags().StringVar(&assignmentsUpdateFlags.course, "course", "", "course code (required)")
	assignmentsUpdateCmd.Flags().StringVar(&assignmentsUpdateFlags.title, "title", "", "assignment title")
	assignmentsUpdateCmd.Flags().IntVar(&assignmentsUpdateFlags.points, "points", -1, "point value")
	assignmentsUpdateCmd.Flags().StringVar(&assignmentsUpdateFlags.due, "due", "", "due date (ISO 8601)")
	assignmentsUpdateCmd.Flags().StringVar(&assignmentsUpdateFlags.file, "file", "", "Markdown body file (use - for stdin)")
	_ = assignmentsUpdateCmd.MarkFlagRequired("course")

	assignmentsDeleteCmd.Flags().StringVar(&assignmentsDeleteFlags.course, "course", "", "course code (required)")
	_ = assignmentsDeleteCmd.MarkFlagRequired("course")

	assignmentsPublishCmd.Flags().StringVar(&assignmentsPublishFlags.course, "course", "", "course code (required)")
	_ = assignmentsPublishCmd.MarkFlagRequired("course")

	assignmentsUnpublishCmd.Flags().StringVar(&assignmentsPublishFlags.course, "course", "", "course code (required)")
	_ = assignmentsUnpublishCmd.MarkFlagRequired("course")

	assignmentsGradeHistoryCmd.Flags().StringVar(&assignmentsGradeHistoryFlags.course, "course", "", "course code (required)")
	assignmentsGradeHistoryCmd.Flags().StringVar(&assignmentsGradeHistoryFlags.student, "student", "", "student user UUID (required)")
	_ = assignmentsGradeHistoryCmd.MarkFlagRequired("course")
	_ = assignmentsGradeHistoryCmd.MarkFlagRequired("student")

	assignmentsOverridesListCmd.Flags().StringVar(&assignmentsOverridesListFlags.course, "course", "", "course code (required)")
	_ = assignmentsOverridesListCmd.MarkFlagRequired("course")

	assignmentsOverridesSetCmd.Flags().StringVar(&assignmentsOverridesSetFlags.course, "course", "", "course code (required)")
	assignmentsOverridesSetCmd.Flags().StringVar(&assignmentsOverridesSetFlags.section, "section", "", "section UUID")
	assignmentsOverridesSetCmd.Flags().StringVar(&assignmentsOverridesSetFlags.user, "user", "", "student user UUID")
	assignmentsOverridesSetCmd.Flags().StringVar(&assignmentsOverridesSetFlags.due, "due", "", "due date (ISO 8601)")
	assignmentsOverridesSetCmd.Flags().StringVar(&assignmentsOverridesSetFlags.availableFrom, "available-from", "", "available from (ISO 8601)")
	assignmentsOverridesSetCmd.Flags().StringVar(&assignmentsOverridesSetFlags.availableUntil, "available-until", "", "available until (ISO 8601)")
	_ = assignmentsOverridesSetCmd.MarkFlagRequired("course")

	assignmentsOverridesDeleteCmd.Flags().StringVar(&assignmentsOverridesDeleteFlags.course, "course", "", "course code (required)")
	assignmentsOverridesDeleteCmd.Flags().StringVar(&assignmentsOverridesDeleteFlags.section, "section", "", "section UUID to remove")
	assignmentsOverridesDeleteCmd.Flags().StringVar(&assignmentsOverridesDeleteFlags.user, "user", "", "student user UUID to remove")
	_ = assignmentsOverridesDeleteCmd.MarkFlagRequired("course")

	assignmentsSubmissionsListCmd.Flags().StringVar(&assignmentsSubmissionsListFlags.course, "course", "", "course code (required)")
	assignmentsSubmissionsListCmd.Flags().StringVar(&assignmentsSubmissionsListFlags.status, "status", "", "filter: graded, ungraded, submitted, missing")
	assignmentsSubmissionsListCmd.Flags().StringVar(&assignmentsSubmissionsListFlags.user, "user", "", "filter to one student user UUID")
	assignmentsSubmissionsListCmd.Flags().BoolVar(&assignmentsSubmissionsListFlags.late, "late", false, "show only late submissions")
	_ = assignmentsSubmissionsListCmd.MarkFlagRequired("course")

	assignmentsSubmissionsGetCmd.Flags().StringVar(&assignmentsSubmissionsGetFlags.course, "course", "", "course code (required)")
	assignmentsSubmissionsGetCmd.Flags().StringVar(&assignmentsSubmissionsGetFlags.user, "user", "", "student user UUID (required)")
	_ = assignmentsSubmissionsGetCmd.MarkFlagRequired("course")
	_ = assignmentsSubmissionsGetCmd.MarkFlagRequired("user")

	assignmentsSubmissionsDownloadCmd.Flags().StringVar(&assignmentsSubmissionsDownloadFlags.course, "course", "", "course code (required)")
	assignmentsSubmissionsDownloadCmd.Flags().StringVar(&assignmentsSubmissionsDownloadFlags.out, "out", "", "output directory (required)")
	assignmentsSubmissionsDownloadCmd.Flags().BoolVar(&assignmentsSubmissionsDownloadFlags.all, "all", false, "download all student submissions")
	assignmentsSubmissionsDownloadCmd.Flags().BoolVar(&assignmentsSubmissionsDownloadFlags.yes, "yes", false, "confirm FERPA-covered bulk export")
	assignmentsSubmissionsDownloadCmd.Flags().StringVar(&assignmentsSubmissionsDownloadFlags.user, "user", "", "download one student's submission")
	assignmentsSubmissionsDownloadCmd.Flags().BoolVar(&assignmentsSubmissionsDownloadFlags.skipExisting, "skip-existing", true, "skip files that already exist")
	_ = assignmentsSubmissionsDownloadCmd.MarkFlagRequired("course")
	_ = assignmentsSubmissionsDownloadCmd.MarkFlagRequired("out")

	assignmentsSubmissionsAnnotateCmd.Flags().StringVar(&assignmentsSubmissionsAnnotateFlags.course, "course", "", "course code (required)")
	assignmentsSubmissionsAnnotateCmd.Flags().StringVar(&assignmentsSubmissionsAnnotateFlags.submission, "submission", "", "submission UUID (required)")
	assignmentsSubmissionsAnnotateCmd.Flags().StringVar(&assignmentsSubmissionsAnnotateFlags.body, "body", "", "annotation comment text (required)")
	assignmentsSubmissionsAnnotateCmd.Flags().StringVar(&assignmentsSubmissionsAnnotateFlags.tool, "tool", "text", "annotation tool: text, highlight, draw, pin, anchor")
	assignmentsSubmissionsAnnotateCmd.Flags().IntVar(&assignmentsSubmissionsAnnotateFlags.page, "page", 1, "page number for PDF annotations")
	_ = assignmentsSubmissionsAnnotateCmd.MarkFlagRequired("course")
	_ = assignmentsSubmissionsAnnotateCmd.MarkFlagRequired("submission")
	_ = assignmentsSubmissionsAnnotateCmd.MarkFlagRequired("body")

	assignmentsSubmissionsCommentCmd.Flags().StringVar(&assignmentsSubmissionsCommentFlags.course, "course", "", "course code (required)")
	assignmentsSubmissionsCommentCmd.Flags().StringVar(&assignmentsSubmissionsCommentFlags.user, "user", "", "student user UUID (required)")
	assignmentsSubmissionsCommentCmd.Flags().StringVar(&assignmentsSubmissionsCommentFlags.comment, "comment", "", "feedback comment (required)")
	_ = assignmentsSubmissionsCommentCmd.MarkFlagRequired("course")
	_ = assignmentsSubmissionsCommentCmd.MarkFlagRequired("user")
	_ = assignmentsSubmissionsCommentCmd.MarkFlagRequired("comment")

	assignmentsOverridesCmd.AddCommand(assignmentsOverridesListCmd, assignmentsOverridesSetCmd, assignmentsOverridesDeleteCmd)
	assignmentsSubmissionsCmd.AddCommand(
		assignmentsSubmissionsListCmd,
		assignmentsSubmissionsGetCmd,
		assignmentsSubmissionsDownloadCmd,
		assignmentsSubmissionsAnnotateCmd,
		assignmentsSubmissionsCommentCmd,
	)
	assignmentsCmd.AddCommand(
		assignmentsUpdateCmd,
		assignmentsDeleteCmd,
		assignmentsPublishCmd,
		assignmentsUnpublishCmd,
		assignmentsGradeHistoryCmd,
		assignmentsOverridesCmd,
		assignmentsSubmissionsCmd,
	)
}

func confirmSensitiveExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaBulkExportWarning)
}

func parseOptionalRFC3339(label, value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return &t, nil
	}
	t, err := parseAssignmentDue(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: use ISO 8601", label, value)
	}
	return &t, nil
}

func assignmentAPIPath(courseCode, suffix string) string {
	return "/api/v1/courses/" + url.PathEscape(courseCode) + suffix
}

func fetchAssignmentRaw(c *client.Client, courseCode, itemID string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, assignmentAPIPath(courseCode, "/assignments/"+url.PathEscape(itemID)), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting assignment: %w", err)
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

func patchAssignment(c *client.Client, courseCode, itemID string, patch map[string]any) ([]byte, error) {
	raw, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding patch: %w", err)
	}
	req, err := c.NewRequest(http.MethodPatch, assignmentAPIPath(courseCode, "/assignments/"+url.PathEscape(itemID)), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating assignment: %w", err)
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

func buildAssignmentPatchFromExisting(existing []byte, markdown *string, points *int, due *time.Time) (map[string]any, error) {
	var current map[string]any
	if err := json.Unmarshal(existing, &current); err != nil {
		return nil, fmt.Errorf("decoding assignment: %w", err)
	}
	patch := map[string]any{
		"markdown":             stringOrDefault(current, "markdown", ""),
		"lateSubmissionPolicy": stringOrDefault(current, "lateSubmissionPolicy", "allow"),
		"postingPolicy":        stringOrDefault(current, "postingPolicy", "automatic"),
		"blindGrading":         boolOrDefault(current, "blindGrading", false),
		"moderatedGrading":     boolOrDefault(current, "moderatedGrading", false),
		"neverDrop":            boolOrDefault(current, "neverDrop", false),
		"replaceWithFinal":     boolOrDefault(current, "replaceWithFinal", false),
	}
	if markdown != nil {
		patch["markdown"] = *markdown
	}
	if points != nil {
		patch["pointsWorth"] = *points
	}
	if due != nil {
		patch["dueAt"] = due.Format(time.RFC3339)
	} else if v, ok := current["dueAt"]; ok && v != nil {
		patch["dueAt"] = v
	}
	for _, key := range []string{
		"availableFrom", "availableUntil", "assignmentGroupId", "latePenaltyPercent",
		"originalityDetection", "originalityStudentVisibility", "gradingType", "releaseAt",
	} {
		if v, ok := current[key]; ok {
			patch[key] = v
		}
	}
	return patch, nil
}

func stringOrDefault(m map[string]any, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

func boolOrDefault(m map[string]any, key string, fallback bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return fallback
}

func fetchAssignmentOverrides(c *client.Client, courseCode, itemID string) (assignmentOverridesBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, assignmentAPIPath(courseCode, "/items/"+url.PathEscape(itemID)+"/overrides"), nil)
	if err != nil {
		return assignmentOverridesBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return assignmentOverridesBody{}, nil, fmt.Errorf("listing overrides: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return assignmentOverridesBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return assignmentOverridesBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out assignmentOverridesBody
	if err := json.Unmarshal(body, &out); err != nil {
		return assignmentOverridesBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func putAssignmentOverrides(c *client.Client, courseCode, itemID string, targets []map[string]any) ([]byte, error) {
	raw, _ := json.Marshal(map[string]any{"targets": targets})
	req, err := c.NewRequest(http.MethodPut, assignmentAPIPath(courseCode, "/items/"+url.PathEscape(itemID)+"/overrides"), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("saving overrides: %w", err)
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

type assignmentOverrideSetOpts struct {
	section        string
	user           string
	due            string
	availableFrom  string
	availableUntil string
}

func overrideTargetWriteFromFlags(flags assignmentOverrideSetOpts) (map[string]any, error) {
	section := strings.TrimSpace(flags.section)
	user := strings.TrimSpace(flags.user)
	if section != "" && user != "" {
		return nil, fmt.Errorf("specify only one of --section or --user")
	}
	targetType := "everyone"
	var targetID *string
	switch {
	case section != "":
		targetType = "section"
		targetID = &section
	case user != "":
		targetType = "student"
		targetID = &user
	}
	due, err := parseOptionalRFC3339("due date", flags.due)
	if err != nil {
		return nil, err
	}
	availableFrom, err := parseOptionalRFC3339("available-from", flags.availableFrom)
	if err != nil {
		return nil, err
	}
	availableUntil, err := parseOptionalRFC3339("available-until", flags.availableUntil)
	if err != nil {
		return nil, err
	}
	if due == nil && availableFrom == nil && availableUntil == nil {
		return nil, fmt.Errorf("provide at least one of --due, --available-from, or --available-until")
	}
	out := map[string]any{"targetType": targetType}
	if targetID != nil {
		out["targetId"] = *targetID
	}
	if due != nil {
		out["dueAt"] = due.Format(time.RFC3339)
	}
	if availableFrom != nil {
		out["availableFrom"] = availableFrom.Format(time.RFC3339)
	}
	if availableUntil != nil {
		out["availableUntil"] = availableUntil.Format(time.RFC3339)
	}
	return out, nil
}

func upsertOverrideTarget(targets []assignmentOverrideTarget, write map[string]any) []map[string]any {
	targetType, _ := write["targetType"].(string)
	targetID, _ := write["targetId"].(string)
	out := make([]map[string]any, 0, len(targets)+1)
	replaced := false
	for _, t := range targets {
		existingID := ""
		if t.TargetID != nil {
			existingID = *t.TargetID
		}
		if t.TargetType == targetType && existingID == targetID {
			out = append(out, write)
			replaced = true
			continue
		}
		entry := map[string]any{"targetType": t.TargetType}
		if t.TargetID != nil {
			entry["targetId"] = *t.TargetID
		}
		if t.DueAt != nil {
			entry["dueAt"] = *t.DueAt
		}
		if t.AvailableFrom != nil {
			entry["availableFrom"] = *t.AvailableFrom
		}
		if t.AvailableUntil != nil {
			entry["availableUntil"] = *t.AvailableUntil
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, write)
	}
	return out
}

func removeOverrideTarget(targets []assignmentOverrideTarget, targetType, targetID string) []map[string]any {
	out := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		existingID := ""
		if t.TargetID != nil {
			existingID = *t.TargetID
		}
		if targetType != "" {
			if t.TargetType == targetType && existingID == targetID {
				continue
			}
		}
		entry := map[string]any{"targetType": t.TargetType}
		if t.TargetID != nil {
			entry["targetId"] = *t.TargetID
		}
		if t.DueAt != nil {
			entry["dueAt"] = *t.DueAt
		}
		if t.AvailableFrom != nil {
			entry["availableFrom"] = *t.AvailableFrom
		}
		if t.AvailableUntil != nil {
			entry["availableUntil"] = *t.AvailableUntil
		}
		out = append(out, entry)
	}
	return out
}

func fetchAssignmentSubmissions(c *client.Client, courseCode, itemID, graded string) (assignmentSubmissionsBody, []byte, error) {
	path := assignmentAPIPath(courseCode, "/assignments/"+url.PathEscape(itemID)+"/submissions")
	if graded != "" {
		path += "?graded=" + url.QueryEscape(graded)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return assignmentSubmissionsBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return assignmentSubmissionsBody{}, nil, fmt.Errorf("listing submissions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return assignmentSubmissionsBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return assignmentSubmissionsBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out assignmentSubmissionsBody
	if err := json.Unmarshal(body, &out); err != nil {
		return assignmentSubmissionsBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func mapSubmissionStatusFilter(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "graded":
		return "graded"
	case "ungraded":
		return "ungraded"
	default:
		return ""
	}
}

func filterSubmissions(entries []assignmentSubmissionEntry, status, user string, lateOnly bool, dueAt *time.Time) []assignmentSubmissionEntry {
	user = strings.TrimSpace(user)
	status = strings.ToLower(strings.TrimSpace(status))
	var out []assignmentSubmissionEntry
	for _, entry := range entries {
		if user != "" && entry.SubmittedBy != user {
			continue
		}
		hasSubmission := strings.TrimSpace(entry.ID) != ""
		switch status {
		case "submitted":
			if !hasSubmission {
				continue
			}
		case "missing":
			if hasSubmission {
				continue
			}
		}
		if lateOnly && dueAt != nil && hasSubmission {
			submittedAt, err := time.Parse(time.RFC3339, entry.SubmittedAt)
			if err != nil || !submittedAt.After(*dueAt) {
				continue
			}
		}
		out = append(out, entry)
	}
	return out
}

func validateOutputDir(dir string) (string, error) {
	clean := filepath.Clean(dir)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid output directory: %s", dir)
	}
	if err := os.MkdirAll(clean, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}
	return clean, nil
}

func safeJoinOutput(baseDir, rel string) (string, error) {
	rel = filepath.Clean(rel)
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid output path: %s", rel)
	}
	full := filepath.Join(baseDir, rel)
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absFull, absBase+string(os.PathSeparator)) && absFull != absBase {
		return "", fmt.Errorf("invalid output path: %s", rel)
	}
	return absFull, nil
}

func downloadHTTPToFile(c *client.Client, contentPath, dest string, skipExisting bool) (int64, bool, error) {
	if skipExisting {
		if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
			return info.Size(), true, nil
		}
	}
	req, err := c.NewRequest(http.MethodGet, contentPath, nil)
	if err != nil {
		return 0, false, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, false, apiErrorBody(resp.StatusCode, body)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, false, err
	}
	f, err := os.Create(dest)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = f.Close() }()
	n, err := io.Copy(f, resp.Body)
	return n, false, err
}

func submissionDownloadJobs(entry assignmentSubmissionEntry) []struct {
	path string
	name string
} {
	var jobs []struct {
		path string
		name string
	}
	for _, att := range entry.Attachments {
		contentPath, _ := att["contentPath"].(string)
		filename, _ := att["filename"].(string)
		if contentPath == "" {
			continue
		}
		if filename == "" {
			filename = "attachment"
		}
		jobs = append(jobs, struct {
			path string
			name string
		}{path: contentPath, name: filename})
	}
	if len(jobs) == 0 && entry.AttachmentContentPath != "" {
		name := entry.AttachmentFilename
		if name == "" {
			name = "attachment"
		}
		jobs = append(jobs, struct {
			path string
			name string
		}{path: entry.AttachmentContentPath, name: name})
	}
	return jobs
}

func downloadSubmissionArchive(c *client.Client, courseCode, itemID, submissionID, dest string, skipExisting bool) (int64, bool, error) {
	if skipExisting {
		if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
			return info.Size(), true, nil
		}
	}
	path := assignmentAPIPath(courseCode, "/assignments/"+url.PathEscape(itemID)+"/submissions/"+url.PathEscape(submissionID)+"/attachments/archive")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return 0, false, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, false, apiErrorBody(resp.StatusCode, body)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, false, err
	}
	f, err := os.Create(dest)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = f.Close() }()
	n, err := io.Copy(f, resp.Body)
	return n, false, err
}

func runAssignmentsUpdate(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := assignmentsUpdateFlags.course

	if assignmentsUpdateFlags.title != "" {
		if err := patchStructureItem(c, courseCode, itemID, structureItemPatchOpts{
			title: &assignmentsUpdateFlags.title,
		}); err != nil {
			return err
		}
	}

	needsAssignmentPatch := assignmentsUpdateFlags.file != "" ||
		assignmentsUpdateFlags.points >= 0 ||
		strings.TrimSpace(assignmentsUpdateFlags.due) != ""
	if !needsAssignmentPatch {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"id": itemID})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated assignment %s\n", itemID)
		return nil
	}

	existing, err := fetchAssignmentRaw(c, courseCode, itemID)
	if err != nil {
		return err
	}
	var markdown *string
	if assignmentsUpdateFlags.file != "" {
		content, err := readInputFile(assignmentsUpdateFlags.file)
		if err != nil {
			return fmt.Errorf("reading assignment file: %w", err)
		}
		s := string(content)
		markdown = &s
	}
	var points *int
	if assignmentsUpdateFlags.points >= 0 {
		points = &assignmentsUpdateFlags.points
	}
	var due *time.Time
	if strings.TrimSpace(assignmentsUpdateFlags.due) != "" {
		t, err := parseAssignmentDue(assignmentsUpdateFlags.due)
		if err != nil {
			return err
		}
		due = &t
	}
	patch, err := buildAssignmentPatchFromExisting(existing, markdown, points, due)
	if err != nil {
		return err
	}
	body, err := patchAssignment(c, courseCode, itemID, patch)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated assignment %s\n", itemID)
	return nil
}

func runAssignmentsDelete(cmd *cobra.Command, args []string) error {
	if err := deleteStructureItem(client.New(Cfg.Server, Cfg.APIKey), assignmentsDeleteFlags.course, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"id": args[0], "deleted": "true"})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted assignment %s\n", args[0])
	return nil
}

func runAssignmentsPublish(cmd *cobra.Command, args []string) error {
	pub := true
	if err := patchStructureItem(client.New(Cfg.Server, Cfg.APIKey), assignmentsPublishFlags.course, args[0], structureItemPatchOpts{published: &pub}); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"id": args[0], "published": true})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Published assignment %s\n", args[0])
	return nil
}

func runAssignmentsUnpublish(cmd *cobra.Command, args []string) error {
	pub := false
	if err := patchStructureItem(client.New(Cfg.Server, Cfg.APIKey), assignmentsPublishFlags.course, args[0], structureItemPatchOpts{published: &pub}); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"id": args[0], "published": false})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Unpublished assignment %s\n", args[0])
	return nil
}

func runAssignmentsGradeHistory(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	path := assignmentAPIPath(
		assignmentsGradeHistoryFlags.course,
		"/assignments/"+url.PathEscape(itemID)+"/grades/"+url.PathEscape(assignmentsGradeHistoryFlags.student)+"/history",
	)
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return fmt.Errorf("getting grade history: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var hist assignmentGradeHistoryBody
	if err := json.Unmarshal(body, &hist); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(hist.Events) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No grade history.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CHANGED_AT\tACTION\tPREVIOUS\tNEW\tREASON")
	for _, e := range hist.Events {
		prev := "-"
		if e.PreviousScore != nil {
			prev = fmt.Sprintf("%.2f", *e.PreviousScore)
		}
		newScore := "-"
		if e.NewScore != nil {
			newScore = fmt.Sprintf("%.2f", *e.NewScore)
		}
		reason := "-"
		if e.Reason != nil {
			reason = *e.Reason
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.ChangedAt, e.Action, prev, newScore, reason)
	}
	return w.Flush()
}

func runAssignmentsOverridesList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchAssignmentOverrides(client.New(Cfg.Server, Cfg.APIKey), assignmentsOverridesListFlags.course, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Targets) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No overrides.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tTARGET\tDUE")
	for _, t := range body.Targets {
		target := "-"
		if t.TargetID != nil {
			target = *t.TargetID
		}
		due := "-"
		if t.DueAt != nil {
			due = *t.DueAt
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.TargetType, target, due)
	}
	return w.Flush()
}

func runAssignmentsOverridesSet(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := assignmentsOverridesSetFlags.course

	current, _, err := fetchAssignmentOverrides(c, courseCode, itemID)
	if err != nil {
		return err
	}
	write, err := overrideTargetWriteFromFlags(assignmentOverrideSetOpts{
		section:        assignmentsOverridesSetFlags.section,
		user:           assignmentsOverridesSetFlags.user,
		due:            assignmentsOverridesSetFlags.due,
		availableFrom:  assignmentsOverridesSetFlags.availableFrom,
		availableUntil: assignmentsOverridesSetFlags.availableUntil,
	})
	if err != nil {
		return err
	}
	targets := upsertOverrideTarget(current.Targets, write)
	body, err := putAssignmentOverrides(c, courseCode, itemID, targets)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Saved override for assignment %s\n", itemID)
	return nil
}

func runAssignmentsOverridesDelete(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := assignmentsOverridesDeleteFlags.course

	current, _, err := fetchAssignmentOverrides(c, courseCode, itemID)
	if err != nil {
		return err
	}
	section := strings.TrimSpace(assignmentsOverridesDeleteFlags.section)
	user := strings.TrimSpace(assignmentsOverridesDeleteFlags.user)
	if section != "" && user != "" {
		return fmt.Errorf("specify only one of --section or --user")
	}
	var targets []map[string]any
	switch {
	case section != "":
		targets = removeOverrideTarget(current.Targets, "section", section)
	case user != "":
		targets = removeOverrideTarget(current.Targets, "student", user)
	default:
		targets = []map[string]any{}
	}
	body, err := putAssignmentOverrides(c, courseCode, itemID, targets)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed override(s) for assignment %s\n", itemID)
	return nil
}

func runAssignmentsSubmissionsList(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := assignmentsSubmissionsListFlags.course

	graded := mapSubmissionStatusFilter(assignmentsSubmissionsListFlags.status)
	body, raw, err := fetchAssignmentSubmissions(c, courseCode, itemID, graded)
	if err != nil {
		return err
	}
	var dueAt *time.Time
	if assignmentsSubmissionsListFlags.late {
		existing, err := fetchAssignmentRaw(c, courseCode, itemID)
		if err != nil {
			return err
		}
		var current map[string]any
		_ = json.Unmarshal(existing, &current)
		if v, ok := current["dueAt"].(string); ok && v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				dueAt = &t
			}
		}
	}
	entries := filterSubmissions(
		body.Submissions,
		assignmentsSubmissionsListFlags.status,
		assignmentsSubmissionsListFlags.user,
		assignmentsSubmissionsListFlags.late,
		dueAt,
	)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"submissions": entries})
	}
	_ = raw
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No submissions.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "USER\tNAME\tSUBMITTED\tGRADED\tSUBMISSION_ID")
	for _, e := range entries {
		submitted := "-"
		if e.SubmittedAt != "" {
			submitted = e.SubmittedAt
		}
		name := e.SubmittedByDisplayName
		if name == "" {
			name = e.SubmittedBy
		}
		subID := e.ID
		if subID == "" {
			subID = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", e.SubmittedBy, name, submitted, e.IsGraded, subID)
	}
	return w.Flush()
}

func runAssignmentsSubmissionsGet(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchAssignmentSubmissions(
		client.New(Cfg.Server, Cfg.APIKey),
		assignmentsSubmissionsGetFlags.course,
		args[0],
		"",
	)
	if err != nil {
		return err
	}
	entries := filterSubmissions(body.Submissions, "", assignmentsSubmissionsGetFlags.user, false, nil)
	if len(entries) == 0 {
		return fmt.Errorf("no submission found for user %s", assignmentsSubmissionsGetFlags.user)
	}
	entry := entries[0]
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(entry)
	}
	_ = raw
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "User:        %s\n", entry.SubmittedBy)
	if entry.SubmittedByDisplayName != "" {
		_, _ = fmt.Fprintf(out, "Name:        %s\n", entry.SubmittedByDisplayName)
	}
	if entry.ID != "" {
		_, _ = fmt.Fprintf(out, "Submission:  %s\n", entry.ID)
		_, _ = fmt.Fprintf(out, "Submitted:   %s\n", entry.SubmittedAt)
		_, _ = fmt.Fprintf(out, "Graded:      %v\n", entry.IsGraded)
		if entry.AttachmentFilename != "" {
			_, _ = fmt.Fprintf(out, "Attachment:  %s\n", entry.AttachmentFilename)
		}
	} else {
		_, _ = fmt.Fprintln(out, "Status:      missing")
	}
	if entry.BodyText != "" {
		_, _ = fmt.Fprintln(out, "---")
		_, _ = fmt.Fprintln(out, entry.BodyText)
	}
	return nil
}

func runAssignmentsSubmissionsDownload(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := assignmentsSubmissionsDownloadFlags.course

	if assignmentsSubmissionsDownloadFlags.all {
		if err := confirmSensitiveExport(assignmentsSubmissionsDownloadFlags.yes); err != nil {
			return err
		}
	}
	if assignmentsSubmissionsDownloadFlags.all && assignmentsSubmissionsDownloadFlags.user != "" {
		return fmt.Errorf("use either --all or --user, not both")
	}
	if !assignmentsSubmissionsDownloadFlags.all && strings.TrimSpace(assignmentsSubmissionsDownloadFlags.user) == "" {
		return fmt.Errorf("specify --user <uuid> or --all")
	}

	outDir, err := validateOutputDir(assignmentsSubmissionsDownloadFlags.out)
	if err != nil {
		return err
	}

	body, _, err := fetchAssignmentSubmissions(c, courseCode, itemID, "")
	if err != nil {
		return err
	}
	entries := body.Submissions
	if assignmentsSubmissionsDownloadFlags.user != "" {
		entries = filterSubmissions(entries, "submitted", assignmentsSubmissionsDownloadFlags.user, false, nil)
	} else {
		var submitted []assignmentSubmissionEntry
		for _, e := range entries {
			if strings.TrimSpace(e.ID) != "" {
				submitted = append(submitted, e)
			}
		}
		entries = submitted
	}

	type job struct {
		userID       string
		submissionID string
		relPath      string
		contentPath  string
		useArchive   bool
	}
	var jobs []job
	for _, entry := range entries {
		userLabel := entry.SubmittedBy
		if entry.SubmittedByDisplayName != "" {
			userLabel = sanitizeFilename(entry.SubmittedByDisplayName)
		}
		attachmentJobs := submissionDownloadJobs(entry)
		if len(attachmentJobs) > 1 {
			rel := filepath.Join(userLabel, "attachments.zip")
			jobs = append(jobs, job{
				userID:       entry.SubmittedBy,
				submissionID: entry.ID,
				relPath:      rel,
				useArchive:   true,
			})
			continue
		}
		for _, att := range attachmentJobs {
			rel := filepath.Join(userLabel, sanitizeFilename(att.name))
			jobs = append(jobs, job{
				userID:       entry.SubmittedBy,
				submissionID: entry.ID,
				relPath:      rel,
				contentPath:  att.path,
			})
		}
	}

	results := make([]assignmentDownloadResult, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, jb job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			dest, err := safeJoinOutput(outDir, jb.relPath)
			if err != nil {
				results[idx] = assignmentDownloadResult{UserID: jb.userID, File: jb.relPath, Error: err.Error()}
				return
			}
			var n int64
			var skipped bool
			if jb.useArchive {
				n, skipped, err = downloadSubmissionArchive(c, courseCode, itemID, jb.submissionID, dest, assignmentsSubmissionsDownloadFlags.skipExisting)
			} else {
				n, skipped, err = downloadHTTPToFile(c, jb.contentPath, dest, assignmentsSubmissionsDownloadFlags.skipExisting)
			}
			if err != nil {
				results[idx] = assignmentDownloadResult{UserID: jb.userID, File: jb.relPath, Error: err.Error()}
				return
			}
			results[idx] = assignmentDownloadResult{
				UserID:  jb.userID,
				File:    jb.relPath,
				Path:    dest,
				Bytes:   n,
				Skipped: skipped,
			}
		}(i, j)
	}
	wg.Wait()

	ok, failed, skipped := 0, 0, 0
	for _, r := range results {
		switch {
		case r.Error != "":
			failed++
		case r.Skipped:
			skipped++
		default:
			ok++
		}
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"results": results,
			"summary": map[string]int{"ok": ok, "skipped": skipped, "failed": failed},
		})
	}
	out := cmd.OutOrStdout()
	for _, r := range results {
		switch {
		case r.Error != "":
			_, _ = fmt.Fprintf(out, "FAIL %s (%s): %s\n", r.File, r.UserID, r.Error)
		case r.Skipped:
			_, _ = fmt.Fprintf(out, "SKIP %s (%s)\n", r.File, r.UserID)
		default:
			_, _ = fmt.Fprintf(out, "OK   %s (%s, %s)\n", r.File, r.UserID, formatFileSize(r.Bytes))
		}
	}
	_, _ = fmt.Fprintf(out, "\nDownloaded %d file(s), skipped %d, failed %d\n", ok, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d download(s) failed", failed)
	}
	return nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "file"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(name)
}

func runAssignmentsSubmissionsAnnotate(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	tool := strings.TrimSpace(assignmentsSubmissionsAnnotateFlags.tool)
	if tool == "" {
		tool = "text"
	}
	clientID := fmt.Sprintf("cli-%d", time.Now().UnixNano())
	bodyText := assignmentsSubmissionsAnnotateFlags.body
	payload := map[string]any{
		"clientId": clientID,
		"page":     assignmentsSubmissionsAnnotateFlags.page,
		"toolType": tool,
		"colour":   "#FFFF00",
		"coordsJson": map[string]any{
			"x": 0, "y": 0, "w": 0, "h": 0,
		},
		"body": bodyText,
	}
	raw, _ := json.Marshal(payload)
	path := assignmentAPIPath(
		assignmentsSubmissionsAnnotateFlags.course,
		"/assignments/"+url.PathEscape(itemID)+"/submissions/"+url.PathEscape(assignmentsSubmissionsAnnotateFlags.submission)+"/annotations",
	)
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return fmt.Errorf("creating annotation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Annotation saved.")
	return nil
}

func runAssignmentsSubmissionsComment(cmd *cobra.Command, args []string) error {
	itemID := args[0]
	payload, _ := json.Marshal(map[string]string{
		"instructorComment": assignmentsSubmissionsCommentFlags.comment,
	})
	path := assignmentAPIPath(
		assignmentsSubmissionsCommentFlags.course,
		"/assignments/"+url.PathEscape(itemID)+"/students/"+url.PathEscape(assignmentsSubmissionsCommentFlags.user)+"/grade",
	)
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return fmt.Errorf("adding comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Comment saved.")
	return nil
}