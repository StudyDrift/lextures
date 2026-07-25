package adaptivecontent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/contentpagegeneration"
)

// Prompt keys and versions.
const (
	// PromptKey is the settings.system_prompts key for adaptive content generation.
	PromptKey = "adaptive_content_variant"
	// PromptVersionCurrent is stamped on each variant when using the seeded prompt.
	PromptVersionCurrent = "v1"
	// EventVariantGenerated is written to adaptive_content_events after generation.
	EventVariantGenerated = "variant_generated"
	// EventVariantRejected is written when fidelity/safety gates reject a variant.
	EventVariantRejected = "variant_rejected"
)

// Sentinel errors for typed fallback.
var (
	// ErrGatewayDenied means aigateway blocked the call (opt-out, COPPA, tenant, etc.).
	ErrGatewayDenied = errors.New("adaptive content: AI gateway denied")
	// ErrBudgetExhausted is reserved for AC.4 budget enforcement.
	ErrBudgetExhausted = errors.New("adaptive content: monthly token budget exhausted")
	// ErrGenerationFailed covers model/parse failures that should fall back to base.
	ErrGenerationFailed = errors.New("adaptive content: generation failed")
	// ErrRejectedFidelity means the variant failed the fidelity gate (still may be stored).
	ErrRejectedFidelity = errors.New("adaptive content: fidelity gate rejected")
	// ErrRejectedSafety means the safety check failed.
	ErrRejectedSafety = errors.New("adaptive content: safety gate rejected")
)

// DefaultSystemPrompt matches settings.system_prompts key adaptive_content_variant when the row is missing.
const DefaultSystemPrompt = `You adapt an existing LMS content page for one learner. You are given the AUTHORITATIVE base content, an emphasis mode, the learner's concept gaps and misconceptions (by name), a target cognitive level, and allowed adaptation axes.

Rewrite or re-emphasize the base to fit the learner. Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object: {"sections":[{"heading":"...","markdown":"..."}]}.

Absolute rules:
- Preserve every fact, definition, standard reference, formula, code block, and listed key term from the base exactly.
- Introduce no claim, statistic, or example not entailed by the base content.
- Never remove required terminology.
- heading: short section title without markdown # prefixes; use "" for a lead-in block with no heading.
- markdown: body content in Markdown only (paragraphs, lists, emphasis, links, fenced code, LaTeX). Do NOT put ## headings inside markdown — use separate section objects instead.
- Prefer 2–12 clear sections; return between 1 and 20 sections.
- Write in the same language as the base content; preserve RTL directionality if present.
- Suggested images must include alt-text placeholders in markdown (e.g. ![description](url)).

Per-mode guidance (from emphasisMode):
- introduce: build from prerequisites; define terms carefully; scaffold foundational understanding.
- reinforce: keep core structure; add one worked example or practice cue on gap concepts.
- compress: condense aggressively for near-masters; skip proven material; keep all key terms and formulas.
- remediate: explicitly name each listed misconception and correct it with a clear contrast to the accurate concept.

If the base content is empty or unusable, return {"sections":[]}.`

// FidelityJudgeSystemPrompt is a fixed rubric for the LLM-judge support score.
const FidelityJudgeSystemPrompt = `You are a fidelity judge for adaptive LMS content. Compare AUTHORITATIVE base content against a GENERATED variant.
Respond with ONLY valid JSON: {"supportScore":0.0,"unsupportedClaims":["..."],"addressesMisconceptions":true,"notes":""}.

Rules:
- supportScore ∈ [0,1]: 1 = every factual claim in the variant is entailed by the base; 0 = many fabricated claims.
- List unsupportedClaims that introduce statistics, facts, or definitions not present in or entailed by the base.
- addressesMisconceptions: true if the variant explicitly names and corrects listed misconceptions (when any); true if none listed.
- Be strict: fabricated numbers or new factual claims must lower the score below 0.85.`

