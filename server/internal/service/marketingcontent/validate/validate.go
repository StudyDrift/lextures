// Package validate enforces the answer-first marketing-content contract.
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lextures/lextures/server/internal/service/marketingcontent/render"
)

type Metadata struct {
	Title, Description, Updated, Author, Cluster, PrimaryQuestion, Locale string
	Keywords                                                              []string
}
type Input struct {
	Kind, BodyMD, Locale string
	Metadata             Metadata
	KnownPaths           map[string]struct{}
}
type Finding struct {
	Rule, Severity, Message string
	Line, Column            int
	Path                    string `json:"path,omitempty"`
}
type Report struct {
	Score          float64            `json:"score"`
	Findings       []Finding          `json:"findings"`
	Stats          render.StatsResult `json:"stats"`
	ValidatorError bool               `json:"validatorError,omitempty"`
	// HTML / PlainText are populated by Service.Lint for editor preview and search extraction (MC.4).
	HTML      string `json:"html,omitempty"`
	PlainText string `json:"plainText,omitempty"`
}
type rule struct {
	id    string
	check func(Input, *analysis) []Finding
}
type analysis struct {
	lines                  []string
	answer, takeaways, faq block
	headings               []heading
	citations              int
	internalLinks          int
	meanPassage            float64
	structured             bool
}
type block struct {
	content string
	line    int
}
type heading struct {
	text string
	line int
}

var allowedDirectives = map[string]bool{"key-takeaways": true, "answer": true, "definition": true, "comparison-table": true, "steps": true, "faq": true, "callout": true, "stat": true, "sources": true}
var directiveStart = regexp.MustCompile(`^:::[ \t]*([\w-]+)(?:[ \t].*)?$`)
var markdownLink = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)\s]+)`)
var rawHTML = regexp.MustCompile(`(?i)</?[a-z][^>]*>|<!--`)
var scriptURL = regexp.MustCompile(`(?i)(?:javascript|data):`)
var numericClaim = regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?%|\b\d{2,}\s+(?:teachers|students|schools|learners|percent)\b`)

func Article(in Input) (out Report) {
	out.Findings = []Finding{}
	defer func() {
		if recover() != nil {
			out = Report{Score: 0, Findings: []Finding{{Rule: "validator_error", Severity: "warn", Message: "Validation could not be completed."}}, Stats: render.Stats(in.BodyMD), ValidatorError: true}
		}
	}()
	if in.Locale == "" {
		in.Locale = in.Metadata.Locale
	}
	a := inspect(in.BodyMD)
	out.Stats = render.Stats(in.BodyMD)
	out.Score = score(in, a)
	for _, r := range rules {
		out.Findings = append(out.Findings, r.check(in, a)...)
	}
	sev := "warn"
	if out.Score < 6 {
		sev = "error"
	}
	if out.Score < 8 {
		out.Findings = append(out.Findings, Finding{Rule: "extractability.score", Severity: sev, Message: fmt.Sprintf("Extractability score %.1f is below 8.0.", out.Score), Line: 1, Column: 1})
	}
	return
}

var rules = []rule{
	{"fm.required", checkMetadata}, {"safety", checkSafety}, {"directive", checkDirectives}, {"structure", checkStructure}, {"passage", checkPassages}, {"citation", checkCitations}, {"link", checkLinks}, {"a11y", checkImages},
}

