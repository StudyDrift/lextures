package coursechecklist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type helpCatalogFile struct {
	Version int `json:"version"`
	Items   map[string]struct {
		ItemID        string   `json:"itemId"`
		HelpRef       string   `json:"helpRef"`
		Title         string   `json:"title"`
		What          string   `json:"what"`
		Why           string   `json:"why"`
		How           string   `json:"how"`
		WhenToDismiss string   `json:"whenToDismiss"`
		Sources       []string `json:"sources"`
	} `json:"items"`
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for d := wd; d != "/" && d != "."; d = filepath.Dir(d) {
		if st, err := os.Stat(filepath.Join(d, "docs", "help", "course-checklist", "items.json")); err == nil && !st.IsDir() {
			return d
		}
	}
	t.Fatal("could not locate docs/help/course-checklist/items.json from working directory")
	return ""
}

// TestHelpRefResolves asserts every registry HelpRef has four-part help content (CC.10 FR-1/FR-2, AC-1).
func TestHelpRefResolves(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("BuildBuiltinRegistry: %v", err)
	}
	root := findRepoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "help", "course-checklist", "items.json"))
	if err != nil {
		t.Fatalf("read help catalog: %v", err)
	}
	var cat helpCatalogFile
	if err := json.Unmarshal(raw, &cat); err != nil {
		t.Fatalf("parse help catalog: %v", err)
	}
	if len(cat.Items) == 0 {
		t.Fatal("help catalog is empty")
	}

	for _, it := range reg.List() {
		ref := strings.TrimSpace(it.HelpRef)
		if ref == "" {
			t.Errorf("item %q: empty HelpRef", it.ID)
			continue
		}
		entry, ok := cat.Items[ref]
		if !ok {
			t.Errorf("item %q: HelpRef %q has no help content (dangling reference)", it.ID, ref)
			continue
		}
		if entry.ItemID != string(it.ID) {
			t.Errorf("item %q: help entry itemId=%q mismatch", it.ID, entry.ItemID)
		}
		for _, field := range []struct {
			name, val string
		}{
			{"what", entry.What},
			{"why", entry.Why},
			{"how", entry.How},
			{"whenToDismiss", entry.WhenToDismiss},
		} {
			if strings.TrimSpace(field.val) == "" {
				t.Errorf("item %q help %q: empty %s", it.ID, ref, field.name)
			}
		}
	}

	// No orphan help entries for unknown item IDs (optional but keeps catalog tight).
	for ref, entry := range cat.Items {
		if reg.Get(ItemID(entry.ItemID)) == nil {
			t.Errorf("help catalog orphan %q itemId=%q (not in registry)", ref, entry.ItemID)
		}
	}
}

// TestActionsKnownKinds validates action declarations on the four assist items (CC.10 FR-5).
func TestActionsKnownKinds(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("BuildBuiltinRegistry: %v", err)
	}
	want := map[ItemID]ActionKind{
		ItemOutcomesAssessmentMapping: ActionKindSuggestOutcomeMappings,
		ItemFeedbackRubricsOnHighStakes: ActionKindBuildRubricAI,
		ItemOrientationWelcomeMessage: ActionKindDraftWelcome,
		ItemA11yImageAltText:          ActionKindSuggestAltText,
	}
	for id, kind := range want {
		it := reg.Get(id)
		if it == nil {
			t.Fatalf("missing item %q", id)
		}
		if it.Action == nil {
			t.Errorf("item %q: expected action %q", id, kind)
			continue
		}
		if it.Action.Kind != kind {
			t.Errorf("item %q: action kind=%q want %q", id, it.Action.Kind, kind)
		}
		if !it.Action.RequiresAI {
			t.Errorf("item %q: RequiresAI should be true", id)
		}
	}
}
