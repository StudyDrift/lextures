package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/conferences"
	"github.com/lextures/lextures/server/internal/repos/orgunit"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/icsgenerator"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func (d Deps) conferenceSchedulingEnabled(w http.ResponseWriter) bool {
	if !d.Config.FFConferenceScheduling {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
			"Conference scheduling is not enabled.")
		return false
	}
	return true
}

type createConferenceAvailabilityBody struct {
	SchoolID     string  `json:"schoolId"`
	Date         string  `json:"date"`
	WindowStart  string  `json:"windowStart"`
	WindowEnd    string  `json:"windowEnd"`
	SlotDuration int     `json:"slotDuration"`
	GapDuration  int     `json:"gapDuration"`
	Location     *string `json:"location"`
	VideoLink    *string `json:"videoLink"`
}

// handleCreateConferenceAvailability — POST /api/v1/teachers/{teacherId}/conference-availability
func (d Deps) handleCreateConferenceAvailability() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		teacherID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "teacherId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid teacher id.")
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if actorID != teacherID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You can only set your own availability.")
			return
		}

		var body createConferenceAvailabilityBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		schoolID, err := uuid.Parse(strings.TrimSpace(body.SchoolID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid school id.")
			return
		}
		if strings.TrimSpace(body.Date) == "" || strings.TrimSpace(body.WindowStart) == "" || strings.TrimSpace(body.WindowEnd) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "date, windowStart, and windowEnd are required.")
			return
		}
		if body.SlotDuration == 0 {
			body.SlotDuration = 15
		}
		if !conferences.AllowedSlotDurations[body.SlotDuration] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "slotDuration must be 5, 10, 15, 20, or 30.")
			return
		}

		av, slots, err := conferences.CreateAvailability(
			r.Context(), d.Pool, teacherID, schoolID,
			body.Date, body.WindowStart, body.WindowEnd,
			body.SlotDuration, body.GapDuration,
			body.Location, body.VideoLink,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create availability.")
			return
		}
		if slots == nil {
			slots = []*conferences.Slot{}
		}
		writeJSON(w, http.StatusCreated, map[string]any{"availability": av, "slots": slots})
	}
}

// handleListConferenceSlots — GET /api/v1/teachers/{teacherId}/conference-slots?date=YYYY-MM-DD
func (d Deps) handleListConferenceSlots() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		teacherID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "teacherId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid teacher id.")
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		date := strings.TrimSpace(r.URL.Query().Get("date"))
		if date == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "date query parameter is required.")
			return
		}

		slots, av, err := conferences.ListSlotsByTeacherDate(r.Context(), d.Pool, teacherID, date)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load slots.")
			return
		}
		if slots == nil {
			slots = []*conferences.Slot{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"availability": av, "slots": slots})
	}
}

type bookConferenceSlotBody struct {
	StudentID string `json:"studentId"`
}

// handleBookConferenceSlot — POST /api/v1/conference-slots/{slotId}/book
func (d Deps) handleBookConferenceSlot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		slotID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "slotId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid slot id.")
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}

		var body bookConferenceSlotBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		childID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, childID); !ok {
			return
		}

		slot, av, err := conferences.BookSlot(r.Context(), d.Pool, slotID, parentID, childID)
		if err == conferences.ErrAlreadyBooked {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "This slot is no longer available.")
			return
		}
		if err == conferences.ErrTeacherNotLinked {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This teacher is not linked to your child.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to book slot.")
			return
		}
		if slot == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Slot not found.")
			return
		}

		d.sendConferenceConfirmationEmails(r, slot, av, parentID, childID, orgID)
		writeJSON(w, http.StatusCreated, map[string]any{"slot": slot, "availability": av})
	}
}

