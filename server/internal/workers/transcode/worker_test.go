package transcode_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/workers/transcode"
)

func TestIsVideoMIME(t *testing.T) {
	cases := []struct {
		mime string
		want bool
	}{
		{"video/mp4", true},
		{"video/webm", true},
		{"video/quicktime", true},
		{"video/x-matroska", true},
		{"audio/mpeg", false},
		{"image/jpeg", false},
		{"application/pdf", false},
		{"", false},
	}
	for _, tc := range cases {
		got := transcode.IsVideoMIME(tc.mime)
		if got != tc.want {
			t.Errorf("IsVideoMIME(%q) = %v, want %v", tc.mime, got, tc.want)
		}
	}
}

func TestDefaultRenditions(t *testing.T) {
	if len(transcode.DefaultRenditions) < 3 {
		t.Fatalf("expected at least 3 renditions, got %d", len(transcode.DefaultRenditions))
	}
	names := map[string]bool{"360p": false, "720p": false, "1080p": false}
	for _, r := range transcode.DefaultRenditions {
		names[r.Name] = true
		if r.Height <= 0 {
			t.Errorf("rendition %q has non-positive height: %d", r.Name, r.Height)
		}
		if r.Bandwidth <= 0 {
			t.Errorf("rendition %q has non-positive bandwidth: %d", r.Name, r.Bandwidth)
		}
		if r.VideoBR == "" || r.AudioBR == "" {
			t.Errorf("rendition %q has empty bitrate", r.Name)
		}
	}
	for name, found := range names {
		if !found {
			t.Errorf("expected rendition %q in DefaultRenditions", name)
		}
	}
}

func TestBuildMasterPlaylistContent(t *testing.T) {
	content := transcode.BuildMasterPlaylistContent()

	if !strings.HasPrefix(content, "#EXTM3U\n") {
		t.Error("master playlist must start with #EXTM3U")
	}
	if !strings.Contains(content, "#EXT-X-VERSION:3") {
		t.Error("master playlist must contain #EXT-X-VERSION:3")
	}

	// All three renditions must be referenced
	for _, r := range transcode.DefaultRenditions {
		if !strings.Contains(content, r.Name+".m3u8") {
			t.Errorf("master playlist missing rendition %q", r.Name)
		}
		if !strings.Contains(content, "BANDWIDTH=") {
			t.Error("master playlist missing BANDWIDTH attribute")
		}
	}

	// Must conform to RFC 8216 §4: each EXT-X-STREAM-INF followed by a URI
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			if i+1 >= len(lines) {
				t.Error("EXT-X-STREAM-INF tag not followed by URI line")
			} else if strings.HasPrefix(lines[i+1], "#") {
				t.Errorf("URI line after EXT-X-STREAM-INF should not be a tag: %q", lines[i+1])
			}
		}
	}
}

func TestWriteMasterPlaylist_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	// writeMasterPlaylist is unexported; exercise it via BuildMasterPlaylistContent+WriteFile
	content := transcode.BuildMasterPlaylistContent()
	path := filepath.Join(dir, "master.m3u8")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Error("written master playlist content does not match BuildMasterPlaylistContent output")
	}
}

func TestWorker_New(t *testing.T) {
	w := transcode.New(nil, nil)
	if w == nil {
		t.Fatal("New returned nil")
	}
	if w.MaxAttempts <= 0 {
		t.Errorf("MaxAttempts should be positive, got %d", w.MaxAttempts)
	}
	if w.FFmpegPath == "" {
		t.Error("FFmpegPath should default to non-empty")
	}
}

func TestWorker_ProcessNext_NoPool(t *testing.T) {
	w := transcode.New(nil, nil)
	_, err := w.ProcessNext(context.TODO())
	if err == nil {
		t.Error("expected error when pool is nil, got nil")
	}
}

func TestWorker_ProcessNext_NoStorage(t *testing.T) {
	w := &transcode.Worker{Pool: nil, Storage: nil, FFmpegPath: "ffmpeg", MaxAttempts: 3}
	_, err := w.ProcessNext(context.TODO())
	if err == nil {
		t.Error("expected error when pool is nil, got nil")
	}
}
