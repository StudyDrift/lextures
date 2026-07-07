package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const (
	ferpaRosterExportWarning = `WARNING: Roster export contains FERPA-covered student records (names, emails, SIS ids).
Re-run with --yes to confirm you are authorized to export this data.`

	defaultEnrollmentImportChunk = 50
)

type enrollmentRow struct {
	ID                string  `json:"id"`
	UserID            string  `json:"userId"`
	DisplayName       *string `json:"displayName"`
	Role              string  `json:"role"`
	SectionID         *string `json:"sectionId"`
	SectionCode       *string `json:"sectionCode"`
	SectionName       *string `json:"sectionName"`
	State             *string `json:"state"`
	InvitationPending bool    `json:"invitationPending"`
}

type enrollmentsListBody struct {
	Enrollments []enrollmentRow `json:"enrollments"`
}

type addEnrollmentsResponse struct {
	Added           []string `json:"added"`
	AlreadyEnrolled []string `json:"alreadyEnrolled"`
	NotFound        []string `json:"notFound"`
}

type enrollmentStatePatchResponse struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type sectionRow struct {
	ID          string  `json:"id"`
	SectionCode string  `json:"sectionCode"`
	Name        *string `json:"name"`
	Status      string  `json:"status"`
	Capacity    *int    `json:"capacity"`
}

type sectionsListBody struct {
	Sections []sectionRow `json:"sections"`
}

type rosterImportRow struct {
	Email      string
	Role       string
	Section    string
	LineNumber int
}

