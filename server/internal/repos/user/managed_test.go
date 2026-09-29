package user

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestManagedEmail(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")
	got := ManagedEmail(id)
	want := "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee@managed.lextures.invalid"
	if got != want {
		t.Fatalf("ManagedEmail = %q, want %q", got, want)
	}
	if !IsManagedEmail(got) {
		t.Fatal("IsManagedEmail should be true")
	}
	if !IsManagedEmail(strings.ToUpper(got)) {
		t.Fatal("IsManagedEmail should be case-insensitive")
	}
	if IsManagedEmail("guide@system.lextures.invalid") {
		t.Fatal("system email must not match managed")
	}
	if IsManagedEmail("kid@school.edu") {
		t.Fatal("real email must not match managed")
	}
}
