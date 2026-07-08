package learnerprofile

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

type fakeDeriver struct {
	key      string
	version  int
	min      int
	result   FacetResult
	err      error
	calls    int
	panicOn  int
}

func (f *fakeDeriver) Key() string { return f.key }

func (f *fakeDeriver) Version() int { return f.version }

func (f *fakeDeriver) MinSignals() int { return f.min }

func (f *fakeDeriver) Derive(ctx context.Context, _ uuid.UUID) (FacetResult, error) {
	f.calls++
	if f.panicOn > 0 && f.calls == f.panicOn {
		panic("deriver boom")
	}
	if f.err != nil {
		return FacetResult{}, f.err
	}
	return f.result, nil
}

func TestFacetResultIdempotentWriteShape(t *testing.T) {
	value, _ := json.Marshal(map[string]int{"n": 7})
	contrib := 1.0
	r := FacetResult{
		State:           "ok",
		Summary:         json.RawMessage(`{"k":1}`),
		Confidence:      0.75,
		ComputedVersion: 2,
		Insights: []InsightResult{
			{
				InsightKey:   "alpha",
				LabelI18nKey: "test.alpha",
				Value:        value,
				Confidence:   0.75,
				Salience:     10,
				Evidence: []EvidenceResult{
					{SourceKind: "engagement_event", SourceTable: "analytics.engagement_events", ObservationCount: 7, Contribution: &contrib},
				},
			},
		},
	}
	w1 := facetResultToWrite(r)
	w2 := facetResultToWrite(r)
	b1, _ := json.Marshal(w1)
	b2, _ := json.Marshal(w2)
	if string(b1) != string(b2) {
		t.Fatalf("write payloads differ:\n%s\n%s", b1, b2)
	}
}

func TestInsufficientDataHasNoInsights(t *testing.T) {
	r := FacetResult{State: "insufficient_data", ComputedVersion: 1}
	w := facetResultToWrite(r)
	if len(w.Insights) != 0 {
		t.Fatalf("expected no insights, got %d", len(w.Insights))
	}
	if w.State != "insufficient_data" {
		t.Fatalf("state=%q", w.State)
	}
}

func TestSafeDeriveRecoversPanic(t *testing.T) {
	bad := &fakeDeriver{key: "study_rhythm", panicOn: 1}
	_, err := safeDerive(context.Background(), bad, uuid.New())
	if err == nil {
		t.Fatal("expected panic error")
	}
}

func TestResolveLabelFallback(t *testing.T) {
	if got := ResolveLabel("en", "learner_profile.study_rhythm.peak_study_window"); got == "" {
		t.Fatal("expected label")
	}
	if got := ResolveLabel("en", "unknown.key"); got != "unknown.key" {
		t.Fatalf("got %q", got)
	}
}