package coursechecklist

import (
	"embed"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/l10n"
)

// MaxSyllabusScanBytes caps syllabus text scanned by keyword heuristics (NFR).
const MaxSyllabusScanBytes = 512 * 1024

//go:embed lexicons/*.json
var lexiconFS embed.FS

type latePolicyLexicon struct {
	Present []string `json:"present"`
	NoLate  []string `json:"noLate"`
}

type lexiconFile struct {
	Locale                string              `json:"locale"`
	StartHereTitles       []string            `json:"startHereTitles"`
	InstructorIntroTitles []string            `json:"instructorIntroTitles"`
	LearnerIntroTitles    []string            `json:"learnerIntroTitles"`
	Contact               []string            `json:"contact"`
	ResponseTime          []string            `json:"responseTime"`
	Participation         []string            `json:"participation"`
	Netiquette            []string            `json:"netiquette"`
	TechRequirements      []string            `json:"techRequirements"`
	SupportLinkHints      []string            `json:"supportLinkHints"`
	GradingPolicy         []string            `json:"gradingPolicy"`
	LatePolicy            latePolicyLexicon   `json:"latePolicy"`
	AcademicIntegrity     []string            `json:"academicIntegrity"`
	Accessibility         []string            `json:"accessibility"`
	PlaceholderTitles     []string            `json:"placeholderTitles"`
	ModuleOverviewTitles  []string            `json:"moduleOverviewTitles"`
	BloomVerbs            map[string][]string `json:"bloomVerbs"`
	NonMeasurableOpeners  []string            `json:"nonMeasurableOpeners"`
	SuggestedReplacements map[string]string   `json:"suggestedReplacements"`
	VagueLinkText         []string            `json:"vagueLinkText"`
	AuthenticActivity     []string            `json:"authenticActivity"`
	ExpressionChoice      []string            `json:"expressionChoice"`
}

// Lexicon is a compiled, locale-specific keyword set for text heuristics (FR-35).
type Lexicon struct {
	Locale                string
	StartHereTitles       *keywordMatcher
	InstructorIntroTitles *keywordMatcher
	LearnerIntroTitles    *keywordMatcher
	Contact               *keywordMatcher
	ResponseTime          *keywordMatcher
	Participation         *keywordMatcher
	Netiquette            *keywordMatcher
	TechRequirements      *keywordMatcher
	SupportLinkHints      *keywordMatcher
	GradingPolicy         *keywordMatcher
	LatePolicyPresent     *keywordMatcher
	LatePolicyNoLate      *keywordMatcher
	AcademicIntegrity     *keywordMatcher
	Accessibility         *keywordMatcher
	PlaceholderTitles     []string
	ModuleOverviewTitles  *keywordMatcher
	BloomVerbs            map[string][]string
	NonMeasurableOpeners  []string
	SuggestedReplacements map[string]string
	HasBloom              bool
	VagueLinkText         *keywordMatcher
	AuthenticActivity     *keywordMatcher
	ExpressionChoice      *keywordMatcher
}

type keywordMatcher struct {
	patterns []*regexp.Regexp
}

func newKeywordMatcher(patterns []string) *keywordMatcher {
	m := &keywordMatcher{patterns: make([]*regexp.Regexp, 0, len(patterns))}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Treat plain phrases as case-insensitive literal substrings; patterns that
		// already contain regex metacharacters are compiled as-is (case-insensitive).
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			re = regexp.MustCompile("(?i)" + regexp.QuoteMeta(p))
		}
		m.patterns = append(m.patterns, re)
	}
	return m
}

