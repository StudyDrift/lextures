package communication

// MessageAction is an interactive button rendered in the inbox for structured messages.
type MessageAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Style string `json:"style,omitempty"` // primary | danger
}

// MessageMetadata holds structured payload for inbox messages (e.g. enrollment invitations).
type MessageMetadata struct {
	Type         string          `json:"type,omitempty"`
	EnrollmentID string          `json:"enrollmentId,omitempty"`
	CourseCode   string          `json:"courseCode,omitempty"`
	CourseTitle  string          `json:"courseTitle,omitempty"`
	Actions      []MessageAction `json:"actions,omitempty"`
	Resolved     *string         `json:"resolved,omitempty"` // approved | declined
}