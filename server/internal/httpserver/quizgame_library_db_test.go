package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
)

func TestQuizLibrary_DeepDuplicateIndependence(t *testing.T) {
	ctx := context.Background()
	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	body, _ := json.Marshal(map[string]any{"title": "Fractions practice", "description": "Grade 7"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create kit: %d %s", rr.Code, rr.Body.String())
	}
	var kit map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &kit)
	kitID := kit["id"].(string)

	qBody, _ := json.Marshal(map[string]any{
		"questionType": "mc_single",
		"prompt":       "1/2 + 1/4 = ?",
		"options": []map[string]any{
			{"id": "a", "text": "3/4"},
			{"id": "b", "text": "1/2"},
		},
		"correctAnswer":    "a",
		"timeLimitSeconds": 20,
		"pointsStyle":      "standard",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/questions", bytes.NewReader(qBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create question: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/duplicate", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("duplicate: %d %s", rr.Code, rr.Body.String())
	}
	var copy map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &copy)
	copyID := copy["id"].(string)
	if copyID == kitID {
		t.Fatal("copy should have new id")
	}
	if int(copy["questionCount"].(float64)) != 1 {
		t.Fatalf("expected 1 question on copy, got %v", copy["questionCount"])
	}
	if copy["derivedFromKitId"] != kitID {
		t.Fatalf("expected derivedFromKitId=%s got %v", kitID, copy["derivedFromKitId"])
	}

	qs, err := quizgame.ListQuestions(ctx, pool, cc, kitID)
	if err != nil || len(qs) != 1 {
		t.Fatalf("list original qs: %v len=%d", err, len(qs))
	}
	newPrompt := "CHANGED ORIGINAL"
	_, err = quizgame.PatchQuestion(ctx, pool, cc, kitID, qs[0].ID, quizgame.PatchQuestionInput{
		ExpectedVersion: qs[0].Version,
		Prompt:          &newPrompt,
	})
	if err != nil {
		t.Fatalf("patch original: %v", err)
	}
	copyQs, err := quizgame.ListQuestions(ctx, pool, cc, copyID)
	if err != nil || len(copyQs) != 1 {
		t.Fatalf("list copy qs: %v", err)
	}
	if copyQs[0].Prompt == newPrompt {
		t.Fatal("copy question should not change when original is edited")
	}
	if copyQs[0].Prompt != "1/2 + 1/4 = ?" {
		t.Fatalf("copy prompt=%q", copyQs[0].Prompt)
	}
}

func TestQuizLibrary_TemplateShareImport(t *testing.T) {
	ctx := context.Background()
	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	body, _ := json.Marshal(map[string]any{"title": "Dept review kit"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var kit map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &kit)
	kitID := kit["id"].(string)

	tmplBody, _ := json.Marshal(map[string]any{"scope": "course", "title": "Dept review template"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/save-as-template", bytes.NewReader(tmplBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("save-as-template: %d %s", rr.Code, rr.Body.String())
	}
	var tmpl map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &tmpl)
	if tmpl["isTemplate"] != true {
		t.Fatalf("expected isTemplate true: %v", tmpl)
	}
	tmplID := tmpl["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/live-quizzes/templates?courseCode="+cc, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list templates: %d %s", rr.Code, rr.Body.String())
	}
	var list struct {
		Templates []map[string]any `json:"templates"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	foundSystem, foundOurs := false, false
	for _, tplt := range list.Templates {
		if tplt["id"] == "b1000000-0000-4000-8000-000000000001" {
			foundSystem = true
		}
		if tplt["id"] == tmplID {
			foundOurs = true
		}
	}
	if !foundSystem {
		t.Fatal("expected Exit ticket system template")
	}
	if !foundOurs {
		t.Fatal("expected saved course template")
	}

	createBody, _ := json.Marshal(map[string]any{"targetCourseCode": cc})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/live-quizzes/templates/b1000000-0000-4000-8000-000000000001/create-kit", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create-from-template: %d %s", rr.Code, rr.Body.String())
	}
	var fromTmpl map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &fromTmpl)
	if fromTmpl["isTemplate"] == true {
		t.Fatal("created kit should not be a template")
	}
	if int(fromTmpl["questionCount"].(float64)) < 1 {
		t.Fatal("expected questions copied from starter")
	}

	shareBody, _ := json.Marshal(map[string]any{"granteeType": "org", "permission": "copy"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/shares", bytes.NewReader(shareBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("share: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/shares", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list shares: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/submit-to-catalog", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("catalog submit without flag should 404, got %d", rr.Code)
	}
}

func TestQuizLibrary_CatalogPendingGate(t *testing.T) {
	ctx := context.Background()
	pool, _, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			FFInteractiveQuizzes: true,
			FFIqPublicKitCatalog: true,
		},
	})

	body, _ := json.Marshal(map[string]any{"title": "Public candidate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var kit map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &kit)
	kitID := kit["id"].(string)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/submit-to-catalog", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", rr.Code, rr.Body.String())
	}
	var pending map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &pending)
	if pending["catalogStatus"] != "pending" {
		t.Fatalf("expected pending, got %v", pending["catalogStatus"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/live-quizzes/library?q=Public", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("library: %d %s", rr.Code, rr.Body.String())
	}
	var lib struct {
		Kits []map[string]any `json:"kits"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &lib)
	for _, k := range lib.Kits {
		if k["id"] == kitID {
			t.Fatal("pending kit must not be listed in library catalog")
		}
	}

	_, err := quizgame.SetCatalogStatus(ctx, pool, kitID, "listed")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/live-quizzes/library?q=Public", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("library after approve: %d", rr.Code)
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &lib)
	found := false
	for _, k := range lib.Kits {
		if k["id"] == kitID {
			found = true
		}
	}
	if !found {
		t.Fatal("listed kit should appear in library when public catalog enabled")
	}

	impBody, _ := json.Marshal(map[string]any{"targetCourseCode": cc})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/live-quizzes/library/"+kitID+"/import", bytes.NewReader(impBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("import: %d %s", rr.Code, rr.Body.String())
	}
	var imported map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &imported)
	if imported["id"] == kitID {
		t.Fatal("import must create a new kit id")
	}
	if imported["derivedFromKitId"] != kitID {
		t.Fatalf("derivedFromKitId=%v", imported["derivedFromKitId"])
	}

	_, _ = quizgame.SetCatalogStatus(ctx, pool, kitID, "unlisted")
	still, err := quizgame.Get(ctx, pool, cc, imported["id"].(string))
	if err != nil || still == nil {
		t.Fatalf("imported copy should remain: %v", err)
	}
}