// Match reports whether any pattern matches text.
func (m *keywordMatcher) Match(text string) bool {
	if m == nil {
		return false
	}
	for _, re := range m.patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// FindAll returns distinct matched substrings (capped).
func (m *keywordMatcher) FindAll(text string, limit int) []string {
	if m == nil || limit <= 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, re := range m.patterns {
		loc := re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		s := text[loc[0]:loc[1]]
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
		if len(out) >= limit {
			break
		}
	}
	return out
}

var (
	lexiconOnce sync.Once
	lexicons    map[string]*Lexicon
	lexiconErr  error
)

func loadLexicons() {
	lexiconOnce.Do(func() {
		lexicons = make(map[string]*Lexicon)
		entries, err := lexiconFS.ReadDir("lexicons")
		if err != nil {
			lexiconErr = err
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			raw, err := lexiconFS.ReadFile("lexicons/" + e.Name())
			if err != nil {
				lexiconErr = err
				return
			}
			var f lexiconFile
			if err := json.Unmarshal(raw, &f); err != nil {
				lexiconErr = err
				return
			}
			locale := strings.ToLower(strings.TrimSpace(f.Locale))
			if locale == "" {
				locale = strings.TrimSuffix(e.Name(), ".json")
			}
			hasBloom := len(f.BloomVerbs) > 0 && len(f.NonMeasurableOpeners) > 0
			lexicons[locale] = &Lexicon{
				Locale:                locale,
				StartHereTitles:       newKeywordMatcher(f.StartHereTitles),
				InstructorIntroTitles: newKeywordMatcher(f.InstructorIntroTitles),
				LearnerIntroTitles:    newKeywordMatcher(f.LearnerIntroTitles),
				Contact:               newKeywordMatcher(f.Contact),
				ResponseTime:          newKeywordMatcher(f.ResponseTime),
				Participation:         newKeywordMatcher(f.Participation),
				Netiquette:            newKeywordMatcher(f.Netiquette),
				TechRequirements:      newKeywordMatcher(f.TechRequirements),
				SupportLinkHints:      newKeywordMatcher(f.SupportLinkHints),
				GradingPolicy:         newKeywordMatcher(f.GradingPolicy),
				LatePolicyPresent:     newKeywordMatcher(f.LatePolicy.Present),
				LatePolicyNoLate:      newKeywordMatcher(f.LatePolicy.NoLate),
				AcademicIntegrity:     newKeywordMatcher(f.AcademicIntegrity),
				Accessibility:         newKeywordMatcher(f.Accessibility),
				PlaceholderTitles:     normalizePhraseList(f.PlaceholderTitles),
				ModuleOverviewTitles:  newKeywordMatcher(f.ModuleOverviewTitles),
				BloomVerbs:            f.BloomVerbs,
				NonMeasurableOpeners:  normalizePhraseList(f.NonMeasurableOpeners),
				SuggestedReplacements: lowerKeyMap(f.SuggestedReplacements),
				HasBloom:              hasBloom,
				VagueLinkText:         newKeywordMatcher(f.VagueLinkText),
				AuthenticActivity:     newKeywordMatcher(f.AuthenticActivity),
				ExpressionChoice:      newKeywordMatcher(f.ExpressionChoice),
			}
		}
		if _, ok := lexicons["en"]; !ok {
			lexiconErr = errMissingEnglishLexicon
		}
	})
}

var errMissingEnglishLexicon = errString("coursechecklist: english lexicon missing")

type errString string

func (e errString) Error() string { return string(e) }

func normalizePhraseList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func lowerKeyMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	return out
}

// LexiconFor returns the lexicon for locale, falling back to English (FR-35).
func LexiconFor(locale string) *Lexicon {
	loadLexicons()
	_ = lexiconErr // loadLexicons records parse failures; English fallback still applies.
	primary := locale
	if norm, err := l10n.NormalizeLocale(locale); err == nil && norm != "" {
		primary = norm
	}
	primary = strings.ToLower(strings.TrimSpace(primary))
	if i := strings.IndexByte(primary, '-'); i > 0 {
		if lx, ok := lexicons[primary]; ok {
			return lx
		}
		primary = primary[:i]
	}
	if lx, ok := lexicons[primary]; ok {
		return lx
	}
	return lexicons["en"]
}

