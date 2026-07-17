package boardexport

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"
)

// CSVHeaders are the standard export columns (VC.9 FR-5).
var CSVHeaders = []string{
	"section",
	"author",
	"content_type",
	"title",
	"body",
	"link",
	"attachment_filename",
	"reaction_count",
	"reaction_avg",
	"comment_count",
	"created_at",
}

// CSVHeadersWithModeration appends moderation columns for manager exports.
var CSVHeadersWithModeration = append(append([]string{}, CSVHeaders...), "status", "hidden", "removed")

// RenderCSV emits UTF-8 CSV bytes for the given cards.
func RenderCSV(cards []CardRow, includeModeration bool) ([]byte, error) {
	var buf bytes.Buffer
	// UTF-8 BOM helps Excel open non-ASCII correctly.
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	headers := CSVHeaders
	if includeModeration {
		headers = CSVHeadersWithModeration
	}
	if err := w.Write(headers); err != nil {
		return nil, err
	}
	for _, c := range cards {
		avg := ""
		if c.ReactionAvg != nil {
			avg = strconv.FormatFloat(*c.ReactionAvg, 'f', 1, 64)
		}
		row := []string{
			c.SectionTitle,
			c.Author,
			c.ContentType,
			c.Title,
			c.BodyText,
			c.Link,
			c.AttachmentFilename,
			strconv.Itoa(c.ReactionCount),
			avg,
			strconv.Itoa(c.CommentCount),
			c.CreatedAt.UTC().Format(time.RFC3339),
		}
		if includeModeration {
			row = append(row,
				c.Status,
				strconv.FormatBool(c.Hidden),
				strconv.FormatBool(c.Removed),
			)
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("boardexport: csv: %w", err)
	}
	return buf.Bytes(), nil
}
