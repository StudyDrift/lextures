package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const ferpaXAPIQueryWarning = `WARNING: xAPI query results may include FERPA-covered learner identity.
Re-run with --yes to confirm you are authorized to export this data.`

type xapiEventRow struct {
	StatementID string          `json:"statementId"`
	Verb        string          `json:"verb"`
	ObjectID    string          `json:"objectId"`
	ObjectTitle *string         `json:"objectTitle,omitempty"`
	StoredAt    string          `json:"storedAt"`
	FullJSON    json.RawMessage `json:"fullJson"`
}

func readNDJSONObjects(path string) ([]json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	defer func() { _ = f.Close() }()
	var out []json.RawMessage
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		out = append(out, json.RawMessage(line))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		var single json.RawMessage
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &single); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
		return []json.RawMessage{single}, nil
	}
	return out, nil
}

func postXAPIStatement(c *client.Client, courseCode, packageID string, statement json.RawMessage) error {
	payload := map[string]any{
		"courseCode": courseCode,
		"packageId":  packageID,
		"statement":  json.RawMessage(statement),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/xapi/statements", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func queryCourseXAPIEvents(c *client.Client, courseCode string, since string) ([]xapiEventRow, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/events"
	if since != "" {
		path += "?since=" + url.QueryEscape(since)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
		Events []xapiEventRow `json:"events"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Events, body, nil
}

func filterXAPIEvents(events []xapiEventRow, verb, activity, actor string) []xapiEventRow {
	verb = strings.TrimSpace(verb)
	activity = strings.TrimSpace(activity)
	actor = strings.TrimSpace(actor)
	if verb == "" && activity == "" && actor == "" {
		return events
	}
	out := make([]xapiEventRow, 0, len(events))
	for _, e := range events {
		if verb != "" && !strings.Contains(strings.ToLower(e.Verb), strings.ToLower(verb)) {
			continue
		}
		if activity != "" && !strings.Contains(strings.ToLower(e.ObjectID), strings.ToLower(activity)) {
			if e.ObjectTitle == nil || !strings.Contains(strings.ToLower(*e.ObjectTitle), strings.ToLower(activity)) {
				continue
			}
		}
		if actor != "" {
			actorMatch := strings.Contains(strings.ToLower(string(e.FullJSON)), strings.ToLower(actor))
			if !actorMatch {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

func statementIDFromJSON(raw json.RawMessage) string {
	var stmt struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &stmt); err != nil || stmt.ID == "" {
		return ""
	}
	return stmt.ID
}

func postEngagementEvents(c *client.Client, events []json.RawMessage) ([]byte, error) {
	var batch []json.RawMessage
	if len(events) == 1 {
		var arr []json.RawMessage
		if err := json.Unmarshal(events[0], &arr); err == nil && len(arr) > 0 {
			batch = arr
		} else {
			batch = events
		}
	} else {
		batch = events
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/analytics/events", bytes.NewReader(marshalJSON(batch)))
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

func postRecommendationEvent(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/recommendations/event", bytes.NewReader(raw))
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

func postScormCommit(c *client.Client, registrationID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/scorm/rte/" + url.PathEscape(registrationID) + "/commit"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
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

func testLRSEndpoint(c *client.Client, endpointID string) ([]byte, error) {
	path := "/api/v1/admin/lrs-config/" + url.PathEscape(endpointID) + "/test"
	req, err := c.NewRequest(http.MethodPost, path, nil)
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

func listLRSDeadLetter(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/lrs-dead-letter", nil)
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

func retryLRSDeadLetter(c *client.Client, id string) error {
	path := "/api/v1/admin/lrs-dead-letter/" + url.PathEscape(id) + "/retry"
	req, err := c.NewRequest(http.MethodPost, path, nil)
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

func marshalJSON(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

func readJSONObjectsFromFile(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("file is empty")
	}
	if data[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(data, &arr); err != nil {
			return nil, fmt.Errorf("parsing JSON array: %w", err)
		}
		return arr, nil
	}
	if strings.Contains(string(data), "\n") {
		return readNDJSONObjects(path)
	}
	return []json.RawMessage{json.RawMessage(data)}, nil
}

func writeJSONToWriter(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}