package gradingagent

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/submissionattachments"
	"github.com/lextures/lextures/server/internal/service/aitutor"
)

// InputModality describes how submission content was sourced for grading.
type InputModality string

const (
	ModalityText       InputModality = "text"
	ModalityFile       InputModality = "file"
	ModalityVision     InputModality = "vision"
	ModalityUnreadable InputModality = "unreadable"

	DefaultMaxVisionPages  = 10
	maxVisionPayloadBytes  = 4 << 20
	maxVisionPDFBytes      = 8 << 20
)

// Failure reasons surfaced in the review queue (mapped to i18n on the client).
const (
	FailureNoSubmissionContent = "No readable submission content"
	FailureEmptyFileText       = "Submission file has no extractable text"
	FailureVisionNotEnabled    = "Vision grading is not enabled for image-only submissions"
	FailureVisionModelMissing  = "Vision grading requires a configured grader model"
	FailureVisionPageCap              = "Submission exceeds the vision page limit"
	FailureVisionWorkflowUnsupported  = "Vision grading is not supported for complex workflow graphs"
)

// ResolveSubmissionContentOptions gates text-entry and vision paths (GA-M2).
type ResolveSubmissionContentOptions struct {
	TextEntryEnabled bool
	VisionEnabled    bool
	MaxVisionPages   int
}

// ResolvedSubmissionContent is the output of the submission content resolver.
type ResolvedSubmissionContent struct {
	Markdowns     []string
	Text          string
	ImageDataURLs []string
	Modality      InputModality
	FailureReason string
}

// ModalityLogLabel returns a short label for dry-run logs.
func (m InputModality) ModalityLogLabel() string {
	switch m {
	case ModalityText:
		return "text-entry"
	case ModalityFile:
		return "file"
	case ModalityVision:
		return "vision"
	default:
		return "unreadable"
	}
}

// SubmissionAttemptableForAgent reports whether a submission row should be included in batch scope.
func SubmissionAttemptableForAgent(row moduleassignmentsubmissions.SubmissionRow, textEntryEnabled bool) bool {
	if row.AttachmentFileID != nil {
		return true
	}
	if textEntryEnabled && moduleassignmentsubmissions.HasBodyText(row) {
		return true
	}
	return false
}

// ResolveSubmissionContent loads gradable submission content using text-entry → file text → vision.
func (s *Service) ResolveSubmissionContent(
	ctx context.Context,
	courseCode string,
	sub *moduleassignmentsubmissions.SubmissionRow,
	opts ResolveSubmissionContentOptions,
) (ResolvedSubmissionContent, error) {
	if sub == nil {
		return ResolvedSubmissionContent{
			Modality:      ModalityUnreadable,
			FailureReason: FailureNoSubmissionContent,
		}, nil
	}
	if s.Pool == nil {
		return ResolvedSubmissionContent{}, fmt.Errorf("database unavailable")
	}

	maxPages := opts.MaxVisionPages
	if maxPages <= 0 {
		maxPages = DefaultMaxVisionPages
	}

	var parts []string
	if opts.TextEntryEnabled {
		if body := strings.TrimSpace(sub.BodyText); body != "" {
			parts = append(parts, aitutor.RedactPII(body))
		}
	}

	fileRefs, err := s.listSubmissionFileRefs(ctx, courseCode, sub)
	if err != nil {
		return ResolvedSubmissionContent{}, err
	}

	var fileMarkdowns []string
	var visionCandidates []submissionFileRef
	for _, ref := range fileRefs {
		md, convErr := s.loadSubmissionFileMarkdown(ctx, courseCode, ref.id, ref.filename, ref.mimeType)
		if convErr == nil && strings.TrimSpace(md) != "" {
			fileMarkdowns = append(fileMarkdowns, md)
			continue
		}
		if isVisionCandidateMIME(ref.mimeType, ref.filename) {
			visionCandidates = append(visionCandidates, ref)
		}
	}

	if len(fileMarkdowns) > 0 {
		parts = append(parts, fileMarkdowns...)
	}

	if len(parts) > 0 {
		text := JoinSubmissions(parts)
		modality := ModalityFile
		if opts.TextEntryEnabled && moduleassignmentsubmissions.HasBodyText(*sub) {
			modality = ModalityText
		}
		return ResolvedSubmissionContent{
			Markdowns: parts,
			Text:      text,
			Modality:  modality,
		}, nil
	}

	if len(visionCandidates) == 0 {
		reason := FailureNoSubmissionContent
		if len(fileRefs) > 0 {
			reason = FailureEmptyFileText
		}
		return ResolvedSubmissionContent{
			Modality:      ModalityUnreadable,
			FailureReason: reason,
		}, nil
	}

	if !opts.VisionEnabled {
		return ResolvedSubmissionContent{
			Modality:      ModalityUnreadable,
			FailureReason: FailureVisionNotEnabled,
		}, nil
	}

	if len(visionCandidates) > maxPages {
		return ResolvedSubmissionContent{
			Modality:      ModalityUnreadable,
			FailureReason: FailureVisionPageCap,
		}, nil
	}

	imageURLs := make([]string, 0, len(visionCandidates))
	for _, ref := range visionCandidates {
		dataURL, encErr := s.encodeSubmissionFileDataURL(ctx, courseCode, ref)
		if encErr != nil {
			return ResolvedSubmissionContent{}, encErr
		}
		imageURLs = append(imageURLs, dataURL)
	}
	if len(imageURLs) == 0 {
		return ResolvedSubmissionContent{
			Modality:      ModalityUnreadable,
			FailureReason: FailureEmptyFileText,
		}, nil
	}

	return ResolvedSubmissionContent{
		ImageDataURLs: imageURLs,
		Modality:      ModalityVision,
	}, nil
}

