package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

var validBCP47 = map[string]bool{
	"en": true, "es": true, "fr": true, "de": true, "ar": true, "zh": true, "ja": true, "pt": true,
}

func validateLocale(code string) error {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return fmt.Errorf("locale is required")
	}
	base := strings.SplitN(code, "-", 2)[0]
	if !validBCP47[base] && len(code) != 2 && len(code) != 5 {
		return fmt.Errorf("unsupported locale %q", code)
	}
	return nil
}

func isWebVTT(data []byte) bool {
	s := strings.TrimSpace(string(data))
	return strings.HasPrefix(s, "WEBVTT")
}

func fetchCourseAccessibility(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/accessibility"
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

func suggestAltText(c *client.Client, course, imageURL, language string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{"imageUrl": imageURL, "language": language})
	if err != nil {
		return nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/alt-text/suggest"
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

func listCaptions(c *client.Client, objectID string) ([]byte, error) {
	path := "/api/v1/files/" + url.PathEscape(objectID) + "/captions"
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

func retriggerCaptions(c *client.Client, objectID string) ([]byte, error) {
	path := "/api/v1/files/" + url.PathEscape(objectID) + "/captions/retrigger"
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func importCaptionVTT(c *client.Client, objectID string, vtt []byte, lang string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{
		"vtt_content": string(vtt),
		"lang":        lang,
	})
	if err != nil {
		return nil, err
	}
	path := "/api/v1/files/" + url.PathEscape(objectID) + "/captions/import"
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func deleteCaption(c *client.Client, objectID, captionID string) error {
	path := fmt.Sprintf("/api/v1/files/%s/captions/%s", url.PathEscape(objectID), url.PathEscape(captionID))
	req, err := c.NewRequest(http.MethodDelete, path, nil)
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

func listCourseTranslations(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/translations"
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

func fetchTranslationCoverage(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/translation-coverage"
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

func draftCourseTranslation(c *client.Client, course, itemID, locale string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/translations/%s/ai-draft",
		url.PathEscape(course), url.PathEscape(itemID))
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader([]byte("{}")))
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
	_ = locale
	return body, nil
}

func setCourseTranslation(c *client.Client, course, itemID, locale, text string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{
		"targetLocale":   locale,
		"translatedText": text,
	})
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/translations/%s",
		url.PathEscape(course), url.PathEscape(itemID))
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
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchTranscodeStatus(c *client.Client, objectID string) ([]byte, error) {
	path := "/api/v1/files/" + url.PathEscape(objectID) + "/transcode-status"
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

func retranscodeFile(c *client.Client, objectID string) ([]byte, error) {
	path := "/api/v1/admin/files/" + url.PathEscape(objectID) + "/retranscode"
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func synthesizeTTS(c *client.Client, text string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/tts/synthesize", bytes.NewReader(payload))
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

func fetchItemReadingLevel(c *client.Client, course, itemID string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/items/%s/reading-level",
		url.PathEscape(course), url.PathEscape(itemID))
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

func waitForTranscode(c *client.Client, objectID string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		body, err := fetchTranscodeStatus(c, objectID)
		if err != nil {
			return nil, err
		}
		var st struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(body, &st) == nil {
			switch strings.ToLower(st.Status) {
			case "completed", "ready", "done":
				return body, nil
			case "failed", "error":
				return body, fmt.Errorf("transcode failed")
			}
		}
		if time.Now().After(deadline) {
			return body, fmt.Errorf("transcode wait timed out after %s", timeout)
		}
		time.Sleep(2 * time.Second)
	}
}

func readOptionalFile(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	return os.ReadFile(path)
}
