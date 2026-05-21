// Package vttformatter converts transcript segments into W3C WebVTT format (plan 8.4).
package vttformatter

import (
	"fmt"
	"strings"
	"time"
)

const lowConfidenceThreshold = 0.70

// Segment is a single timed transcript segment.
type Segment struct {
	Start      time.Duration
	End        time.Duration
	Text       string
	Confidence float32
}

// Format produces a WebVTT-conformant string from the given segments.
// Low-confidence segments (< 0.70) are annotated with a <c.low-confidence> class
// so the player can highlight them for instructor review.
func Format(segments []Segment) string {
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	for i, seg := range segments {
		// cue identifier (1-based)
		fmt.Fprintf(&sb, "%d\n", i+1)
		// timestamps
		fmt.Fprintf(&sb, "%s --> %s\n", formatTimestamp(seg.Start), formatTimestamp(seg.End))
		text := seg.Text
		if seg.Confidence > 0 && seg.Confidence < lowConfidenceThreshold {
			text = fmt.Sprintf("<c.low-confidence>%s</c>", text)
		}
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// ParseVTT parses a WebVTT string back into segments. Used for instructor edits.
// It is lenient: cue identifiers and blank lines are skipped gracefully.
func ParseVTT(vtt string) ([]Segment, error) {
	lines := strings.Split(strings.ReplaceAll(vtt, "\r\n", "\n"), "\n")
	var segments []Segment
	i := 0
	// skip WEBVTT header
	for i < len(lines) && !strings.HasPrefix(lines[i], "WEBVTT") {
		i++
	}
	i++ // skip header line
	for i < len(lines) {
		// skip blanks and cue ids
		for i < len(lines) && (lines[i] == "" || !strings.Contains(lines[i], " --> ")) {
			i++
		}
		if i >= len(lines) {
			break
		}
		// timestamp line
		parts := strings.SplitN(lines[i], " --> ", 2)
		if len(parts) != 2 {
			i++
			continue
		}
		start, err := parseTimestamp(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("vttformatter: bad start timestamp %q: %w", parts[0], err)
		}
		end, err := parseTimestamp(strings.TrimSpace(strings.Fields(parts[1])[0]))
		if err != nil {
			return nil, fmt.Errorf("vttformatter: bad end timestamp %q: %w", parts[1], err)
		}
		i++
		// collect text lines until blank
		var textLines []string
		for i < len(lines) && lines[i] != "" {
			textLines = append(textLines, lines[i])
			i++
		}
		segments = append(segments, Segment{
			Start: start,
			End:   end,
			Text:  strings.Join(textLines, "\n"),
		})
	}
	return segments, nil
}

// ConfidenceStats computes average confidence and whether any segment is below the threshold.
func ConfidenceStats(segments []Segment) (avg float32, hasLow bool) {
	if len(segments) == 0 {
		return 0, false
	}
	var sum float32
	for _, s := range segments {
		sum += s.Confidence
		if s.Confidence > 0 && s.Confidence < lowConfidenceThreshold {
			hasLow = true
		}
	}
	return sum / float32(len(segments)), hasLow
}

// PlainText returns the transcript as a single string without WebVTT markup.
func PlainText(segments []Segment) string {
	texts := make([]string, 0, len(segments))
	for _, s := range segments {
		t := stripVTTTags(s.Text)
		if t != "" {
			texts = append(texts, t)
		}
	}
	return strings.Join(texts, " ")
}

// stripVTTTags removes WebVTT cue tags (e.g. <c.low-confidence>, </c>).
func stripVTTTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// formatTimestamp formats a duration as HH:MM:SS.mmm for WebVTT.
func formatTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// parseTimestamp parses HH:MM:SS.mmm or MM:SS.mmm into a duration.
func parseTimestamp(s string) (time.Duration, error) {
	var h, m, sec, ms int
	// Try HH:MM:SS.mmm first
	n, err := fmt.Sscanf(s, "%d:%d:%d.%d", &h, &m, &sec, &ms)
	if err != nil || n != 4 {
		// Try MM:SS.mmm
		n, err = fmt.Sscanf(s, "%d:%d.%d", &m, &sec, &ms)
		if err != nil || n != 3 {
			return 0, fmt.Errorf("unrecognised timestamp format: %q", s)
		}
	}
	d := time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second +
		time.Duration(ms)*time.Millisecond
	return d, nil
}
