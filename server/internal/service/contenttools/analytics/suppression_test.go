package analytics

import "testing"

func TestClampMinRespondents(t *testing.T) {
	if got := ClampMinRespondents(0); got != DefaultSmallN {
		t.Fatalf("default: got %d", got)
	}
	if got := ClampMinRespondents(2); got != MinRespondentsFloor {
		t.Fatalf("floor: got %d", got)
	}
	if got := ClampMinRespondents(7); got != 7 {
		t.Fatalf("passthrough: got %d", got)
	}
}

func TestAggregateWithSuppression_SmallN(t *testing.T) {
	counts := map[string]int{"a": 2, "b": 1}
	got := AggregateWithSuppression(counts, 3, 5, true)
	if !got.Suppressed || got.Reason != "small_n" {
		t.Fatalf("want suppressed small_n, got %#v", got)
	}
	if len(got.Options) != 0 {
		t.Fatalf("options must be withheld: %#v", got.Options)
	}
	if got.Learners != 3 {
		t.Fatalf("learners: %d", got.Learners)
	}
}

func TestAggregateWithSuppression_Percents(t *testing.T) {
	counts := map[string]int{"a": 3, "b": 2}
	got := AggregateWithSuppression(counts, 5, 5, true)
	if got.Suppressed {
		t.Fatal("should not suppress")
	}
	if len(got.Options) != 2 {
		t.Fatalf("options: %#v", got.Options)
	}
	if got.Options[0].OptionID != "a" || got.Options[0].Count != 3 || got.Options[0].Percent != 60 {
		t.Fatalf("a: %#v", got.Options[0])
	}
	if got.Options[1].OptionID != "b" || got.Options[1].Percent != 40 {
		t.Fatalf("b: %#v", got.Options[1])
	}
}
