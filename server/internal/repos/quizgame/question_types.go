package quizgame

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Supported game question types (IQ.2).
const (
	QTypeMCSingle   = "mc_single"
	QTypeMCMultiple = "mc_multiple"
	QTypeTrueFalse  = "true_false"
	QTypeTypeAnswer = "type_answer"
	QTypeNumeric    = "numeric"
	QTypePoll       = "poll"
	QTypeOrdering   = "ordering"
	QTypeWordCloud  = "word_cloud"
)

const (
	PointsStandard = "standard"
	PointsDouble   = "double"
	PointsNone     = "no_points"
)

const (
	minTimeLimit = 5
	maxTimeLimit = 240
	defTimeLimit = 20
	maxPromptLen = 4000
	maxOptionLen = 500
	maxExplainLen = 2000
	maxOptions    = 6
	minOptions    = 2
	maxAccepted   = 20
)

// Option is one MC / poll / ordering choice.
type Option struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	MediaRef  *string `json:"mediaRef,omitempty"`
	MediaAlt  *string `json:"mediaAlt,omitempty"`
	IsCorrect bool    `json:"isCorrect"`
}

// AcceptedAnswer is one type_answer entry.
type AcceptedAnswer struct {
	Text      string `json:"text"`
	MatchMode string `json:"matchMode"` // exact | case_insensitive | trim | fuzzy
	FuzzyMax  *int   `json:"fuzzyMax,omitempty"`
}

// NumericCorrect is the numeric correct_answer payload.
type NumericCorrect struct {
	Value     float64  `json:"value"`
	Tolerance float64  `json:"tolerance"`
	Unit      *string  `json:"unit,omitempty"`
}

// TypeAnswerCorrect wraps accepted answers.
type TypeAnswerCorrect struct {
	Accepted []AcceptedAnswer `json:"accepted"`
}

// OrderingCorrect stores the correct option id sequence.
type OrderingCorrect struct {
	Order []string `json:"order"`
}

