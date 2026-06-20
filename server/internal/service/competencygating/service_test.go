package competencygating

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/conditionalrelease"
	crrepo "github.com/lextures/lextures/server/internal/repos/conditionalrelease"
)

func TestModuleComplete_AllItems(t *testing.T) {
	modID := uuid.New()
	item1, item2 := uuid.New(), uuid.New()
	cc := &courseContext{
		moduleReqs: map[uuid.UUID]conditionalrelease.ModuleRequirement{
			modID: {ModuleID: modID, CompletionMode: conditionalrelease.CompletionAllItems},
		},
		itemRules: map[uuid.UUID]conditionalrelease.ItemRule{
			item1: {ItemID: item1, RuleType: conditionalrelease.RuleMustMarkDone},
			item2: {ItemID: item2, RuleType: conditionalrelease.RuleMustMarkDone},
		},
		itemProgress: map[uuid.UUID]conditionalrelease.ItemProgress{
			item1: {ItemID: item1, Status: "complete"},
		},
		moduleItems: map[uuid.UUID][]crrepo.ModuleLeafItem{
			modID: {
				{ItemID: item1, Title: "A"},
				{ItemID: item2, Title: "B"},
			},
		},
	}
	if cc.moduleComplete(modID) {
		t.Fatal("module should be incomplete when only one of two rule items is met")
	}
	cc.itemProgress[item2] = conditionalrelease.ItemProgress{ItemID: item2, Status: "complete"}
	if !cc.moduleComplete(modID) {
		t.Fatal("module should be complete when all rule items are met")
	}
}

func TestModuleComplete_OneItem(t *testing.T) {
	modID := uuid.New()
	item1, item2 := uuid.New(), uuid.New()
	cc := &courseContext{
		moduleReqs: map[uuid.UUID]conditionalrelease.ModuleRequirement{
			modID: {ModuleID: modID, CompletionMode: conditionalrelease.CompletionOneItem},
		},
		itemRules: map[uuid.UUID]conditionalrelease.ItemRule{
			item1: {ItemID: item1, RuleType: conditionalrelease.RuleMustMarkDone},
			item2: {ItemID: item2, RuleType: conditionalrelease.RuleMustMarkDone},
		},
		itemProgress: map[uuid.UUID]conditionalrelease.ItemProgress{
			item2: {ItemID: item2, Status: "complete"},
		},
		moduleItems: map[uuid.UUID][]crrepo.ModuleLeafItem{
			modID: {{ItemID: item1}, {ItemID: item2}},
		},
	}
	if !cc.moduleComplete(modID) {
		t.Fatal("one_item mode should complete when any rule item is met")
	}
}

func TestItemLocked_SequentialOrder(t *testing.T) {
	modID := uuid.New()
	item1, item2, item3 := uuid.New(), uuid.New(), uuid.New()
	cc := &courseContext{
		modules: []crrepo.CourseModule{{ModuleID: modID, Title: "Mod 1"}},
		moduleReqs: map[uuid.UUID]conditionalrelease.ModuleRequirement{
			modID: {ModuleID: modID, CompletionMode: conditionalrelease.CompletionSequentialOrder},
		},
		itemRules: map[uuid.UUID]conditionalrelease.ItemRule{
			item1: {ItemID: item1, RuleType: conditionalrelease.RuleMustMarkDone},
			item2: {ItemID: item2, RuleType: conditionalrelease.RuleMustMarkDone},
			item3: {ItemID: item3, RuleType: conditionalrelease.RuleMustMarkDone},
		},
		itemProgress: map[uuid.UUID]conditionalrelease.ItemProgress{},
		moduleItems: map[uuid.UUID][]crrepo.ModuleLeafItem{
			modID: {
				{ItemID: item1, Title: "Step 1"},
				{ItemID: item2, Title: "Step 2"},
				{ItemID: item3, Title: "Step 3"},
			},
		},
	}
	now := time.Now().UTC()
	locked, reason := cc.itemLocked(item3, now)
	if !locked {
		t.Fatal("item 3 should be locked before item 2 is complete")
	}
	if reason == nil || reason.Code != "sequential_order" {
		t.Fatalf("expected sequential_order reason, got %+v", reason)
	}
	cc.itemProgress[item1] = conditionalrelease.ItemProgress{ItemID: item1, Status: "complete"}
	cc.itemProgress[item2] = conditionalrelease.ItemProgress{ItemID: item2, Status: "complete"}
	locked, _ = cc.itemLocked(item3, now)
	if locked {
		t.Fatal("item 3 should unlock after prior items are complete")
	}
}

func TestModuleUnlocked_Prerequisite(t *testing.T) {
	modA, modB := uuid.New(), uuid.New()
	itemA := uuid.New()
	cc := &courseContext{
		modules: []crrepo.CourseModule{
			{ModuleID: modA, Title: "Module A"},
			{ModuleID: modB, Title: "Module B"},
		},
		moduleReqs: map[uuid.UUID]conditionalrelease.ModuleRequirement{
			modA: {ModuleID: modA, CompletionMode: conditionalrelease.CompletionAllItems},
			modB: {ModuleID: modB, PrerequisiteIDs: []uuid.UUID{modA}},
		},
		itemRules: map[uuid.UUID]conditionalrelease.ItemRule{
			itemA: {ItemID: itemA, RuleType: conditionalrelease.RuleMustMarkDone},
		},
		itemProgress: map[uuid.UUID]conditionalrelease.ItemProgress{},
		moduleItems: map[uuid.UUID][]crrepo.ModuleLeafItem{
			modA: {{ItemID: itemA, Title: "Lesson A"}},
			modB: {},
		},
	}
	now := time.Now().UTC()
	unlocked, reason := cc.moduleUnlocked(modB, now)
	if unlocked {
		t.Fatal("module B should be locked until module A is complete")
	}
	if reason == nil || reason.Code != "module_prerequisite" {
		t.Fatalf("expected module_prerequisite reason, got %+v", reason)
	}
	cc.itemProgress[itemA] = conditionalrelease.ItemProgress{ItemID: itemA, Status: "complete"}
	unlocked, reason = cc.moduleUnlocked(modB, now)
	if !unlocked {
		t.Fatalf("module B should unlock after module A is complete, reason=%+v", reason)
	}
}

func TestModuleUnlocked_UnlockDate(t *testing.T) {
	modID := uuid.New()
	future := time.Now().UTC().Add(24 * time.Hour)
	cc := &courseContext{
		moduleReqs: map[uuid.UUID]conditionalrelease.ModuleRequirement{
			modID: {ModuleID: modID, UnlockAt: &future},
		},
		moduleItems: map[uuid.UUID][]crrepo.ModuleLeafItem{modID: {}},
	}
	unlocked, reason := cc.moduleUnlocked(modID, time.Now().UTC())
	if unlocked {
		t.Fatal("module should be locked before unlock date")
	}
	if reason == nil || reason.Code != "unlock_date" {
		t.Fatalf("expected unlock_date reason, got %+v", reason)
	}
}

func TestService_Health(t *testing.T) {
	s := New(nil)
	got, err := s.Health(t.Context())
	if err != nil || got != "competencygating:ok" {
		t.Fatalf("Health() = %q, %v", got, err)
	}
}
