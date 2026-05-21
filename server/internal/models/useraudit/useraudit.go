// Package useraudit holds API types for user learning-activity audit (user.user_audit).
package useraudit

// PostCourseContextRequest is the body for POST /api/v1/courses/{course_code}/course-context.
// The path is named "course-context" in Rust to avoid ad blockers on "analytics" style names.
type PostCourseContextRequest struct {
	// Kind is course_visit (no structure item), content_open / content_leave (with structureItemId),
	// or equation_inserted / equation_editor_open (optional structureItemId).
	Kind string `json:"kind"`
	// StructureItemID is required for content_open and content_leave; must be a content_page in this course.
	StructureItemID *string `json:"structureItemId,omitempty"`
}
