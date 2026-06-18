// Package onboarding implements self-learner onboarding placement and recommendations (plan 15.11).
package onboarding

import (
	"context"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Question is a lightweight calibration item for the onboarding diagnostic.
type Question struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Choices []string `json:"choices"`
	Answer  int      `json:"-"`
}

// EffectiveLevel returns the placement level from self-assessment and optional diagnostic score.
func EffectiveLevel(priorLevel string, diagnosticScore *float64, diagnosticSkipped bool) string {
	if !diagnosticSkipped && diagnosticScore != nil {
		score := *diagnosticScore
		switch {
		case score >= 70:
			return "advanced"
		case score >= 40:
			return "intermediate"
		default:
			return "beginner"
		}
	}
	switch priorLevel {
	case "intermediate", "advanced":
		return priorLevel
	default:
		return "beginner"
	}
}

// ScoreDiagnostic computes a 0–100 score from answer indices keyed by question id.
func ScoreDiagnostic(topic string, answers map[string]int) float64 {
	questions := QuestionsForTopic(topic)
	if len(questions) == 0 {
		return 0
	}
	correct := 0
	for _, q := range questions {
		if answers[q.ID] == q.Answer {
			correct++
		}
	}
	return math.Round(float64(correct)/float64(len(questions))*10000) / 100
}

// QuestionsForTopic returns five calibration questions for the chosen topic.
func QuestionsForTopic(topic string) []Question {
	key := strings.ToLower(strings.TrimSpace(topic))
	if bank, ok := questionBanks[key]; ok {
		return bank
	}
	return questionBanks["general"]
}

// RecommendCourse finds a published self-paced course matching topic and level.
func RecommendCourse(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, topic, level string) (code, title string, ok bool) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", "", false
	}
	pattern := "%" + topic + "%"
	rows, err := pool.Query(ctx, `
SELECT course_code, title
FROM course.courses
WHERE organization_id = $1
  AND published = TRUE
  AND course_mode = 'self_paced'
  AND (title ILIKE $2 OR description ILIKE $2)
ORDER BY title ASC
LIMIT 50
`, orgID, pattern)
	if err != nil {
		return "", "", false
	}
	defer rows.Close()

	type candidate struct {
		code  string
		title string
	}
	var matches []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.code, &c.title); err != nil {
			continue
		}
		matches = append(matches, c)
	}
	if len(matches) == 0 {
		return "", "", false
	}

	levelKeywords := levelKeywordsFor(level)
	for _, c := range matches {
		lower := strings.ToLower(c.title)
		for _, kw := range levelKeywords {
			if strings.Contains(lower, kw) {
				return c.code, c.title, true
			}
		}
	}
	// Fallback: first topic match without level filtering.
	c := matches[0]
	return c.code, c.title, true
}

func levelKeywordsFor(level string) []string {
	switch level {
	case "advanced":
		return []string{"advanced", "expert", "mastery", "professional"}
	case "intermediate":
		return []string{"intermediate", "beyond basics", "level 2", "part 2"}
	default:
		return []string{"beginner", "introduction", "intro", "fundamentals", "basics", "101", "getting started"}
	}
}

var questionBanks = map[string][]Question{
	"python": {
		{ID: "py1", Prompt: "What does `len([1, 2, 3])` return in Python?", Choices: []string{"2", "3", "Error", "None"}, Answer: 1},
		{ID: "py2", Prompt: "Which keyword defines a function in Python?", Choices: []string{"func", "def", "fn", "function"}, Answer: 1},
		{ID: "py3", Prompt: "What is the output of `print(2 ** 3)`?", Choices: []string{"6", "8", "9", "5"}, Answer: 1},
		{ID: "py4", Prompt: "Which type is mutable?", Choices: []string{"tuple", "str", "list", "int"}, Answer: 2},
		{ID: "py5", Prompt: "How do you start a comment in Python?", Choices: []string{"//", "#", "--", "/*"}, Answer: 1},
	},
	"javascript": {
		{ID: "js1", Prompt: "Which keyword declares a block-scoped variable?", Choices: []string{"var", "let", "define", "const only"}, Answer: 1},
		{ID: "js2", Prompt: "What does `typeof []` return?", Choices: []string{"array", "object", "list", "undefined"}, Answer: 1},
		{ID: "js3", Prompt: "Which method adds an item to the end of an array?", Choices: []string{"append", "push", "add", "insert"}, Answer: 1},
		{ID: "js4", Prompt: "What is `null == undefined`?", Choices: []string{"false", "true", "Error", "null"}, Answer: 1},
		{ID: "js5", Prompt: "Which symbol starts a single-line comment?", Choices: []string{"#", "//", "--", "/*"}, Answer: 1},
	},
	"data-science": {
		{ID: "ds1", Prompt: "Which library is commonly used for tabular data in Python?", Choices: []string{"NumPy only", "pandas", "React", "Express"}, Answer: 1},
		{ID: "ds2", Prompt: "A histogram shows:", Choices: []string{"correlation", "distribution", "geography", "time zones"}, Answer: 1},
		{ID: "ds3", Prompt: "Mean is sensitive to:", Choices: []string{"outliers", "sample size only", "column names", "file format"}, Answer: 0},
		{ID: "ds4", Prompt: "Train/test split helps measure:", Choices: []string{"UI latency", "generalization", "SMTP delivery", "GPU temperature"}, Answer: 1},
		{ID: "ds5", Prompt: "CSV files store data as:", Choices: []string{"binary images", "delimited text", "compiled bytecode", "3D meshes"}, Answer: 1},
	},
	"general": {
		{ID: "g1", Prompt: "Spaced repetition helps you:", Choices: []string{"forget faster", "retain knowledge", "skip practice", "avoid review"}, Answer: 1},
		{ID: "g2", Prompt: "A learning goal should be:", Choices: []string{"vague", "specific and measurable", "secret", "unchangeable"}, Answer: 1},
		{ID: "g3", Prompt: "Active recall means:", Choices: []string{"re-reading only", "testing yourself", "highlighting everything", "skipping hard parts"}, Answer: 1},
		{ID: "g4", Prompt: "Short daily sessions often beat:", Choices: []string{"consistent practice", "rare marathon cramming", "taking notes", "asking questions"}, Answer: 1},
		{ID: "g5", Prompt: "When stuck, a good first step is:", Choices: []string{"give up", "break the problem down", "ignore instructions", "skip ahead randomly"}, Answer: 1},
	},
}
