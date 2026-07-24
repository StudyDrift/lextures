package badgesextraction

import (
	"strings"
	"testing"
)

func TestParseDraftBadgesJSON_FromOutcomes(t *testing.T) {
	t.Parallel()
	valid := map[string]struct{}{"o1": {}, "o2": {}}
	raw := `{"badges":[
		{"outcomeId":"o1","name":"  Data Analysis  ","description":" Shows mastery of analysis. "},
		{"outcomeId":"bogus","name":"Keep Without Link","description":"ok"},
		{"outcomeId":"o1","name":"Duplicate Outcome","description":"skip"},
		{"outcomeId":"o2","name":"","description":"skip empty name"}
	]}`
	got, err := ParseDraftBadgesJSON(raw, valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2: %#v", len(got), got)
	}
	if got[0].Name != "Data Analysis" || got[0].OutcomeID == nil || *got[0].OutcomeID != "o1" {
		t.Fatalf("first = %#v", got[0])
	}
	if got[1].Name != "Keep Without Link" || got[1].OutcomeID != nil {
		t.Fatalf("second = %#v", got[1])
	}
}

func TestParseDraftBadgesJSON_FromSyllabus(t *testing.T) {
	t.Parallel()
	raw := "```json\n{\"badges\":[{\"name\":\"Critical Thinking\",\"description\":\"Evaluates sources.\"}]}\n```"
	got, err := ParseDraftBadgesJSON(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Critical Thinking" || got[0].OutcomeID != nil {
		t.Fatalf("got %#v", got)
	}
}

func TestParseDraftBadgesJSON_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := ParseDraftBadgesJSON("not json", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDraftBadgesJSON_CapsNameAndCount(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("a", MaxNameRunes+10)
	var b strings.Builder
	b.WriteString(`{"badges":[`)
	for i := 0; i < MaxBadges+3; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"name":"`)
		if i == 0 {
			b.WriteString(longName)
		} else {
			b.WriteString("Badge ")
			b.WriteString(strings.Repeat("x", i+1))
		}
		b.WriteString(`","description":""}`)
	}
	b.WriteString(`]}`)
	got, err := ParseDraftBadgesJSON(b.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxBadges {
		t.Fatalf("len=%d want %d", len(got), MaxBadges)
	}
	if utf8Len(got[0].Name) != MaxNameRunes {
		t.Fatalf("name runes=%d want %d", utf8Len(got[0].Name), MaxNameRunes)
	}
}

func utf8Len(s string) int {
	return len([]rune(s))
}
