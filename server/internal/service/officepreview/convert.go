package officepreview

import (
	"bytes"
	"fmt"
	"strings"

	markitdown "github.com/conductor-oss/markitdown"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	converter = markitdown.New(markitdown.WithKeepDataURIs(true))
	mdToHTML  = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	sanitizePolicy = func() *bluemonday.Policy {
		p := bluemonday.UGCPolicy()
		p.AllowImages()
		p.AllowDataURIImages()
		return p
	}()
)

// ConvertToHTML converts a DOCX, XLSX, or PPTX blob to a full HTML document.
func ConvertToHTML(data []byte, filename string, mimeType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("officepreview: empty file")
	}
	if len(data) > maxPreviewBytes {
		return "", fmt.Errorf("officepreview: file exceeds preview size limit (%d bytes)", maxPreviewBytes)
	}
	format, ok := DetectFormat(filename, mimeType)
	if !ok {
		return "", fmt.Errorf("officepreview: unsupported format")
	}

	switch format {
	case FormatPPTX:
		return convertPptxToHTML(data, filename, mimeType)
	case FormatDOCX:
		return convertDocxToHTML(data, filename, mimeType)
	case FormatXLSX:
		return convertMarkdownOfficeToHTML(data, filename, mimeType, format)
	default:
		return "", fmt.Errorf("officepreview: unsupported format")
	}
}

func convertMarkdownOfficeToHTML(data []byte, filename, mimeType string, format Format) (string, error) {
	ext := extensionForFormat(format)
	result, err := converter.ConvertReader(bytes.NewReader(data), markitdown.StreamInfo{
		Extension: ext,
		Filename:  filename,
		MIMEType:  mimeType,
	})
	if err != nil {
		return "", fmt.Errorf("officepreview: convert: %w", err)
	}
	md := strings.TrimSpace(result.Markdown)
	if md == "" {
		md = "*This document has no previewable text content.*"
	}
	fragment := markdownFragmentToHTML(md)
	return wrapHTMLDocument(fragment), nil
}

func markdownFragmentToHTML(md string) string {
	// DOCX converter may leave inline HTML img tags in the markdown stream.
	if strings.Contains(md, "<img ") {
		parts := strings.Split(md, "<img ")
		var b strings.Builder
		for i, part := range parts {
			if i == 0 {
				b.WriteString(renderMarkdown(part))
				continue
			}
			end := strings.Index(part, ">")
			if end < 0 {
				b.WriteString(renderMarkdown(part))
				continue
			}
			tag := "<img " + part[:end+1]
			b.WriteString(sanitizeHTML(tag))
			b.WriteString(renderMarkdown(part[end+1:]))
		}
		return sanitizeHTML(b.String())
	}
	return sanitizeHTML(renderMarkdown(md))
}

func renderMarkdown(md string) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return ""
	}
	var body bytes.Buffer
	if err := mdToHTML.Convert([]byte(md), &body); err != nil {
		return ""
	}
	return body.String()
}

func sanitizeHTML(fragment string) string {
	return sanitizePolicy.Sanitize(fragment)
}

func wrapHTMLDocument(fragment string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  :root { color-scheme: light; }
  body {
    font-family: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
    margin: 0;
    padding: 1.25rem 1.5rem 2.5rem;
    line-height: 1.55;
    color: #1e293b;
    background: #fff;
    word-wrap: break-word;
  }
  h1, h2, h3, h4 { line-height: 1.25; margin: 1.25em 0 0.5em; font-weight: 600; }
  h1 { font-size: 1.75rem; }
  h2 { font-size: 1.375rem; border-bottom: 1px solid #e2e8f0; padding-bottom: 0.25rem; }
  h3 { font-size: 1.125rem; }
  p { margin: 0.6em 0; }
  strong, b { font-weight: 600; }
  em, i { font-style: italic; }
  ul, ol { margin: 0.6em 0; padding-left: 1.5rem; }
  li { margin: 0.25em 0; }
  a { color: #4f46e5; text-decoration: underline; }
  table { border-collapse: collapse; width: 100%; margin: 1rem 0; font-size: 0.875rem; }
  th, td { border: 1px solid #cbd5e1; padding: 0.5rem 0.625rem; text-align: left; vertical-align: top; }
  th { background: #f1f5f9; font-weight: 600; }
  tr:nth-child(even) td { background: #f8fafc; }
  img { max-width: 100%; height: auto; display: block; margin: 0.75rem auto; }
  figure.slide-figure { margin: 1rem 0; text-align: center; }
  pre, code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.875em; }
  pre { overflow-x: auto; padding: 0.75rem; background: #f8fafc; border-radius: 0.375rem; border: 1px solid #e2e8f0; }
  blockquote { margin: 1rem 0; padding: 0.5rem 0 0.5rem 1rem; border-left: 3px solid #cbd5e1; color: #475569; }
  hr { border: none; border-top: 1px solid #e2e8f0; margin: 1.5rem 0; }
  .slide {
    margin: 0 0 2rem;
    padding: 1rem 1.25rem 1.25rem;
    border: 1px solid #e2e8f0;
    border-radius: 0.5rem;
    background: #fafafa;
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.06);
  }
  .slide-header {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: #64748b;
    margin-bottom: 0.75rem;
  }
  .slide-body > :first-child { margin-top: 0; }
  .pptx-empty { color: #64748b; font-style: italic; }
</style>
</head>
<body>` + fragment + `</body>
</html>`
}
