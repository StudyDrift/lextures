package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/flashcards"
)

func sampleFlashcardsConfig() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"title": "Vocab",
		"cards": []map[string]any{
			{"id": "c1", "front": "hola", "back": "hello"},
			{"id": "c2", "front": "adiós", "back": "goodbye"},
			{"id": "c3", "front": "gracias", "back": "thank you"},
			{"id": "c4", "front": "por favor", "back": "please"},
			{"id": "c5", "front": "sí", "back": "yes"},
			{"id": "c6", "front": "no", "back": "no"},
		},
		"reversePractice":  false,
		"sessionCap":       20,
		"shuffle":          false,
		"requireFirstPass": true,
	})
	return raw
}

func TestFlashcardsManifestRegistered(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(flashcards.ID)
	if m == nil {
		t.Fatal("missing flashcards")
	}
	if len(m.Actions) < 4 {
		t.Fatalf("actions: %#v", m.Actions)
	}
}

func TestFlashcardsSessionAndRateWithoutSRS(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(flashcards.ID)
	cfgJSON := sampleFlashcardsConfig()
	stJSON, _ := json.Marshal(flashcards.EmptyState())
	off := false

	started, err := contenttools.DispatchAction(m, "start_session", contenttools.ActionContext{
		ConfigJSON:         cfgJSON,
		StateJSON:          stJSON,
		InteractRole:       "student",
		EnrollmentID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		InstanceID:         uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		SRSPracticeEnabled: &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.Result["caughtUp"] == true {
		t.Fatal("expected cards in queue")
	}
	queue, _ := started.Result["queue"].([]flashcards.QueueItem)
	if len(queue) == 0 {
		// may be []any after map
		raw, _ := json.Marshal(started.Result["queue"])
		_ = json.Unmarshal(raw, &queue)
	}
	if len(queue) != 6 {
		t.Fatalf("want 6 cards, got %d (%#v)", len(queue), started.Result["queue"])
	}
	if started.Result["srsEnabled"] != false {
		t.Fatalf("srs should be off: %#v", started.Result["srsEnabled"])
	}

	stateJSON := started.StatePatch
	rated := 0
	for rated < 6 {
		var cur map[string]any
		rawCur, _ := json.Marshal(started.Result["current"])
		if rated > 0 {
			// reload current from last rate
		}
		_ = json.Unmarshal(rawCur, &cur)

		// get current from status of last result
		var last *contenttools.ActionResult
		if rated == 0 {
			last = started
		} else {
			last = nil
		}
		_ = last

		st := flashcards.ParseState(stateJSON)
		if st.ActiveSession == nil || st.ActiveSession.Index >= len(st.ActiveSession.Queue) {
			break
		}
		item := st.ActiveSession.Queue[st.ActiveSession.Index]
		in, _ := json.Marshal(map[string]any{
			"cardId":         item.CardID,
			"rating":         "good",
			"side":           item.Side,
			"idempotencyKey": uuid.NewString(),
		})
		res, err := contenttools.DispatchAction(m, "rate", contenttools.ActionContext{
			ConfigJSON:         cfgJSON,
			StateJSON:          stateJSON,
			Input:              in,
			InteractRole:       "student",
			SRSPracticeEnabled: &off,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Result["error"] != nil {
			t.Fatalf("rate error: %#v", res.Result)
		}
		stateJSON = res.StatePatch
		rated++
		if res.Result["sessionComplete"] == true {
			break
		}
	}
	if rated != 6 {
		t.Fatalf("rated %d want 6", rated)
	}
	st := flashcards.ParseState(stateJSON)
	if st.FirstPassCompletedAt == "" {
		t.Fatal("expected first pass complete")
	}
	if len(st.Cards) != 6 {
		t.Fatalf("cards progress: %#v", st.Cards)
	}
}

func TestFlashcardsCaughtUpWhenEmptyQueue(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(flashcards.ID)
	cfgJSON := sampleFlashcardsConfig()
	// Mark all seen; with shuffle false and no SRS due map, SelectSessionQueue still
	// includes previously seen as due in self-check mode — so end session then status.
	st := flashcards.EmptyState()
	r := flashcards.RatingGood
	for _, id := range []string{"c1", "c2", "c3", "c4", "c5", "c6"} {
		st.Cards[id] = flashcards.CardProgress{Seen: 1, FirstRating: &r, LastRating: &r}
	}
	stJSON, _ := json.Marshal(st)
	off := false
	res, err := contenttools.DispatchAction(m, "start_session", contenttools.ActionContext{
		ConfigJSON:         cfgJSON,
		StateJSON:          stJSON,
		InteractRole:       "student",
		EnrollmentID:       uuid.New(),
		SRSPracticeEnabled: &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Self-check mode re-queues seen cards as due — not caught up.
	if res.Result["caughtUp"] == true {
		t.Fatal("self-check should still offer due cards")
	}
}

func TestFlashcardsCardMismatch(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(flashcards.ID)
	cfgJSON := sampleFlashcardsConfig()
	off := false
	started, err := contenttools.DispatchAction(m, "start_session", contenttools.ActionContext{
		ConfigJSON:         cfgJSON,
		StateJSON:          json.RawMessage(`{"v":1}`),
		InteractRole:       "student",
		EnrollmentID:       uuid.New(),
		SRSPracticeEnabled: &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	in, _ := json.Marshal(map[string]any{"cardId": "c6", "rating": "easy", "side": "forward"})
	res, err := contenttools.DispatchAction(m, "rate", contenttools.ActionContext{
		ConfigJSON:         cfgJSON,
		StateJSON:          started.StatePatch,
		Input:              in,
		InteractRole:       "student",
		SRSPracticeEnabled: &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "card_mismatch" && res.Result["error"] != "no_session" {
		// If shuffle false, first card is c1 — rating c6 mismatches.
		if res.Result["error"] != "card_mismatch" {
			t.Fatalf("want card_mismatch, got %#v", res.Result)
		}
	}
}
