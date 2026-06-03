package officepreview

import (
	"path/filepath"
	"strings"
)

// Format is an Office Open XML type we can convert server-side.
type Format string

const (
	FormatDOCX Format = "docx"
	FormatXLSX Format = "xlsx"
	FormatPPTX Format = "pptx"
)

const maxPreviewBytes = 32 << 20 // 32 MiB

// DetectFormat returns the Open XML format from filename and optional MIME type.
func DetectFormat(filename, mimeType string) (Format, bool) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch ext {
	case ".docx":
		return FormatDOCX, true
	case ".xlsx":
		return FormatXLSX, true
	case ".pptx":
		return FormatPPTX, true
	}
	mt := strings.ToLower(strings.TrimSpace(mimeType))
	switch mt {
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return FormatDOCX, true
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return FormatXLSX, true
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return FormatPPTX, true
	}
	return "", false
}

func extensionForFormat(f Format) string {
	switch f {
	case FormatDOCX:
		return ".docx"
	case FormatXLSX:
		return ".xlsx"
	case FormatPPTX:
		return ".pptx"
	default:
		return ""
	}
}
