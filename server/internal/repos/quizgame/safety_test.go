package quizgame

import (
	"testing"

	"github.com/lextures/lextures/server/internal/service/boardfilter"
)

func TestScreenNickname_BlocksProfanity(t *testing.T) {
	if err := ScreenNickname("Ada"); err != nil {
		t.Fatalf("clean nickname: %v", err)
	}
	term := boardfilter.DefaultEnglish[0]
	if err := ScreenNickname(term); err != ErrNicknameDenied {
		t.Fatalf("want ErrNicknameDenied, got %v", err)
	}
	if err := ScreenNickname("xx" + term + "yy"); err != ErrNicknameDenied {
		t.Fatalf("embedded term: %v", err)
	}
}

func TestScreenOpenText_FilterDistribution(t *testing.T) {
	term := boardfilter.DefaultEnglish[0]
	denied, matched := ScreenOpenText("hello " + term)
	if !denied || matched == "" {
		t.Fatalf("expected deny, denied=%v term=%q", denied, matched)
	}
	dist := map[string]int{
		"a":    3,
		term:   2,
		"nice": 1,
	}
	filtered := FilterDistributionForProjector(dist)
	if _, ok := filtered[term]; ok {
		t.Fatal("profane key should be withheld from projector")
	}
	if filtered["a"] != 3 || filtered["nice"] != 1 {
		t.Fatalf("unexpected filtered: %#v", filtered)
	}
}

func TestNormalizeOneSessionRule(t *testing.T) {
	if NormalizeOneSessionRule("") != OneSessionTakeover {
		t.Fatal("default takeover")
	}
	if NormalizeOneSessionRule("refuse") != OneSessionRefuse {
		t.Fatal("refuse")
	}
	if NormalizeOneSessionRule("OFF") != OneSessionOff {
		t.Fatal("off")
	}
}

func TestDisplayNickname_Muted(t *testing.T) {
	if got := DisplayNickname(true, 0, "Ada"); got != "Player 1" {
		t.Fatalf("got %q", got)
	}
	if got := DisplayNickname(false, 0, "Ada"); got != "Ada" {
		t.Fatalf("got %q", got)
	}
}

func TestHashJoinIP_StablePerSession(t *testing.T) {
	a := HashJoinIP("sess-1", "1.2.3.4")
	b := HashJoinIP("sess-1", "1.2.3.4")
	c := HashJoinIP("sess-2", "1.2.3.4")
	if a == "" || a != b {
		t.Fatalf("stable hash failed: %q %q", a, b)
	}
	if a == c {
		t.Fatal("salt should differ per session")
	}
}
