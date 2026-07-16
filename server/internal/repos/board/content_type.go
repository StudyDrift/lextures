package board

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Supported post content types (FR-2).
const (
	ContentTypeText    = "text"
	ContentTypeImage   = "image"
	ContentTypeFile    = "file"
	ContentTypeLink    = "link"
	ContentTypeVideo   = "video"
	ContentTypeAudio   = "audio"
	ContentTypeDrawing = "drawing"
)

// ValidContentType reports whether ct is a known board post content type.
func ValidContentType(ct string) bool {
	switch ct {
	case ContentTypeText, ContentTypeImage, ContentTypeFile, ContentTypeLink,
		ContentTypeVideo, ContentTypeAudio, ContentTypeDrawing:
		return true
	default:
		return false
	}
}

// FileBackedContentType is true for types that require an attachment.
func FileBackedContentType(ct string) bool {
	switch ct {
	case ContentTypeImage, ContentTypeFile, ContentTypeVideo, ContentTypeAudio:
		return true
	default:
		return false
	}
}

// CreatePostInput is validated before insert.
type CreatePostInput struct {
	ContentType  string
	Title        string
	Body         json.RawMessage
	LinkURL      string
	DrawingData  json.RawMessage
	AttachmentID *string
	// Status defaults to approved when empty (VC.7).
	Status string
}

// ValidateCreatePost enforces content-type-specific required fields (FR-3).
func ValidateCreatePost(in CreatePostInput) error {
	ct := strings.TrimSpace(strings.ToLower(in.ContentType))
	if !ValidContentType(ct) {
		return fmt.Errorf("board: invalid content_type %q", in.ContentType)
	}
	switch ct {
	case ContentTypeText:
		if len(in.Body) == 0 && strings.TrimSpace(in.Title) == "" {
			return fmt.Errorf("board: text posts require title or body")
		}
	case ContentTypeLink:
		if strings.TrimSpace(in.LinkURL) == "" {
			return fmt.Errorf("board: link posts require link_url")
		}
		if err := validateHTTPURL(in.LinkURL); err != nil {
			return err
		}
	case ContentTypeImage, ContentTypeFile, ContentTypeVideo, ContentTypeAudio:
		if in.AttachmentID == nil || strings.TrimSpace(*in.AttachmentID) == "" {
			return fmt.Errorf("board: %s posts require attachment_id", ct)
		}
	case ContentTypeDrawing:
		if len(in.DrawingData) == 0 || string(in.DrawingData) == "null" {
			return fmt.Errorf("board: drawing posts require drawing_data")
		}
	}
	if in.LinkURL != "" && ct != ContentTypeLink && ct != ContentTypeVideo {
		if err := validateHTTPURL(in.LinkURL); err != nil {
			return err
		}
	}
	return nil
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("board: link_url must be a valid http(s) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("board: link_url must be http or https")
	}
	return nil
}
