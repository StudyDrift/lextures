package quizgame

import "testing"

func TestValidateNickname(t *testing.T) {
	ok, err := ValidateNickname("  Ada  ")
	if err != nil || ok != "Ada" {
		t.Fatalf("got %q err=%v", ok, err)
	}
	if _, err := ValidateNickname(""); err == nil {
		t.Fatal("empty should fail")
	}
	if _, err := ValidateNickname("this-nickname-is-way-too-long!!"); err == nil {
		t.Fatal("too long should fail")
	}
	if _, err := ValidateNickname("bad@name"); err == nil {
		t.Fatal("charset should fail")
	}
	if _, err := ValidateNickname("OK_Name-1!"); err != nil {
		t.Fatalf("allowed charset failed: %v", err)
	}
}
