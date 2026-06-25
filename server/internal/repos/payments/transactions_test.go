package payments

import (
	"testing"
)

func TestCreateIdempotent_RequiresKey(t *testing.T) {
	_, _, err := CreateIdempotent(t.Context(), nil, CreateTransactionInput{})
	if err == nil {
		t.Fatal("expected error for missing idempotency key")
	}
}