type enrollmentImportSummary struct {
	Added    int      `json:"added"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	NotFound int      `json:"notFound"`
	Errors   []string `json:"errors,omitempty"`
}

type crossListGroup struct {
	ID               string `json:"id"`
	CourseID         string `json:"courseId"`
	Name             *string `json:"name"`
	PrimarySectionID *string `json:"primarySectionId"`
}

type crossListGroupsBody struct {
	Groups []crossListGroup `json:"groups"`
}

func confirmRosterExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaRosterExportWarning)
}

func normalizeEnrollmentStateAlias(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "active", "waitlist", "dropped", "withdrawn", "audit", "no_credit", "incomplete":
		return s, nil
	case "concluded", "conclude":
		return "withdrawn", nil
	case "deactivated", "deactivate", "inactive":
		return "dropped", nil
	case "reactivate", "reactivated":
		return "active", nil
	case "invited":
		return "", fmt.Errorf("invitation state is managed by the enrollment invitation flow, not set-state")
	default:
		return "", fmt.Errorf("invalid enrollment state %q (use active, withdrawn, dropped, concluded, deactivated, or reactivate)", raw)
	}
}

func parseRosterCSV(raw []byte, defaultRole string) ([]rosterImportRow, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	header := make([]string, len(records[0]))
	for i, col := range records[0] {
		header[i] = strings.ToLower(strings.TrimSpace(col))
	}
	emailIdx, sisIdx, roleIdx, sectionIdx := -1, -1, -1, -1
	for i, col := range header {
		switch col {
		case "email", "email_address", "email address":
			emailIdx = i
		case "sis_id", "sisid", "sis id", "sis_user_id":
			sisIdx = i
		case "role", "course_role", "course role":
			roleIdx = i
		case "section", "section_code", "section code":
			sectionIdx = i
		}
	}
	if emailIdx < 0 && sisIdx < 0 {
		return nil, fmt.Errorf("CSV must include an email or sis_id column")
	}

	start := 0
	if emailIdx >= 0 || sisIdx >= 0 || roleIdx >= 0 || sectionIdx >= 0 {
		start = 1
	}

	var rows []rosterImportRow
	for line := start; line < len(records); line++ {
		rec := records[line]
		if len(rec) == 0 || strings.TrimSpace(strings.Join(rec, "")) == "" {
			continue
		}
		email := ""
		if emailIdx >= 0 && emailIdx < len(rec) {
			email = strings.TrimSpace(rec[emailIdx])
		}
		if email == "" && sisIdx >= 0 && sisIdx < len(rec) {
			email = strings.TrimSpace(rec[sisIdx])
			if email != "" && !strings.Contains(email, "@") {
				return nil, fmt.Errorf("line %d: sis_id lookup is not supported yet; provide email", line+1)
			}
		}
		if email == "" {
			return nil, fmt.Errorf("line %d: missing email", line+1)
		}
		role := strings.TrimSpace(defaultRole)
		if roleIdx >= 0 && roleIdx < len(rec) && strings.TrimSpace(rec[roleIdx]) != "" {
			role = strings.TrimSpace(rec[roleIdx])
		}
		if role == "" {
			return nil, fmt.Errorf("line %d: role is required (set --role or include a role column)", line+1)
		}
		section := ""
		if sectionIdx >= 0 && sectionIdx < len(rec) {
			section = strings.TrimSpace(rec[sectionIdx])
		}
		rows = append(rows, rosterImportRow{
			Email:      strings.ToLower(email),
			Role:       strings.ToLower(role),
			Section:    section,
			LineNumber: line + 1,
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV contains no data rows")
	}
	return rows, nil
}

func chunkStrings(items []string, size int) [][]string {
	if size < 1 {
		size = defaultEnrollmentImportChunk
	}
	var chunks [][]string
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func filterEnrollments(rows []enrollmentRow, role, section, state string) []enrollmentRow {
	role = strings.ToLower(strings.TrimSpace(role))
	section = strings.TrimSpace(section)
	state = strings.ToLower(strings.TrimSpace(state))
	if role == "" && section == "" && state == "" {
		return rows
	}
	out := make([]enrollmentRow, 0, len(rows))
	for _, row := range rows {
		if role != "" && !strings.EqualFold(row.Role, role) {
			continue
		}
		if section != "" {
			match := false
			if row.SectionID != nil && strings.EqualFold(*row.SectionID, section) {
				match = true
			}
			if row.SectionCode != nil && strings.EqualFold(*row.SectionCode, section) {
				match = true
			}
			if !match {
				continue
			}
		}
		if state != "" {
			if state == "invited" {
				if !row.InvitationPending {
					continue
				}
			} else {
				rowState := "active"
				if row.State != nil && strings.TrimSpace(*row.State) != "" {
					rowState = strings.ToLower(strings.TrimSpace(*row.State))
				}
				if rowState != state {
					continue
				}
			}
		}
		out = append(out, row)
	}
	return out
}

func writeEnrollmentsExportCSV(w io.Writer, rows []enrollmentRow) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"enrollment_id", "user_id", "display_name", "role", "section_code", "section_id", "state", "invitation_pending",
	}); err != nil {
		return err
	}
	for _, row := range rows {
		display := ""
		if row.DisplayName != nil {
			display = *row.DisplayName
		}
		sectionCode := ""
		if row.SectionCode != nil {
			sectionCode = *row.SectionCode
		}
		sectionID := ""
		if row.SectionID != nil {
			sectionID = *row.SectionID
		}
		state := "active"
		if row.State != nil && strings.TrimSpace(*row.State) != "" {
			state = *row.State
		}
		if err := cw.Write([]string{
			row.ID, row.UserID, display, row.Role, sectionCode, sectionID, state, fmt.Sprintf("%t", row.InvitationPending),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func fetchEnrollments(c *client.Client, course string) ([]enrollmentRow, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+url.PathEscape(course)+"/enrollments", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing enrollments: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed enrollmentsListBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Enrollments, nil
}

func postEnrollments(c *client.Client, course string, emails []string, role string) (addEnrollmentsResponse, error) {
	payload := map[string]any{
		"emails":     strings.Join(emails, "\n"),
		"courseRole": role,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return addEnrollmentsResponse{}, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/enrollments"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return addEnrollmentsResponse{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return addEnrollmentsResponse{}, fmt.Errorf("enrolling users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return addEnrollmentsResponse{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return addEnrollmentsResponse{}, apiErrorBody(resp.StatusCode, body)
	}
	var out addEnrollmentsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return addEnrollmentsResponse{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func deleteEnrollment(c *client.Client, course, enrollmentID string) error {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/enrollments/" + url.PathEscape(enrollmentID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("removing enrollment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return apiErrorBody(resp.StatusCode, body)
}

func patchEnrollmentState(c *client.Client, course, enrollmentID, state, reason string) (enrollmentStatePatchResponse, error) {
	payload := map[string]any{"state": state}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = strings.TrimSpace(reason)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return enrollmentStatePatchResponse{}, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/enrollments/" + url.PathEscape(enrollmentID) + "/state"
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return enrollmentStatePatchResponse{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return enrollmentStatePatchResponse{}, fmt.Errorf("updating enrollment state: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return enrollmentStatePatchResponse{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return enrollmentStatePatchResponse{}, apiErrorBody(resp.StatusCode, body)
	}
	var out enrollmentStatePatchResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return enrollmentStatePatchResponse{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func transferEnrollmentSection(c *client.Client, enrollmentID, sectionID string) error {
	payload := map[string]string{"sectionId": sectionID}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/enrollments/" + url.PathEscape(enrollmentID) + "/section"
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("transferring section: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return apiErrorBody(resp.StatusCode, body)
}

func fetchSections(c *client.Client, course string) ([]sectionRow, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+url.PathEscape(course)+"/sections", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing sections: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed sectionsListBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Sections, nil
}

func resolveSectionRef(sections []sectionRow, ref string) (sectionRow, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return sectionRow{}, fmt.Errorf("section reference is required")
	}
	if looksLikeUUID(ref) {
		for _, sec := range sections {
			if strings.EqualFold(sec.ID, ref) {
				return sec, nil
			}
		}
		return sectionRow{}, fmt.Errorf("section %q not found", ref)
	}
	for _, sec := range sections {
		if strings.EqualFold(sec.SectionCode, ref) {
			return sec, nil
		}
	}
	return sectionRow{}, fmt.Errorf("section %q not found", ref)
}

func findEnrollmentsForUser(rows []enrollmentRow, userRef, role string) []enrollmentRow {
	userRef = strings.TrimSpace(userRef)
	role = strings.ToLower(strings.TrimSpace(role))
	var matches []enrollmentRow
	for _, row := range rows {
		if role != "" && !strings.EqualFold(row.Role, role) {
			continue
		}
		if strings.EqualFold(row.UserID, userRef) {
			matches = append(matches, row)
			continue
		}
		if row.DisplayName != nil && strings.EqualFold(strings.TrimSpace(*row.DisplayName), userRef) {
			matches = append(matches, row)
		}
	}
	return matches
}

func resolveUserEmail(c *client.Client, idOrEmail string) (string, error) {
	if !looksLikeUUID(idOrEmail) {
		return strings.TrimSpace(idOrEmail), nil
	}
	req, err := c.NewRequest(http.MethodGet, "/api/v1/users/"+url.PathEscape(idOrEmail), nil)
	if err != nil {
		return "", err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", apiErrorBody(resp.StatusCode, body)
	}
	var u userPublic
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", fmt.Errorf("decoding user: %w", err)
	}
	if strings.TrimSpace(u.Email) == "" {
		return "", fmt.Errorf("user %s has no email", idOrEmail)
	}
	return u.Email, nil
}

func importRosterRows(c *client.Client, course string, rows []rosterImportRow, chunkSize int, createMissing bool, progress func(done, total int)) (enrollmentImportSummary, error) {
	summary := enrollmentImportSummary{}
	if chunkSize < 1 {
		chunkSize = defaultEnrollmentImportChunk
	}

	byRole := map[string][]rosterImportRow{}
	for _, row := range rows {
		byRole[row.Role] = append(byRole[row.Role], row)
	}

	emailToSection := map[string]string{}
	for _, row := range rows {
		if row.Section != "" {
			emailToSection[row.Email] = row.Section
		}
	}

	total := len(rows)
	done := 0
	for role, roleRows := range byRole {
		emails := make([]string, len(roleRows))
		for i, row := range roleRows {
			emails[i] = row.Email
		}
		for _, chunk := range chunkStrings(emails, chunkSize) {
			if createMissing {
				for _, email := range chunk {
					if err := ensureUserExists(c, email, ""); err != nil {
						summary.Failed++
						summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", email, err))
					}
				}
			}
			resp, err := postEnrollments(c, course, chunk, role)
			if err != nil {
				summary.Failed += len(chunk)
				summary.Errors = append(summary.Errors, err.Error())
				done += len(chunk)
				if progress != nil {
					progress(done, total)
				}
				continue
			}
			summary.Added += len(resp.Added)
			summary.Skipped += len(resp.AlreadyEnrolled)
			summary.NotFound += len(resp.NotFound)
			done += len(chunk)
			if progress != nil {
				progress(done, total)
			}
		}
	}

	if len(emailToSection) == 0 {
		return summary, nil
	}

	sections, err := fetchSections(c, course)
	if err != nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("section assignment skipped: %v", err))
		return summary, nil
	}
	enrollments, err := fetchEnrollments(c, course)
	if err != nil {
		summary.Errors = append(summary.Errors, fmt.Sprintf("section assignment skipped: %v", err))
		return summary, nil
	}

	emailByUserID := map[string]string{}
	for email := range emailToSection {
		userID, _, err := resolveUserID(c, email)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line lookup %s: %v", email, err))
			continue
		}
		emailByUserID[userID] = email
	}

	for _, en := range enrollments {
		email, ok := emailByUserID[en.UserID]
		if !ok {
			continue
		}
		sectionRef, ok := emailToSection[email]
		if !ok || sectionRef == "" {
			continue
		}
		sec, err := resolveSectionRef(sections, sectionRef)
		if err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", email, err))
			continue
		}
		current := ""
		if en.SectionID != nil {
			current = *en.SectionID
		}
		if strings.EqualFold(current, sec.ID) {
			continue
		}
		if err := transferEnrollmentSection(c, en.ID, sec.ID); err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s section move: %v", email, err))
			continue
		}
		summary.Updated++
	}
	return summary, nil
}

func ensureUserExists(c *client.Client, email, name string) error {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/users/"+url.PathEscape(email), nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return apiErrorBody(resp.StatusCode, body)
	}
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	payload := map[string]any{"email": email, "name": name, "role": "student"}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	createReq, err := c.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	createResp, err := doWithRetry(c, createReq)
	if err != nil {
		return err
	}
	defer func() { _ = createResp.Body.Close() }()
	if createResp.StatusCode == http.StatusCreated || createResp.StatusCode == http.StatusConflict {
		return nil
	}
	body, _ := io.ReadAll(createResp.Body)
	return apiErrorBody(createResp.StatusCode, body)
}

func resolveEnrollmentForUser(c *client.Client, course, userRef, role string) (enrollmentRow, error) {
	rows, err := fetchEnrollments(c, course)
	if err != nil {
		return enrollmentRow{}, err
	}
	userID := userRef
	if !looksLikeUUID(userRef) {
		id, _, err := resolveUserID(c, userRef)
		if err != nil {
			return enrollmentRow{}, fmt.Errorf("resolving user: %w", err)
		}
		userID = id
	}
	matches := findEnrollmentsForUser(rows, userID, role)
	if len(matches) == 0 {
		return enrollmentRow{}, fmt.Errorf("no enrollment found for user %q in course %s", userRef, course)
	}
	if len(matches) > 1 && role == "" {
		return enrollmentRow{}, fmt.Errorf("user %q has multiple enrollments; pass --role to disambiguate", userRef)
	}
	return matches[0], nil
}

func postSelfEnroll(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/self-enroll"
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("self-enrolling: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchCrossListGroups(c *client.Client, orgID string) ([]crossListGroup, error) {
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/cross-list-groups"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing cross-list groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed crossListGroupsBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Groups, nil
}

func createCrossListGroup(c *client.Client, orgID, course, primarySectionID string, name *string) ([]byte, error) {
	payload := map[string]any{
		"courseCode":       course,
		"primarySectionId": primarySectionID,
	}
	if name != nil && strings.TrimSpace(*name) != "" {
		payload["name"] = strings.TrimSpace(*name)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/cross-list-groups"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("creating cross-list group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func addCrossListMember(c *client.Client, orgID, groupID, sectionID string) ([]byte, error) {
	payload := map[string]string{"sectionId": sectionID}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/cross-list-groups/" + url.PathEscape(groupID) + "/members"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("adding cross-list member: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}