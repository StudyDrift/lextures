package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type portfolioRow struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	IsPublic bool    `json:"isPublic"`
	Slug     *string `json:"publicSlug"`
}

func listPortfolios(c *client.Client) ([]portfolioRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/portfolios", nil)
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
		Portfolios []portfolioRow `json:"portfolios"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Portfolios, body, nil
}

func getPortfolio(c *client.Client, id string) ([]byte, error) {
	path := "/api/v1/me/portfolios/" + url.PathEscape(id)
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

func createPortfolio(c *client.Client, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/portfolios", bytes.NewReader(b))
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

func deletePortfolio(c *client.Client, id string) error {
	path := "/api/v1/me/portfolios/" + url.PathEscape(id)
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

func patchPortfolio(c *client.Client, id string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/me/portfolios/" + url.PathEscape(id)
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

func createPortfolioArtifact(c *client.Client, portfolioID string, payload map[string]any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/me/portfolios/" + url.PathEscape(portfolioID) + "/artifacts"
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

func uploadPortfolioArtifact(c *client.Client, portfolioID, filePath, title string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if title == "" {
		title = filepath.Base(filePath)
	}
	_ = w.WriteField("title", title)
	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	_ = w.Close()

	path := "/api/v1/me/portfolios/" + url.PathEscape(portfolioID) + "/artifacts/upload"
	req, err := http.NewRequest(http.MethodPost, c.BaseURL()+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if key := c.APIKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := c.Do(req)
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

func deletePortfolioArtifact(c *client.Client, portfolioID, artifactID string) error {
	path := fmt.Sprintf("/api/v1/me/portfolios/%s/artifacts/%s",
		url.PathEscape(portfolioID), url.PathEscape(artifactID))
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

func exportPortfolio(c *client.Client, id string) ([]byte, error) {
	return getPortfolio(c, id)
}