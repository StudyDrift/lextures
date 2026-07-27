package contenttools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	ctctx "github.com/lextures/lextures/server/internal/service/contenttools/context"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/ask_questions"
)

func init() {
	RegisterActionHandler(ask_questions.ID, "ask", handleAskQuestionsAsk)
	RegisterActionHandler(ask_questions.ID, "clear", handleAskQuestionsClear)
}

func handleAskQuestionsClear(ctx ActionContext) (*ActionResult, error) {
	if ctx.Pool != nil {
		settings, err := ctrepo.GetSettings(ctx.Ctx, ctx.Pool, ctx.CourseID)
		if err != nil {
			return nil, err
		}
		m := MustDefault().Get(ask_questions.ID)
		if m == nil || !m.AllowsSelfReset {
			return nil, fmt.Errorf("this tool does not allow clearing the conversation")
		}
		if settings == nil || !settings.StudentResetAllowed {
			return nil, fmt.Errorf("self-clear is not enabled for this course")
		}
	}
	st := ask_questions.EmptyState()
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveAskQuestionsTurn("clear")
	return &ActionResult{
		Result:     map[string]any{"cleared": true},
		StatePatch: patch,
		Status:     StatusInProgress,
	}, nil
}

func handleAskQuestionsAsk(ctx ActionContext) (*ActionResult, error) {
	cfg := ask_questions.ParseConfig(ctx.ConfigJSON)
	st := ask_questions.ParseState(ctx.StateJSON)

	var in struct {
		Question string `json:"question"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid ask input: %w", err)
		}
	}
	question := strings.TrimSpace(in.Question)
	if question == "" {
		question = strings.TrimSpace(st.Draft)
	}
	if question == "" {
		return nil, fmt.Errorf("question is required")
	}
	if len(question) > 4000 {
		return nil, fmt.Errorf("question is too long (max 4000 characters)")
	}

	now := time.Now().UTC()
	if ask_questions.QuestionsRemaining(st, cfg.MaxQuestionsPerDay, now) <= 0 {
		ObserveAskQuestionsTurn("rate_limited")
		resetAt := now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		return &ActionResult{
			Result: map[string]any{
				"error":           "rate_limited",
				"message":         "You've reached today's question limit for this activity. Try again after the daily reset.",
				"resetAt":         resetAt.Format(time.RFC3339),
				"maxPerDay":       cfg.MaxQuestionsPerDay,
				"questionsLeft":   0,
				"preserveInput":   true,
			},
		}, nil
	}

	// Content filter (CT.8) before egress.
	screen := ScreenFreeText(question, FilterActionFlag, true)
	if screen.Action == FilterActionBlock {
		ObserveAskQuestionsTurn("filtered")
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"preserveInput": true,
			},
		}, nil
	}
	if screen.Crisis {
		ObserveAskQuestionsTurn("crisis")
		// Still allow a supportive redirect path without calling the model for crisis content.
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"crisis":        true,
				"preserveInput": true,
			},
		}, nil
	}

	if ctx.Pool == nil || ctx.Completer == nil {
		return nil, fmt.Errorf("ask action requires AI runtime deps")
	}

	taskPrompt := ask_questions.BuildTaskPrompt(cfg, ctx.ReadingLevel)
	if st.Summary != "" {
		taskPrompt += "\nPrior conversation summary:\n" + st.Summary + "\n"
	}

	call, err := ctctx.RunMediatedCall(ctx.Ctx, ctx.Pool, ctctx.CallOpts{
		InstanceID:  ctx.InstanceID,
		CourseID:    ctx.CourseID,
		OrgID:       ctx.OrgID,
		UserID:      ctx.PrincipalID,
		ToolID:      ask_questions.ID,
		FeatureID:   ask_questions.FeatureID,
		TaskPrompt:  taskPrompt,
		LearnerText: question,
		Model:       ctx.Model,
		Completer:   ctx.Completer,
		GatewayCfg:  ctx.GatewayCfg,
		BuildOpts: ctctx.BuildOpts{
			Query:         question,
			EnqueueIngest: true,
			PinnedNotes:   cfg.GroundingNotes,
			ConfigLinks:   cfg.ExtraSourceURLs,
			TokenBudget:   ctctx.DefaultRequestContextTokens,
		},
		MaxTokens: 800,
	})
	if err != nil {
		var ge *ctctx.GatewayError
		var be *ctctx.BudgetError
		switch {
		case errors.As(err, &ge):
			ObserveAskQuestionsTurn("gateway_denied")
			code := "opt_out"
			if ge.Reason == string(aigateway.BlockCoppaAI) {
				code = "coppa"
			}
			return &ActionResult{
				Result: map[string]any{
					"error":         code,
					"message":       ge.Message,
					"preserveInput": true,
					"askInstructor": true,
				},
			}, nil
		case errors.As(err, &be):
			ObserveAskQuestionsTurn("budget")
			return &ActionResult{
				Result: map[string]any{
					"error":         "budget",
					"message":       be.Message,
					"preserveInput": true,
				},
			}, nil
		case errors.Is(err, ctctx.ErrProviderUnavailable):
			ObserveAskQuestionsTurn("provider_error")
			return &ActionResult{
				Result: map[string]any{
					"error":         "provider_unavailable",
					"message":       "The assistant is temporarily unavailable. Your question was not lost — try again.",
					"preserveInput": true,
				},
			}, nil
		default:
			ObserveAskQuestionsTurn("provider_error")
			return &ActionResult{
				Result: map[string]any{
					"error":         "provider_unavailable",
					"message":       "The assistant is temporarily unavailable. Your question was not lost — try again.",
					"preserveInput": true,
				},
			}, nil
		}
	}

	packCites := make([]ask_questions.Citation, 0, len(call.Citations))
	for _, c := range call.Citations {
		packCites = append(packCites, ask_questions.Citation{
			Kind:  string(c.Kind),
			ID:    c.ID,
			Title: c.Title,
			URL:   c.URL,
		})
	}
	fromText, dropped := ask_questions.CitationsFromText(call.Text, packCites)
	if dropped > 0 {
		ObserveAskCitationsDropped(dropped)
	}
	cites := ask_questions.MergeCitationLists(fromText, packCites, cfg.ShowCitations)

	createdAt := now.Format(time.RFC3339)
	userTurn := ask_questions.Turn{
		ID:        ask_questions.NewTurnID(),
		Role:      "user",
		Text:      question,
		CreatedAt: createdAt,
	}
	tok := call.Usage.TotalTokens
	assistantTurn := ask_questions.Turn{
		ID:        ask_questions.NewTurnID(),
		Role:      "assistant",
		Text:      call.Text,
		Citations: cites,
		CreatedAt: createdAt,
		Tokens:    &tok,
	}

	ask_questions.IncrementAskedToday(&st, now)
	ask_questions.AppendTurns(&st, userTurn, assistantTurn, cfg.MaxTurns)
	st.Draft = ""

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveAskQuestionsTurn("ok")
	left := ask_questions.QuestionsRemaining(st, cfg.MaxQuestionsPerDay, now)
	return &ActionResult{
		Result: map[string]any{
			"turn":          assistantTurn,
			"questionsLeft": left,
			"citationCount": len(cites),
		},
		StatePatch: patch,
		Status:     StatusInProgress,
	}, nil
}
