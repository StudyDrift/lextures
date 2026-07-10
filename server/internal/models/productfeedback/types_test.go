package productfeedback

import (
	"strings"
	"testing"
)

func TestValidateMessage(t *testing.T) {
	got, err := ValidateMessage("  hello  ")
	if err != nil || got != "hello" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := ValidateMessage(""); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := ValidateMessage(strings.Repeat("a", MaxMessageLen+1)); err == nil {
		t.Fatal("expected oversize error")
	}
	stripped, err := ValidateMessage("ok\x00\x01there")
	if err != nil || stripped != "okthere" {
		t.Fatalf("strip: %q err %v", stripped, err)
	}
}

func TestNormalizeCategory(t *testing.T) {
	if NormalizeCategory("BUG") != CategoryBug {
		t.Fatal("bug")
	}
	if NormalizeCategory("nope") != CategoryOther {
		t.Fatal("other default")
	}
}

func TestReconcileSource(t *testing.T) {
	if got := ReconcileSource(SourceWeb, "Mozilla/5.0"); got != SourceWeb {
		t.Fatalf("web: %s", got)
	}
	if got := ReconcileSource(SourceIOS, "lextures-android/1.0"); got != SourceAndroid {
		t.Fatalf("android ua wins: %s", got)
	}
	if got := ReconcileSource(SourceWeb, "lextures-ios/2.0"); got != SourceIOS {
		t.Fatalf("ios ua: %s", got)
	}
}

func TestStatusTerminal(t *testing.T) {
	if !StatusResolved.IsTerminal() {
		t.Fatal("resolved terminal")
	}
	if StatusNew.IsTerminal() {
		t.Fatal("new not terminal")
	}
}
