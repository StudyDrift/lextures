package contenttools

import (
	"strings"
	"testing"
)

func TestParseSerializeLexToolFenceRoundTrip(t *testing.T) {
	p := LexToolFencePayload{
		InstanceID: "11111111-1111-1111-1111-111111111111",
		ToolID:     "noop_probe",
		V:          1,
	}
	fence := SerializeLexToolFence(p)
	want := "```lex-tool\n{\"instanceId\":\"11111111-1111-1111-1111-111111111111\",\"toolId\":\"noop_probe\",\"v\":1}\n```"
	if fence != want {
		t.Fatalf("serialize mismatch:\n got: %s\nwant: %s", fence, want)
	}
	parsed := ParseLexToolFences("intro\n" + fence + "\noutro")
	if len(parsed) != 1 {
		t.Fatalf("expected 1 fence, got %d", len(parsed))
	}
	if parsed[0] != p {
		t.Fatalf("parse mismatch: %+v", parsed[0])
	}
}

func TestParseLexToolFencesSkipsInvalid(t *testing.T) {
	md := "x\n```lex-tool\nnot-json\n```\n```lex-tool\n{\"instanceId\":\"aaa\",\"toolId\":\"t\",\"v\":1}\n```\n"
	parsed := ParseLexToolFences(md)
	if len(parsed) != 1 || parsed[0].InstanceID != "aaa" {
		t.Fatalf("got %+v", parsed)
	}
}

func TestStripInvalidLexToolFences(t *testing.T) {
	good := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	bad := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	md := "before\n" +
		SerializeLexToolFence(LexToolFencePayload{InstanceID: good, ToolID: "noop_probe", V: 1}) +
		"\nmiddle\n" +
		SerializeLexToolFence(LexToolFencePayload{InstanceID: bad, ToolID: "noop_probe", V: 1}) +
		"\nafter\n```lex-tool\n{bad\n```\n"
	out := StripInvalidLexToolFences(md, map[string]bool{good: true})
	if !strings.Contains(out, good) {
		t.Fatalf("expected valid fence kept: %s", out)
	}
	if strings.Contains(out, bad) {
		t.Fatalf("expected invalid id stripped: %s", out)
	}
	if strings.Contains(out, "{bad") {
		t.Fatalf("expected invalid JSON fence stripped: %s", out)
	}
}

func TestRewriteLexToolFencesUsesStableKeys(t *testing.T) {
	md := "```lex-tool\n{\"instanceId\":\"old\",\"toolId\":\"noop_probe\",\"v\":1}\n```"
	out := RewriteLexToolFences(md, map[string]string{"old": "new"})
	want := "```lex-tool\n{\"instanceId\":\"new\",\"toolId\":\"noop_probe\",\"v\":1}\n```"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}
