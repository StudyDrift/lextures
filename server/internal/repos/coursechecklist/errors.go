package coursechecklist

import "errors"

var (
	// ErrInvalidReason is returned when dismiss_reason is not in the allow-list.
	ErrInvalidReason = errors.New("coursechecklist: invalid dismiss reason")
	// ErrNoteTooLong is returned when dismiss_note exceeds MaxDismissNoteLen.
	ErrNoteTooLong = errors.New("coursechecklist: dismiss note too long")
	// ErrPayloadTooLarge is returned when a snapshot payload exceeds 256 KiB even after truncation.
	ErrPayloadTooLarge = errors.New("coursechecklist: snapshot payload too large")
)
