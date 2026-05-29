// Package captions provides WebVTT/SRT parsing and conversion for the accessibility layer (plan 12.4).
package captions

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/service/vttformatter"
)

var srtBlockRe = regexp.MustCompile(`(?ms)(\d+)\s*\n(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})\s*\n([\s\S]*?)(?:\n\n|\z)`)

// ParseSRT converts SubRip content to WebVTT segments.
func ParseSRT(srt string) ([]vttformatter.Segment, error) {
	srt = strings.TrimSpace(strings.ReplaceAll(srt, "\r\n", "\n"))
	if srt == "" {
		return nil, fmt.Errorf("captions: empty SRT")
	}
	matches := srtBlockRe.FindAllStringSubmatch(srt, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("captions: no SRT cues found")
	}
	segments := make([]vttformatter.Segment, 0, len(matches))
	for _, m := range matches {
		start, err := parseSRTTimestamp(m[2])
		if err != nil {
			return nil, err
		}
		end, err := parseSRTTimestamp(m[3])
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(m[4])
		if text == "" {
			continue
		}
		segments = append(segments, vttformatter.Segment{Start: start, End: end, Text: text})
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("captions: no SRT cues with text")
	}
	return segments, nil
}

// SRTFromVTT serializes segments to SubRip format.
func SRTFromVTT(segments []vttformatter.Segment) string {
	var sb strings.Builder
	for i, seg := range segments {
		fmt.Fprintf(&sb, "%d\n", i+1)
		fmt.Fprintf(&sb, "%s --> %s\n", formatSRTTimestamp(seg.Start), formatSRTTimestamp(seg.End))
		sb.WriteString(vttformatter.PlainText([]vttformatter.Segment{seg}))
		if seg.Text != "" && !strings.HasSuffix(seg.Text, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// VTTFromSRT converts SRT file content to a WebVTT string.
func VTTFromSRT(srt string) (string, error) {
	segments, err := ParseSRT(srt)
	if err != nil {
		return "", err
	}
	return vttformatter.Format(segments), nil
}

// ParseVTT parses WebVTT into segments (delegates to vttformatter).
func ParseVTT(vtt string) ([]vttformatter.Segment, error) {
	return vttformatter.ParseVTT(vtt)
}

// FormatVTT formats segments as WebVTT.
func FormatVTT(segments []vttformatter.Segment) string {
	return vttformatter.Format(segments)
}

// PlainTranscript returns cue text joined for search/display.
func PlainTranscript(segments []vttformatter.Segment) string {
	return vttformatter.PlainText(segments)
}

func parseSRTTimestamp(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	var h, m, sec, ms int
	n, err := fmt.Sscanf(s, "%d:%d:%d,%d", &h, &m, &sec, &ms)
	if err != nil || n != 4 {
		return 0, fmt.Errorf("captions: bad SRT timestamp %q", s)
	}
	return time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second +
		time.Duration(ms)*time.Millisecond, nil
}

func formatSRTTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// CourseFileIDPattern matches course-file content URLs in markdown/HTML.
var CourseFileIDPattern = regexp.MustCompile(`/course-files/([0-9a-fA-F-]{36})/content`)
