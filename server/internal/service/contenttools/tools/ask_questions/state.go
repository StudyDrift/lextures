package ask_questions

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

var citationIDPattern = regexp.MustCompile(`\[([a-zA-Z0-9_.:\-]+)\]`)

// TodayUTC returns YYYY-MM-DD in UTC.
func TodayUTC(now time.Time) string {
	return now.UTC().Format("2006-01-02")
}

// QuestionsRemaining returns how many questions the learner may still ask today.
func QuestionsRemaining(st State, maxPerDay int, now time.Time) int {
	if maxPerDay <= 0 {
		maxPerDay = 20
	}
	today := TodayUTC(now)
	used := 0
	if st.AskedToday != nil && st.AskedToday.Date == today {
		used = st.AskedToday.Count
	}
	left := maxPerDay - used
	if left < 0 {
		return 0
	}
	return left
}

// IncrementAskedToday bumps the daily counter.
func IncrementAskedToday(st *State, now time.Time) {
	today := TodayUTC(now)
	if st.AskedToday == nil || st.AskedToday.Date != today {
		st.AskedToday = &AskedToday{Date: today, Count: 1}
		return
	}
	st.AskedToday.Count++
}

// NewTurnID returns a unique turn id.
func NewTurnID() string {
	return uuid.NewString()
}

// AppendTurns adds user + assistant turns and trims oldest pairs when over maxTurns.
// When trimming, a rolling summary is updated from dropped user texts.
func AppendTurns(st *State, user, assistant Turn, maxTurns int) {
	if maxTurns < 4 {
		maxTurns = 40
	}
	st.Turns = append(st.Turns, user, assistant)
	for len(st.Turns) > maxTurns {
		dropped := st.Turns[0]
		st.Turns = st.Turns[1:]
		if dropped.Role == "user" && strings.TrimSpace(dropped.Text) != "" {
			if st.Summary != "" {
				st.Summary += " "
			}
			st.Summary += summarizeClip(dropped.Text, 120)
			// Keep summary bounded.
			if len(st.Summary) > 2000 {
				st.Summary = st.Summary[len(st.Summary)-2000:]
			}
		}
	}
}

func summarizeClip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// CitationsFromText extracts [id] markers and resolves them against allowed citations.
// Unresolvable citations are dropped (FR-3).
func CitationsFromText(text string, allowed []Citation) ([]Citation, int) {
	if len(allowed) == 0 {
		return nil, 0
	}
	byID := map[string]Citation{}
	for _, c := range allowed {
		byID[c.ID] = c
	}
	seen := map[string]struct{}{}
	var out []Citation
	dropped := 0
	for _, m := range citationIDPattern.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		id := m[1]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if c, ok := byID[id]; ok {
			out = append(out, c)
		} else {
			dropped++
		}
	}
	return out, dropped
}

// MergeCitationLists prefers text-resolved cites; falls back to pack cites limited to those referenced or all pack if none.
func MergeCitationLists(fromText, fromPack []Citation, show bool) []Citation {
	if !show {
		return nil
	}
	if len(fromText) > 0 {
		return fromText
	}
	// Cap pack fallback to avoid dumping every segment.
	const maxFallback = 5
	if len(fromPack) > maxFallback {
		return fromPack[:maxFallback]
	}
	return fromPack
}

// ThemeCluster is an anonymized question theme for instructor insights (FR-11).
type ThemeCluster struct {
	Theme                 string   `json:"theme"`
	Count                 int      `json:"count"`
	RepresentativeExamples []string `json:"representativeExamples"`
}

// ClusterQuestions groups user questions into coarse keyword themes (no learner ids).
func ClusterQuestions(questions []string, maxThemes int) []ThemeCluster {
	if maxThemes <= 0 {
		maxThemes = 8
	}
	type bucket struct {
		key   string
		count int
		ex    []string
	}
	buckets := map[string]*bucket{}
	order := []string{}
	for _, q := range questions {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		key := themeKey(q)
		b, ok := buckets[key]
		if !ok {
			b = &bucket{key: key}
			buckets[key] = b
			order = append(order, key)
		}
		b.count++
		if len(b.ex) < 3 {
			b.ex = append(b.ex, summarizeClip(q, 140))
		}
	}
	// Sort by count desc.
	type pair struct {
		key   string
		count int
	}
	var ranked []pair
	for _, k := range order {
		ranked = append(ranked, pair{key: k, count: buckets[k].count})
	}
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].count > ranked[i].count {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if len(ranked) > maxThemes {
		ranked = ranked[:maxThemes]
	}
	out := make([]ThemeCluster, 0, len(ranked))
	for _, r := range ranked {
		b := buckets[r.key]
		out = append(out, ThemeCluster{
			Theme:                  displayTheme(b.key),
			Count:                  b.count,
			RepresentativeExamples: append([]string{}, b.ex...),
		})
	}
	return out
}

func themeKey(q string) string {
	words := tokenize(strings.ToLower(q))
	stop := map[string]struct{}{
		"a": {}, "an": {}, "the": {}, "is": {}, "are": {}, "was": {}, "were": {},
		"what": {}, "why": {}, "how": {}, "when": {}, "where": {}, "who": {},
		"does": {}, "do": {}, "did": {}, "can": {}, "could": {}, "would": {},
		"i": {}, "me": {}, "my": {}, "we": {}, "you": {}, "your": {},
		"this": {}, "that": {}, "it": {}, "to": {}, "of": {}, "in": {}, "on": {},
		"for": {}, "with": {}, "about": {}, "mean": {}, "means": {}, "here": {},
	}
	var keep []string
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		if _, ok := stop[w]; ok {
			continue
		}
		keep = append(keep, w)
		if len(keep) >= 3 {
			break
		}
	}
	if len(keep) == 0 {
		return "general"
	}
	return strings.Join(keep, " ")
}

func displayTheme(key string) string {
	if key == "general" {
		return "General questions"
	}
	parts := strings.Split(key, " ")
	for i, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}

func tokenize(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// CollectUserQuestions extracts user turn texts from state JSON blobs.
func CollectUserQuestions(states []State) []string {
	var out []string
	for _, st := range states {
		for _, t := range st.Turns {
			if t.Role == "user" && strings.TrimSpace(t.Text) != "" {
				out = append(out, t.Text)
			}
		}
	}
	return out
}
