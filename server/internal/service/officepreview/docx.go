package officepreview

// convertDocxToHTML renders Word documents from OOXML (fallback: markitdown extraction).
func convertDocxToHTML(data []byte, filename, mimeType string) (string, error) {
	if html, err := convertDocxToVisualHTML(data, filename, mimeType); err == nil {
		return html, nil
	}
	return convertMarkdownOfficeToHTML(data, filename, mimeType, FormatDOCX)
}
