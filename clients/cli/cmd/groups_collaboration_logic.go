package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type groupPublic struct {
	ID          string `json:"id"`
	GroupSetID  string `json:"groupSetId"`
	Name        string `json:"name"`
	SortOrder   int    `json:"sortOrder"`
	MemberCount int64  `json:"memberCount"`
	CreatedAt   string `json:"createdAt"`
}

type groupsListBody struct {
	Groups []groupPublic `json:"groups"`
}

type enrollmentGroupPublic struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	SortOrder     int32    `json:"sortOrder"`
	EnrollmentIDs []string `json:"enrollmentIds"`
}

type enrollmentGroupSetPublic struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	SortOrder int32                   `json:"sortOrder"`
	Groups    []enrollmentGroupPublic `json:"groups"`
}

type enrollmentGroupsTreeBody struct {
	GroupSets []enrollmentGroupSetPublic `json:"groupSets"`
}

type enrollmentRow struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

type enrollmentsListBody struct {
	Enrollments []enrollmentRow `json:"enrollments"`
}

func enrollmentGroupsBase(course string) string {
	return "/api/v1/courses/" + url.PathEscape(course) + "/enrollment-groups"
}

func fetchCourseGroups(c *client.Client, course string) (groupsListBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+url.PathEscape(course)+"/groups", nil)
	if err != nil {
		return groupsListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return groupsListBody{}, nil, fmt.Errorf("listing groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return groupsListBody{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return groupsListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out groupsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return groupsListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func fetchEnrollmentGroupsTree(c *client.Client, course string) (enrollmentGroupsTreeBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, enrollmentGroupsBase(course), nil)
	if err != nil {
		return enrollmentGroupsTreeBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return enrollmentGroupsTreeBody{}, nil, fmt.Errorf("loading enrollment groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return enrollmentGroupsTreeBody{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return enrollmentGroupsTreeBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out enrollmentGroupsTreeBody
	if err := json.Unmarshal(body, &out); err != nil {
		return enrollmentGroupsTreeBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func postEnrollmentGroupSet(c *client.Client, course, name string) (string, []byte, error) {
	raw, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "", nil, err
	}
	req, err := c.NewRequest(http.MethodPost, enrollmentGroupsBase(course)+"/sets", bytes.NewReader(raw))
	if err != nil {
		return "", nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", nil, fmt.Errorf("creating group set: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", body, fmt.Errorf("decoding response: %w", err)
	}
	return out.ID, body, nil
}

func postEnrollmentGroup(c *client.Client, course, setID, name string) (string, []byte, error) {
	raw, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "", nil, err
	}
	path := enrollmentGroupsBase(course) + "/sets/" + url.PathEscape(setID) + "/groups"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return "", nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", nil, fmt.Errorf("creating group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", body, fmt.Errorf("decoding response: %w", err)
	}
	return out.ID, body, nil
}

func deleteEnrollmentGroup(c *client.Client, course, groupID string) error {
	path := enrollmentGroupsBase(course) + "/groups/" + url.PathEscape(groupID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func deleteEnrollmentGroupSet(c *client.Client, course, setID string) error {
	path := enrollmentGroupsBase(course) + "/sets/" + url.PathEscape(setID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting group set: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func putEnrollmentGroupMembership(c *client.Client, course, enrollmentID, setID, groupID string) error {
	payload := map[string]any{
		"enrollmentId": enrollmentID,
		"groupSetId":   setID,
		"groupId":      groupID,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPut, enrollmentGroupsBase(course)+"/memberships", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating membership: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func removeEnrollmentGroupMembership(c *client.Client, course, enrollmentID, setID string) error {
	payload := map[string]any{
		"enrollmentId": enrollmentID,
		"groupSetId":   setID,
		"groupId":      nil,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPut, enrollmentGroupsBase(course)+"/memberships", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("removing membership: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func findGroupInTree(tree enrollmentGroupsTreeBody, groupID string) (*enrollmentGroupPublic, *enrollmentGroupSetPublic) {
	for i := range tree.GroupSets {
		set := &tree.GroupSets[i]
		for j := range set.Groups {
			if set.Groups[j].ID == groupID {
				return &set.Groups[j], set
			}
		}
	}
	return nil, nil
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
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
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

func studentEnrollmentIDs(rows []enrollmentRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		role := strings.ToLower(strings.TrimSpace(row.Role))
		if role == "student" {
			out = append(out, row.ID)
		}
	}
	return out
}

func enrollmentIDForUser(rows []enrollmentRow, userID string) (string, error) {
	for _, row := range rows {
		if row.UserID == userID {
			return row.ID, nil
		}
	}
	return "", fmt.Errorf("user %q is not enrolled in this course", userID)
}

// planAutoAssignGroups returns group count and deterministic student order for auto-assignment.
func planAutoAssignGroups(studentIDs []string, size int, seed int64) (groupCount int, ordered []string) {
	if size < 1 {
		size = 1
	}
	ordered = append([]string(nil), studentIDs...)
	if seed != 0 {
		rng := rand.New(rand.NewSource(seed))
		rng.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })
	} else {
		sort.Strings(ordered)
	}
	groupCount = int(math.Ceil(float64(len(ordered)) / float64(size)))
	if groupCount < 1 {
		groupCount = 1
	}
	return groupCount, ordered
}

func assignStudentsRoundRobin(c *client.Client, course, setID string, groupIDs, enrollmentIDs []string) error {
	for i, enrollmentID := range enrollmentIDs {
		groupID := groupIDs[i%len(groupIDs)]
		if err := putEnrollmentGroupMembership(c, course, enrollmentID, setID, groupID); err != nil {
			return fmt.Errorf("row %d: %w", i+1, err)
		}
	}
	return nil
}