package publicapi

import "time"

// CourseResource is the public API projection of a course.
type CourseResource struct {
	ID          string `json:"id"`
	CourseCode  string `json:"courseCode"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Published   bool   `json:"published"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

// UserResource is the public API projection of a user.
type UserResource struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"displayName,omitempty"`
	Email       *string `json:"email,omitempty"`
	FirstName   *string `json:"firstName,omitempty"`
	LastName    *string `json:"lastName,omitempty"`
}

// EnrollmentResource is the public API projection of an enrollment.
type EnrollmentResource struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Role      string     `json:"role"`
	State     string     `json:"state"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// AssignmentResource is the public API projection of an assignment.
type AssignmentResource struct {
	ID        string     `json:"id"`
	CourseID  string     `json:"courseId"`
	Title     string     `json:"title"`
	DueAt     *time.Time `json:"dueAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// GradeResource is the public API projection of a posted grade cell.
type GradeResource struct {
	CourseID      string     `json:"courseId"`
	StudentUserID string     `json:"studentUserId"`
	ModuleItemID  string     `json:"moduleItemId"`
	PointsEarned  string     `json:"pointsEarned"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}
