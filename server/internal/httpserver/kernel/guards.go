package kernel

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

// ErrWritten means a wrapped helper already wrote the HTTP response. The
// toolkit must not write a second body.
var ErrWritten = errors.New("kernel: response already written")

// Access is the adapter httpserver implements around Deps. Guards call these
// methods so they wrap requireCourseAccess / meUserID rather than reimplement
// them (FR-5). Returning ok=false means the adapter already wrote the error.
type Access interface {
	Authenticate(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool)
	RequireCourseAccess(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, ok bool)
	UserHasPermission(ctx context.Context, userID uuid.UUID, perm string) (bool, error)
	LookupCourseID(ctx context.Context, courseCode string) (*uuid.UUID, error)
}

// Ctx is the per-request toolkit context. Identity fields are filled by guards.
type Ctx struct {
	context.Context
	W          http.ResponseWriter
	R          *http.Request
	Access     Access
	Viewer     uuid.UUID
	CourseCode string
	CourseID   uuid.UUID
	status     int
	public     bool
}

// Status overrides the success status for this request (e.g. 201, 204).
func (c *Ctx) Status(code int) {
	if c != nil {
		c.status = code
	}
}

// Guard authorises a request. Zero value is not public: handler wrappers
// replace it with Authenticated() and count the route as unguarded (FR-6).
type Guard struct {
	name   string
	public bool
	run    func(*Ctx) error
}

// Name is the declared guard identifier (for the FR-11 ratchet).
func (g Guard) Name() string { return g.name }

// Public reports whether this guard is the explicit Public() marker.
func (g Guard) Public() bool { return g.public }

// Check runs the guard. A zero Guard returns an error so it cannot silently
// authorise a request if a wrapper forgets to substitute Authenticated().
func (g Guard) Check(c *Ctx) error {
	if g.run == nil {
		return Internal("Server misconfiguration.", errors.New("kernel: nil guard"))
	}
	return g.run(c)
}

func newGuard(name string, public bool, run func(*Ctx) error) Guard {
	return Guard{name: name, public: public, run: run}
}

// Authenticated requires a signed-in user via Access.Authenticate (meUserID).
func Authenticated() Guard {
	return newGuard("Authenticated", false, func(c *Ctx) error {
		if c.Access == nil {
			return Internal("Server misconfiguration.", errors.New("kernel: nil Access"))
		}
		id, ok := c.Access.Authenticate(c.W, c.R)
		if !ok {
			return ErrWritten
		}
		c.Viewer = id
		c.Context = c.R.Context()
		return nil
	})
}

// RequireCourseAccess wraps the existing requireCourseAccess helper.
func RequireCourseAccess() Guard {
	return newGuard("RequireCourseAccess", false, func(c *Ctx) error {
		if err := loadCourseAccess(c); err != nil {
			return err
		}
		return loadCourseID(c)
	})
}

// RequireCoursePermission requires course access plus course:{code}:{action}.
// denyMessage must match the hand-rolled handler when converting an endpoint.
func RequireCoursePermission(action, denyMessage string) Guard {
	return newGuard("RequireCoursePermission:"+action, false, func(c *Ctx) error {
		if err := loadCourseAccess(c); err != nil {
			return err
		}
		if err := loadCourseID(c); err != nil {
			return err
		}
		perm := "course:" + c.CourseCode + ":" + action
		has, err := c.Access.UserHasPermission(c.R.Context(), c.Viewer, perm)
		if err != nil {
			return Internal("Failed to verify permissions.", err)
		}
		if !has {
			msg := denyMessage
			if msg == "" {
				msg = "You do not have permission for this action."
			}
			return Forbidden(msg)
		}
		return nil
	})
}

// Public marks a route as intentionally unauthenticated. It is counted as
// unguarded by the FR-11 ratchet.
func Public() Guard {
	return newGuard("Public", true, func(c *Ctx) error {
		c.public = true
		return nil
	})
}

func loadCourseAccess(c *Ctx) error {
	if c.Access == nil {
		return Internal("Server misconfiguration.", errors.New("kernel: nil Access"))
	}
	code, viewer, ok := c.Access.RequireCourseAccess(c.W, c.R)
	if !ok {
		return ErrWritten
	}
	c.CourseCode = code
	c.Viewer = viewer
	c.Context = c.R.Context()
	return nil
}

func loadCourseID(c *Ctx) error {
	if c.Access == nil {
		return Internal("Server misconfiguration.", errors.New("kernel: nil Access"))
	}
	id, err := c.Access.LookupCourseID(c.R.Context(), c.CourseCode)
	if err != nil {
		return Internal("Failed to load course.", err)
	}
	if id == nil {
		return NotFound("Course not found.")
	}
	c.CourseID = *id
	return nil
}
