package selfpaced

import (
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
)

func TestProgressPercent(t *testing.T) {
	cases := []struct {
		completed, total, want int
	}{
		{0, 0, 0},
		{0, 10, 0},
		{3, 10, 30},
		{1, 3, 33},
		{10, 10, 100},
		{11, 10, 100}, // never exceeds 100
		{5, 0, 0},     // no items
	}
	for _, c := range cases {
		if got := ProgressPercent(c.completed, c.total); got != c.want {
			t.Errorf("ProgressPercent(%d,%d) = %d, want %d", c.completed, c.total, got, c.want)
		}
	}
}

func TestIsCourseComplete(t *testing.T) {
	if IsCourseComplete(0, 0) {
		t.Error("empty course should not be complete")
	}
	if IsCourseComplete(2, 3) {
		t.Error("partial course should not be complete")
	}
	if !IsCourseComplete(3, 3) {
		t.Error("fully completed course should be complete")
	}
}

func mods(specs ...[3]int) []learnerprogress.ModuleProgress {
	out := make([]learnerprogress.ModuleProgress, 0, len(specs))
	for i, s := range specs {
		out = append(out, learnerprogress.ModuleProgress{
			ModuleID:       uuid.New(),
			SortOrder:      i,
			TotalItems:     s[1],
			CompletedItems: s[2],
		})
	}
	return out
}

func TestBuildModuleViews_GatingDisabled(t *testing.T) {
	m := mods([3]int{0, 2, 0}, [3]int{0, 2, 0})
	views := BuildModuleViews(m, false)
	for _, v := range views {
		if v.Locked {
			t.Errorf("module %s should not be locked when gating disabled", v.ModuleID)
		}
	}
}

func TestBuildModuleViews_GatingLocksAfterIncomplete(t *testing.T) {
	// Module 0: 1/2 done (incomplete). Module 1 & 2 should be locked.
	m := mods([3]int{0, 2, 1}, [3]int{0, 2, 0}, [3]int{0, 2, 0})
	views := BuildModuleViews(m, true)
	if views[0].Locked {
		t.Error("first module must never be locked")
	}
	if !views[1].Locked {
		t.Error("module after incomplete module must be locked")
	}
	if !views[2].Locked {
		t.Error("module two after incomplete must stay locked")
	}
}

func TestBuildModuleViews_GatingUnlocksAfterComplete(t *testing.T) {
	// Module 0 complete, module 1 unlocked but incomplete, module 2 locked.
	m := mods([3]int{0, 2, 2}, [3]int{0, 3, 1}, [3]int{0, 2, 0})
	views := BuildModuleViews(m, true)
	if views[0].Locked || views[1].Locked {
		t.Error("completed module and the next module must be unlocked")
	}
	if !views[2].Locked {
		t.Error("module after an incomplete (but unlocked) module must be locked")
	}
	if views[0].ProgressPercent != 100 {
		t.Errorf("module 0 percent = %d, want 100", views[0].ProgressPercent)
	}
}

func TestBuildModuleViews_EmptyModuleNeverBlocks(t *testing.T) {
	// Module 0 has no items (treated complete); module 1 must be unlocked.
	m := mods([3]int{0, 0, 0}, [3]int{0, 2, 0})
	views := BuildModuleViews(m, true)
	if views[1].Locked {
		t.Error("empty module must not block the next module")
	}
}

func TestModuleIsLocked(t *testing.T) {
	m := mods([3]int{0, 2, 1}, [3]int{0, 2, 0})
	if ModuleIsLocked(m, false, m[1].ModuleID) {
		t.Error("gating disabled should never lock")
	}
	if ModuleIsLocked(m, true, m[0].ModuleID) {
		t.Error("first module must not be locked")
	}
	if !ModuleIsLocked(m, true, m[1].ModuleID) {
		t.Error("second module after incomplete must be locked")
	}
	if ModuleIsLocked(m, true, uuid.New()) {
		t.Error("unknown module id should default to unlocked")
	}
}
