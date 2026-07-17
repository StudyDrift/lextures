package bannerevents

import "testing"

func TestHub_PublishAndSubscribe(t *testing.T) {
	t.Parallel()
	h := New()
	ch, unsub := h.Subscribe()
	defer unsub()

	h.Cleared("banner-1", "global", "")
	select {
	case ev := <-ch:
		if ev.Type != "banner_changed" || ev.Action != "cleared" || ev.ID != "banner-1" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	default:
		t.Fatal("expected event on subscribe channel")
	}
}

func TestHub_NilSafe(t *testing.T) {
	t.Parallel()
	var h *Hub
	ch, unsub := h.Subscribe()
	unsub()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("nil hub subscribe channel should be closed")
		}
	default:
		t.Fatal("nil hub subscribe channel should be closed")
	}
	h.Cleared("x", "global", "")
	h.Upserted("x", "org", "org-1")
}
