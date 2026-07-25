package adaptivecontent

import (
	"regexp"
	"strings"
	"unicode"
)

// DefaultMinFidelity is the unit default when min_fidelity is unset.
const DefaultMinFidelity = 0.85

// FidelityResult is the outcome of hard checks + composite score.
type FidelityResult struct {
	// Score is in [0,1]. Hard-check failure forces 0.
	Score float64
	// HardPass is false when any required key term, math, or code block is missing.
	HardPass bool
	// Flags are machine-readable reasons (e.g. missing_key_term:Photosynthesis).
	Flags []string
	// UnsupportedClaims from the LLM-judge (informational; already reflected in JudgeScore).
	UnsupportedClaims []string
	// JudgeScore is the LLM-judge support score in [0,1], or -1 if judge was skipped.
	JudgeScore float64
}

// ExtractFencedCodeBlocks returns the interior of ``` ... ``` fences (language tag stripped).
func ExtractFencedCodeBlocks(md string) []string {
	re := regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")
	matches := re.FindAllStringSubmatch(md, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		body := strings.TrimSpace(m[1])
		if body != "" {
			out = append(out, body)
		}
	}
	return out
}

// ExtractLatexBlocks returns $...$, $$...$$, and \(...\) / \[...\] spans (normalized).
func ExtractLatexBlocks(md string) []string {
	var out []string
	// Display $$...$$
	reDD := regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)
	for _, m := range reDD.FindAllStringSubmatch(md, -1) {
		if len(m) >= 2 {
			t := strings.TrimSpace(m[1])
			if t != "" {
				out = append(out, normalizeLatex(t))
			}
		}
	}
	// Inline $...$ (avoid matching $$ already consumed by not matching empty)
	reD := regexp.MustCompile(`\$([^$\n]+)\$`)
	for _, m := range reD.FindAllStringSubmatch(md, -1) {
		if len(m) >= 2 {
			t := strings.TrimSpace(m[1])
			if t != "" {
				out = append(out, normalizeLatex(t))
			}
		}
	}
	// \( ... \) and \[ ... \]
	reParen := regexp.MustCompile(`\\\((.+?)\\\)`)
	for _, m := range reParen.FindAllStringSubmatch(md, -1) {
		if len(m) >= 2 {
			t := strings.TrimSpace(m[1])
			if t != "" {
				out = append(out, normalizeLatex(t))
			}
		}
	}
	reBracket := regexp.MustCompile(`(?s)\\\[(.+?)\\\]`)
	for _, m := range reBracket.FindAllStringSubmatch(md, -1) {
		if len(m) >= 2 {
			t := strings.TrimSpace(m[1])
			if t != "" {
				out = append(out, normalizeLatex(t))
			}
		}
	}
	return out
}

func normalizeLatex(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// CheckKeyTerms verifies must-appear terms are present case-insensitively as substrings.
func CheckKeyTerms(variantMarkdown string, terms []string) (pass bool, missing []string) {
	hay := strings.ToLower(variantMarkdown)
	pass = true
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !strings.Contains(hay, strings.ToLower(t)) {
			pass = false
			missing = append(missing, t)
		}
	}
	return pass, missing
}

// CheckCodeBlocksSurvive requires every base fenced code body to appear byte-verifiably in the variant
// (after trimming surrounding whitespace on each block body).
func CheckCodeBlocksSurvive(base, variant string) (pass bool, missing []string) {
	baseBlocks := ExtractFencedCodeBlocks(base)
	if len(baseBlocks) == 0 {
		return true, nil
	}
	// Allow either still-fenced or raw presence of the body.
	pass = true
	for _, b := range baseBlocks {
		if !strings.Contains(variant, b) {
			pass = false
			// Truncate for flag readability.
			snip := b
			if len(snip) > 48 {
				snip = snip[:48] + "…"
			}
			missing = append(missing, snip)
		}
	}
	return pass, missing
}

// CheckLatexSurvives requires every base LaTeX body (normalized whitespace) to appear in the variant.
func CheckLatexSurvives(base, variant string) (pass bool, missing []string) {
	baseLatex := ExtractLatexBlocks(base)
	if len(baseLatex) == 0 {
		return true, nil
	}
	variantNorm := normalizeLatex(variant)
	// Also check raw variant for exact latex spans.
	pass = true
	for _, l := range baseLatex {
		if strings.Contains(variant, l) || strings.Contains(variantNorm, l) {
			continue
		}
		// Try with surrounding $ restored for loose match of original formula text.
		pass = false
		snip := l
		if len(snip) > 48 {
			snip = snip[:48] + "…"
		}
		missing = append(missing, snip)
	}
	return pass, missing
}

// RunHardFidelityChecks runs key-term / math / code presence checks.
func RunHardFidelityChecks(baseMarkdown, variantMarkdown string, keyTerms []string) FidelityResult {
	res := FidelityResult{HardPass: true, JudgeScore: -1, Score: 1}
	var flags []string

	if ok, missing := CheckKeyTerms(variantMarkdown, keyTerms); !ok {
		res.HardPass = false
		for _, m := range missing {
			flags = append(flags, "missing_key_term:"+m)
		}
	}
	if ok, missing := CheckCodeBlocksSurvive(baseMarkdown, variantMarkdown); !ok {
		res.HardPass = false
		for _, m := range missing {
			flags = append(flags, "missing_code_block:"+m)
		}
	}
	if ok, missing := CheckLatexSurvives(baseMarkdown, variantMarkdown); !ok {
		res.HardPass = false
		for _, m := range missing {
			flags = append(flags, "missing_latex:"+m)
		}
	}
	res.Flags = flags
	if !res.HardPass {
		res.Score = 0
	}
	return res
}

