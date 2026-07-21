package parentlinkinvites

import "testing"

func TestHashToken_Stable(t *testing.T) {
	t.Parallel()
	a := HashToken("abc")
	b := HashToken("abc")
	c := HashToken("xyz")
	if a != b {
		t.Fatal("hash should be stable")
	}
	if a == c {
		t.Fatal("different tokens should differ")
	}
	if len(a) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(a))
	}
}
