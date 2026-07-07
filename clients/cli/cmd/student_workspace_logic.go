package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
)

type notebookSummary struct {
	CourseCode string `json:"courseCode"`
	UpdatedAt  string `json:"updatedAt"`
}

type notebookTaskRow struct {
	ID         string  `json:"id"`
	CourseCode string  `json:"courseCode"`
	TaskText   string  `json:"taskText"`
	Completed  bool    `json:"completed"`
	DueAt      *string `json:"dueAt"`
}

type todoPlacement struct {
	ItemKey  string `json:"itemKey"`
	ColumnID string `json:"columnId"`
}

func listNotebooks(c *client.Client) ([]notebookSummary, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/notebooks", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Notebooks []notebookSummary `json:"notebooks"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Notebooks, body, nil
}

func putNotebook(c *client.Client, course string, payload []byte) ([]byte, error) {
	path := "/api/v1/me/notebooks?courseCode=" + url.QueryEscape(course)
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func notebookFromTextFile(text string) ([]byte, error) {
	pageID := uuid.New().String()
	payload := map[string]any{
		"formatVersion": 2,
		"updatedAt":     time.Now().UTC().Format(time.RFC3339),
		"pages": []map[string]any{{
			"id":      pageID,
			"title":   "Notes",
			"content": text,
		}},
	}
	return json.Marshal(payload)
}

func listNotebookTasks(c *client.Client) ([]notebookTaskRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/notebook-tasks", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Tasks []notebookTaskRow `json:"tasks"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Tasks, body, nil
}

func upsertNotebookTask(c *client.Client, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/notebook-tasks", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func patchNotebookTask(c *client.Client, id string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/me/notebook-tasks/" + url.PathEscape(id)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func getTodoBoard(c *client.Client) (map[string][]string, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/student-todo-board", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Placements []todoPlacement `json:"placements"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	columns := map[string][]string{}
	for _, p := range out.Placements {
		columns[p.ColumnID] = append(columns[p.ColumnID], p.ItemKey)
	}
	return columns, body, nil
}

func putTodoBoard(c *client.Client, columns map[string][]string) error {
	payload, _ := json.Marshal(map[string]any{"columns": columns})
	req, err := c.NewRequest(http.MethodPut, "/api/v1/me/student-todo-board", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
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

func parseTodoDueColumn(due string, tz string) (string, error) {
	due = strings.TrimSpace(strings.ToLower(due))
	if due == "" {
		return "mon", nil
	}
	if due == "done" {
		return "done", nil
	}
	switch due {
	case "tomorrow":
		t := time.Now().Add(24 * time.Hour)
		return strings.ToLower(t.Weekday().String())[:3], nil
	case "monday", "mon", "tuesday", "tue", "wednesday", "wed", "thursday", "thu",
		"friday", "fri", "saturday", "sat", "sunday", "sun", "done":
		if len(due) > 3 {
			return due[:3], nil
		}
		return due, nil
	}
	t, err := cli.ParseRFC3339InTZ(due, tz)
	if err != nil {
		return "", err
	}
	weekday := strings.ToLower(t.Weekday().String())
	return weekday[:3], nil
}

func meGET(c *client.Client, path string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func mePATCH(c *client.Client, path string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func mePUT(c *client.Client, path string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func mePOST(c *client.Client, path string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func getGamification(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/gamification")
}

func getLeaderboard(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/leaderboard"
	return meGET(c, path)
}

func getCoachingTips(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/coaching-tips")
}

func todoItemKey(text string) string {
	return "cli:" + uuid.New().String()[:8] + ":" + strings.TrimSpace(text)
}

func ensureTodoItemKey(columns map[string][]string, id string) (string, bool) {
	for _, keys := range columns {
		for _, k := range keys {
			if k == id || strings.HasSuffix(k, ":"+id) {
				return k, true
			}
		}
	}
	return "", false
}

func removeTodoItem(columns map[string][]string, key string) {
	for col, keys := range columns {
		filtered := keys[:0]
		for _, k := range keys {
			if k != key {
				filtered = append(filtered, k)
			}
		}
		columns[col] = filtered
	}
}

func moveTodoItem(columns map[string][]string, key, dest string) {
	removeTodoItem(columns, key)
	columns[dest] = append(columns[dest], key)
}

func journalAdd(c *client.Client, text string) ([]byte, error) {
	return mePOST(c, "/api/v1/me/reflection-journal", map[string]any{"text": text})
}

func journalList(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/reflection-journal")
}

func goalsGet(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/study-goal")
}

func goalsSet(c *client.Client, payload map[string]any) ([]byte, error) {
	return mePUT(c, "/api/v1/me/study-goal", payload)
}

func remindersGet(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/reminder-config")
}

func remindersSet(c *client.Client, payload map[string]any) ([]byte, error) {
	return mePATCH(c, "/api/v1/me/reminder-config", payload)
}

func readingPrefsGet(c *client.Client) ([]byte, error) {
	return meGET(c, "/api/v1/me/reading-preferences")
}

func readingPrefsSet(c *client.Client, payload map[string]any) ([]byte, error) {
	return mePATCH(c, "/api/v1/me/reading-preferences", payload)
}

func parseNotebookDeleteCourse(args []string, flagCourse string) (string, error) {
	if flagCourse != "" {
		return flagCourse, nil
	}
	if len(args) > 0 {
		return args[0], nil
	}
	return "", fmt.Errorf("course code required")
}