// GenerateInput is the pure generation request (no DB I/O inside GenerateVariant).
type GenerateInput struct {
	// Base content (authoritative).
	BaseMarkdown string
	BaseTitle    string
	// Profile decision (from AC.2).
	Profile ProfileResult
	// AllowedAxes is the effective axis set (unit ∩ course).
	AllowedAxes []string
	// KeyTerms that must appear (instructor-marked).
	KeyTerms []string
	// ConceptLabels maps concept id → display name (optional; gaps use id string if missing).
	ConceptLabels map[uuid.UUID]string
	// MisconceptionLabels maps misconception id string → name/description.
	MisconceptionLabels map[string]string
	// Model id for the provider.
	Model string
	// SystemPrompt override; empty ⇒ DefaultSystemPrompt.
	SystemPrompt string
	// PromptVersion stamped on the result (default PromptVersionCurrent).
	PromptVersion string
	// MinFidelity threshold (default DefaultMinFidelity).
	MinFidelity float64
	// ContentVersion of the unit base content.
	ContentVersion int32
	// RequireInstructorApproval forces pending_review instead of auto_served on pass.
	RequireInstructorApproval bool
	// SkipJudge skips the LLM fidelity judge (uses lexical overlap only). Useful in tests.
	SkipJudge bool
	// GatewayAllowed must be true; when false GenerateVariant returns ErrGatewayDenied without calling the model.
	GatewayAllowed bool
	// BudgetExhausted when true returns ErrBudgetExhausted (AC.4).
	BudgetExhausted bool
}

// Variant is the generation output (sections + gates + meta).
type Variant struct {
	Sections         []contentpagegeneration.DraftSection
	Markdown         string
	AxesApplied      []string
	FidelityScore    float64
	SafetyFlags      []string
	A11yFlags        []string
	Model            string
	PromptVersion    string
	PromptTokens     int
	CompletionTokens int
	// Status is draft | rejected | pending_review | auto_served.
	Status string
	// Fallback is true when the caller should serve base content.
	Fallback bool
	// FallbackReason is a short machine code when Fallback is true.
	FallbackReason string
	// ContentVersion echoed from input.
	ContentVersion int32
	// ProfileSignature echoed from profile.
	ProfileSignature string
	// CacheHit is true when the caller supplied a pre-existing variant (set by orchestrators).
	CacheHit bool
	// UnsupportedClaims from the fidelity judge.
	UnsupportedClaims []string
}

// IsNeutralLike reports whether this result is a neutral/base short-circuit.
func (v Variant) IsNeutralLike() bool {
	return v.ProfileSignature == NeutralSignature || v.FallbackReason == "neutral_profile"
}

