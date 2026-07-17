package quizgame

import "testing"

func TestBoundOrgOverrides_ClampsToPlatform(t *testing.T) {
	platMax := 10
	platKits := 5
	platAI := 20
	plat := PlatformSettings{
		MaxConcurrentGames:        &platMax,
		MaxPlayersPerGame:         100,
		MaxKitsPerCourse:          &platKits,
		RetentionDays:             365,
		GuestJoinPolicy:           GuestJoinTeacherMediated,
		DefaultMode:               DefaultMode,
		DefaultLeaderboardPrivacy: DefaultLeaderboardPrivacy,
		AIGenerationEnabled:       true,
		AIGenerationsPerDay:       &platAI,
	}
	ovMax := 50
	ovPlayers := 500
	ovKits := 20
	ovRet := 999
	ovAI := 100
	open := GuestJoinOpen
	aiOn := true
	ov := OrgOverrides{
		MaxConcurrentGames:  &ovMax,
		MaxPlayersPerGame:   &ovPlayers,
		MaxKitsPerCourse:    &ovKits,
		RetentionDays:       &ovRet,
		GuestJoinPolicy:     &open,
		AIGenerationEnabled: &aiOn,
		AIGenerationsPerDay: &ovAI,
	}
	got, err := BoundOrgOverrides(plat, ov)
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxConcurrentGames == nil || *got.MaxConcurrentGames != 10 {
		t.Fatalf("concurrent=%v want 10", got.MaxConcurrentGames)
	}
	if got.MaxPlayersPerGame == nil || *got.MaxPlayersPerGame != 100 {
		t.Fatalf("players=%v want 100", got.MaxPlayersPerGame)
	}
	if got.MaxKitsPerCourse == nil || *got.MaxKitsPerCourse != 5 {
		t.Fatalf("kits=%v want 5", got.MaxKitsPerCourse)
	}
	if got.RetentionDays == nil || *got.RetentionDays != 365 {
		t.Fatalf("retention=%v want 365", got.RetentionDays)
	}
	if got.GuestJoinPolicy == nil || *got.GuestJoinPolicy != GuestJoinTeacherMediated {
		t.Fatalf("guest=%v", got.GuestJoinPolicy)
	}
	if got.AIGenerationsPerDay == nil || *got.AIGenerationsPerDay != 20 {
		t.Fatalf("ai/day=%v want 20", got.AIGenerationsPerDay)
	}
}

func TestGuestRetentionDays(t *testing.T) {
	if GuestRetentionDays(365) != DefaultGuestRetentionDays {
		t.Fatalf("got %d", GuestRetentionDays(365))
	}
	if GuestRetentionDays(10) != 10 {
		t.Fatalf("got %d", GuestRetentionDays(10))
	}
}

func TestNormalizeGuestJoinPolicy(t *testing.T) {
	if normalizeGuestJoinPolicy("OPEN") != GuestJoinOpen {
		t.Fatal("open")
	}
	if normalizeGuestJoinPolicy("nope") != "" {
		t.Fatal("invalid")
	}
}
