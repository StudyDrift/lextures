package gradingagent

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	markitdown "github.com/conductor-oss/markitdown"

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

const maxSubmissionMarkdownBytes = 512 << 10

var submissionMarkdownConverter = markitdown.New()

// JoinSubmissions flattens markdown exports into a double-newline delimited string.
func JoinSubmissions(submissions []string) string {
	parts := make([]string, 0, len(submissions))
	for _, submission := range submissions {
		if trimmed := strings.TrimSpace(submission); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, "\n\n")
}

// LoadSubmissionMarkdownsForSubmission returns text/file markdown parts (not vision images).
func (s *Service) LoadSubmissionMarkdownsForSubmission(ctx context.Context, courseCode string, sub *moduleassignmentsubmissions.SubmissionRow) ([]string, error) {
	resolved, err := s.ResolveSubmissionContent(ctx, courseCode, sub, ResolveSubmissionContentOptions{
		TextEntryEnabled: true,
	})
	if err != nil {
		return nil, err
	}
	if resolved.FailureReason != "" {
		return nil, fmt.Errorf("%s", resolved.FailureReason)
	}
	if len(resolved.Markdowns) == 0 {
		return nil, fmt.Errorf("no submission text available")
	}
	return resolved.Markdowns, nil
}

// LoadSubmissionTextForSubmission returns all submission files joined as markdown text.
func (s *Service) LoadSubmissionTextForSubmission(ctx context.Context, courseCode string, sub *moduleassignmentsubmissions.SubmissionRow) (string, error) {
	submissions, err := s.LoadSubmissionMarkdownsForSubmission(ctx, courseCode, sub)
	if err != nil {
		return "", err
	}
	return JoinSubmissions(submissions), nil
}

// LoadReferenceFileMarkdown extracts text from a course file for reference material nodes.
func (s *Service) LoadReferenceFileMarkdown(ctx context.Context, courseCode string, fileID uuid.UUID) (string, error) {
	row, err := coursefiles.GetForCourse(ctx, s.Pool, courseCode, fileID)
	if err != nil || row == nil {
		return "", fmt.Errorf("reference file not found")
	}
	return s.loadSubmissionFileMarkdown(ctx, courseCode, fileID, row.OriginalFilename, row.MimeType)
}

func (s *Service) loadSubmissionFileMarkdown(ctx context.Context, courseCode string, fileID uuid.UUID, filename, mimeType string) (string, error) {
	row, err := coursefiles.GetForCourse(ctx, s.Pool, courseCode, fileID)
	if err != nil || row == nil {
		return "", fmt.Errorf("submission file not found")
	}
	b, err := s.readSubmissionBlob(ctx, courseCode, row)
	if err != nil {
		return "", err
	}
	if len(b) > maxSubmissionMarkdownBytes {
		b = b[:maxSubmissionMarkdownBytes]
	}
	if len(b) == 0 {
		return "", fmt.Errorf("empty submission text")
	}

	name := strings.TrimSpace(filename)
	if name == "" {
		name = strings.TrimSpace(row.OriginalFilename)
	}
	mime := strings.TrimSpace(mimeType)
	if mime == "" {
		mime = strings.TrimSpace(row.MimeType)
	}

	result, err := submissionMarkdownConverter.ConvertReader(bytes.NewReader(b), markitdown.StreamInfo{
		Extension: filepath.Ext(name),
		Filename:  name,
		MIMEType:  mime,
	})
	if err != nil {
		return "", fmt.Errorf("convert submission file: %w", err)
	}
	md := strings.TrimSpace(result.Markdown)
	if md == "" {
		return "", fmt.Errorf("empty submission text")
	}
	if len(md) > maxSubmissionMarkdownBytes {
		md = md[:maxSubmissionMarkdownBytes]
	}
	return md, nil
}