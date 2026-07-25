package course

import (
	"reflect"
	"testing"
)

func TestNormalizeGradeLevels(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		out, ok := NormalizeGradeLevels(nil)
		if !ok || out != nil {
			t.Fatalf("got %v ok=%v", out, ok)
		}
		out, ok = NormalizeGradeLevels([]string{})
		if !ok || out != nil {
			t.Fatalf("got %v ok=%v", out, ok)
		}
	})

	t.Run("dedupe and trim", func(t *testing.T) {
		out, ok := NormalizeGradeLevels([]string{" 5 ", "3", "5", "", "4"})
		if !ok {
			t.Fatal("expected ok")
		}
		want := []string{"5", "3", "4"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got %v want %v", out, want)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, ok := NormalizeGradeLevels([]string{"5", "INVALID"})
		if ok {
			t.Fatal("expected invalid")
		}
	})

	t.Run("bands", func(t *testing.T) {
		out, ok := NormalizeGradeLevels([]string{"K-2", "3-5"})
		if !ok {
			t.Fatal("expected ok")
		}
		want := []string{"K-2", "3-5"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got %v want %v", out, want)
		}
	})
}

func TestValidGradeLevel(t *testing.T) {
	t.Parallel()
	if !ValidGradeLevel("K") || !ValidGradeLevel("12") || !ValidGradeLevel("K-12") {
		t.Fatal("expected valid tokens")
	}
	if ValidGradeLevel("") || ValidGradeLevel("13") || ValidGradeLevel("k") {
		t.Fatal("expected invalid tokens")
	}
}
