package tts

import (
	"testing"
)

func TestStubWAVNonEmpty(t *testing.T) {
	wav := StubWAV("Hello world from read aloud.", 1.0)
	if len(wav) < 44 {
		t.Fatalf("expected WAV header + data, got %d bytes", len(wav))
	}
	if string(wav[0:4]) != "RIFF" {
		t.Fatalf("expected RIFF header, got %q", wav[0:4])
	}
}

func TestStubWAVScalesWithSpeed(t *testing.T) {
	slow := StubWAV("one two three four five six seven eight", 0.75)
	fast := StubWAV("one two three four five six seven eight", 2.0)
	if len(slow) <= len(fast) {
		t.Fatalf("slower speed should produce longer audio: slow=%d fast=%d", len(slow), len(fast))
	}
}
