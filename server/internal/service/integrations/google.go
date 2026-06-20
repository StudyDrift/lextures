package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ExternalCourse is a class as returned by a provider course-list endpoint.
type ExternalCourse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Section string `json:"section,omitempty"`
}

// ExternalMember is a student or co-teacher from a provider roster.
type ExternalMember struct {
	ExternalUserID string `json:"externalUserId"`
	Email          string `json:"email"`
	FullName       string `json:"fullName"`
	Role           string `json:"role"` // "student" | "teacher"
}

// ExternalAssignment is a piece of coursework from a provider.
type ExternalAssignment struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	MaxPoints   float64    `json:"maxPoints"`
	Attachments []string   `json:"attachments,omitempty"`
}

// ClassroomClient abstracts the Google Classroom REST API so the import pipeline
// can be unit-tested with a stub.
type ClassroomClient interface {
	ListCourses(ctx context.Context, accessToken string) ([]ExternalCourse, error)
	ListMembers(ctx context.Context, accessToken, courseID string) ([]ExternalMember, error)
	ListCourseWork(ctx context.Context, accessToken, courseID string) ([]ExternalAssignment, error)
}

// httpClassroomClient is the production Google Classroom client.
type httpClassroomClient struct {
	http    *http.Client
	baseURL string // overridable in tests; defaults to the Google API base
}

const googleClassroomBase = "https://classroom.googleapis.com/v1"

func (c *httpClassroomClient) base() string {
	if c.baseURL != "" {
		return strings.TrimRight(c.baseURL, "/")
	}
	return googleClassroomBase
}

func (c *httpClassroomClient) get(ctx context.Context, accessToken, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("integrations: classroom %s status %d", path, resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

func (c *httpClassroomClient) ListCourses(ctx context.Context, accessToken string) ([]ExternalCourse, error) {
	var resp struct {
		Courses []struct {
			ID, Name, Section string
		} `json:"courses"`
	}
	if err := c.get(ctx, accessToken, "/courses?courseStates=ACTIVE", &resp); err != nil {
		return nil, err
	}
	out := make([]ExternalCourse, 0, len(resp.Courses))
	for _, c := range resp.Courses {
		out = append(out, ExternalCourse{ID: c.ID, Name: c.Name, Section: c.Section})
	}
	return out, nil
}

func (c *httpClassroomClient) ListMembers(ctx context.Context, accessToken, courseID string) ([]ExternalMember, error) {
	out := make([]ExternalMember, 0)
	var students struct {
		Students []struct {
			ID      string `json:"userId"`
			Profile struct {
				Name struct {
					FullName string `json:"fullName"`
				} `json:"name"`
				EmailAddress string `json:"emailAddress"`
			} `json:"profile"`
		} `json:"students"`
	}
	if err := c.get(ctx, accessToken, "/courses/"+url.PathEscape(courseID)+"/students", &students); err != nil {
		return nil, err
	}
	for _, s := range students.Students {
		out = append(out, ExternalMember{ExternalUserID: s.ID, Email: s.Profile.EmailAddress, FullName: s.Profile.Name.FullName, Role: "student"})
	}
	var teachers struct {
		Teachers []struct {
			ID      string `json:"userId"`
			Profile struct {
				Name struct {
					FullName string `json:"fullName"`
				} `json:"name"`
				EmailAddress string `json:"emailAddress"`
			} `json:"profile"`
		} `json:"teachers"`
	}
	if err := c.get(ctx, accessToken, "/courses/"+url.PathEscape(courseID)+"/teachers", &teachers); err != nil {
		return nil, err
	}
	for _, t := range teachers.Teachers {
		out = append(out, ExternalMember{ExternalUserID: t.ID, Email: t.Profile.EmailAddress, FullName: t.Profile.Name.FullName, Role: "teacher"})
	}
	return out, nil
}

func (c *httpClassroomClient) ListCourseWork(ctx context.Context, accessToken, courseID string) ([]ExternalAssignment, error) {
	var resp struct {
		CourseWork []struct {
			ID        string  `json:"id"`
			Title     string  `json:"title"`
			MaxPoints float64 `json:"maxPoints"`
			DueDate   *struct {
				Year, Month, Day int
			} `json:"dueDate"`
		} `json:"courseWork"`
	}
	if err := c.get(ctx, accessToken, "/courses/"+url.PathEscape(courseID)+"/courseWork", &resp); err != nil {
		return nil, err
	}
	out := make([]ExternalAssignment, 0, len(resp.CourseWork))
	for _, w := range resp.CourseWork {
		a := ExternalAssignment{ID: w.ID, Title: w.Title, MaxPoints: w.MaxPoints}
		if w.DueDate != nil && w.DueDate.Year > 0 {
			due := time.Date(w.DueDate.Year, time.Month(w.DueDate.Month), w.DueDate.Day, 23, 59, 0, 0, time.UTC)
			a.DueDate = &due
		}
		out = append(out, a)
	}
	return out, nil
}

// RosterDiff describes how an imported roster differs from the current Lextures
// enrollment, so an instructor can review before committing (plan 16.4 FR-8).
type RosterDiff struct {
	Added     []ExternalMember `json:"added"`
	Unchanged []ExternalMember `json:"unchanged"`
	Removed   []string         `json:"removed"` // emails present in Lextures but not the source
}

// ComputeRosterDiff is a pure function comparing an external roster against the
// set of emails currently enrolled in the Lextures course. Email comparison is
// case-insensitive. It is the unit-tested core of import preview and sync.
func ComputeRosterDiff(external []ExternalMember, currentEmails []string) RosterDiff {
	current := make(map[string]bool, len(currentEmails))
	for _, e := range currentEmails {
		current[normalizeEmail(e)] = true
	}
	seen := make(map[string]bool, len(external))
	diff := RosterDiff{Added: []ExternalMember{}, Unchanged: []ExternalMember{}, Removed: []string{}}
	for _, m := range external {
		key := normalizeEmail(m.Email)
		if key == "" {
			continue
		}
		seen[key] = true
		if current[key] {
			diff.Unchanged = append(diff.Unchanged, m)
		} else {
			diff.Added = append(diff.Added, m)
		}
	}
	for _, e := range currentEmails {
		key := normalizeEmail(e)
		if key != "" && !seen[key] {
			diff.Removed = append(diff.Removed, e)
		}
	}
	sort.Strings(diff.Removed)
	return diff
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}