// BloomLexiconFor returns a lexicon with Bloom verbs for locale, or nil when unsupported.
// Unlike LexiconFor, this does NOT fall back to English (CC.4 FR-14).
func BloomLexiconFor(locale string) *Lexicon {
	loadLexicons()
	primary := locale
	if norm, err := l10n.NormalizeLocale(locale); err == nil && norm != "" {
		primary = norm
	}
	primary = strings.ToLower(strings.TrimSpace(primary))
	candidates := []string{primary}
	if i := strings.IndexByte(primary, '-'); i > 0 {
		candidates = append(candidates, primary[:i])
	}
	for _, c := range candidates {
		if lx, ok := lexicons[c]; ok && lx != nil && lx.HasBloom {
			return lx
		}
	}
	return nil
}

// PlaceholderLexiconFor returns placeholder titles for locale, falling back to English (FR-3).
func PlaceholderLexiconFor(locale string) []string {
	lx := LexiconFor(locale)
	if lx != nil && len(lx.PlaceholderTitles) > 0 {
		return lx.PlaceholderTitles
	}
	en := LexiconFor("en")
	if en == nil {
		return nil
	}
	return en.PlaceholderTitles
}

// SyllabusPlainText concatenates syllabus section titles + markdown, capped at MaxSyllabusScanBytes.
// truncated is true when the combined text exceeded the cap.
func SyllabusPlainText(snap CourseSnapshot) (text string, truncated bool) {
	var b strings.Builder
	for _, s := range snap.SyllabusSections {
		if s.Title != "" {
			b.WriteString(s.Title)
			b.WriteByte('\n')
		}
		if s.Markdown != "" {
			b.WriteString(s.Markdown)
			b.WriteByte('\n')
		}
		if b.Len() >= MaxSyllabusScanBytes {
			truncated = true
			break
		}
	}
	out := b.String()
	if len(out) > MaxSyllabusScanBytes {
		// Truncate on a rune boundary.
		out = out[:MaxSyllabusScanBytes]
		for len(out) > 0 && !utf8.ValidString(out) {
			out = out[:len(out)-1]
		}
		truncated = true
	}
	if snap.SyllabusCheckedTruncated {
		truncated = true
	}
	return out, truncated
}

// TitleMatches reports whether title matches any of the matcher phrases.
func TitleMatches(m *keywordMatcher, title string) bool {
	return m != nil && m.Match(strings.TrimSpace(title))
}

// isStructurePlaceholderTitle reports whole-title or title-prefix placeholder matches (FR-3).
func isStructurePlaceholderTitle(title string, placeholders []string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	if t == "" {
		return true
	}
	for _, p := range placeholders {
		if t == p || strings.HasPrefix(t, p+" ") || strings.HasPrefix(t, p+":") || strings.HasPrefix(t, p+"-") {
			return true
		}
	}
	return false
}

// measurableOutcomeVerb reports whether title starts with a Bloom verb, or flags a non-measurable opener.
// suggested is a replacement verb when flagged.
func measurableOutcomeVerb(title string, lx *Lexicon) (ok bool, flagged bool, suggested string) {
	if lx == nil || !lx.HasBloom {
		return false, false, ""
	}
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return false, true, "identify"
	}
	lower := strings.ToLower(trimmed)
	for _, opener := range lx.NonMeasurableOpeners {
		if lower == opener || strings.HasPrefix(lower, opener+" ") || strings.HasPrefix(lower, opener+":") {
			sug := lx.SuggestedReplacements[opener]
			if sug == "" {
				sug = "explain"
			}
			return false, true, sug
		}
	}
	first := firstWord(lower)
	for _, verbs := range lx.BloomVerbs {
		for _, v := range verbs {
			if first == strings.ToLower(v) {
				return true, false, ""
			}
		}
	}
	// Not a known Bloom opener and not in the non-measurable list — treat as OK (avoid false todos).
	return true, false, ""
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if r == ' ' || r == '\t' || r == ':' || r == '-' || r == '—' {
			return s[:i]
		}
	}
	return s
}
