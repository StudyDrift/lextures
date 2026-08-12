package marketingcontent

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestTransitionMatrix(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	future := now.Add(time.Hour)
	tests := []struct {
		s    Status
		a    Action
		want Status
		ok   bool
	}{{StatusDraft, ActionSubmit, StatusReview, true}, {StatusReview, ActionApprove, StatusDraft, true}, {StatusReview, ActionRequestChanges, StatusChanges, true}, {StatusChanges, ActionRestoreDraft, StatusDraft, true}, {StatusDraft, ActionPublish, StatusPublished, true}, {StatusPublished, ActionUnpublish, StatusDraft, true}, {StatusPublished, ActionArchive, StatusArchived, true}, {StatusArchived, ActionRestoreDraft, StatusDraft, true}, {StatusPublished, ActionSubmit, "", false}}
	for _, tt := range tests {
		got, err := NextStatus(tt.s, tt.a, &future, now)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("%s/%s got %s,%v", tt.s, tt.a, got, err)
		}
	}
}
func TestScheduleRequiresFuture(t *testing.T) {
	now := time.Now()
	if _, err := NextStatus(StatusDraft, ActionSchedule, nil, now); err == nil {
		t.Fatal("missing schedule accepted")
	}
	past := now.Add(-time.Second)
	if _, err := NextStatus(StatusDraft, ActionSchedule, &past, now); err == nil {
		t.Fatal("past schedule accepted")
	}
}
func TestPreviewTokenScopedAndExpires(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := Service{PreviewSecret: []byte("secret"), Now: func() time.Time { return now }}
	id := uuid.New()
	token, _, err := s.MintPreviewToken(id, 5, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyPreviewToken(token, id, 5); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyPreviewToken(token, uuid.New(), 5); err == nil {
		t.Fatal("wrong article accepted")
	}
	s.Now = func() time.Time { return now.Add(2 * time.Minute) }
	if err := s.VerifyPreviewToken(token, id, 5); err == nil {
		t.Fatal("expired token accepted")
	}
}
