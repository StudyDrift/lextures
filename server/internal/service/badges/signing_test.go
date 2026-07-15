package badges

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

func TestAchievementSigningRoundTrip(t *testing.T) {
	key, err := vcsigning.GenerateKey("https://self.lextures.com")
	if err != nil {
		t.Fatal(err)
	}
	subject := map[string]any{
		"id":   "urn:uuid:user:" + uuid.NewString(),
		"type": []string{"AchievementSubject"},
		"name": "Test Learner",
		"achievement": map[string]any{
			"id":          "https://self.lextures.com/achievements/badge/" + uuid.NewString(),
			"type":        []string{"Achievement"},
			"name":        "Algebra — Linear Equations",
			"description": "Mastered solving linear equations",
			"criteria": map[string]any{
				"narrative": "Demonstrated proficiency on aligned assessments.",
			},
		},
	}
	vc, err := vcsigning.SignAchievementCredential(subject, "Lextures", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(vc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	ok, err := vcsigning.VerifyCredential(decoded, key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected verified credential")
	}
	// Tamper
	decoded["issuanceDate"] = "2000-01-01T00:00:00Z"
	ok, err = vcsigning.VerifyCredential(decoded, key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("tampered credential must not verify")
	}
}