// GenerateVariant produces a content variant from base content + profile.
// It never mutates base content. On gateway deny / budget / model error it returns
// a Fallback=true Variant with a typed error.
//
// FR-1 / FR-10: neutral profiles short-circuit without a model call.
func GenerateVariant(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	in GenerateInput,
) (Variant, aiprovider.CallMeta, error) {
	start := time.Now()
	defer func() {
		ObserveGenerate(float64(time.Since(start).Milliseconds()))
	}()

	minFid := in.MinFidelity
	if minFid <= 0 {
		minFid = DefaultMinFidelity
	}
	promptVersion := strings.TrimSpace(in.PromptVersion)
	if promptVersion == "" {
		promptVersion = PromptVersionCurrent
	}
	model := strings.TrimSpace(in.Model)
	axes := NormalizeAxes(in.AllowedAxes)
	if len(axes) == 0 {
		axes = NormalizeAxes(in.Profile.AxisSet)
	}

	// FR-10: neutral / base signature short-circuit — never call the model.
	if in.Profile.IsNeutral || in.Profile.ProfileSignature == NeutralSignature || in.Profile.ProfileSignature == "neutral" {
		sections := baseAsSections(in.BaseMarkdown, in.BaseTitle)
		md := SectionsToMarkdown(sections)
		IncGenerated("neutral")
		IncCacheHit() // treat as free hit / no model
		return Variant{
			Sections:         sections,
			Markdown:         md,
			AxesApplied:      []string{},
			FidelityScore:    1,
			SafetyFlags:      []string{},
			A11yFlags:        LintA11y(md),
			Model:            "",
			PromptVersion:    promptVersion,
			Status:           "auto_served",
			Fallback:         true,
			FallbackReason:   "neutral_profile",
			ContentVersion:   in.ContentVersion,
			ProfileSignature: NeutralSignature,
		}, aiprovider.CallMeta{}, nil
	}

	if in.BudgetExhausted {
		IncGenerated("budget")
		return fallbackVariant(in, promptVersion, "budget_exhausted"), aiprovider.CallMeta{}, ErrBudgetExhausted
	}
	if !in.GatewayAllowed {
		IncGenerated("gateway_denied")
		return fallbackVariant(in, promptVersion, "gateway_denied"), aiprovider.CallMeta{}, ErrGatewayDenied
	}
	if client == nil {
		IncGenerated("no_client")
		return fallbackVariant(in, promptVersion, "no_client"), aiprovider.CallMeta{}, fmt.Errorf("%w: nil client", ErrGenerationFailed)
	}

	// PII guard: profile inputs must not carry free-text names/emails (AC.2 construction).
	if err := AssertNoPII(in); err != nil {
		IncGenerated("pii_guard")
		return fallbackVariant(in, promptVersion, "pii_guard"), aiprovider.CallMeta{}, err
	}

	sys := strings.TrimSpace(in.SystemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}
	userPrompt := BuildUserPrompt(in)

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: userPrompt},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		IncGenerated("model_error")
		return fallbackVariant(in, promptVersion, "model_error"), meta, fmt.Errorf("%w: %v", ErrGenerationFailed, err)
	}

	sections, err := contentpagegeneration.ParseDraftSectionsJSON(res.Text)
	if err != nil {
		IncGenerated("parse_error")
		return fallbackVariant(in, promptVersion, "parse_error"), meta, fmt.Errorf("%w: %v", ErrGenerationFailed, err)
	}
	// Sanitize each section markdown.
	for i := range sections {
		sections[i].Markdown = SanitizeVariantMarkdown(sections[i].Markdown)
		sections[i].Heading = SanitizeVariantMarkdown(sections[i].Heading)
	}
	md := SectionsToMarkdown(sections)
	md = SanitizeVariantMarkdown(md)

	// Safety gate (FR-5).
	safetyFlags := SafetyScan(md)
	a11yFlags := LintA11y(md)

	// Fidelity hard checks (FR-3).
	hard := RunHardFidelityChecks(in.BaseMarkdown, md, in.KeyTerms)

	// LLM judge or lexical fallback.
	judgeScore := -1.0
	var unsupported []string
	if !in.SkipJudge {
		js, claims, jMeta, jErr := RunFidelityJudge(ctx, client, model, in.BaseMarkdown, md, in.Profile, in.MisconceptionLabels)
		// Accumulate judge tokens into meta usage.
		meta.Usage.PromptTokens += jMeta.Usage.PromptTokens
		meta.Usage.CompletionTokens += jMeta.Usage.CompletionTokens
		meta.Usage.TotalTokens += jMeta.Usage.TotalTokens
		meta.Usage.CostUSD += jMeta.Usage.CostUSD
		if jErr == nil {
			judgeScore = js
			unsupported = claims
		}
	}
	if judgeScore < 0 {
		// Hybrid: lexical claim overlap as embedding substitute.
		judgeScore = LexicalClaimOverlap(in.BaseMarkdown, md)
	}
	// Mode-specific soft checks.
	if in.Profile.EmphasisMode == EmphasisRemediate && len(in.Profile.Payload.Misconceptions) > 0 {
		// Prefer judge addressesMisconceptions; fallback: name must appear.
		for _, mid := range in.Profile.Payload.Misconceptions {
			label := mid
			if in.MisconceptionLabels != nil {
				if n, ok := in.MisconceptionLabels[mid]; ok && n != "" {
					label = n
				}
			}
			if label != "" && !strings.Contains(strings.ToLower(md), strings.ToLower(label)) {
				// Soft: reduce judge score slightly rather than hard-fail unless judge already handled it.
				if judgeScore > 0.5 {
					judgeScore = 0.5
				}
				hard.Flags = append(hard.Flags, "misconception_not_addressed:"+label)
			}
		}
	}

	fid := CompositeFidelityScore(hard, judgeScore)
	fid.UnsupportedClaims = unsupported
	ObserveFidelity(fid.Score)

	v := Variant{
		Sections:          sections,
		Markdown:          md,
		AxesApplied:       axesApplied(in.Profile, axes),
		FidelityScore:     fid.Score,
		SafetyFlags:       safetyFlags,
		A11yFlags:         a11yFlags,
		Model:             model,
		PromptVersion:     promptVersion,
		PromptTokens:      meta.Usage.PromptTokens,
		CompletionTokens:  meta.Usage.CompletionTokens,
		ContentVersion:    in.ContentVersion,
		ProfileSignature:  in.Profile.ProfileSignature,
		UnsupportedClaims: unsupported,
	}

	// Gate decision (FR-4 / FR-5).
	if len(safetyFlags) > 0 {
		v.Status = "rejected"
		v.Fallback = true
		v.FallbackReason = "safety"
		v.SafetyFlags = append(v.SafetyFlags, fid.Flags...)
		IncRejectedSafety()
		IncGenerated("rejected_safety")
		return v, meta, ErrRejectedSafety
	}
	if !fid.HardPass || fid.Score < minFid {
		v.Status = "rejected"
		v.Fallback = true
		v.FallbackReason = "fidelity"
		v.SafetyFlags = append(v.SafetyFlags, fid.Flags...)
		IncRejectedFidelity()
		IncGenerated("rejected_fidelity")
		return v, meta, ErrRejectedFidelity
	}

	if in.RequireInstructorApproval {
		v.Status = "pending_review"
	} else {
		v.Status = "auto_served"
		IncAutoServed()
	}
	IncGenerated("ok")
	return v, meta, nil
}

