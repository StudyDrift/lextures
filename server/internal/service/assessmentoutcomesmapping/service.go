package assessmentoutcomesmapping

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const (
	// MaxItems caps unmapped assessments per suggestion run (CC.10 §11 cost budget).
	MaxItems = 200
	// MaxSuggestions is the hard ceiling on proposals returned.
	MaxSuggestions = 200
	// MaxPromptMaterial bounds the user prompt rune count.
	MaxPromptMaterial = 80_000
	// MaxRationaleRunes truncates instructor-facing rationale.
	MaxRationaleRunes = 400
	// MaxItemTitle truncates assessment titles in the prompt.
	MaxItemTitle = 200

	defaultMeasurement = "summative"
	defaultIntensity   = "medium"
	defaultConfidence  = 0.5
)

// OutcomeInput is a course learning outcome offered for mapping.
type OutcomeInput struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// AssessmentInput is an unmapped assignment or quiz structure item.
// Must not include learner data, submissions, or grades (CC.10 §11).
type AssessmentInput struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind"` // assignment | quiz
	Module   string `json:"module,omitempty"`
	Points   *int   `json:"points,omitempty"`
	HasBody  bool   `json:"hasBody,omitempty"`
	Criteria string `json:"criteria,omitempty"` // short rubric/instructions excerpt only
}

// SuggestInput is the material sent to the model.
type SuggestInput struct {
	CourseTitle  string
	CourseLang   string
	Outcomes     []OutcomeInput
	Assessments  []AssessmentInput
}

// Proposal is one proposed structure-item → outcome link (not persisted).
type Proposal struct {
	StructureItemID  string  `json:"structureItemId"`
	ItemTitle        string  `json:"itemTitle,omitempty"`
	ItemKind         string  `json:"itemKind,omitempty"` // assignment | quiz
	OutcomeID        string  `json:"outcomeId"`
	OutcomeTitle     string  `json:"outcomeTitle,omitempty"`
	MeasurementLevel string  `json:"measurementLevel"`
	IntensityLevel   string  `json:"intensityLevel"`
	Confidence       float64 `json:"confidence"`
	Rationale        string  `json:"rationale"`
}

// DefaultSystemPrompt instructs the model to return structured mapping JSON only.
const DefaultSystemPrompt = `You map course assessments (assignments and quizzes) to learning outcomes for an LMS.
Respond with ONLY valid JSON (no markdown fences, no commentary).

The JSON must be an object:
{"proposals":[{"structureItemId":"...","outcomeId":"...","measurementLevel":"diagnostic"|"formative"|"summative"|"performance","intensityLevel":"low"|"medium"|"high","confidence":0.0-1.0,"rationale":"..."}]}

Rules:
- Use only structureItemId values from the provided assessments list (exact match).
- Use only outcomeId values from the provided outcomes list (exact match).
- Prefer the strongest relevant outcomes; do not force weak matches.
- An assessment may link to multiple outcomes when justified; avoid redundant near-duplicates.
- measurementLevel: diagnostic (pre-check), formative (practice/feedback), summative (graded mastery), performance (authentic/applied).
- intensityLevel: how strongly this item evidences the outcome (low/medium/high).
- confidence: 0–1 model confidence that the link is appropriate.
- rationale: short instructor-facing reason (one sentence). Do not invent student data.
- If nothing maps well, return {"proposals":[]}.
- Return at most 200 proposals.
- Never include student names, submissions, grades, or accommodations.`

// Suggest asks the model for draft assessment → outcome links.
func Suggest(
	ctx context.Context,
	client aiprovider.ScopedCompleter,
	model, systemPrompt string,
	in SuggestInput,
) ([]Proposal, aiprovider.CallMeta, error) {
	if len(in.Outcomes) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("outcomes are required")
	}
	if len(in.Assessments) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("assessments are required")
	}

	validOutcomes := make(map[string]struct{}, len(in.Outcomes))
	outcomeTitles := make(map[string]string, len(in.Outcomes))
	outcomes := make([]OutcomeInput, 0, len(in.Outcomes))
	for _, o := range in.Outcomes {
		id := strings.TrimSpace(o.ID)
		title := strings.TrimSpace(o.Title)
		if id == "" || title == "" {
			continue
		}
		validOutcomes[id] = struct{}{}
		outcomeTitles[id] = title
		outcomes = append(outcomes, OutcomeInput{
			ID:          id,
			Title:       title,
			Description: truncateRunes(strings.TrimSpace(o.Description), 500),
		})
	}
	if len(outcomes) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("outcomes are required")
	}

	validItems := make(map[string]struct{}, len(in.Assessments))
	itemTitles := make(map[string]string, len(in.Assessments))
	itemKinds := make(map[string]string, len(in.Assessments))
	assessments := make([]AssessmentInput, 0, len(in.Assessments))
	for i, a := range in.Assessments {
		if i >= MaxItems {
			break
		}
		id := strings.TrimSpace(a.ID)
		title := strings.TrimSpace(a.Title)
		if id == "" || title == "" {
			continue
		}
		// Reject prompt material that looks like learner PII fields (CC.10 §11).
		if looksLikeLearnerField(title) || looksLikeLearnerField(a.Criteria) {
			return nil, aiprovider.CallMeta{}, fmt.Errorf("assessment prompt rejected: learner data fields are not allowed")
		}
		kind := strings.TrimSpace(strings.ToLower(a.Kind))
		if kind != "assignment" && kind != "quiz" {
			kind = "assignment"
		}
		validItems[id] = struct{}{}
		itemTitles[id] = title
		itemKinds[id] = kind
		assessments = append(assessments, AssessmentInput{
			ID:       id,
			Title:    truncateRunes(title, MaxItemTitle),
			Kind:     kind,
			Module:   truncateRunes(strings.TrimSpace(a.Module), 120),
			Points:   a.Points,
			HasBody:  a.HasBody,
			Criteria: truncateRunes(strings.TrimSpace(a.Criteria), 400),
		})
	}
	if len(assessments) == 0 {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("assessments are required")
	}

	payload := map[string]any{
		"courseTitle": strings.TrimSpace(in.CourseTitle),
		"language":    strings.TrimSpace(in.CourseLang),
		"outcomes":    outcomes,
		"assessments": assessments,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, aiprovider.CallMeta{}, err
	}
	user := "Suggest outcome mappings for unmapped assessments. Input JSON:\n" + string(encoded)
	if utf8.RuneCountInString(user) > MaxPromptMaterial {
		return nil, aiprovider.CallMeta{}, fmt.Errorf("mapping prompt is too long (max %d characters)", MaxPromptMaterial)
	}

	sys := strings.TrimSpace(systemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}

	res, meta, err := client.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}, aiprovider.ChatOptions{JSONMode: true})
	if err != nil {
		return nil, meta, err
	}
	proposals, err := ParseProposalsJSON(res.Text, validOutcomes, validItems, outcomeTitles, itemTitles, itemKinds)
	if err != nil {
		return nil, meta, err
	}
	return proposals, meta, nil
}

