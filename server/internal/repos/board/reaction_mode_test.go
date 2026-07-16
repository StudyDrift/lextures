package board

import "testing"

func TestNormalizeReactionMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"none", ReactionModeNone, false},
		{"LIKE", ReactionModeLike, false},
		{" vote ", ReactionModeVote, false},
		{"star", ReactionModeStar, false},
		{"grade", ReactionModeGrade, false},
		{"emoji", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		got, err := NormalizeReactionMode(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("NormalizeReactionMode(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("NormalizeReactionMode(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestModeToKind(t *testing.T) {
	t.Parallel()
	if ModeToKind(ReactionModeNone) != "" {
		t.Fatal("none should map to empty kind")
	}
	if ModeToKind(ReactionModeLike) != ReactionKindLike {
		t.Fatal("like kind")
	}
	if ModeToKind(ReactionModeStar) != ReactionKindStar {
		t.Fatal("star kind")
	}
}

func TestEngagementScore(t *testing.T) {
	t.Parallel()
	avg := 4.0
	grade := 92.0
	if EngagementScore(PostEngagement{ReactionCount: 5}, ReactionModeLike) != 5 {
		t.Fatal("like score")
	}
	if EngagementScore(PostEngagement{AvgStars: &avg, StarCount: 3}, ReactionModeStar) != 4003 {
		t.Fatalf("star score = %v", EngagementScore(PostEngagement{AvgStars: &avg, StarCount: 3}, ReactionModeStar))
	}
	if EngagementScore(PostEngagement{Grade: &grade}, ReactionModeGrade) != 92 {
		t.Fatal("grade score")
	}
}

func TestValidateReactionKindValue(t *testing.T) {
	t.Parallel()
	if err := validateReactionKindValue(ReactionKindLike, nil); err != nil {
		t.Fatal(err)
	}
	v := 3.0
	if err := validateReactionKindValue(ReactionKindLike, &v); err == nil {
		t.Fatal("like must reject value")
	}
	if err := validateReactionKindValue(ReactionKindStar, &v); err != nil {
		t.Fatal(err)
	}
	bad := 6.0
	if err := validateReactionKindValue(ReactionKindStar, &bad); err == nil {
		t.Fatal("star must reject out of range")
	}
	if err := validateReactionKindValue(ReactionKindGrade, nil); err == nil {
		t.Fatal("grade requires value")
	}
}
