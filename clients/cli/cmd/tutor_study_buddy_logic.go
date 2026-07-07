package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
)

func postTutorMessage(c *client.Client, course, message string) (*http.Response, error) {
	payload, _ := json.Marshal(map[string]string{"message": message})
	path := "/api/v1/courses/" + url.PathEscape(course) + "/tutor/message"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return cli.StreamHTTPPost(c.HTTPClient(), req)
}

func getTutorConversation(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/tutor/conversation"
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

func postStudyBuddyMessage(c *client.Client, course, message string) ([]byte, error) {
	payload, _ := json.Marshal(map[string]string{"message": message})
	path := "/api/v1/courses/" + url.PathEscape(course) + "/study-buddy/message"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
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

func getStudyBuddyMemory(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/study-buddy/memory"
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

func resolveEnrollmentForCourse(c *client.Client, course string) (string, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/enrollments", nil)
	if err != nil {
		return "", err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Enrollments []struct {
			ID         string `json:"id"`
			CourseCode string `json:"courseCode"`
		} `json:"enrollments"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	for _, e := range out.Enrollments {
		if strings.EqualFold(e.CourseCode, course) {
			return e.ID, nil
		}
	}
	return "", fmt.Errorf("no enrollment found for course %q — pass --enrollment", course)
}

func startDiagnostic(c *client.Client, enrollmentID string) ([]byte, error) {
	path := "/api/v1/enrollments/" + url.PathEscape(enrollmentID) + "/diagnostic/start"
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func getDiagnosticGate(c *client.Client, enrollmentID string) ([]byte, error) {
	path := "/api/v1/enrollments/" + url.PathEscape(enrollmentID) + "/diagnostic"
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

func getDiagnosticConfig(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/diagnostic-config"
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

func listMyPaths(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/paths", nil)
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

func getPathProgress(c *client.Client, pathID string) ([]byte, error) {
	path := "/api/v1/me/paths/" + url.PathEscape(pathID) + "/progress"
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

func listConcepts(c *client.Client, query string) ([]byte, error) {
	path := "/api/v1/concepts"
	if query != "" {
		path += "?q=" + url.QueryEscape(query)
	}
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

func getConcept(c *client.Client, id string) ([]byte, error) {
	path := "/api/v1/concepts/" + url.PathEscape(id)
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

func getLearnerConcepts(c *client.Client, userID string) ([]byte, error) {
	path := "/api/v1/learners/" + url.PathEscape(userID) + "/concepts"
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