// ParseProposalsJSON parses and normalizes model JSON into draft proposals.
func ParseProposalsJSON(
	raw string,
	validOutcomeIDs, validItemIDs map[string]struct{},
	outcomeTitles, itemTitles, itemKinds map[string]string,
) ([]Proposal, error) {
	text := stripJSONFences(raw)
	var payload struct {
		Proposals []Proposal `json:"proposals"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("parse assessment outcome mapping JSON: %w", err)
	}
	return normalizeProposals(payload.Proposals, validOutcomeIDs, validItemIDs, outcomeTitles, itemTitles, itemKinds), nil
}

func normalizeProposals(
	in []Proposal,
	validOutcomeIDs, validItemIDs map[string]struct{},
	outcomeTitles, itemTitles, itemKinds map[string]string,
) []Proposal {
	out := make([]Proposal, 0, len(in))
	seen := make(map[string]struct{})
	for _, p := range in {
		itemID := strings.TrimSpace(p.StructureItemID)
		outcomeID := strings.TrimSpace(p.OutcomeID)
		if itemID == "" || outcomeID == "" {
			continue
		}
		if _, ok := validItemIDs[itemID]; !ok {
			continue
		}
		if _, ok := validOutcomeIDs[outcomeID]; !ok {
			continue
		}
		measurement := strings.TrimSpace(strings.ToLower(p.MeasurementLevel))
		if !containsString(courseoutcomes.MeasurementLevels, measurement) {
			measurement = defaultMeasurement
		}
		intensity := strings.TrimSpace(strings.ToLower(p.IntensityLevel))
		if !containsString(courseoutcomes.IntensityLevels, intensity) {
			intensity = defaultIntensity
		}
		conf := p.Confidence
		if conf < 0 || conf > 1 || conf == 0 {
			conf = defaultConfidence
		}
		rationale := strings.TrimSpace(p.Rationale)
		if utf8.RuneCountInString(rationale) > MaxRationaleRunes {
			rationale = string([]rune(rationale)[:MaxRationaleRunes])
		}
		key := itemID + "|" + outcomeID + "|" + measurement
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		kind := itemKinds[itemID]
		if kind != "quiz" {
			kind = "assignment"
		}
		out = append(out, Proposal{
			StructureItemID:  itemID,
			ItemTitle:        itemTitles[itemID],
			ItemKind:         kind,
			OutcomeID:        outcomeID,
			OutcomeTitle:     outcomeTitles[outcomeID],
			MeasurementLevel: measurement,
			IntensityLevel:   intensity,
			Confidence:       conf,
			Rationale:        rationale,
		})
		if len(out) >= MaxSuggestions {
			break
		}
	}
	return out
}

// looksLikeLearnerField rejects prompt fields that appear to carry student PII (CC.10 §11 test hook).
func looksLikeLearnerField(s string) bool {
	lower := strings.ToLower(s)
	for _, banned := range []string{
		"student name", "submission", "grade for", "accommodation",
		"email@", "enrollment id", "learner id",
	} {
		if strings.Contains(lower, banned) {
			return true
		}
	}
	return false
}

func containsString(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}

func stripJSONFences(raw string) string {
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```json"); idx != -1 {
		text = text[idx+7:]
		if endIdx := strings.Index(text, "```"); endIdx != -1 {
			text = text[:endIdx]
		}
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if endIdx := strings.Index(text, "```"); endIdx != -1 {
			text = text[:endIdx]
		}
	}
	return strings.TrimSpace(text)
}
