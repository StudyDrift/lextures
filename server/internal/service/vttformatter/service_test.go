package vttformatter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/vttformatter"
)

func TestFormat_EmptySegments(t *testing.T) {
	out := vttformatter.Format(nil)
	if !strings.HasPrefix(out, "WEBVTT") {
		t.Fatal("output must start with WEBVTT")
	}
}

func TestFormat_SingleSegment(t *testing.T) {
	segs := []vttformatter.Segment{
		{Start: 0, End: 2 * time.Second, Text: "Hello world", Confidence: 0.95},
	}
	out := vttformatter.Format(segs)
	if !strings.Contains(out, "WEBVTT") {
		t.Error("must contain WEBVTT header")
	}
	if !strings.Contains(out, "00:00:00.000 --> 00:00:02.000") {
		t.Errorf("unexpected timestamp in:\n%s", out)
	}
	if !strings.Contains(out, "Hello world") {
		t.Error("missing cue text")
	}
}

func TestFormat_LowConfidenceAnnotated(t *testing.T) {
	segs := []vttformatter.Segment{
		{Start: 0, End: time.Second, Text: "uncertain words", Confidence: 0.50},
	}
	out := vttformatter.Format(segs)
	if !strings.Contains(out, "low-confidence") {
		t.Errorf("low-confidence segment should be annotated:\n%s", out)
	}
}

func TestFormat_HighConfidenceNotAnnotated(t *testing.T) {
	segs := []vttformatter.Segment{
		{Start: 0, End: time.Second, Text: "clear speech", Confidence: 0.95},
	}
	out := vttformatter.Format(segs)
	if strings.Contains(out, "low-confidence") {
		t.Errorf("high-confidence segment should not be annotated:\n%s", out)
	}
}

func TestFormat_Timestamps(t *testing.T) {
	segs := []vttformatter.Segment{
		{
			Start: 1*time.Hour + 2*time.Minute + 3*time.Second + 456*time.Millisecond,
			End:   1*time.Hour + 2*time.Minute + 4*time.Second,
			Text:  "one",
			Confidence: 0.90,
		},
	}
	out := vttformatter.Format(segs)
	if !strings.Contains(out, "01:02:03.456 --> 01:02:04.000") {
		t.Errorf("wrong timestamp format:\n%s", out)
	}
}

func TestParseVTT_RoundTrip(t *testing.T) {
	segs := []vttformatter.Segment{
		{Start: 0, End: 2 * time.Second, Text: "First cue", Confidence: 0.9},
		{Start: 2 * time.Second, End: 5 * time.Second, Text: "Second cue", Confidence: 0.8},
	}
	vtt := vttformatter.Format(segs)
	parsed, err := vttformatter.ParseVTT(vtt)
	if err != nil {
		t.Fatalf("ParseVTT: %v", err)
	}
	if len(parsed) != len(segs) {
		t.Fatalf("expected %d segments, got %d", len(segs), len(parsed))
	}
	for i, got := range parsed {
		want := segs[i]
		if got.Start != want.Start {
			t.Errorf("[%d] Start: got %v want %v", i, got.Start, want.Start)
		}
		if got.End != want.End {
			t.Errorf("[%d] End: got %v want %v", i, got.End, want.End)
		}
		// Text may have annotation tags stripped; check plain text contains cue text
		plain := vttformatter.PlainText([]vttformatter.Segment{got})
		if !strings.Contains(plain, stripTags(want.Text)) {
			t.Errorf("[%d] Text mismatch: got %q, want to contain %q", i, plain, want.Text)
		}
	}
}

func TestConfidenceStats(t *testing.T) {
	segs := []vttformatter.Segment{
		{Confidence: 0.9},
		{Confidence: 0.5}, // low
		{Confidence: 0.8},
	}
	avg, hasLow := vttformatter.ConfidenceStats(segs)
	if !hasLow {
		t.Error("expected hasLow=true")
	}
	want := float32((0.9 + 0.5 + 0.8) / 3.0)
	if avg < want-0.01 || avg > want+0.01 {
		t.Errorf("avg: got %v want ~%v", avg, want)
	}
}

func TestConfidenceStats_AllHigh(t *testing.T) {
	segs := []vttformatter.Segment{
		{Confidence: 0.9},
		{Confidence: 0.85},
	}
	_, hasLow := vttformatter.ConfidenceStats(segs)
	if hasLow {
		t.Error("expected hasLow=false when all confidences are high")
	}
}

func TestConfidenceStats_Empty(t *testing.T) {
	avg, hasLow := vttformatter.ConfidenceStats(nil)
	if avg != 0 || hasLow {
		t.Errorf("empty segments: got avg=%v hasLow=%v", avg, hasLow)
	}
}

func TestPlainText_StripsTagsAndJoins(t *testing.T) {
	segs := []vttformatter.Segment{
		{Text: "<c.low-confidence>hello</c>"},
		{Text: "world"},
	}
	got := vttformatter.PlainText(segs)
	if got != "hello world" {
		t.Errorf("PlainText: got %q want %q", got, "hello world")
	}
}

func TestParseVTT_BadTimestamp(t *testing.T) {
	bad := "WEBVTT\n\n1\nbad --> 00:00:01.000\ntext\n\n"
	_, err := vttformatter.ParseVTT(bad)
	if err == nil {
		t.Error("expected error for bad timestamp, got nil")
	}
}

// stripTags is a test helper to remove <tag> patterns from expected text.
func stripTags(s string) string {
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
