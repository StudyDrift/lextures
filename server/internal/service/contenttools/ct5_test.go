package contenttools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestClassifySchemaDiff_MajorOnRemovedField(t *testing.T) {
	oldSchema := json.RawMessage(`{
		"type":"object",
		"required":["prompt"],
		"properties":{"prompt":{"type":"string"},"answerKey":{"type":"string"}}
	}`)
	newSchema := json.RawMessage(`{
		"type":"object",
		"required":["prompt"],
		"properties":{"prompt":{"type":"string"}}
	}`)
	kind, findings, err := ClassifySchemaDiff(oldSchema, newSchema)
	if err != nil {
		t.Fatal(err)
	}
	if kind != BumpMajor {
		t.Fatalf("expected major, got %s findings=%v", kind, findings)
	}
	err = AssertVersionCoversSchemaDiff("1.0.0", "1.1.0", oldSchema, newSchema)
	if err == nil || !strings.Contains(err.Error(), "answerKey") {
		t.Fatalf("expected CI failure naming answerKey, got %v", err)
	}
}

func TestClassifySchemaDiff_MinorOnAdditiveOptional(t *testing.T) {
	oldSchema := json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}}}`)
	newSchema := json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}}}`)
	kind, _, err := ClassifySchemaDiff(oldSchema, newSchema)
	if err != nil {
		t.Fatal(err)
	}
	if kind != BumpMinor {
		t.Fatalf("expected minor, got %s", kind)
	}
	if err := AssertVersionCoversSchemaDiff("1.0.0", "1.1.0", oldSchema, newSchema); err != nil {
		t.Fatal(err)
	}
}

func TestResolveWithinMajor(t *testing.T) {
	got, err := ResolveWithinMajor("1.4.0", []string{"1.2.0", "1.5.1", "2.0.0", "1.4.2"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.5.1" {
		t.Fatalf("got %s want 1.5.1", got)
	}
}

func TestApplyStateMigrations_LazyAndQuarantine(t *testing.T) {
	table := &MigrationTable{
		ToolID:             "demo",
		StateSchemaVersion: 2,
		State: map[int]DocMigration{
			1: func(doc json.RawMessage) (json.RawMessage, error) {
				var m map[string]any
				_ = json.Unmarshal(doc, &m)
				if m == nil {
					m = map[string]any{}
				}
				m["migrated"] = true
				return json.Marshal(m)
			},
		},
	}
	res := ApplyStateMigrations(table, 1, json.RawMessage(`{"response":"hi"}`))
	if res.Quarantine || res.ToVersion != 2 {
		t.Fatalf("unexpected %#v", res)
	}
	if !strings.Contains(string(res.Doc), `"migrated":true`) {
		t.Fatalf("doc=%s", res.Doc)
	}
	// Idempotent when already at target.
	res2 := ApplyStateMigrations(table, 2, res.Doc)
	if !res2.Unchanged {
		t.Fatal("expected unchanged")
	}
	failTable := &MigrationTable{
		ToolID: "demo", StateSchemaVersion: 2,
		State: map[int]DocMigration{
			1: func(doc json.RawMessage) (json.RawMessage, error) {
				return nil, errMigrationBoom
			},
		},
	}
	bad := ApplyStateMigrations(failTable, 1, json.RawMessage(`{"x":1}`))
	if !bad.Quarantine || string(bad.Doc) != `{"x":1}` {
		t.Fatalf("quarantine should preserve original: %#v", bad)
	}
}

var errMigrationBoom = errStringError("boom")

type errStringError string

func (e errStringError) Error() string { return string(e) }

func TestBreakerTrips(t *testing.T) {
	b := NewBreaker(3, time.Minute)
	now := time.Now().UTC()
	b.RecordFailure("t", "e1", now)
	b.RecordFailure("t", "e2", now.Add(time.Second))
	if b.IsOpen("t") {
		t.Fatal("should still be closed")
	}
	st := b.RecordFailure("t", "e3", now.Add(2*time.Second))
	if !st.Open || !b.IsOpen("t") {
		t.Fatal("expected open")
	}
	b.Reset("t")
	if b.IsOpen("t") {
		t.Fatal("expected reset")
	}
}

func TestEffectiveSandboxMode(t *testing.T) {
	t.Setenv(EnvSandboxMode, SandboxModeOff)
	if EffectiveSandboxMode(SandboxIframe) != SandboxInProcess {
		t.Fatal("off should force inprocess")
	}
	t.Setenv(EnvSandboxMode, SandboxModeRequired)
	if EffectiveSandboxMode(SandboxInProcess) != SandboxIframe {
		t.Fatal("required should force iframe")
	}
	t.Setenv(EnvSandboxMode, SandboxModeOptIn)
	if EffectiveSandboxMode(SandboxIframe) != SandboxIframe {
		t.Fatal("optin should honor manifest iframe")
	}
	if EffectiveSandboxMode("") != SandboxInProcess {
		t.Fatal("optin default inprocess")
	}
}

func TestSandboxProbeRegistered(t *testing.T) {
	m := MustDefault().Get("sandbox_probe")
	if m == nil {
		t.Fatal("missing sandbox_probe")
	}
	if m.Sandbox != SandboxIframe {
		t.Fatalf("sandbox=%s", m.Sandbox)
	}
	if LookupActionHandler("sandbox_probe", "grade") == nil {
		t.Fatal("missing grade handler")
	}
}

func TestFilterCatalog_HidesDeprecated(t *testing.T) {
	base := MustDefault()
	probe := base.Get("noop_probe")
	dep := *probe
	dep.Deprecated = true
	dep.ID = "deprecated_probe"
	reg, err := NewRegistry([]*CompiledManifest{probe, &dep})
	if err != nil {
		t.Fatal(err)
	}
	out := FilterCatalog(reg, nil, "", nil)
	for _, ttool := range out {
		if ttool.ID == "deprecated_probe" {
			t.Fatal("deprecated tool should be hidden from palette")
		}
	}
}
