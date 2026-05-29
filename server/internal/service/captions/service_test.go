package captions

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/service/vttformatter"
)

const sampleSRT = `1
00:00:01,000 --> 00:00:04,000
Hello world.

2
00:00:05,000 --> 00:00:08,500
Second cue.
`

func TestVTTFromSRT(t *testing.T) {
	vtt, err := VTTFromSRT(sampleSRT)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(vtt, "WEBVTT") {
		t.Fatalf("expected WEBVTT header, got %q", vtt[:20])
	}
	if !strings.Contains(vtt, "Hello world.") {
		t.Fatalf("missing cue text: %s", vtt)
	}
}

func TestSRTFromVTT_roundTrip(t *testing.T) {
	segments, err := ParseSRT(sampleSRT)
	if err != nil {
		t.Fatal(err)
	}
	vtt := FormatVTT(segments)
	back, err := ParseVTT(vtt)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(back))
	}
	if PlainTranscript(back) == "" {
		t.Fatal("empty transcript")
	}
}

func TestParseSRT_empty(t *testing.T) {
	if _, err := ParseSRT(""); err == nil {
		t.Fatal("expected error for empty SRT")
	}
}

func TestCourseFileIDPattern(t *testing.T) {
	md := `See [video](/api/v1/courses/C-ABC/course-files/550e8400-e29b-41d4-a716-446655440000/content).`
	m := CourseFileIDPattern.FindStringSubmatch(md)
	if m == nil || m[1] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected match: %v", m)
	}
}

func TestPlainTranscript_stripsTags(t *testing.T) {
	segs := []vttformatter.Segment{{Text: "<c.low-confidence>oops</c>"}}
	if got := PlainTranscript(segs); got != "oops" {
		t.Fatalf("got %q", got)
	}
}
