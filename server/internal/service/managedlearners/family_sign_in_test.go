package managedlearners

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeSecret_Pin(t *testing.T) {
	for _, bad := range []string{"", "123", "1234567", "12a4", "12 34"} {
		if _, err := normalizeSecret(SignInMethodPin, bad, nil); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("pin %q: want ErrInvalidInput, got %v", bad, err)
		}
	}
	for _, good := range []string{"1234", " 0000 ", "123456"} {
		got, err := normalizeSecret(SignInMethodPin, good, nil)
		if err != nil {
			t.Fatalf("pin %q: %v", good, err)
		}
		if got != "pin:"+strings.TrimSpace(good) {
			t.Fatalf("pin %q canonical %q", good, got)
		}
	}
}

func TestNormalizeSecret_Picture(t *testing.T) {
	if _, err := normalizeSecret(SignInMethodPicture, "", []string{"cat", "dog"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("too short: %v", err)
	}
	if _, err := normalizeSecret(SignInMethodPicture, "", []string{"cat", "dog", "sun", "sun", "sun", "sun", "sun"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("too long: %v", err)
	}
	if _, err := normalizeSecret(SignInMethodPicture, "", []string{"cat", "dog", "dragon"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown picture: %v", err)
	}
	got, err := normalizeSecret(SignInMethodPicture, "", []string{" Cat", "dog", "sun", "cat"})
	if err != nil {
		t.Fatalf("valid: %v", err)
	}
	if got != "picture:cat,dog,sun,cat" {
		t.Fatalf("canonical %q", got)
	}
	// A PIN and a picture sequence never share a canonical form.
	pin, _ := normalizeSecret(SignInMethodPin, "1234", nil)
	if pin == got {
		t.Fatal("pin and picture canonical forms must differ")
	}
}

func TestNormalizeSecret_UnknownMethod(t *testing.T) {
	if _, err := normalizeSecret("password", "hunter2", nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestDeviceTokenHelpers(t *testing.T) {
	a, err := newDeviceToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := newDeviceToken()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || len(a) < 40 {
		t.Fatalf("tokens should be long and unique: %q %q", a, b)
	}
	h := hashDeviceToken(a)
	if h == a || len(h) != 64 || h == hashDeviceToken(b) {
		t.Fatal("hash must be a stable hex sha256 distinct from the token")
	}
	if normalizeDeviceLabel("  ") != defaultDeviceLabel {
		t.Fatal("blank label should default")
	}
	if got := normalizeDeviceLabel(strings.Repeat("x", 200)); len([]rune(got)) != deviceLabelMaxRunes {
		t.Fatalf("label not truncated: %d", len(got))
	}
}
