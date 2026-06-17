package credentials

import (
	"testing"
	"time"

	"github.com/google/uuid"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

func TestBuildAchievementSubject(t *testing.T) {
	uid := uuid.New()
	subject := BuildAchievementSubject(uid, "Ada Lovelace", "Intro to CS", "A great course", "Completed all items.")
	achievement, ok := subject["achievement"].(map[string]any)
	if !ok {
		t.Fatal("expected achievement map")
	}
	if achievement["name"] != "Intro to CS" {
		t.Fatalf("unexpected name: %v", achievement["name"])
	}
}

func TestSignAndVerifyAchievementCredential(t *testing.T) {
	key, err := vcsigning.GenerateKey("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	uid := uuid.New()
	subject := BuildAchievementSubject(uid, "Learner", "Self-Paced 101", "desc", CriteriaNarrativeForSource("course"))
	vc, err := vcsigning.SignAchievementCredential(subject, "Lextures", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := vcsigning.VerifyCredential(vc, key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected valid credential")
	}
	types, ok := vc["type"].([]string)
	if !ok {
		t.Fatalf("unexpected type field: %T", vc["type"])
	}
	foundOB := false
	for _, ty := range types {
		if ty == "OpenBadgeCredential" {
			foundOB = true
		}
	}
	if !foundOB {
		t.Fatal("expected OpenBadgeCredential type")
	}
}

func TestDefaultAchievementNamePath(t *testing.T) {
	got := DefaultAchievementName("path", "Data Science")
	if got != "Data Science — Learning Path" {
		t.Fatalf("got %q", got)
	}
}