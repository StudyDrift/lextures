package board

import (
	"errors"
	"time"
)

// WriteKind classifies a board write for the shared moderation write-gate (VC.7).
type WriteKind string

const (
	WritePost     WriteKind = "post"
	WriteComment  WriteKind = "comment"
	WriteReact    WriteKind = "react"
	WriteArrange  WriteKind = "arrange"
	WriteSync     WriteKind = "sync" // Y.js / WS persist
)

var (
	// ErrBoardLocked is returned when the board is locked and the actor is not a manager.
	ErrBoardLocked = errors.New("board: board is locked")
	// ErrBoardFrozen is returned when posting is frozen and the actor is not a manager.
	ErrBoardFrozen = errors.New("board: posting is frozen")
)

// IsFrozen reports whether frozen_until is set and still in the future.
func (b *Board) IsFrozen(now time.Time) bool {
	if b == nil || b.FrozenUntil == nil {
		return false
	}
	return b.FrozenUntil.After(now)
}

// CheckWriteAllowed enforces lock/freeze for non-managers on all write paths (REST + WS).
// Managers always pass. Freeze only blocks post/comment creation; lock blocks all writes.
func CheckWriteAllowed(b *Board, isManager bool, kind WriteKind, now time.Time) error {
	if b == nil || isManager {
		return nil
	}
	if b.Locked {
		return ErrBoardLocked
	}
	switch kind {
	case WritePost, WriteComment:
		if b.IsFrozen(now) {
			return ErrBoardFrozen
		}
	}
	return nil
}
