package badges

import (
	"strings"
	"testing"
)

func TestValidateHandleFormat(t *testing.T) {
	cases := []struct {
		in   string
		want error
	}{
		{"willden", nil},
		{"ab", ErrInvalidHandle},
		{"a", ErrInvalidHandle},
		{"-abc", ErrInvalidHandle},
		{"abc-", ErrInvalidHandle},
		{"ab--cd", ErrInvalidHandle},
		{"Admin", ErrHandleReserved},
		{"api", ErrHandleReserved},
		{"verify", ErrHandleReserved},
		{"good-handle-1", nil},
		{"u" + strings.Repeat("a", 30) + "x", nil}, // 32
		{"u" + strings.Repeat("a", 31) + "x", ErrInvalidHandle},
		{"HasUpper", nil}, // lowercased: hasupper
		{"with_underscore", ErrInvalidHandle},
		{"", ErrInvalidHandle},
	}
	for _, tc := range cases {
		got := ValidateHandleFormat(tc.in)
		if got != tc.want {
			t.Errorf("ValidateHandleFormat(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Algebra Linear Equations", "algebra-linear-equations"},
		{"  Hello   World  ", "hello-world"},
		{"C++ Basics!", "c-basics"},
		{"", "badge"},
		{"---", "badge"},
		{"Already-slug", "already-slug"},
	}
	for _, tc := range cases {
		if got := Slugify(tc.in); got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidateSlugFormat(t *testing.T) {
	if err := ValidateSlugFormat("algebra-linear-equations"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSlugFormat("Bad_Slug"); err != ErrInvalidSlug {
		t.Fatalf("want ErrInvalidSlug, got %v", err)
	}
	if err := ValidateSlugFormat("-leading"); err != ErrInvalidSlug {
		t.Fatalf("want ErrInvalidSlug, got %v", err)
	}
	if err := ValidateSlugFormat(""); err != ErrInvalidSlug {
		t.Fatalf("want ErrInvalidSlug, got %v", err)
	}
}
