// Package kidmessaging enforces kid-safe inbox rules for managed learners (#775).
//
// A managed learner may only exchange inbox messages with their parent or guardian
// (an active parent_student_links row) and with course teachers that parent approved.
// The rule is symmetric: other people cannot message a managed learner either.
package kidmessaging

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/kidmessaging"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// ErrRecipientNotAllowed is returned when a message would reach or leave a managed learner
// outside their allowed contacts.
var ErrRecipientNotAllowed = errors.New("kidmessaging: recipient not allowed")

// ErrNotCandidate is returned when a parent tries to approve someone who does not teach the learner.
var ErrNotCandidate = errors.New("kidmessaging: contact does not teach this learner")

// Recipient is an allowed contact as shown to a managed learner's Compose.
type Recipient struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"displayName"`
	Kind         string   `json:"kind"`
	CourseTitles []string `json:"courseTitles,omitempty"`
}

// IsManaged reports whether row is a parent-created managed learner account.
func IsManaged(row *user.Row) bool {
	return row != nil && row.AccountType == user.AccountTypeManaged
}

// AllowedContacts returns the learner's parents plus approved course teachers (parents first, deduped).
func AllowedContacts(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]repo.Contact, error) {
	parents, err := repo.ListParentContacts(ctx, pool, studentID)
	if err != nil {
		return nil, err
	}
	teachers, err := repo.ListTeacherCandidates(ctx, pool, studentID)
	if err != nil {
		return nil, err
	}
	return mergeAllowed(parents, teachers), nil
}

// TeacherCandidates returns course teachers the parent could approve, excluding the parents themselves.
func TeacherCandidates(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]repo.Contact, error) {
	parents, err := repo.ListParentContacts(ctx, pool, studentID)
	if err != nil {
		return nil, err
	}
	teachers, err := repo.ListTeacherCandidates(ctx, pool, studentID)
	if err != nil {
		return nil, err
	}
	return excludeParents(parents, teachers), nil
}

// ApproveTeacher lets contactID message the learner. The contact must currently teach one of
// the learner's active courses.
func ApproveTeacher(ctx context.Context, pool *pgxpool.Pool, studentID, contactID, parentID uuid.UUID) error {
	candidates, err := TeacherCandidates(ctx, pool, studentID)
	if err != nil {
		return err
	}
	if !containsUser(candidates, contactID) {
		return ErrNotCandidate
	}
	return repo.ApproveContact(ctx, pool, studentID, contactID, parentID)
}

// RevokeTeacher removes a teacher approval. Missing approvals are not an error.
func RevokeTeacher(ctx context.Context, pool *pgxpool.Pool, studentID, contactID uuid.UUID) error {
	_, err := repo.RevokeContact(ctx, pool, studentID, contactID)
	return err
}

// CheckSend returns ErrRecipientNotAllowed when either side is a managed learner and the
// other side is not one of that learner's allowed contacts.
func CheckSend(ctx context.Context, pool *pgxpool.Pool, sender, recipient *user.Row) error {
	if sender == nil || recipient == nil {
		return ErrRecipientNotAllowed
	}
	senderID, err := uuid.Parse(sender.ID)
	if err != nil {
		return err
	}
	recipientID, err := uuid.Parse(recipient.ID)
	if err != nil {
		return err
	}
	if IsManaged(sender) {
		ok, err := isAllowed(ctx, pool, senderID, recipientID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRecipientNotAllowed
		}
	}
	if IsManaged(recipient) && senderID != recipientID {
		ok, err := isAllowed(ctx, pool, recipientID, senderID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRecipientNotAllowed
		}
	}
	return nil
}

func isAllowed(ctx context.Context, pool *pgxpool.Pool, studentID, otherID uuid.UUID) (bool, error) {
	if studentID == otherID {
		return false, nil
	}
	allowed, err := AllowedContacts(ctx, pool, studentID)
	if err != nil {
		return false, err
	}
	return containsUser(allowed, otherID), nil
}

// ToRecipients maps contacts to the learner-facing shape (no email addresses).
func ToRecipients(contacts []repo.Contact) []Recipient {
	out := make([]Recipient, 0, len(contacts))
	for _, c := range contacts {
		out = append(out, Recipient{
			ID:           c.UserID.String(),
			DisplayName:  c.DisplayName,
			Kind:         c.Kind,
			CourseTitles: c.CourseTitles,
		})
	}
	return out
}

func mergeAllowed(parents, teachers []repo.Contact) []repo.Contact {
	out := make([]repo.Contact, 0, len(parents)+len(teachers))
	seen := make(map[uuid.UUID]struct{}, len(parents)+len(teachers))
	for _, p := range parents {
		if _, ok := seen[p.UserID]; ok {
			continue
		}
		seen[p.UserID] = struct{}{}
		out = append(out, p)
	}
	for _, t := range teachers {
		if !t.Approved {
			continue
		}
		if _, ok := seen[t.UserID]; ok {
			continue
		}
		seen[t.UserID] = struct{}{}
		out = append(out, t)
	}
	return out
}

func excludeParents(parents, teachers []repo.Contact) []repo.Contact {
	out := make([]repo.Contact, 0, len(teachers))
	for _, t := range teachers {
		if containsUser(parents, t.UserID) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func containsUser(list []repo.Contact, id uuid.UUID) bool {
	for _, c := range list {
		if c.UserID == id {
			return true
		}
	}
	return false
}