// handleCancelConferenceBooking — DELETE /api/v1/conference-slots/{slotId}/book
func (d Deps) handleCancelConferenceBooking() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		slotID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "slotId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid slot id.")
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}

		slot, av, err := conferences.CancelBooking(r.Context(), d.Pool, slotID, parentID)
		if err == conferences.ErrNotBookedByParent {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Booking not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to cancel booking.")
			return
		}

		if slot.BookedForChild != nil {
			childID, _ := uuid.Parse(*slot.BookedForChild)
			d.sendConferenceCancellationEmails(r, slot, av, parentID, childID, orgID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"slot": slot})
	}
}

// handleParentConferenceTeachers — GET /api/v1/parent/conference-teachers?studentId=...
func (d Deps) handleParentConferenceTeachers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		studentID, ok := d.parseStudentIDParam(w, r)
		if !ok {
			studentIDStr := strings.TrimSpace(r.URL.Query().Get("studentId"))
			if studentIDStr == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "studentId is required.")
				return
			}
			var err error
			studentID, err = uuid.Parse(studentIDStr)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
				return
			}
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}

		teachers, err := conferences.ListTeachersForStudent(r.Context(), d.Pool, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load teachers.")
			return
		}
		if teachers == nil {
			teachers = []conferences.TeacherSummary{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"teachers": teachers})
	}
}

// handleSchoolConferenceSchedule — GET /api/v1/admin/org-units/{orgUnitId}/conference-schedule?date=YYYY-MM-DD
func (d Deps) handleSchoolConferenceSchedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		orgUnitID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgUnitId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org unit id.")
			return
		}
		unit, err := orgunit.GetByID(r.Context(), d.Pool, orgUnitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org unit.")
			return
		}
		if unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Org unit not found.")
			return
		}
		actorID, global, ok := d.adminOrgOrUnitAccess(w, r, unit.OrgID)
		if !ok {
			return
		}
		if !global {
			allowed, err := d.unitAdminAllowedSubtree(r.Context(), actorID, unit.OrgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify org unit access.")
				return
			}
			if !d.unitInAllowedSubtree(orgUnitID, allowed) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this org unit.")
				return
			}
		}
		date := strings.TrimSpace(r.URL.Query().Get("date"))
		if date == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "date query parameter is required.")
			return
		}

		entries, err := conferences.ListSchoolSchedule(r.Context(), d.Pool, orgUnitID, date)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load schedule.")
			return
		}
		if entries == nil {
			entries = []conferences.ScheduleEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"schedule": entries})
	}
}

