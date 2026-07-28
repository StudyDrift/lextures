package flashcards

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/service/srs"
)

func TestGradeToQualityMapping(t *testing.T) {
	cases := map[string]float64{
		"again": 0,
		"hard":  2,
		"good":  4,
		"easy":  5,
		"AGAIN": 0,
	}
	for g, want := range cases {
		got, ok := srs.GradeToQuality(g)
		if !ok || got != want {
			t.Fatalf("%s: got %v ok=%v want %v", g, got, ok, want)
		}
	}
	if _, ok := srs.GradeToQuality("meh"); ok {
		t.Fatal("expected invalid grade")
	}
}

func TestSelectSessionQueueNewAndCap(t *testing.T) {
	cfg := Config{
		Cards: []Card{
			{ID: "a", Front: "A", Back: "1"},
			{ID: "b", Front: "B", Back: "2"},
			{ID: "c", Front: "C", Back: "3"},
			{ID: "d", Front: "D", Back: "4"},
			{ID: "e", Front: "E", Back: "5"},
		},
		SessionCap: 3,
		Shuffle:    false,
	}
	st := EmptyState()
	q := SelectSessionQueue(cfg, st, nil, time.Now().UTC(), "seed")
	if len(q) != 3 {
		t.Fatalf("want 3, got %d: %#v", len(q), q)
	}
	for _, it := range q {
		if it.Side != SideForward {
			t.Fatalf("unexpected side %#v", it)
		}
	}
}

func TestSelectSessionQueueDuePrefer(t *testing.T) {
	cfg := Config{
		Cards: []Card{
			{ID: "a", Front: "A", Back: "1"},
			{ID: "b", Front: "B", Back: "2"},
			{ID: "c", Front: "C", Back: "3"},
		},
		SessionCap: 10,
		Shuffle:    false,
	}
	st := EmptyState()
	due := map[string]CardDueInfo{
		"b|forward": {CardID: "b", Side: SideForward, IsDue: true},
		"a|forward": {CardID: "a", Side: SideForward, IsNew: true},
		"c|forward": {CardID: "c", Side: SideForward, IsNew: true},
	}
	q := SelectSessionQueue(cfg, st, due, time.Now().UTC(), "x")
	if len(q) < 1 || q[0].CardID != "b" {
		t.Fatalf("due should come first: %#v", q)
	}
}

func TestSelectSessionQueueReverseIndependent(t *testing.T) {
	cfg := Config{
		Cards: []Card{
			{ID: "a", Front: "A", Back: "1"},
			{ID: "b", Front: "B", Back: "2"},
			{ID: "c", Front: "C", Back: "3"},
		},
		ReversePractice: true,
		SessionCap:      20,
		Shuffle:         false,
	}
	q := SelectSessionQueue(cfg, EmptyState(), nil, time.Now().UTC(), "s")
	if len(q) != 6 {
		t.Fatalf("want 6 (3 forward + 3 reverse), got %d", len(q))
	}
}

func TestFirstPassCompleteAndApplyRating(t *testing.T) {
	cfg := Config{
		Cards: []Card{
			{ID: "a", Front: "A", Back: "1"},
			{ID: "b", Front: "B", Back: "2"},
			{ID: "c", Front: "C", Back: "3"},
		},
		RequireFirstPass: true,
	}
	st := EmptyState()
	st.ActiveSession = &ActiveSession{
		StartedAt: NowRFC3339(),
		Queue:     []QueueItem{{CardID: "a", Side: SideForward}},
		Index:     0,
	}
	ApplyRating(&st, "a", RatingGood, NowRFC3339())
	if FirstPassComplete(cfg, st) {
		t.Fatal("not complete yet")
	}
	ApplyRating(&st, "b", RatingHard, NowRFC3339())
	ApplyRating(&st, "c", RatingEasy, NowRFC3339())
	if !FirstPassComplete(cfg, st) {
		t.Fatal("expected complete")
	}
	if st.Cards["a"].FirstRating == nil || *st.Cards["a"].FirstRating != RatingGood {
		t.Fatalf("first rating: %#v", st.Cards["a"])
	}
}

func TestParseConfigDefaults(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"cards": []map[string]any{
			{"id": "1", "front": "f1", "back": "b1"},
			{"id": "2", "front": "f2", "back": "b2"},
			{"id": "3", "front": "f3", "back": "b3"},
		},
	})
	cfg := ParseConfig(raw)
	if !cfg.Shuffle || !cfg.RequireFirstPass || cfg.SessionCap != 20 {
		t.Fatalf("defaults: %#v", cfg)
	}
	if len(cfg.Cards) != 3 {
		t.Fatalf("cards: %#v", cfg.Cards)
	}
}

func TestQuestionIDStable(t *testing.T) {
	inst := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	a := QuestionIDFor(inst, "card-a", SideForward)
	b := QuestionIDFor(inst, "card-a", SideForward)
	c := QuestionIDFor(inst, "card-a", SideReverse)
	if a != b {
		t.Fatal("ids must be stable")
	}
	if a == c {
		t.Fatal("forward/reverse must differ")
	}
}