// ValidIssue is one blocking validation problem.
type ValidIssue struct {
	QuestionID string `json:"questionId"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func isValidQuestionType(t string) bool {
	switch t {
	case QTypeMCSingle, QTypeMCMultiple, QTypeTrueFalse, QTypeTypeAnswer,
		QTypeNumeric, QTypePoll, QTypeOrdering, QTypeWordCloud:
		return true
	default:
		return false
	}
}

func isValidPointsStyle(s string) bool {
	switch s {
	case PointsStandard, PointsDouble, PointsNone:
		return true
	default:
		return false
	}
}

func sanitizePlainText(s string, max int) string {
	s = strings.TrimSpace(s)
	// Strip angle-bracket tags to keep projector HTML-safe (markdown-lite only).
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(out) > max {
		runes := []rune(out)
		out = string(runes[:max])
	}
	return out
}

func defaultOptionsForType(qtype string) []Option {
	switch qtype {
	case QTypeTrueFalse:
		return []Option{
			{ID: "true", Text: "True", IsCorrect: true},
			{ID: "false", Text: "False", IsCorrect: false},
		}
	case QTypeMCSingle, QTypeMCMultiple, QTypePoll:
		return []Option{
			{ID: "a", Text: "", IsCorrect: false},
			{ID: "b", Text: "", IsCorrect: false},
			{ID: "c", Text: "", IsCorrect: false},
			{ID: "d", Text: "", IsCorrect: false},
		}
	case QTypeOrdering:
		return []Option{
			{ID: "1", Text: "", IsCorrect: false},
			{ID: "2", Text: "", IsCorrect: false},
			{ID: "3", Text: "", IsCorrect: false},
		}
	default:
		return []Option{}
	}
}

func marshalOptions(opts []Option) (json.RawMessage, error) {
	if opts == nil {
		opts = []Option{}
	}
	b, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func unmarshalOptions(raw json.RawMessage) ([]Option, error) {
	if len(raw) == 0 {
		return []Option{}, nil
	}
	var opts []Option
	if err := json.Unmarshal(raw, &opts); err != nil {
		return nil, fmt.Errorf("quizgame: invalid options JSON")
	}
	if opts == nil {
		opts = []Option{}
	}
	return opts, nil
}

// NormalizeCreateInput fills defaults and sanitises a create/patch payload.
func NormalizeCreateInput(in *CreateQuestionInput) error {
	if in.QuestionType == "" {
		in.QuestionType = QTypeMCSingle
	}
	in.QuestionType = strings.TrimSpace(strings.ToLower(in.QuestionType))
	if !isValidQuestionType(in.QuestionType) {
		return fmt.Errorf("quizgame: invalid question type")
	}
	in.Prompt = sanitizePlainText(in.Prompt, maxPromptLen)
	if in.Explanation != nil {
		e := sanitizePlainText(*in.Explanation, maxExplainLen)
		in.Explanation = &e
	}
	if in.PromptMediaAlt != nil {
		a := sanitizePlainText(*in.PromptMediaAlt, 500)
		in.PromptMediaAlt = &a
	}
	if in.TimeLimitSeconds <= 0 {
		in.TimeLimitSeconds = defTimeLimit
	}
	if in.TimeLimitSeconds < minTimeLimit || in.TimeLimitSeconds > maxTimeLimit {
		return fmt.Errorf("quizgame: time_limit_seconds must be between %d and %d", minTimeLimit, maxTimeLimit)
	}
	if in.PointsStyle == "" {
		in.PointsStyle = PointsStandard
	}
	in.PointsStyle = strings.TrimSpace(strings.ToLower(in.PointsStyle))
	if !isValidPointsStyle(in.PointsStyle) {
		return fmt.Errorf("quizgame: invalid points_style")
	}
	if len(in.Options) == 0 {
		in.Options = defaultOptionsForType(in.QuestionType)
	}
	for i := range in.Options {
		in.Options[i].Text = sanitizePlainText(in.Options[i].Text, maxOptionLen)
		if in.Options[i].ID == "" {
			in.Options[i].ID = fmt.Sprintf("opt-%d", i+1)
		}
		if in.Options[i].MediaAlt != nil {
			a := sanitizePlainText(*in.Options[i].MediaAlt, 500)
			in.Options[i].MediaAlt = &a
		}
	}
	if err := normalizeCorrectForType(in); err != nil {
		return err
	}
	return nil
}

func normalizeCorrectForType(in *CreateQuestionInput) error {
	switch in.QuestionType {
	case QTypeTypeAnswer:
		var ca TypeAnswerCorrect
		if len(in.CorrectAnswer) > 0 {
			if err := json.Unmarshal(in.CorrectAnswer, &ca); err != nil {
				return fmt.Errorf("quizgame: invalid type_answer correct_answer")
			}
		}
		for i := range ca.Accepted {
			ca.Accepted[i].Text = sanitizePlainText(ca.Accepted[i].Text, maxOptionLen)
			m := strings.TrimSpace(strings.ToLower(ca.Accepted[i].MatchMode))
			if m == "" {
				m = "case_insensitive"
			}
			switch m {
			case "exact", "case_insensitive", "trim", "fuzzy":
				ca.Accepted[i].MatchMode = m
			default:
				return fmt.Errorf("quizgame: invalid match mode")
			}
			if m == "fuzzy" && (ca.Accepted[i].FuzzyMax == nil || *ca.Accepted[i].FuzzyMax < 0) {
				n := 1
				ca.Accepted[i].FuzzyMax = &n
			}
		}
		if len(ca.Accepted) > maxAccepted {
			return fmt.Errorf("quizgame: too many accepted answers")
		}
		b, err := json.Marshal(ca)
		if err != nil {
			return err
		}
		in.CorrectAnswer = b
	case QTypeNumeric:
		var ca NumericCorrect
		if len(in.CorrectAnswer) > 0 {
			if err := json.Unmarshal(in.CorrectAnswer, &ca); err != nil {
				return fmt.Errorf("quizgame: invalid numeric correct_answer")
			}
		}
		if ca.Tolerance < 0 {
			return fmt.Errorf("quizgame: tolerance must be >= 0")
		}
		if ca.Unit != nil {
			u := sanitizePlainText(*ca.Unit, 32)
			ca.Unit = &u
		}
		b, err := json.Marshal(ca)
		if err != nil {
			return err
		}
		in.CorrectAnswer = b
	case QTypeOrdering:
		var ca OrderingCorrect
		if len(in.CorrectAnswer) > 0 {
			_ = json.Unmarshal(in.CorrectAnswer, &ca)
		}
		if len(ca.Order) == 0 {
			for _, o := range in.Options {
				ca.Order = append(ca.Order, o.ID)
			}
		}
		b, err := json.Marshal(ca)
		if err != nil {
			return err
		}
		in.CorrectAnswer = b
	case QTypePoll, QTypeWordCloud:
		in.CorrectAnswer = nil
	}
	return nil
}

// ValidateQuestionReady returns blocking issues for "ready to host".
func ValidateQuestionReady(q Question) []ValidIssue {
	var issues []ValidIssue
	add := func(code, msg string) {
		issues = append(issues, ValidIssue{QuestionID: q.ID, Code: code, Message: msg})
	}
	if strings.TrimSpace(q.Prompt) == "" {
		add("missing_prompt", "Question needs a prompt.")
	}
	if q.TimeLimitSeconds < minTimeLimit || q.TimeLimitSeconds > maxTimeLimit {
		add("invalid_timer", "Timer must be between 5 and 240 seconds.")
	}
	if q.PromptMediaRef != nil && strings.TrimSpace(*q.PromptMediaRef) != "" {
		if q.PromptMediaAlt == nil || strings.TrimSpace(*q.PromptMediaAlt) == "" {
			add("missing_media_alt", "Prompt media requires alt text or a caption.")
		}
	}
	opts, err := unmarshalOptions(q.Options)
	if err != nil {
		add("invalid_options", "Options JSON is invalid.")
		return issues
	}
	for _, o := range opts {
		if o.MediaRef != nil && strings.TrimSpace(*o.MediaRef) != "" {
			if o.MediaAlt == nil || strings.TrimSpace(*o.MediaAlt) == "" {
				add("missing_option_media_alt", "Option media requires alt text or a caption.")
				break
			}
		}
	}
	correctCount := 0
	for _, o := range opts {
		if o.IsCorrect {
			correctCount++
		}
	}
	switch q.QuestionType {
	case QTypeMCSingle:
		if len(opts) < minOptions || len(opts) > maxOptions {
			add("invalid_option_count", "Multiple choice needs 2–6 options.")
		}
		for _, o := range opts {
			if strings.TrimSpace(o.Text) == "" {
				add("empty_option", "Every option needs text.")
				break
			}
		}
		if correctCount != 1 {
			add("missing_correct", "Mark exactly one correct answer.")
		}
	case QTypeMCMultiple:
		if len(opts) < minOptions || len(opts) > maxOptions {
			add("invalid_option_count", "Multiple choice needs 2–6 options.")
		}
		for _, o := range opts {
			if strings.TrimSpace(o.Text) == "" {
				add("empty_option", "Every option needs text.")
				break
			}
		}
		if correctCount < 1 {
			add("missing_correct", "Mark at least one correct answer.")
		}
	case QTypeTrueFalse:
		if len(opts) != 2 || correctCount != 1 {
			add("invalid_true_false", "True/False needs two options with exactly one correct.")
		}
	case QTypePoll:
		if len(opts) < minOptions || len(opts) > maxOptions {
			add("invalid_option_count", "Polls need 2–6 options.")
		}
		for _, o := range opts {
			if strings.TrimSpace(o.Text) == "" {
				add("empty_option", "Every option needs text.")
				break
			}
		}
		if correctCount != 0 {
			add("poll_has_correct", "Polls must not mark a correct answer.")
		}
	case QTypeOrdering:
		if len(opts) < 2 {
			add("invalid_option_count", "Ordering needs at least two items.")
		}
		for _, o := range opts {
			if strings.TrimSpace(o.Text) == "" {
				add("empty_option", "Every ordering item needs text.")
				break
			}
		}
		var ca OrderingCorrect
		if len(q.CorrectAnswer) == 0 || json.Unmarshal(q.CorrectAnswer, &ca) != nil || len(ca.Order) != len(opts) {
			add("invalid_order", "Ordering needs a complete correct sequence.")
		}
	case QTypeTypeAnswer:
		var ca TypeAnswerCorrect
		if len(q.CorrectAnswer) == 0 || json.Unmarshal(q.CorrectAnswer, &ca) != nil {
			add("missing_accepted", "Add at least one accepted answer.")
		} else {
			ok := false
			for _, a := range ca.Accepted {
				if strings.TrimSpace(a.Text) != "" {
					ok = true
					break
				}
			}
			if !ok {
				add("missing_accepted", "Add at least one accepted answer.")
			}
		}
	case QTypeNumeric:
		var ca NumericCorrect
		if len(q.CorrectAnswer) == 0 || json.Unmarshal(q.CorrectAnswer, &ca) != nil {
			add("missing_numeric", "Numeric questions need a value and tolerance.")
		} else if ca.Tolerance < 0 {
			add("invalid_tolerance", "Tolerance must be >= 0.")
		}
	case QTypeWordCloud:
		// Open text; no correct answer required.
	}
	return issues
}

// MapBankQuestionType maps course.question_type → quizgame.question_type.
func MapBankQuestionType(bankType string) (string, bool) {
	switch strings.TrimSpace(bankType) {
	case "mc_single":
		return QTypeMCSingle, true
	case "mc_multiple":
		return QTypeMCMultiple, true
	case "true_false":
		return QTypeTrueFalse, true
	case "short_answer":
		return QTypeTypeAnswer, true
	case "numeric":
		return QTypeNumeric, true
	case "ordering":
		return QTypeOrdering, true
	default:
		return "", false
	}
}

// MapToBankQuestionType maps quizgame → course.question_type for push-to-bank.
func MapToBankQuestionType(gameType string) (string, bool) {
	switch gameType {
	case QTypeMCSingle:
		return "mc_single", true
	case QTypeMCMultiple:
		return "mc_multiple", true
	case QTypeTrueFalse:
		return "true_false", true
	case QTypeTypeAnswer, QTypeWordCloud:
		return "short_answer", true
	case QTypeNumeric:
		return "numeric", true
	case QTypeOrdering:
		return "ordering", true
	case QTypePoll:
		return "mc_single", true // closest; no graded correct
	default:
		return "", false
	}
}