func finding(rule, severity, message string, line, column int) Finding {
	return Finding{Rule: rule, Severity: severity, Message: message, Line: line, Column: column}
}
func checkMetadata(in Input, _ *analysis) []Finding {
	fields := []struct{ n, v string }{{"title", in.Metadata.Title}, {"description", in.Metadata.Description}, {"updated", in.Metadata.Updated}, {"author", in.Metadata.Author}, {"cluster", in.Metadata.Cluster}, {"primaryQuestion", in.Metadata.PrimaryQuestion}}
	out := []Finding{}
	for _, f := range fields {
		if strings.TrimSpace(f.v) == "" {
			x := finding("fm."+f.n, "error", "Required metadata field is missing.", 0, 0)
			x.Path = f.n
			out = append(out, x)
		}
	}
	if len(in.Metadata.Keywords) == 0 {
		x := finding("fm.keywords", "error", "At least one keyword is required.", 0, 0)
		x.Path = "keywords"
		out = append(out, x)
	}
	return out
}
func checkSafety(in Input, _ *analysis) []Finding {
	out := []Finding{}
	for i, line := range strings.Split(in.BodyMD, "\n") {
		if loc := rawHTML.FindStringIndex(line); loc != nil {
			out = append(out, finding("safety.raw-html", "error", "Raw HTML is not allowed.", i+1, loc[0]+1))
		}
		if loc := scriptURL.FindStringIndex(line); loc != nil {
			out = append(out, finding("safety.script-url", "error", "Executable or data URLs are not allowed.", i+1, loc[0]+1))
		}
	}
	return out
}
func checkDirectives(in Input, _ *analysis) []Finding {
	out := []Finding{}
	open := ""
	lineNo := 0
	for i, line := range strings.Split(in.BodyMD, "\n") {
		if strings.TrimSpace(line) == ":::" {
			open = ""
			continue
		}
		m := directiveStart.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if open != "" {
			out = append(out, finding("directive.malformed", "error", "Directive is nested or missing a closing marker.", i+1, 1))
		}
		open = m[1]
		lineNo = i + 1
		if !allowedDirectives[strings.ToLower(open)] {
			out = append(out, finding("directive.unknown", "error", fmt.Sprintf("Directive %q is not supported.", open), i+1, 1))
		}
	}
	if open != "" {
		out = append(out, finding("directive.malformed", "error", "Directive is missing a closing marker.", lineNo, 1))
	}
	return out
}
func checkStructure(in Input, a *analysis) []Finding {
	out := []Finding{}
	n := bulletCount(a.takeaways.content)
	if n < 3 || n > 5 {
		out = append(out, finding("struct.key-takeaways", "error", fmt.Sprintf("Key takeaways must contain 3–5 bullets; found %d.", n), max(1, a.takeaways.line), 1))
	}
	m := metricsFor(in.Locale)
	aw := textLen(a.answer.content, m.chars)
	if aw == 0 {
		out = append(out, finding("struct.answer-block", "error", "A direct answer block is required.", 1, 1))
	} else if aw < m.answerMin || aw > m.answerMax {
		out = append(out, finding("passage.length", "warn", fmt.Sprintf("Direct answer is %d %s; target is %d–%d.", aw, m.unit, m.answerMin, m.answerMax), a.answer.line, 1))
	}
	fq := len(regexp.MustCompile(`(?m)^###\s+.+\?\s*$`).FindAllString(a.faq.content, -1))
	if fq < 3 || fq > 6 {
		out = append(out, finding("struct.faq-count", "warn", fmt.Sprintf("FAQ must contain 3–6 questions; found %d.", fq), max(1, a.faq.line), 1))
	}
	q := 0
	for _, h := range a.headings {
		if questionHeading(h.text) {
			q++
		}
	}
	if len(a.headings) > 0 && float64(q)/float64(len(a.headings)) < .6 {
		out = append(out, finding("struct.heading-questions", "warn", "At least 60% of level-two headings should be questions.", a.headings[0].line, 1))
	}
	return out
}
func checkPassages(in Input, a *analysis) []Finding {
	m := metricsFor(in.Locale)
	if m.chars {
		return nil
	}
	if a.meanPassage > 0 && (a.meanPassage < float64(m.passageMin) || a.meanPassage > float64(m.passageMax)) {
		return []Finding{finding("passage.self-contained", "warn", fmt.Sprintf("Mean passage length is %.0f %s; target is %d–%d.", a.meanPassage, m.unit, m.passageMin, m.passageMax), 1, 1)}
	}
	return nil
}
func checkCitations(in Input, _ *analysis) []Finding {
	out := []Finding{}
	lines := strings.Split(in.BodyMD, "\n")
	defs := map[string]bool{}
	defRE := regexp.MustCompile(`^\[\^(\d+)\]:\s+https?://`)
	for _, line := range lines {
		if m := defRE.FindStringSubmatch(line); m != nil {
			defs[m[1]] = true
		}
	}
	refRE := regexp.MustCompile(`\[\^(\d+)\]`)
	for i, line := range lines {
		if numericClaim.MatchString(line) && !refRE.MatchString(line) {
			out = append(out, finding("cite.numeric-claim", "warn", "Numeric claim needs an inline citation.", i+1, 1))
		}
		for _, m := range refRE.FindAllStringSubmatch(line, -1) {
			if !strings.HasPrefix(line, "[^"+m[1]+"]:") && !defs[m[1]] {
				out = append(out, finding("cite.source-resolvable", "warn", "Citation has no resolvable source definition.", i+1, 1))
			}
		}
	}
	return out
}
func checkLinks(in Input, _ *analysis) []Finding {
	out := []Finding{}
	for i, line := range strings.Split(in.BodyMD, "\n") {
		for _, m := range markdownLink.FindAllStringSubmatchIndex(line, -1) {
			anchor := line[m[2]:m[3]]
			target := line[m[4]:m[5]]
			if strings.HasPrefix(target, "/") {
				path := strings.SplitN(strings.SplitN(target, "#", 2)[0], "?", 2)[0]
				if _, ok := in.KnownPaths[path]; !ok {
					out = append(out, finding("link.internal-resolves", "error", fmt.Sprintf("Internal link %q does not resolve.", path), i+1, m[4]+1))
				}
				if regexp.MustCompile(`(?i)^(here|read more|click here)$`).MatchString(strings.TrimSpace(anchor)) {
					out = append(out, finding("link.descriptive-anchor", "warn", "Use a descriptive link label.", i+1, m[2]+1))
				}
			}
		}
	}
	return out
}
func checkImages(in Input, _ *analysis) []Finding {
	out := []Finding{}
	re := regexp.MustCompile(`!\[([^\]]*)\]\(`)
	for i, line := range strings.Split(in.BodyMD, "\n") {
		for _, m := range re.FindAllStringSubmatchIndex(line, -1) {
			if strings.TrimSpace(line[m[2]:m[3]]) == "" {
				out = append(out, finding("a11y.image-alt", "error", "Images require alternative text.", i+1, m[0]+1))
			}
		}
	}
	return out
}