// handleConferenceSlotIcal — GET /api/v1/conference-slots/{slotId}/ical
func (d Deps) handleConferenceSlotIcal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.conferenceSchedulingEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		slotID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "slotId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid slot id.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		slot, av, err := conferences.GetSlotByID(r.Context(), d.Pool, slotID)
		if err != nil || slot == nil || av == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Slot not found.")
			return
		}

		isParent := slot.BookedByParent != nil && *slot.BookedByParent == userID.String()
		isTeacher := av.TeacherID == userID.String()
		if !isParent && !isTeacher {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
			return
		}

		summary, location := conferenceEventDetails(r.Context(), d, slot, av)
		ics := icsgenerator.BuildEvent(icsgenerator.Event{
			UID:       icsgenerator.ConferenceUID(slot.ID),
			Summary:   summary,
			Location:  location,
			Start:     slot.StartAt,
			End:       slot.EndAt,
			Organizer: "School",
		})
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="conference-%s.ics"`, slot.ID[:8]))
		_, _ = w.Write([]byte(ics))
	}
}

func conferenceEventDetails(ctx context.Context, d Deps, slot *conferences.Slot, av *conferences.Availability) (summary, location string) {
	teacherName := "Teacher"
	if tid, err := uuid.Parse(av.TeacherID); err == nil {
		if row, _ := user.FindByID(ctx, d.Pool, tid); row != nil && row.DisplayName != nil {
			teacherName = strings.TrimSpace(*row.DisplayName)
		}
	}
	childName := ""
	if slot.BookedForChild != nil {
		if cid, err := uuid.Parse(*slot.BookedForChild); err == nil {
			if row, _ := user.FindByID(ctx, d.Pool, cid); row != nil && row.DisplayName != nil {
				childName = strings.TrimSpace(*row.DisplayName)
			}
		}
	}
	summary = fmt.Sprintf("Parent-Teacher Conference with %s", teacherName)
	if childName != "" {
		summary = fmt.Sprintf("%s (%s)", summary, childName)
	}
	if av.VideoLink != nil && strings.TrimSpace(*av.VideoLink) != "" {
		location = strings.TrimSpace(*av.VideoLink)
	} else if av.Location != nil {
		location = strings.TrimSpace(*av.Location)
	}
	return summary, location
}

func (d Deps) sendConferenceConfirmationEmails(r *http.Request, slot *conferences.Slot, av *conferences.Availability, parentID, childID, orgID uuid.UUID) {
	if !d.Config.EmailNotificationsEnabled {
		return
	}
	summary, location := conferenceEventDetails(r.Context(), d, slot, av)
	when := slot.StartAt.UTC().Format("Mon Jan 2, 2006 3:04 PM MST")
	ics := icsgenerator.BuildEvent(icsgenerator.Event{
		UID:       icsgenerator.ConferenceUID(slot.ID),
		Summary:   summary,
		Location:  location,
		Start:     slot.StartAt,
		End:       slot.EndAt,
		Organizer: "School",
	})

	ns := &notifications.Service{Pool: d.Pool, Config: d.Config}
	vars := map[string]string{
		"when":        when,
		"summary":     summary,
		"location":    location,
		"icsContent":  ics,
		"icsFilename": fmt.Sprintf("conference-%s.ics", slot.ID[:8]),
	}
	teacherID, _ := uuid.Parse(av.TeacherID)
	_ = ns.EnqueueEmail(r.Context(), parentID, notifications.EventConferenceConfirmed, "conference_confirmed", vars, &orgID)
	_ = ns.EnqueueEmail(r.Context(), teacherID, notifications.EventConferenceConfirmed, "conference_confirmed", vars, &orgID)
}

func (d Deps) sendConferenceCancellationEmails(r *http.Request, slot *conferences.Slot, av *conferences.Availability, parentID, childID, orgID uuid.UUID) {
	if !d.Config.EmailNotificationsEnabled {
		return
	}
	summary, location := conferenceEventDetails(r.Context(), d, slot, av)
	when := slot.StartAt.UTC().Format("Mon Jan 2, 2006 3:04 PM MST")
	ns := &notifications.Service{Pool: d.Pool, Config: d.Config}
	vars := map[string]string{
		"when":     when,
		"summary":  summary,
		"location": location,
		"cancelled": "true",
	}
	teacherID, _ := uuid.Parse(av.TeacherID)
	_ = ns.EnqueueEmail(r.Context(), parentID, notifications.EventConferenceConfirmed, "conference_cancelled", vars, &orgID)
	_ = ns.EnqueueEmail(r.Context(), teacherID, notifications.EventConferenceConfirmed, "conference_cancelled", vars, &orgID)
}

func (d Deps) registerConferenceRoutes(r chi.Router) {
	r.Post("/api/v1/teachers/{teacherId}/conference-availability", d.handleCreateConferenceAvailability())
	r.Get("/api/v1/teachers/{teacherId}/conference-slots", d.handleListConferenceSlots())
	r.Post("/api/v1/conference-slots/{slotId}/book", d.handleBookConferenceSlot())
	r.Delete("/api/v1/conference-slots/{slotId}/book", d.handleCancelConferenceBooking())
	r.Get("/api/v1/conference-slots/{slotId}/ical", d.handleConferenceSlotIcal())
	r.Get("/api/v1/parent/conference-teachers", d.handleParentConferenceTeachers())
	r.Get("/api/v1/admin/org-units/{orgUnitId}/conference-schedule", d.handleSchoolConferenceSchedule())
}
