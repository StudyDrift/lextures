package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SSEHandler is called for each SSE data payload.
type SSEHandler func(eventType string, data json.RawMessage) error

// ReadSSE consumes a text/event-stream response body.
func ReadSSE(body io.Reader, onEvent SSEHandler) error {
	if onEvent == nil {
		return fmt.Errorf("SSE handler required")
	}
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var dataLines []string
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			if len(dataLines) == 0 {
				continue
			}
			payload := strings.Join(dataLines, "\n")
			dataLines = nil
			var envelope struct {
				Type string          `json:"type"`
				Text string          `json:"text"`
				Data json.RawMessage `json:"-"`
			}
			_ = json.Unmarshal([]byte(payload), &envelope)
			evtType := envelope.Type
			if evtType == "" {
				evtType = "message"
			}
			if err := onEvent(evtType, json.RawMessage(payload)); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return sc.Err()
}

// StreamHTTPPost opens an SSE stream from a POST request. Caller must close resp.Body.
func StreamHTTPPost(client *http.Client, req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "text/event-stream")
	return client.Do(req)
}

// TutorStreamResult holds the final tutor SSE outcome.
type TutorStreamResult struct {
	ConversationID string
	Text           string
	Usage          map[string]any
}

// CollectTutorSSE renders content chunks to stderrWriter and returns the final result.
func CollectTutorSSE(body io.Reader, stderr io.Writer, jsonOut bool) (TutorStreamResult, error) {
	var result TutorStreamResult
	var text strings.Builder
	err := ReadSSE(body, func(eventType string, data json.RawMessage) error {
		var envelope struct {
			Type             string `json:"type"`
			Text             string `json:"text"`
			ConversationID   string `json:"conversationId"`
			PromptTokens     int    `json:"promptTokens"`
			CompletionTokens int    `json:"completionTokens"`
			Message          string `json:"message"`
		}
		_ = json.Unmarshal(data, &envelope)
		switch envelope.Type {
		case "content":
			if envelope.Text != "" {
				text.WriteString(envelope.Text)
				if !jsonOut && stderr != nil {
					_, _ = io.WriteString(stderr, envelope.Text)
				}
			}
		case "done":
			result.ConversationID = envelope.ConversationID
		case "error":
			if envelope.Message != "" {
				return fmt.Errorf("%s", envelope.Message)
			}
		}
		return nil
	})
	result.Text = text.String()
	return result, err
}