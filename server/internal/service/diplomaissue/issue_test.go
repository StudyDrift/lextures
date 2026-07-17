package diplomaissue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/config"
)

func TestIssue_FeatureDisabled(t *testing.T) {
	_, err := Issue(context.Background(), nil, config.Config{FFDiplomas: false}, IssueParams{
		OrgID:       uuid.New(),
		TemplateID:  uuid.New(),
		UserID:      uuid.New(),
		ConferredAt: time.Now().UTC(),
	})
	if err != ErrFeatureDisabled {
		t.Fatalf("got %v want ErrFeatureDisabled", err)
	}
}

func TestRevoke_FeatureDisabled(t *testing.T) {
	_, err := Revoke(context.Background(), nil, config.Config{}, uuid.New(), "test")
	if err != ErrFeatureDisabled {
		t.Fatalf("got %v want ErrFeatureDisabled", err)
	}
}
