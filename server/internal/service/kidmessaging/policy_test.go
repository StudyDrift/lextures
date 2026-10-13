package kidmessaging

import (
	"testing"

	"github.com/google/uuid"

	repo "github.com/lextures/lextures/server/internal/repos/kidmessaging"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestMergeAllowed_ParentsFirstApprovedTeachersOnly(t *testing.T) {
	p := uuid.New()
	approved := uuid.New()
	pending := uuid.New()
	parents := []repo.Contact{{UserID: p, Kind: repo.KindParent, Approved: true, DisplayName: "Mom"}}
	teachers := []repo.Contact{
		{UserID: p, Kind: repo.KindTeacher, Approved: false, DisplayName: "Mom"},
		{UserID: approved, Kind: repo.KindTeacher, Approved: true, DisplayName: "Ms. A"},
		{UserID: pending, Kind: repo.KindTeacher, Approved: false, DisplayName: "Mr. P"},
	}
	got := mergeAllowed(parents, teachers)
	if len(got) != 2 {
		t.Fatalf("want 2 contacts, got %d: %+v", len(got), got)
	}
	if got[0].UserID != p || got[0].Kind != repo.KindParent {
		t.Fatalf("parent should be first and keep parent kind: %+v", got[0])
	}
	if got[1].UserID != approved {
		t.Fatalf("approved teacher missing: %+v", got[1])
	}
	if containsUser(got, pending) {
		t.Fatal("unapproved teacher must not be allowed")
	}
}

func TestExcludeParents(t *testing.T) {
	p := uuid.New()
	other := uuid.New()
	got := excludeParents(
		[]repo.Contact{{UserID: p}},
		[]repo.Contact{{UserID: p}, {UserID: other}},
	)
	if len(got) != 1 || got[0].UserID != other {
		t.Fatalf("parents should be excluded from teacher candidates: %+v", got)
	}
}

func TestToRecipients_NoEmail(t *testing.T) {
	id := uuid.New()
	got := ToRecipients([]repo.Contact{{UserID: id, DisplayName: "Dad", Email: "dad@example.com", Kind: repo.KindParent}})
	if len(got) != 1 || got[0].ID != id.String() || got[0].DisplayName != "Dad" || got[0].Kind != "parent" {
		t.Fatalf("unexpected recipients: %+v", got)
	}
}

func TestIsManaged(t *testing.T) {
	if IsManaged(nil) {
		t.Fatal("nil row is not managed")
	}
	if !IsManaged(&user.Row{AccountType: user.AccountTypeManaged}) {
		t.Fatal("managed row should be managed")
	}
	if IsManaged(&user.Row{AccountType: user.AccountTypeStandard}) {
		t.Fatal("standard row is not managed")
	}
}

func TestCheckSend_NilRows(t *testing.T) {
	if err := CheckSend(t.Context(), nil, nil, &user.Row{}); err != ErrRecipientNotAllowed {
		t.Fatalf("nil sender: %v", err)
	}
}

func TestCheckSend_StandardUsersSkipLookup(t *testing.T) {
	a := &user.Row{ID: uuid.NewString(), AccountType: user.AccountTypeStandard}
	b := &user.Row{ID: uuid.NewString(), AccountType: user.AccountTypeParent}
	// nil pool is fine: neither side is managed so no query runs.
	if err := CheckSend(t.Context(), nil, a, b); err != nil {
		t.Fatalf("standard → parent should be allowed: %v", err)
	}
}