type submissionFileRef struct {
	id       uuid.UUID
	filename string
	mimeType string
}

func (s *Service) listSubmissionFileRefs(ctx context.Context, courseCode string, sub *moduleassignmentsubmissions.SubmissionRow) ([]submissionFileRef, error) {
	refs := make([]submissionFileRef, 0, 4)
	attachments, err := submissionattachments.ListForSubmission(ctx, s.Pool, sub.ID)
	if err != nil {
		return nil, err
	}
	for _, att := range attachments {
		refs = append(refs, submissionFileRef{id: att.FileID, filename: att.OriginalFilename, mimeType: att.MimeType})
	}
	if len(refs) == 0 && sub.AttachmentFileID != nil {
		row, rowErr := coursefiles.GetForCourse(ctx, s.Pool, courseCode, *sub.AttachmentFileID)
		if rowErr != nil || row == nil {
			return nil, fmt.Errorf("submission file not found")
		}
		refs = append(refs, submissionFileRef{id: row.ID, filename: row.OriginalFilename, mimeType: row.MimeType})
	}
	return refs, nil
}

func isVisionCandidateMIME(mimeType, filename string) bool {
	mime := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(mime, "image/") {
		return true
	}
	if mime == "application/pdf" {
		return true
	}
	ext := strings.ToLower(filename)
	return strings.HasSuffix(ext, ".pdf") ||
		strings.HasSuffix(ext, ".png") ||
		strings.HasSuffix(ext, ".jpg") ||
		strings.HasSuffix(ext, ".jpeg") ||
		strings.HasSuffix(ext, ".gif") ||
		strings.HasSuffix(ext, ".webp")
}

func (s *Service) encodeSubmissionFileDataURL(ctx context.Context, courseCode string, ref submissionFileRef) (string, error) {
	row, err := coursefiles.GetForCourse(ctx, s.Pool, courseCode, ref.id)
	if err != nil || row == nil {
		return "", fmt.Errorf("submission file not found")
	}
	b, err := s.readSubmissionBlob(ctx, courseCode, row)
	if err != nil {
		return "", err
	}
	mime := strings.TrimSpace(row.MimeType)
	if mime == "" {
		mime = strings.TrimSpace(ref.mimeType)
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	limit := maxVisionPayloadBytes
	if mime == "application/pdf" {
		limit = maxVisionPDFBytes
	}
	if len(b) > limit {
		b = b[:limit]
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	return fmt.Sprintf("data:%s;base64,%s", mime, encoded), nil
}