func inspect(body string) *analysis {
	a := &analysis{lines: strings.Split(body, "\n")}
	open := ""
	start := 0
	var content []string
	for i, line := range a.lines {
		if strings.TrimSpace(line) == ":::" && open != "" {
			b := block{content: strings.Join(content, "\n"), line: start}
			switch open {
			case "answer":
				a.answer = b
			case "key-takeaways":
				a.takeaways = b
			case "faq":
				a.faq = b
			}
			open = ""
			content = nil
			continue
		}
		if open != "" {
			content = append(content, line)
			continue
		}
		if m := directiveStart.FindStringSubmatch(line); m != nil && allowedDirectives[m[1]] {
			open = m[1]
			start = i + 1
			continue
		}
		if m := regexp.MustCompile(`^##\s+(.+)$`).FindStringSubmatch(line); m != nil {
			a.headings = append(a.headings, heading{text: m[1], line: i + 1})
		}
	}
	a.citations = len(regexp.MustCompile(`(?m)^\[\^\d+\]:\s+https?://`).FindAllString(body, -1))
	a.internalLinks = len(regexp.MustCompile(`\[[^\]]+\]\(/`).FindAllString(body, -1))
	a.structured = regexp.MustCompile(`(?m)^(?:[-*]|\d+\.)\s+`).MatchString(body) || strings.Contains(body, ":::steps") || strings.Contains(body, ":::comparison-table")
	s := render.Stats(body)
	if len(s.PassageLengths) > 0 {
		for _, n := range s.PassageLengths {
			a.meanPassage += float64(n)
		}
		a.meanPassage /= float64(len(s.PassageLengths))
	}
	return a
}
func score(in Input, a *analysis) float64 {
	s := 0.0
	if n := bulletCount(a.takeaways.content); n >= 3 && n <= 5 {
		s += 1
	}
	if n := textLen(a.answer.content, metricsFor(in.Locale).chars); n >= metricsFor(in.Locale).answerMin && n <= metricsFor(in.Locale).answerMax {
		s += 1.5
	}
	q := 0
	for _, h := range a.headings {
		if questionHeading(h.text) {
			q++
		}
	}
	if len(a.headings) > 0 && float64(q)/float64(len(a.headings)) >= .6 {
		s += 1.5
	}
	if m := metricsFor(in.Locale); !m.chars && a.meanPassage >= float64(m.passageMin) && a.meanPassage <= float64(m.passageMax) {
		s += 1.5
	}
	if a.citations >= max(1, (render.Stats(in.BodyMD).WordCount+399)/400) {
		s += 2
	}
	if a.structured {
		s += 1
	}
	if n := len(regexp.MustCompile(`(?m)^###\s+.+\?\s*$`).FindAllString(a.faq.content, -1)); n >= 3 && n <= 6 {
		s += 1
	}
	if a.internalLinks >= 3 {
		s += .5
	}
	return s
}
func wordCount(s string) int {
	return len(strings.Fields(regexp.MustCompile(`[#>*_`+"`"+`\[\](){}|:-]`).ReplaceAllString(s, " ")))
}

type localeMetrics struct {
	answerMin, answerMax, passageMin, passageMax int
	chars                                        bool
	unit                                         string
}

func metricsFor(locale string) localeMetrics {
	lang := strings.ToLower(strings.Split(locale, "-")[0])
	switch lang {
	case "zh", "ja", "ko":
		return localeMetrics{80, 160, 240, 400, true, "characters"}
	default:
		return localeMetrics{40, 60, 120, 180, false, "words"}
	}
}

func textLen(s string, chars bool) int {
	if !chars {
		return wordCount(s)
	}
	stripped := regexp.MustCompile(`[#>*_` + "`" + `\[\](){}|:\-\s]`).ReplaceAllString(s, "")
	return len([]rune(stripped))
}
func bulletCount(s string) int {
	return len(regexp.MustCompile(`(?m)^\s*[-*]\s+`).FindAllString(s, -1))
}
func questionHeading(s string) bool {
	return strings.HasSuffix(strings.TrimSpace(s), "?") || regexp.MustCompile(`(?i)^(how|what|when|where|why|who|which|can|should|does|do|is|are)\b`).MatchString(s)
}
