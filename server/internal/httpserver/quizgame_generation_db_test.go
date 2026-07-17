package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/service/quizgameai"
)

func TestQuizKitGenerate_FeatureGate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTestWithCfg(t, ctx, "teacher", true, true, func(c *config.Config) {
		c.FFIqAiGeneration = false
	})
	defer pool.Close()

	// Create a kit first.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader([]byte(`{"title":"AI Kit"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("create kit: %d %s", rr.Code, rr.Body.String())
	}
	var kitBody map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &kitBody)
	kitID, _ := kitBody["id"].(string)
	if kitID == "" {
		if kit, ok := kitBody["kit"].(map[string]any); ok {
			kitID, _ = kit["id"].(string)
		}
	}
	if kitID == "" {
		t.Fatalf("no kit id: %s", rr.Body.String())
	}

	body, _ := json.Marshal(map[string]any{
		"sourceType": "topic",
		"sourceRef":  map[string]any{"topic": "photosynthesis grade 8"},
		"params":     map[string]any{"count": 3, "types": []string{"mc_single"}},
	})
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/generate", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when AI generation flag off, got %d %s", rr2.Code, rr2.Body.String())
	}
}

func TestQuizKitGenerate_Validation_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTestWithCfg(t, ctx, "teacher", true, true, func(c *config.Config) {
		c.FFIqAiGeneration = true
	})
	defer pool.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader([]byte(`{"title":"AI Kit 2"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	var kitBody map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &kitBody)
	kitID, _ := kitBody["id"].(string)
	if kitID == "" {
		if kit, ok := kitBody["kit"].(map[string]any); ok {
			kitID, _ = kit["id"].(string)
		}
	}
	if kitID == "" {
		t.Fatalf("create kit failed: %d %s", rr.Code, rr.Body.String())
	}

	// Missing topic → 400
	body, _ := json.Marshal(map[string]any{
		"sourceType": "topic",
		"sourceRef":  map[string]any{},
		"params":     map[string]any{"count": 2},
	})
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/generate", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty topic, got %d %s", rr2.Code, rr2.Body.String())
	}
}

func TestQuizgameai_InsertProvenance_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()
	_ = tok

	// Resolve a user id for Create via an existing enrollment row.
	var createdBy string
	if err := pool.QueryRow(ctx, `
SELECT u.id::text FROM "user".users u
INNER JOIN course.course_enrollments e ON e.user_id = u.id
INNER JOIN course.courses c ON c.id = e.course_id
WHERE c.course_code = $1 LIMIT 1
`, cc).Scan(&createdBy); err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, err := uuid.Parse(createdBy)
	if err != nil {
		t.Fatal(err)
	}
	kit, err := quizgame.Create(ctx, pool, cc, uid, "Provenance kit", "", nil)
	if err != nil || kit == nil {
		t.Fatalf("create kit: %v", err)
	}
	needs := true
	conf := 0.4
	q, err := quizgame.CreateQuestion(ctx, pool, cc, kit.ID, quizgame.CreateQuestionInput{
		QuestionType: quizgame.QTypeTrueFalse,
		Prompt:       "Water boils at 100C at sea level.",
		Options: []quizgame.Option{
			{ID: "true", Text: "True", IsCorrect: true},
			{ID: "false", Text: "False", IsCorrect: false},
		},
		TimeLimitSeconds:     20,
		Source:               quizgame.QuestionSourceAIGenerated,
		NeedsReview:          &needs,
		GenerationConfidence: &conf,
	})
	if err != nil || q == nil {
		t.Fatalf("create q: %v", err)
	}
	if q.Source != quizgame.QuestionSourceAIGenerated {
		t.Fatalf("source: %q", q.Source)
	}
	if !q.NeedsReview {
		t.Fatal("expected needs_review")
	}
	got, err := quizgame.GetQuestion(ctx, pool, cc, kit.ID, q.ID)
	if err != nil || got == nil {
		t.Fatalf("get: %v", err)
	}
	if got.GenerationConfidence == nil {
		t.Fatal("expected confidence")
	}
	if diff := *got.GenerationConfidence - 0.4; diff > 0.001 || diff < -0.001 {
		t.Fatalf("confidence: %v", *got.GenerationConfidence)
	}

	// Malformed drafts drop; valid insert path via filter.
	res := quizgameai.ValidateAndFilter([]quizgameai.DraftQuestion{
		{QuestionType: "mc_single", Prompt: ""},
	}, []string{"mc_single"}, true, "")
	if res.Dropped != 1 || len(res.Inputs) != 0 {
		t.Fatalf("expected drop, got %+v", res)
	}
}
