// Package studentprogress defines API shapes for per-student progress dashboards (plan 9.1).
package studentprogress

import "time"

// Summary is the top-level progress payload.
type Summary struct {
	EnrollmentID            string     `json:"enrollmentId"`
	CourseID                string     `json:"courseId"`
	StudentUserID           string     `json:"studentUserId"`
	StudentDisplayName      string     `json:"studentDisplayName"`
	AssignmentsSubmittedPct float64    `json:"assignmentsSubmittedPct"`
	ModulesViewedPct        float64    `json:"modulesViewedPct"`
	AvgQuizScore            *float64   `json:"avgQuizScore,omitempty"`
	AvgGradePercent         *float64   `json:"avgGradePercent,omitempty"`
	LastActiveAt            *time.Time `json:"lastActiveAt,omitempty"`
	MissingCount            int        `json:"missingCount"`
	DataAsOf                time.Time  `json:"dataAsOf"`
	StaleMinutes            int        `json:"staleMinutes"`
	CanManageNotes          bool       `json:"canManageNotes"`
}

// MissingItem is overdue work not yet submitted.
type MissingItem struct {
	ItemID      string  `json:"itemId"`
	Title       string  `json:"title"`
	Kind        string  `json:"kind"`
	DueAt       *string `json:"dueAt,omitempty"`
	DaysOverdue int     `json:"daysOverdue"`
	GradeStatus string  `json:"gradeStatus"`
}

// AssignmentRow is one gradable assignment for the student.
type AssignmentRow struct {
	ItemID      string  `json:"itemId"`
	Title       string  `json:"title"`
	DueAt       *string `json:"dueAt,omitempty"`
	SubmittedAt *string `json:"submittedAt,omitempty"`
	Grade       string  `json:"grade"`
	Status      string  `json:"status"`
}

// QuizRow is one quiz attempt summary.
type QuizRow struct {
	AttemptID   string  `json:"attemptId"`
	ItemID      string  `json:"itemId"`
	Title       string  `json:"title"`
	SubmittedAt string  `json:"submittedAt"`
	ScorePercent *float64 `json:"scorePercent,omitempty"`
	TimeSpentSec *int    `json:"timeSpentSec,omitempty"`
}

// ActivityEvent is one timeline entry (day-level grouping done client-side).
type ActivityEvent struct {
	OccurredAt string `json:"occurredAt"`
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
}

// ActivityPage is a paginated activity timeline.
type ActivityPage struct {
	Events     []ActivityEvent `json:"events"`
	NextCursor *string         `json:"nextCursor,omitempty"`
}

// Note is a private instructor note.
type Note struct {
	ID        string `json:"id"`
	AuthorID  string `json:"authorId"`
	NoteText  string `json:"noteText"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ProgressResponse is GET .../progress.
type ProgressResponse struct {
	Summary     Summary         `json:"summary"`
	Missing     []MissingItem   `json:"missing"`
	Assignments []AssignmentRow `json:"assignments"`
	Quizzes     []QuizRow       `json:"quizzes"`
	Notes       []Note          `json:"notes,omitempty"`
}

// CreateNoteRequest is POST .../notes.
type CreateNoteRequest struct {
	NoteText string `json:"noteText"`
}

// UpdateNoteRequest is PUT .../notes/:nid.
type UpdateNoteRequest struct {
	NoteText string `json:"noteText"`
}
