package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type catalogListing struct {
	IsPublic        bool    `json:"isPublic"`
	Category        *string `json:"category"`
	DifficultyLevel *string `json:"difficultyLevel"`
	Language        string  `json:"language"`
	PriceCents      int     `json:"priceCents"`
	Slug            string  `json:"slug"`
}

type libraryBook struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Author *string `json:"author"`
	ISBN   *string `json:"isbn"`
}

func fetchPublicCatalog(c *client.Client, q url.Values) ([]byte, error) {
	path := "/api/v1/public/catalog/courses"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
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

func fetchCatalogSections(c *client.Client, q url.Values) ([]byte, error) {
	path := "/api/v1/catalog/sections"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
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

func fetchCourseCatalogListing(c *client.Client, course string) (catalogListing, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/catalog-listing"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return catalogListing{}, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return catalogListing{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return catalogListing{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return catalogListing{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Listing catalogListing `json:"listing"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return catalogListing{}, body, err
	}
	return out.Listing, body, nil
}

func putCourseCatalogListing(c *client.Client, course string, listing catalogListing) ([]byte, error) {
	raw, err := json.Marshal(listing)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/catalog-listing"
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
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

func fetchOrgLibrary(c *client.Client, orgID string, q url.Values) ([]libraryBook, []byte, error) {
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/library"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
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
		Books []libraryBook `json:"books"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Books, body, nil
}

func searchLibrary(c *client.Client, query string) ([]byte, error) {
	path := "/api/v1/library/search?q=" + url.QueryEscape(query)
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

func linkLibraryResource(c *client.Client, course, moduleID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/library-resources",
		url.PathEscape(course), url.PathEscape(moduleID))
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchOERProviders(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/oer/providers", nil)
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

func searchOER(c *client.Client, q url.Values) ([]byte, error) {
	path := "/api/v1/oer/search"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
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

func linkOERResource(c *client.Client, course, moduleID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/oer-import",
		url.PathEscape(course), url.PathEscape(moduleID))
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchTextbookResource(c *client.Client, course, itemID string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/textbook-resources/%s",
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

func patchTextbookResource(c *client.Client, course, itemID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/textbook-resources/%s",
		url.PathEscape(course), url.PathEscape(itemID))
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
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

func fetchInclusiveAccess(c *client.Client, course string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/inclusive-access"
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

func setInclusiveAccess(c *client.Client, course string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/inclusive-access"
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func patchLibraryResource(c *client.Client, course, itemID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/courses/%s/library-resources/%s",
		url.PathEscape(course), url.PathEscape(itemID))
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
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

func filterLibraryBooks(books []libraryBook, query string) []libraryBook {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return books
	}
	out := make([]libraryBook, 0, len(books))
	for _, b := range books {
		if strings.Contains(strings.ToLower(b.Title), q) {
			out = append(out, b)
			continue
		}
		if b.Author != nil && strings.Contains(strings.ToLower(*b.Author), q) {
			out = append(out, b)
		}
	}
	return out
}
