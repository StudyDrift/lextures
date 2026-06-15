package accessibility

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildWrite_ExtendedTimePresets(t *testing.T) {
	w := BuildWrite([]string{TypeExtendedTime15}, nil)
	if w.TimeMultiplier != 1.5 {
		t.Fatalf("1.5x preset: want 1.5 got %v", w.TimeMultiplier)
	}
	w = BuildWrite([]string{TypeExtendedTime2}, nil)
	if w.TimeMultiplier != 2.0 {
		t.Fatalf("2x preset: want 2.0 got %v", w.TimeMultiplier)
	}
}

func TestBuildWrite_CustomMultiplierWins(t *testing.T) {
	w := BuildWrite([]string{TypeExtendedTime15}, json.RawMessage(`{"timeMultiplier":1.75}`))
	if w.TimeMultiplier != 1.75 {
		t.Fatalf("custom multiplier: want 1.75 got %v", w.TimeMultiplier)
	}
}

func TestBuildWrite_FlagsAndFormat(t *testing.T) {
	w := BuildWrite(
		[]string{TypeSeparateTesting, TypeScreenReader, TypeSpeechToText, TypeReducedDistract, TypeAlternateFormat},
		json.RawMessage(`{"alternateFormat":"braille"}`),
	)
	if !w.SeparateSetting || !w.TTSEnabled || !w.SpeechToTextEnabled || !w.ReducedDistraction {
		t.Fatalf("expected all flags set: %+v", w)
	}
	if w.AlternativeFormat == nil || *w.AlternativeFormat != "braille" {
		t.Fatalf("alternate format: want braille got %v", w.AlternativeFormat)
	}
	if w.TimeMultiplier != 1.0 {
		t.Fatalf("no extended time: want 1.0 got %v", w.TimeMultiplier)
	}
}

func TestValidTypes(t *testing.T) {
	if ValidTypes(nil) {
		t.Fatal("empty should be invalid")
	}
	if !ValidTypes([]string{TypeExtendedTime15, TypeOther}) {
		t.Fatal("valid list rejected")
	}
	if ValidTypes([]string{"bogus"}) {
		t.Fatal("bogus type accepted")
	}
}

func TestRenderLetter_NoDisabilityDisclosure(t *testing.T) {
	letter := RenderLetter("Jordan Lee", "2026-06-14", []string{TypeExtendedTime15})
	if !strings.Contains(letter, "Jordan Lee") {
		t.Fatal("letter missing student name")
	}
	if !strings.Contains(letter, "1.5x Extended Time") {
		t.Fatal("letter missing accommodation label")
	}
	for _, banned := range []string{"disability", "diagnosis", "diagnos", "medical"} {
		// The letter explicitly states it does not disclose the disability; ensure no
		// diagnosis-style wording leaks beyond that single reassurance sentence.
		if strings.Count(strings.ToLower(letter), banned) > 1 {
			t.Fatalf("letter appears to reference %q more than the reassurance line", banned)
		}
	}
}
