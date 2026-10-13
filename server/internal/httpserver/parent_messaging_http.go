package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	commodels "github.com/lextures/lextures/server/internal/models/communication"
	"github.com/lextures/lextures/server/internal/repos/communication"
	repokid "github.com/lextures/lextures/server/internal/repos/kidmessaging"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/kidmessaging"
)

// registerParentMessagingRoutes wires the parent's view of a managed learner's inbox and the
// teacher approvals for kid-safe messaging (#775).
func (d Deps) registerParentMessagingRoutes(r chi.Router) {
	r.Get("/api/v1/parent/students/{sid}/messages", d.handleParentStudentMessages())
	r.Get("/api/v1/parent/students/{sid}/messages/{id}", d.handleParentStudentMessage())
	r.Get("/api/v1/parent/students/{sid}/messaging/contacts", d.handleParentStudentMessagingContacts())
	r.Put("/api/v1/parent/students/{sid}/messaging/contacts/{contactId}", d.handleParentStudentMessagingContactApprove())
	r.Delete("/api/v1/parent/students/{sid}/messaging/contacts/{contactId}", d.handleParentStudentMessagingContactRevoke())
}

// parentManagedChildAuth requires a parent viewer with a link to {sid}, and {sid} must be a
// managed learner: only parent-created kid accounts expose their mailbox to the parent.
func (d Deps) parentManagedChildAuth(w http.ResponseWriter, r *http.Request) (parentID, studentID uuid.UUID, ok bool) {
	parentID, orgID, ok := d.requireParentViewer(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	studentID, ok = d.parseStudentIDParam(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if _, ok = d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	row, err := user.FindByID(r.Context(), d.Pool, studentID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load learner.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if !kidmessaging.IsManaged(row) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Messages are only available for learners you manage.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return parentID, studentID, true
}

func redactManagedEmails(msgs []commodels.MailboxMessage) {
	for i := range msgs {
		redactManagedEmail(&msgs[i])
	}
}

func redactManagedEmail(m *commodels.MailboxMessage) {
	if user.IsManagedEmail(m.From.Email) {
		m.From.Email = ""
	}
	if user.IsManagedEmail(m.To) {
		m.To = ""
	}
}

// handleParentStudentMessages is GET /api/v1/parent/students/{sid}/messages?folder=inbox|sent&q=
func (d Deps) handleParentStudentMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, studentID, ok := d.parentManagedChildAuth(w, r)
		if !ok {
			return
		}
		folder := strings.TrimSpace(r.URL.Query().Get("folder"))
		if folder == "" {
			folder = "inbox"
		}
		if folder != "inbox" && folder != "sent" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "folder must be inbox or sent.")
			return
		}
		msgs, err := communication.ListForUser(r.Context(), d.Pool, studentID, folder, r.URL.Query().Get("q"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list messages.")
			return
		}
		if msgs == nil {
			msgs = []commodels.MailboxMessage{}
		}
		redactManagedEmails(msgs)
		writeJSON(w, http.StatusOK, commodels.MailboxListResponse{Messages: msgs})
	}
}

// handleParentStudentMessage is GET /api/v1/parent/students/{sid}/messages/{id} (read-only; does not mark read).
func (d Deps) handleParentStudentMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, studentID, ok := d.parentManagedChildAuth(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid message id.")
			return
		}
		msg, err := communication.GetForUser(r.Context(), d.Pool, studentID, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load message.")
			return
		}
		if msg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		redactManagedEmail(msg)
		writeJSON(w, http.StatusOK, msg)
	}
}

type parentMessagingContactOut struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"displayName"`
	Kind         string   `json:"kind"`
	CourseTitles []string `json:"courseTitles"`
	Approved     bool     `json:"approved"`
}

func toParentMessagingContacts(list []repokid.Contact) []parentMessagingContactOut {
	out := make([]parentMessagingContactOut, 0, len(list))
	for _, c := range list {
		titles := c.CourseTitles
		if titles == nil {
			titles = []string{}
		}
		out = append(out, parentMessagingContactOut{
			ID:           c.UserID.String(),
			DisplayName:  c.DisplayName,
			Kind:         c.Kind,
			CourseTitles: titles,
			Approved:     c.Approved,
		})
	}
	return out
}

func (d Deps) writeParentMessagingContacts(w http.ResponseWriter, r *http.Request, studentID uuid.UUID) {
	parents, err := repokid.ListParentContacts(r.Context(), d.Pool, studentID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load message contacts.")
		return
	}
	teachers, err := kidmessaging.TeacherCandidates(r.Context(), d.Pool, studentID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load message contacts.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"parents":  toParentMessagingContacts(parents),
		"teachers": toParentMessagingContacts(teachers),
	})
}

// handleParentStudentMessagingContacts is GET /api/v1/parent/students/{sid}/messaging/contacts
func (d Deps) handleParentStudentMessagingContacts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, studentID, ok := d.parentManagedChildAuth(w, r)
		if !ok {
			return
		}
		d.writeParentMessagingContacts(w, r, studentID)
	}
}

func parseContactIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "contactId")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid contact id.")
		return uuid.UUID{}, false
	}
	return id, true
}

// handleParentStudentMessagingContactApprove is PUT /api/v1/parent/students/{sid}/messaging/contacts/{contactId}
func (d Deps) handleParentStudentMessagingContactApprove() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, studentID, ok := d.parentManagedChildAuth(w, r)
		if !ok {
			return
		}
		contactID, ok := parseContactIDParam(w, r)
		if !ok {
			return
		}
		if err := kidmessaging.ApproveTeacher(r.Context(), d.Pool, studentID, contactID, parentID); err != nil {
			if errors.Is(err, kidmessaging.ErrNotCandidate) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Only teachers of this learner's courses can be approved.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not approve contact.")
			return
		}
		d.writeParentMessagingContacts(w, r, studentID)
	}
}

// handleParentStudentMessagingContactRevoke is DELETE /api/v1/parent/students/{sid}/messaging/contacts/{contactId}
func (d Deps) handleParentStudentMessagingContactRevoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, studentID, ok := d.parentManagedChildAuth(w, r)
		if !ok {
			return
		}
		contactID, ok := parseContactIDParam(w, r)
		if !ok {
			return
		}
		if err := kidmessaging.RevokeTeacher(r.Context(), d.Pool, studentID, contactID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not remove contact.")
			return
		}
		d.writeParentMessagingContacts(w, r, studentID)
	}
}
