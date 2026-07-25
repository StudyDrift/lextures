package adaptivecontent

import (
	"testing"

	"github.com/google/uuid"
)

func TestProfileResultFromRow_Nil(t *testing.T) {
	t.Parallel()
	p := ProfileResultFromRow(nil)
	if p.ProfileSignature != "" {
		t.Fatal("expected empty")
	}
}

func TestEnqueue_SkipsNeutral(t *testing.T) {
	t.Parallel()
	// Neutral signatures return early without touching the pool.
	id, created, err := Enqueue(nil, nil, uuid.Nil, NeutralSignature, 1, PriorityPrewarm)
	if err != nil {
		t.Fatal(err)
	}
	if created || id != uuid.Nil {
		t.Fatal("neutral should not enqueue")
	}
	_, created, err = Enqueue(nil, nil, uuid.Nil, "neutral", 1, PriorityPrewarm)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("case-insensitive neutral should not enqueue")
	}
	_, created, err = Enqueue(nil, nil, uuid.Nil, "", 1, PriorityPrewarm)
	if err != nil || created {
		t.Fatal("empty signature should not enqueue")
	}
}
