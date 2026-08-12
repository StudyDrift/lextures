// Package marketingcontent owns marketing-content authoring workflows.
package marketingcontent

import (
	"errors"
	"fmt"
	"time"
)

type Status string
type Action string

const (
	StatusDraft     Status = "draft"
	StatusReview    Status = "in_review"
	StatusChanges   Status = "changes_requested"
	StatusPublished Status = "published"
	StatusScheduled Status = "scheduled"
	StatusArchived  Status = "archived"

	ActionSubmit         Action = "submit_review"
	ActionApprove        Action = "approve"
	ActionRequestChanges Action = "request_changes"
	ActionPublish        Action = "publish"
	ActionSchedule       Action = "schedule"
	ActionUnpublish      Action = "unpublish"
	ActionArchive        Action = "archive"
	ActionRestoreDraft   Action = "restore_draft"
)

var (
	ErrInvalidTransition = errors.New("marketingcontent: invalid transition")
	ErrScheduledInPast   = errors.New("marketingcontent: scheduled time must be in the future")
)

var transitions = map[Status]map[Action]Status{
	StatusDraft:     {ActionSubmit: StatusReview, ActionPublish: StatusPublished, ActionSchedule: StatusScheduled, ActionArchive: StatusArchived},
	StatusReview:    {ActionApprove: StatusDraft, ActionRequestChanges: StatusChanges},
	StatusChanges:   {ActionRestoreDraft: StatusDraft},
	StatusPublished: {ActionUnpublish: StatusDraft, ActionArchive: StatusArchived, ActionSchedule: StatusScheduled},
	StatusScheduled: {ActionPublish: StatusPublished, ActionUnpublish: StatusDraft, ActionArchive: StatusArchived},
	StatusArchived:  {ActionRestoreDraft: StatusDraft},
}

func LegalActions(status Status) []Action {
	m := transitions[status]
	out := make([]Action, 0, len(m))
	order := []Action{ActionSubmit, ActionApprove, ActionRequestChanges, ActionPublish, ActionSchedule, ActionUnpublish, ActionArchive, ActionRestoreDraft}
	for _, action := range order {
		if _, ok := m[action]; ok {
			out = append(out, action)
		}
	}
	return out
}

func NextStatus(status Status, action Action, scheduledFor *time.Time, now time.Time) (Status, error) {
	next, ok := transitions[status][action]
	if !ok {
		return "", fmt.Errorf("%w: current status %q; legal actions: %v", ErrInvalidTransition, status, LegalActions(status))
	}
	if action == ActionSchedule && (scheduledFor == nil || !scheduledFor.After(now)) {
		return "", ErrScheduledInPast
	}
	return next, nil
}