func fallbackVariant(in GenerateInput, promptVersion, reason string) Variant {
	sections := baseAsSections(in.BaseMarkdown, in.BaseTitle)
	md := SectionsToMarkdown(sections)
	return Variant{
		Sections:         sections,
		Markdown:         md,
		AxesApplied:      []string{},
		FidelityScore:    0,
		SafetyFlags:      []string{},
		A11yFlags:        []string{},
		Model:            strings.TrimSpace(in.Model),
		PromptVersion:    promptVersion,
		Status:           "rejected",
		Fallback:         true,
		FallbackReason:   reason,
		ContentVersion:   in.ContentVersion,
		ProfileSignature: in.Profile.ProfileSignature,
	}
}

// BuildUserPrompt encodes base content + profile + axes for the model (FR-2).
func BuildUserPrompt(in GenerateInput) string {
	var b strings.Builder
	if title := strings.TrimSpace(in.BaseTitle); title != "" {
		fmt.Fprintf(&b, "Page title: %s\n\n", title)
	}
	fmt.Fprintf(&b, "Emphasis mode: %s\n", strings.TrimSpace(in.Profile.EmphasisMode))
	if bloom := strings.TrimSpace(in.Profile.TargetBloom); bloom != "" {
		fmt.Fprintf(&b, "Target Bloom level: %s\n", bloom)
	}
	axes := NormalizeAxes(in.AllowedAxes)
	if len(axes) == 0 {
		axes = NormalizeAxes(in.Profile.AxisSet)
	}
	fmt.Fprintf(&b, "Allowed axes: %s\n", strings.Join(axes, ", "))
	if pref := strings.TrimSpace(in.Profile.ReadingLevelPref); pref != "" && pref != "default" {
		fmt.Fprintf(&b, "Reading level preference: %s\n", pref)
	}
	if pref := strings.TrimSpace(in.Profile.ModalityPref); pref != "" && pref != "default" {
		fmt.Fprintf(&b, "Modality preference: %s\n", pref)
	}

	// Concept gaps by name.
	if len(in.Profile.Payload.ConceptGaps) > 0 {
		b.WriteString("Concept gaps (name → gap 0–1, higher = weaker):\n")
		for _, g := range in.Profile.Payload.ConceptGaps {
			name := g.ConceptID.String()
			if in.ConceptLabels != nil {
				if n, ok := in.ConceptLabels[g.ConceptID]; ok && n != "" {
					name = n
				}
			}
			fmt.Fprintf(&b, "- %s: %.2f\n", name, g.Gap)
		}
	}

	// Misconceptions by name/description.
	if len(in.Profile.Payload.Misconceptions) > 0 {
		b.WriteString("Misconceptions to address (by name):\n")
		for _, mid := range in.Profile.Payload.Misconceptions {
			label := mid
			if in.MisconceptionLabels != nil {
				if n, ok := in.MisconceptionLabels[mid]; ok && n != "" {
					label = n
				}
			}
			fmt.Fprintf(&b, "- %s\n", label)
		}
	}

	if len(in.KeyTerms) > 0 {
		b.WriteString("Key terms that MUST appear exactly (do not omit or rephrase away):\n")
		for _, t := range in.KeyTerms {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}

	base := strings.TrimSpace(in.BaseMarkdown)
	if utf8.RuneCountInString(base) > contentpagegeneration.MaxExistingMarkdownRunes {
		base = string([]rune(base)[:contentpagegeneration.MaxExistingMarkdownRunes])
	}
	fmt.Fprintf(&b, "\nAUTHORITATIVE base content (preserve all facts, formulas, code, and key terms):\n%s\n", base)
	b.WriteString("\nReturn ONLY the JSON object with sections as specified.")
	return b.String()
}

// RunFidelityJudge asks the model for a support score + unsupported claims.
func RunFidelityJudge(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, base, variant string,
	profile ProfileResult,
	misLabels map[string]string,
) (score float64, unsupported []string, meta aiprovider.CallMeta, err error) {
	var user strings.Builder
	user.WriteString("AUTHORITATIVE base content:\n")
	user.WriteString(truncateRunes(base, 40_000))
	user.WriteString("\n\nGENERATED variant:\n")
	user.WriteString(truncateRunes(variant, 40_000))
	if len(profile.Payload.Misconceptions) > 0 {
		user.WriteString("\n\nMisconceptions that should be addressed:\n")
		for _, mid := range profile.Payload.Misconceptions {
			label := mid
			if misLabels != nil {
				if n, ok := misLabels[mid]; ok && n != "" {
					label = n
				}
			}
			fmt.Fprintf(&user, "- %s\n", label)
		}
	}
	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: FidelityJudgeSystemPrompt},
		{Role: "user", Content: user.String()},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return -1, nil, meta, err
	}
	score, unsupported, err = ParseFidelityJudgeJSON(res.Text)
	return score, unsupported, meta, err
}

