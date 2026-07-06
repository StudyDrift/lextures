package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type modulePackageUploadOpts struct {
	courseCode string
	moduleID   string
	segment    string // "scorm" or "h5p"
	localPath  string
	title      string
	quiet      bool
	progress   io.Writer
}

func uploadModulePackage(c *client.Client, opts modulePackageUploadOpts) (structureItemPublic, []byte, error) {
	localPath := filepath.Clean(opts.localPath)
	if strings.Contains(localPath, "..") {
		return structureItemPublic{}, nil, fmt.Errorf("invalid file path: %s", opts.localPath)
	}
	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return structureItemPublic{}, nil, fmt.Errorf("file not found: %s", localPath)
		}
		return structureItemPublic{}, nil, fmt.Errorf("accessing file %s: %w", localPath, err)
	}
	if info.IsDir() {
		return structureItemPublic{}, nil, fmt.Errorf("%s is a directory, not a file", localPath)
	}

	progressOut := opts.progress
	if progressOut == nil {
		progressOut = io.Discard
	}
	if !opts.quiet {
		_, _ = fmt.Fprintf(progressOut, "Uploading %s (%s)...\n",
			filepath.Base(localPath), formatFileBytes(info.Size()))
	}

	body, contentType, err := buildMultipartPackageBody(localPath, opts.title)
	if err != nil {
		return structureItemPublic{}, nil, err
	}

	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/%s",
		opts.courseCode, opts.moduleID, opts.segment)
	req, err := http.NewRequest(http.MethodPost, c.BaseURL()+path, body)
	if err != nil {
		return structureItemPublic{}, nil, fmt.Errorf("building request: %w", err)
	}
	if c.APIKey() != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey())
	}
	if client.DefaultUserAgent != "" {
		req.Header.Set("User-Agent", client.DefaultUserAgent)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.Do(req)
	if err != nil {
		return structureItemPublic{}, nil, fmt.Errorf("uploading package: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return structureItemPublic{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return structureItemPublic{}, respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	var item structureItemPublic
	if err := json.Unmarshal(respBody, &item); err != nil {
		return structureItemPublic{}, respBody, fmt.Errorf("decoding response: %w", err)
	}
	return item, respBody, nil
}

func buildMultipartPackageBody(localPath, title string) (io.Reader, string, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		var copyErr error
		defer func() {
			_ = writer.Close()
			if copyErr != nil {
				_ = pw.CloseWithError(copyErr)
				return
			}
			_ = pw.Close()
		}()
		if strings.TrimSpace(title) != "" {
			if err := writer.WriteField("title", title); err != nil {
				copyErr = err
				return
			}
		}
		part, err := writer.CreateFormFile("file", filepath.Base(localPath))
		if err != nil {
			copyErr = err
			return
		}
		f, err := os.Open(localPath)
		if err != nil {
			copyErr = err
			return
		}
		defer func() { _ = f.Close() }()
		if _, err := io.Copy(part, f); err != nil {
			copyErr = err
		}
	}()
	return pr, writer.FormDataContentType(), nil
}