// CompositeFidelityScore combines hard checks with an optional LLM-judge score.
// composite = 0 when hard checks fail; else min(1, judge) when judge >= 0; else 1 for hard-only.
func CompositeFidelityScore(hard FidelityResult, judgeScore float64) FidelityResult {
	out := hard
	out.JudgeScore = judgeScore
	if !hard.HardPass {
		out.Score = 0
		return out
	}
	if judgeScore < 0 {
		out.Score = 1
		return out
	}
	if judgeScore > 1 {
		judgeScore = 1
	}
	if judgeScore < 0 {
		judgeScore = 0
	}
	out.Score = judgeScore
	if judgeScore < DefaultMinFidelity {
		out.Flags = append(out.Flags, "low_judge_score")
	}
	return out
}

// LexicalClaimOverlap is a cheap embedding substitute: fraction of base content "claims"
// (sentence-like chunks) that appear (token-overlap ≥ 0.5) in the variant.
// Returns score in [0,1]. Used when the LLM judge is unavailable.
func LexicalClaimOverlap(base, variant string) float64 {
	baseClaims := splitClaims(base)
	if len(baseClaims) == 0 {
		return 1
	}
	varHits := 0
	for _, c := range baseClaims {
		if claimSupported(c, variant) {
			varHits++
		}
	}
	return float64(varHits) / float64(len(baseClaims))
}

func splitClaims(md string) []string {
	// Strip code fences so code is not treated as prose claims.
	stripped := regexp.MustCompile("(?s)```.*?```").ReplaceAllString(md, " ")
	// Split on sentence terminators and newlines.
	parts := regexp.MustCompile(`[.!?\n]+`).Split(stripped, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Drop headings markers and very short fragments.
		p = strings.TrimLeft(p, "#*- ")
		if runeCount(p) < 24 {
			continue
		}
		out = append(out, p)
	}
	return out
}

func claimSupported(claim, variant string) bool {
	claimTokens := tokenize(claim)
	if len(claimTokens) == 0 {
		return true
	}
	variantSet := make(map[string]struct{})
	for _, t := range tokenize(variant) {
		variantSet[t] = struct{}{}
	}
	hits := 0
	for _, t := range claimTokens {
		if _, ok := variantSet[t]; ok {
			hits++
		}
	}
	return float64(hits)/float64(len(claimTokens)) >= 0.5
}

func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 3 {
			continue
		}
		// Drop ultra-common stop words.
		switch f {
		case "the", "and", "for", "that", "with", "this", "from", "are", "was", "were", "have", "has":
			continue
		}
		out = append(out, f)
	}
	return out
}

func runeCount(s string) int {
	return len([]rune(s))
}

// LintA11y runs lightweight WCAG-oriented structure checks on generated markdown.
// Returns flag codes (empty = clean). AC.8 owns enforcement; AC.3 emits the signal.
func LintA11y(sectionsMarkdown string) []string {
	var flags []string
	// Images without alt text: ![ ]( or ![](
	if regexp.MustCompile(`!\[\s*\]\([^)]+\)`).MatchString(sectionsMarkdown) {
		flags = append(flags, "image_missing_alt")
	}
	// Heading jump detection on leading # lines inside combined markdown.
	// (Our sections use separate headings; check embedded ## jumps only.)
	lines := strings.Split(sectionsMarkdown, "\n")
	prevLevel := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		level := 0
		for _, r := range trimmed {
			if r == '#' {
				level++
			} else {
				break
			}
		}
		if level == 0 || level > 6 {
			continue
		}
		if prevLevel > 0 && level > prevLevel+1 {
			flags = append(flags, "heading_level_skip")
			break
		}
		prevLevel = level
	}
	return flags
}

// SanitizeVariantMarkdown strips script/HTML injection vectors while keeping markdown.
func SanitizeVariantMarkdown(md string) string {
	// Remove script/style tags and their contents.
	reScript := regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	reStyle := regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	out := reScript.ReplaceAllString(md, "")
	out = reStyle.ReplaceAllString(out, "")
	// Strip remaining HTML tags (block editor is markdown-first).
	reTag := regexp.MustCompile(`(?i)</?[a-z][^>]*>`)
	out = reTag.ReplaceAllString(out, "")
	// Neutralize javascript: URLs.
	reJS := regexp.MustCompile(`(?i)javascript\s*:`)
	out = reJS.ReplaceAllString(out, "blocked:")
	return out
}

// SafetyScan flags prohibited / injection patterns. Returns flags (empty = pass).
func SafetyScan(md string) []string {
	var flags []string
	lower := strings.ToLower(md)
	if strings.Contains(lower, "<script") {
		flags = append(flags, "script_tag")
	}
	if strings.Contains(lower, "javascript:") {
		flags = append(flags, "javascript_url")
	}
	if strings.Contains(lower, "onerror=") || strings.Contains(lower, "onload=") {
		flags = append(flags, "inline_event_handler")
	}
	// Extremely crude age-inappropriate token list (expand in AC.8).
	prohibited := []string{"how to make a bomb", "child pornography", "sexually explicit"}
	for _, p := range prohibited {
		if strings.Contains(lower, p) {
			flags = append(flags, "prohibited_content")
			break
		}
	}
	return flags
}
