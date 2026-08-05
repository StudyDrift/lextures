package coursechecklist

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// updateCatalogTargets regenerates testdata/catalog_targets.json when set:
//
//	go test ./internal/service/coursechecklist/ -run TestCatalogTargetsFixture -update
var updateCatalogTargets = flag.Bool("update", false, "update golden fixtures")

// catalogTargetRow is one (itemId, route, anchor) emitted by the server catalog (CC.8 FR-18).
type catalogTargetRow struct {
	ItemID string `json:"itemId"`
	Route  string `json:"route"`
	Anchor string `json:"anchor,omitempty"`
}

func collectCatalogTargets(t *testing.T) []catalogTargetRow {
	t.Helper()
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("BuildBuiltinRegistry: %v", err)
	}
	rows := make([]catalogTargetRow, 0, reg.Size())
	for _, it := range reg.List() {
		row := catalogTargetRow{
			ItemID: string(it.ID),
			Route:  it.Target.Route,
			Anchor: it.Target.Anchor,
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ItemID != rows[j].ItemID {
			return rows[i].ItemID < rows[j].ItemID
		}
		if rows[i].Route != rows[j].Route {
			return rows[i].Route < rows[j].Route
		}
		return rows[i].Anchor < rows[j].Anchor
	})
	return rows
}

func TestCatalogTargetsFixture(t *testing.T) {
	rows := collectCatalogTargets(t)
	path := filepath.Join("testdata", "catalog_targets.json")

	if *updateCatalogTargets {
		payload, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		payload = append(payload, '\n')
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("updated %s (%d rows)", path, len(rows))
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (run with -update to generate): %v", path, err)
	}
	var want []catalogTargetRow
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	gotJSON, _ := json.MarshalIndent(rows, "", "  ")
	wantJSON, _ := json.MarshalIndent(want, "", "  ")
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("catalog_targets.json out of date.\nRun: go test ./internal/service/coursechecklist/ -run TestCatalogTargetsFixture -update\n\ngot %d rows, want %d", len(rows), len(want))
	}
}

func TestCatalogTargetAnchorsValidFormat(t *testing.T) {
	rows := collectCatalogTargets(t)
	for _, row := range rows {
		if row.Anchor == "" {
			continue
		}
		if !ItemIDPattern.MatchString(row.Anchor) {
			t.Errorf("item %q anchor %q does not match ItemIDPattern (CC.8 FR-2)", row.ItemID, row.Anchor)
		}
	}
}
