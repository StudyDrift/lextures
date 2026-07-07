package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
)

type meetingRow struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Provider       string  `json:"provider"`
	Status         string  `json:"status"`
	ScheduledStart *string `json:"scheduledStart"`
	ScheduledEnd   *string `json:"scheduledEnd"`
}

type availabilitySlot struct {
	ID        string  `json:"id"`
	StartTime string  `json:"startTime"`
	EndTime   string  `json:"endTime"`
	Status    string  `json:"status"`
}

type conferenceSlot struct {
	ID        string `json:"id"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Status    string `json:"status"`
}

func listMeetings(c *client.Client, course string) ([]meetingRow, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/meetings"
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
		Meetings []meetingRow `json:"meetings"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Meetings, body, nil
}

func createMeeting(c *client.Client, course string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/meetings"
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func patchMeeting(c *client.Client, meetingID string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/meetings/" + url.PathEscape(meetingID)
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

func buildMeetingPayload(title, start, duration, tz, provider string) (map[string]any, error) {
	payload := map[string]any{"title": title}
	if provider != "" {
		payload["provider"] = provider
	}
	if start != "" {
		t, err := cli.ParseRFC3339InTZ(start, tz)
		if err != nil {
			return nil, err
		}
		startStr, err := cli.FormatRFC3339(t, tz)
		if err != nil {
			return nil, err
		}
		payload["scheduledStart"] = startStr
		if duration != "" {
			var mins int
			if _, err := fmt.Sscanf(duration, "%d", &mins); err != nil || mins <= 0 {
				return nil, fmt.Errorf("invalid --duration")
			}
			end := t.Add(time.Duration(mins) * time.Minute)
			endStr, err := cli.FormatRFC3339(end, tz)
			if err != nil {
				return nil, err
			}
			payload["scheduledEnd"] = endStr
		}
	}
	return payload, nil
}

func setOfficeHoursAvailability(c *client.Client, course string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/availability"
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func getOfficeHoursAvailability(c *client.Client, course string) ([]availabilitySlot, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/availability"
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
		Slots []availabilitySlot `json:"slots"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Slots, body, nil
}

func listConferenceSlots(c *client.Client, teacherID string) ([]conferenceSlot, []byte, error) {
	path := "/api/v1/teachers/" + url.PathEscape(teacherID) + "/conference-slots"
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
		Slots []conferenceSlot `json:"slots"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Slots, body, nil
}

func createConferenceAvailability(c *client.Client, teacherID string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/teachers/" + url.PathEscape(teacherID) + "/conference-availability"
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func getCalendarToken(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/calendar-token", nil)
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

func rotateCalendarToken(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/calendar-token", bytes.NewReader([]byte("{}")))
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

func exportCalendarICal(c *client.Client, token string) ([]byte, error) {
	path := "/api/v1/me/calendar.ics"
	if token != "" {
		path += "?token=" + url.QueryEscape(token)
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