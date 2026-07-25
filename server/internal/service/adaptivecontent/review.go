package adaptivecontent

import (
	"errors"
	"strings"
	"time"
)

// Review event types written to adaptive_content_events (AC.5).
// Note: EventVariantRejected ("variant_rejected") is already defined in generate.go for gate rejects.
const (
	EventVariantApproved          = "variant_approved"
	EventVariantRejectedByReview  = "variant_rejected_by_instructor"
	EventVariantEdited            = "variant_edited"
	EventVariantRevoked           = "variant_revoked"
	EventVariantsBulk             = "variants_bulk"
)

// Review-related sentinel errors.
var (
	ErrVariantNotPending       = errors.New("variant is not pending review")
	ErrVariantNotRevocable     = errors.New("only approved or auto-served variants can be revoked")
	ErrHardKeyTermFailure      = errors.New("cannot approve a variant that is missing a required key term")
	ErrGateFailedNoOverride    = errors.New("variant failed fidelity or safety gates; set overrideGate to force-approve (key-term hard failures cannot be overridden)")
	ErrInvalidReviewAction     = errors.New("action must be approve or reject")
	ErrEmptyVariantMarkdown    = errors.New("variant markdown cannot be empty")
	ErrBulkEmpty               = errors.New("variantIds must not be empty")
	ErrBulkTooLarge            = errors.New("bulk actions are limited to 100 variants")
)

// MaxBulkReviewVariants is the cap for bulk approve/reject.
const MaxBulkReviewVariants = 100

// HasHardKeyTermFailure reports whether safety/fidelity flags include a missing must-appear term.
func HasHardKeyTermFailure(flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "missing_key_term:") {
			return true
		}
	}
	return false
}

// ValidateBulkAction checks bulk review request bounds.
func ValidateBulkAction(action string, n int) error {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve", "reject":
	default:
		return ErrInvalidReviewAction
	}
	if n == 0 {
		return ErrBulkEmpty
	}
	if n > MaxBulkReviewVariants {
		return ErrBulkTooLarge
	}
	return nil
}

// CanApproveStatus reports whether the current status may transition to approved via review.
func CanApproveStatus(status string) bool {
	switch status {
	case "pending_review", "rejected", "draft":
		return true
	default:
		return false
	}
}

// CanRejectStatus reports whether the current status may transition to rejected via review.
func CanRejectStatus(status string) bool {
	switch status {
	case "pending_review", "approved", "auto_served", "draft":
		return true
	default:
		return false
	}
}

// CanRevokeStatus reports whether the variant can be revoked to superseded.
func CanRevokeStatus(status string) bool {
	return status == "approved" || status == "auto_served"
}

// ReviewNotePtr returns a trimmed optional note pointer (nil when empty).
func ReviewNotePtr(note string) *string {
	n := strings.TrimSpace(note)
	if n == "" {
		return nil
	}
	return &n
}

// TimeInQueueMs approximates time-in-queue from created_at to now.
func TimeInQueueMs(createdAt time.Time) float64 {
	if createdAt.IsZero() {
		return 0
	}
	d := time.Since(createdAt)
	if d < 0 {
		return 0
	}
	return float64(d.Milliseconds())
}
