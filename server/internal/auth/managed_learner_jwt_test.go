package auth

import "testing"

func TestSignManagedLearnerVia_FamilyDevice(t *testing.T) {
	signer := NewJWTSigner("01234567890123456789012345678901")
	actorID := "11111111-1111-4111-8111-111111111111"
	userID := "22222222-2222-4222-8222-222222222222"
	tok, ml, err := signer.SignManagedLearnerVia(actorID, userID, userID+"@managed.lextures.invalid", "", "", ManagedLearnerViaFamilyDevice)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if ml.Via != ManagedLearnerViaFamilyDevice {
		t.Fatalf("via %q", ml.Via)
	}
	got, err := signer.VerifyManagedLearner(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Via != ManagedLearnerViaFamilyDevice || got.ActorID != actorID || got.TargetUserID != userID {
		t.Fatalf("mismatch: %+v", got)
	}

	plain, _, err := signer.SignManagedLearner(actorID, userID, userID+"@managed.lextures.invalid", "", "")
	if err != nil {
		t.Fatalf("sign plain: %v", err)
	}
	gotPlain, err := signer.VerifyManagedLearner(plain)
	if err != nil || gotPlain.Via != "" {
		t.Fatalf("plain Learn-as must have empty via: %+v err %v", gotPlain, err)
	}

	if _, _, err := signer.SignManagedLearnerVia(actorID, userID, userID+"@managed.lextures.invalid", "", "", "bogus"); err == nil {
		t.Fatal("unknown via must be rejected")
	}
}