// ParseFidelityJudgeJSON parses the judge response.
func ParseFidelityJudgeJSON(raw string) (score float64, unsupported []string, err error) {
	text := strings.TrimSpace(raw)
	// Reuse fence stripping via a minimal approach.
	if idx := strings.Index(text, "```"); idx != -1 {
		text = text[idx+3:]
		if strings.HasPrefix(strings.ToLower(text), "json") {
			text = text[4:]
		}
		if end := strings.Index(text, "```"); end != -1 {
			text = text[:end]
		}
	}
	if start := strings.Index(text, "{"); start != -1 {
		if end := strings.LastIndex(text, "}"); end > start {
			text = text[start : end+1]
		}
	}
	var payload struct {
		SupportScore          float64  `json:"supportScore"`
		UnsupportedClaims     []string `json:"unsupportedClaims"`
		AddressesMisconceptions *bool  `json:"addressesMisconceptions"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &payload); err != nil {
		return -1, nil, fmt.Errorf("parse fidelity judge JSON: %w", err)
	}
	score = payload.SupportScore
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	if payload.AddressesMisconceptions != nil && !*payload.AddressesMisconceptions {
		if score > 0.7 {
			score = 0.7
		}
	}
	if payload.UnsupportedClaims == nil {
		payload.UnsupportedClaims = []string{}
	}
	return score, payload.UnsupportedClaims, nil
}

// SectionsToMarkdown joins draft sections into a single markdown document.
func SectionsToMarkdown(sections []contentpagegeneration.DraftSection) string {
	var b strings.Builder
	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n\n")
		}
		h := strings.TrimSpace(s.Heading)
		if h != "" {
			fmt.Fprintf(&b, "## %s\n\n", h)
		}
		b.WriteString(strings.TrimSpace(s.Markdown))
	}
	return b.String()
}

// baseAsSections wraps base markdown as a single section (optionally titled).
func baseAsSections(baseMarkdown, title string) []contentpagegeneration.DraftSection {
	md := strings.TrimSpace(baseMarkdown)
	if utf8.RuneCountInString(md) > contentpagegeneration.MaxMarkdownRunes {
		md = string([]rune(md)[:contentpagegeneration.MaxMarkdownRunes])
	}
	heading := strings.TrimSpace(title)
	if utf8.RuneCountInString(heading) > contentpagegeneration.MaxHeadingRunes {
		heading = string([]rune(heading)[:contentpagegeneration.MaxHeadingRunes])
	}
	if md == "" && heading == "" {
		return []contentpagegeneration.DraftSection{}
	}
	return []contentpagegeneration.DraftSection{{Heading: heading, Markdown: md}}
}

func axesApplied(profile ProfileResult, allowed []string) []string {
	// Always include emphasis when non-neutral; include other axes present on profile prefs.
	out := make([]string, 0, 4)
	seen := map[string]struct{}{}
	add := func(a string) {
		a = strings.TrimSpace(a)
		if a == "" {
			return
		}
		if _, ok := seen[a]; ok {
			return
		}
		// Only record axes that are allowed (or emphasis always).
		if a != "emphasis" {
			allowedOK := false
			for _, x := range allowed {
				if x == a {
					allowedOK = true
					break
				}
			}
			if !allowedOK {
				return
			}
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	add("emphasis")
	if len(profile.Payload.Misconceptions) > 0 {
		add("misconception")
	}
	if pref := strings.TrimSpace(profile.ReadingLevelPref); pref != "" && pref != "default" {
		add("reading_level")
	}
	if pref := strings.TrimSpace(profile.ModalityPref); pref != "" && pref != "default" {
		add("modality")
	}
	// Scaffolding is implied by introduce/remediate emphasis when allowed.
	if profile.EmphasisMode == EmphasisIntroduce || profile.EmphasisMode == EmphasisRemediate {
		add("scaffolding")
	}
	return out
}

// AssertNoPII is a defence-in-depth guard: profile payloads must not include email-like
// strings or obvious name fields. Concept/misconception labels are course content, not PII.
func AssertNoPII(in GenerateInput) error {
	// Check free-text labels for email patterns only (names of concepts are fine).
	emailRe := strings.Contains
	check := func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		// Simple email detection.
		if strings.Contains(s, "@") && strings.Contains(s, ".") {
			// Allow if it looks like a code sample; still block obvious emails.
			at := strings.Index(s, "@")
			if at > 0 && at < len(s)-3 {
				return fmt.Errorf("adaptive content: PII guard blocked prompt input containing email-like text")
			}
		}
		_ = emailRe
		return nil
	}
	for _, t := range in.KeyTerms {
		if err := check(t); err != nil {
			return err
		}
	}
	return nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// SyntheticProfileFromRequest builds a ProfileResult from instructor preview controls.
func SyntheticProfileFromRequest(
	emphasisMode string,
	targetBloom string,
	readingLevel string,
	modality string,
	conceptGaps []ConceptGap,
	misconceptionIDs []string,
	axisSet []string,
) ProfileResult {
	mode := strings.TrimSpace(emphasisMode)
	if mode == "" {
		mode = EmphasisReinforce
	}
	switch mode {
	case EmphasisIntroduce, EmphasisReinforce, EmphasisCompress, EmphasisRemediate:
	default:
		mode = EmphasisReinforce
	}
	bloom := strings.TrimSpace(targetBloom)
	if bloom == "" {
		switch mode {
		case EmphasisIntroduce:
			bloom = "remember"
		case EmphasisCompress:
			bloom = "analyze"
		case EmphasisRemediate:
			bloom = "understand"
		default:
			bloom = "understand"
		}
	}
	if readingLevel == "" {
		readingLevel = "default"
	}
	if modality == "" {
		modality = "default"
	}
	axes := NormalizeAxes(axisSet)
	if len(axes) == 0 {
		axes = []string{"emphasis", "scaffolding", "reading_level", "misconception"}
	}
	payload := ProfilePayload{
		ConceptGaps:    conceptGaps,
		Misconceptions: misconceptionIDs,
		PriorRecord:    true,
	}
	if len(conceptGaps) > 0 {
		var sum float64
		for _, g := range conceptGaps {
			sum += g.Gap
		}
		payload.MeanGap = sum / float64(len(conceptGaps))
	}
	// Stable signature for synthetic previews (not learner-linked).
	sig, err := ProfileSignature(map[string]any{
		"emphasis":       mode,
		"bloom":          bloom,
		"reading":        readingLevel,
		"modality":       modality,
		"axes":           axes,
		"gaps":           conceptGaps,
		"misconceptions": misconceptionIDs,
		"synthetic":      true,
	})
	if err != nil {
		sig = "synthetic"
	}
	return ProfileResult{
		EmphasisMode:     mode,
		TargetBloom:      bloom,
		ProfileSignature: sig,
		IsNeutral:        false,
		ReadingLevelPref: readingLevel,
		ModalityPref:     modality,
		AxisSet:          axes,
		Payload:          payload,
	}
}
