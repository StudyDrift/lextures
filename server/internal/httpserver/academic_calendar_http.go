package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func (d Deps) requireAcademicCalendar(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFAcademicCalendar && !d.Config.FFAcademicCalendar {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Academic calendar is not enabled.")
		return false
	}
	return true
}

type calendarEventRow struct {
	ID         string  `json:"id"`
	OrgID      string  `json:"orgId"`
	TermID     *string `json:"termId,omitempty"`
	EventType  string  `json:"eventType"`
	EventName  string  `json:"eventName"`
	StartDate  string  `json:"startDate"`
	EndDate    *string `json:"endDate,omitempty"`
	AllDay     bool    `json:"allDay"`
	Notes      *string `json:"notes,omitempty"`
	SisID      *string `json:"sisId,omitempty"`
	CreatedBy  *string `json:"createdBy,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}

func (d Deps) handleCalendarEventsGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAcademicCalendar(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}

		var termFilter *uuid.UUID
		if q := strings.TrimSpace(r.URL.Query().Get("term_id")); q != "" {
			tid, err := uuid.Parse(q)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid term_id.")
				return
			}
			termFilter = &tid
		}

		rows, err := d.listCalendarEvents(r, orgID, termFilter)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load calendar events.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": rows})
	}
}

func (d Deps) listCalendarEvents(r *http.Request, orgID uuid.UUID, termID *uuid.UUID) ([]calendarEventRow, error) {
	ctx := r.Context()
	var args []any
	q := `
SELECT id, org_id, term_id, event_type, event_name, start_date, end_date, all_day, notes, sis_id, created_by, created_at
FROM tenant.academic_calendar_events
WHERE org_id = $1`
	args = append(args, orgID)
	if termID != nil {
		args = append(args, *termID)
		q += fmt.Sprintf(" AND term_id = $%d", len(args))
	}
	q += " ORDER BY start_date ASC, created_at ASC"

	pgRows, err := d.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	var out []calendarEventRow
	for pgRows.Next() {
		var (
			id, orgIDScan       uuid.UUID
			termIDScan          *uuid.UUID
			eventType, name     string
			startDate           time.Time
			endDate             *time.Time
			allDay              bool
			notes, sisID        *string
			createdBy           *uuid.UUID
			createdAt           time.Time
		)
		if err := pgRows.Scan(&id, &orgIDScan, &termIDScan, &eventType, &name, &startDate, &endDate, &allDay, &notes, &sisID, &createdBy, &createdAt); err != nil {
			return nil, err
		}
		row := calendarEventRow{
			ID:        id.String(),
			OrgID:     orgIDScan.String(),
			EventType: eventType,
			EventName: name,
			StartDate: startDate.Format("2006-01-02"),
			AllDay:    allDay,
			Notes:     notes,
			SisID:     sisID,
			CreatedAt: createdAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		if termIDScan != nil {
			s := termIDScan.String()
			row.TermID = &s
		}
		if endDate != nil {
			s := endDate.Format("2006-01-02")
			row.EndDate = &s
		}
		if createdBy != nil {
			s := createdBy.String()
			row.CreatedBy = &s
		}
		out = append(out, row)
	}
	if err := pgRows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []calendarEventRow{}
	}
	return out, nil
}

func (d Deps) handleAdminCalendarEventPost() http.HandlerFunc {
	type reqBody struct {
		TermID    *string `json:"termId"`
		EventType string  `json:"eventType"`
		EventName string  `json:"eventName"`
		StartDate string  `json:"startDate"`
		EndDate   *string `json:"endDate"`
		AllDay    *bool   `json:"allDay"`
		Notes     *string `json:"notes"`
		SisID     *string `json:"sisId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAcademicCalendar(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.EventName) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "eventName is required.")
			return
		}
		if !validCalendarEventType(body.EventType) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid eventType.")
			return
		}
		startDate, err := time.Parse("2006-01-02", strings.TrimSpace(body.StartDate))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid startDate (use YYYY-MM-DD).")
			return
		}
		var endDate *time.Time
		if body.EndDate != nil && strings.TrimSpace(*body.EndDate) != "" {
			ed, err := time.Parse("2006-01-02", strings.TrimSpace(*body.EndDate))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid endDate (use YYYY-MM-DD).")
				return
			}
			if ed.Before(startDate) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "endDate must not be before startDate.")
				return
			}
			endDate = &ed
		}
		allDay := true
		if body.AllDay != nil {
			allDay = *body.AllDay
		}
		var termID *uuid.UUID
		if body.TermID != nil && strings.TrimSpace(*body.TermID) != "" {
			tid, err := uuid.Parse(strings.TrimSpace(*body.TermID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid termId.")
				return
			}
			termID = &tid
		}

		var newID uuid.UUID
		var createdAt time.Time
		err = d.Pool.QueryRow(r.Context(), `
INSERT INTO tenant.academic_calendar_events
    (org_id, term_id, event_type, event_name, start_date, end_date, all_day, notes, sis_id, created_by)
VALUES ($1, $2, $3::tenant.calendar_event_type, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, created_at`,
			orgID, termID, body.EventType, strings.TrimSpace(body.EventName),
			startDate, endDate, allDay, body.Notes, body.SisID, viewer,
		).Scan(&newID, &createdAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create calendar event.")
			return
		}

		out := calendarEventRow{
			ID:        newID.String(),
			OrgID:     orgID.String(),
			EventType: body.EventType,
			EventName: strings.TrimSpace(body.EventName),
			StartDate: startDate.Format("2006-01-02"),
			AllDay:    allDay,
			Notes:     body.Notes,
			SisID:     body.SisID,
			CreatedAt: createdAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		if termID != nil {
			s := termID.String()
			out.TermID = &s
		}
		if endDate != nil {
			s := endDate.Format("2006-01-02")
			out.EndDate = &s
		}
		vStr := viewer.String()
		out.CreatedBy = &vStr

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"event": out})
	}
}

func (d Deps) handleAdminCalendarEventPatch() http.HandlerFunc {
	type reqBody struct {
		EventType *string `json:"eventType"`
		EventName *string `json:"eventName"`
		StartDate *string `json:"startDate"`
		EndDate   *string `json:"endDate"`
		AllDay    *bool   `json:"allDay"`
		Notes     *string `json:"notes"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAcademicCalendar(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		eventID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "eventId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid event id.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.EventType != nil && !validCalendarEventType(*body.EventType) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid eventType.")
			return
		}

		setClauses := []string{}
		args := []any{}
		nextArg := func(v any) string {
			args = append(args, v)
			return fmt.Sprintf("$%d", len(args))
		}
		if body.EventType != nil {
			setClauses = append(setClauses, "event_type = "+nextArg(*body.EventType)+"::tenant.calendar_event_type")
		}
		if body.EventName != nil {
			if strings.TrimSpace(*body.EventName) == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "eventName cannot be empty.")
				return
			}
			setClauses = append(setClauses, "event_name = "+nextArg(strings.TrimSpace(*body.EventName)))
		}
		if body.StartDate != nil {
			sd, err := time.Parse("2006-01-02", strings.TrimSpace(*body.StartDate))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid startDate.")
				return
			}
			setClauses = append(setClauses, "start_date = "+nextArg(sd))
		}
		if body.EndDate != nil {
			if strings.TrimSpace(*body.EndDate) == "" {
				setClauses = append(setClauses, "end_date = NULL")
			} else {
				ed, err := time.Parse("2006-01-02", strings.TrimSpace(*body.EndDate))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid endDate.")
					return
				}
				setClauses = append(setClauses, "end_date = "+nextArg(ed))
			}
		}
		if body.AllDay != nil {
			setClauses = append(setClauses, "all_day = "+nextArg(*body.AllDay))
		}
		if body.Notes != nil {
			if strings.TrimSpace(*body.Notes) == "" {
				setClauses = append(setClauses, "notes = NULL")
			} else {
				setClauses = append(setClauses, "notes = "+nextArg(*body.Notes))
			}
		}
		if len(setClauses) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No fields to update.")
			return
		}
		args = append(args, eventID, orgID)
		q := fmt.Sprintf(`
UPDATE tenant.academic_calendar_events
SET %s
WHERE id = $%d AND org_id = $%d
RETURNING id, org_id, term_id, event_type, event_name, start_date, end_date, all_day, notes, sis_id, created_by, created_at`,
			strings.Join(setClauses, ", "), len(args)-1, len(args))

		var (
			id, orgIDScan       uuid.UUID
			termIDScan          *uuid.UUID
			eventType, name     string
			startDate           time.Time
			endDate             *time.Time
			allDay              bool
			notes, sisID        *string
			createdBy           *uuid.UUID
			createdAt           time.Time
		)
		err = d.Pool.QueryRow(r.Context(), q, args...).Scan(
			&id, &orgIDScan, &termIDScan, &eventType, &name, &startDate, &endDate, &allDay, &notes, &sisID, &createdBy, &createdAt,
		)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Calendar event not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update calendar event.")
			return
		}
		out := calendarEventRow{
			ID:        id.String(),
			OrgID:     orgIDScan.String(),
			EventType: eventType,
			EventName: name,
			StartDate: startDate.Format("2006-01-02"),
			AllDay:    allDay,
			Notes:     notes,
			SisID:     sisID,
			CreatedAt: createdAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		if termIDScan != nil {
			s := termIDScan.String()
			out.TermID = &s
		}
		if endDate != nil {
			s := endDate.Format("2006-01-02")
			out.EndDate = &s
		}
		if createdBy != nil {
			s := createdBy.String()
			out.CreatedBy = &s
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"event": out})
	}
}

func (d Deps) handleAdminCalendarEventDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAcademicCalendar(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		eventID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "eventId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid event id.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		tag, err := d.Pool.Exec(r.Context(),
			`DELETE FROM tenant.academic_calendar_events WHERE id = $1 AND org_id = $2`,
			eventID, orgID,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete calendar event.")
			return
		}
		if tag.RowsAffected() == 0 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Calendar event not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleCalendarTermICAL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAcademicCalendar(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		termID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "termId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid term id.")
			return
		}

		events, err := d.listCalendarEvents(r, orgID, &termID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load calendar events.")
			return
		}

		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="term-%s.ics"`, termID))

		var b strings.Builder
		b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Lextures//Academic Calendar//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n")
		now := time.Now().UTC().Format("20060102T150405Z")
		for _, ev := range events {
			uid := fmt.Sprintf("%s@lextures", ev.ID)
			dtstart := strings.ReplaceAll(ev.StartDate, "-", "")
			dtend := dtstart
			if ev.EndDate != nil {
				// End date in iCal all-day events is exclusive (next day).
				ed, perr := time.Parse("2006-01-02", *ev.EndDate)
				if perr == nil {
					dtend = ed.AddDate(0, 0, 1).Format("20060102")
				}
			} else {
				// Single-day event: end = start + 1 day.
				sd, perr := time.Parse("2006-01-02", ev.StartDate)
				if perr == nil {
					dtend = sd.AddDate(0, 0, 1).Format("20060102")
				}
			}
			b.WriteString("BEGIN:VEVENT\r\n")
			_, _ = fmt.Fprintf(&b, "UID:%s\r\n", uid)
			_, _ = fmt.Fprintf(&b, "DTSTAMP:%s\r\n", now)
			_, _ = fmt.Fprintf(&b, "DTSTART;VALUE=DATE:%s\r\n", dtstart)
			_, _ = fmt.Fprintf(&b, "DTEND;VALUE=DATE:%s\r\n", dtend)
			_, _ = fmt.Fprintf(&b, "SUMMARY:%s\r\n", icalEscape(ev.EventName))
			if ev.Notes != nil && strings.TrimSpace(*ev.Notes) != "" {
				_, _ = fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", icalEscape(*ev.Notes))
			}
			b.WriteString("END:VEVENT\r\n")
		}
		b.WriteString("END:VCALENDAR\r\n")
		_, _ = w.Write([]byte(b.String()))
	}
}

func validCalendarEventType(t string) bool {
	switch t {
	case "term_start", "term_end", "add_drop_deadline", "withdrawal_deadline",
		"finals_start", "finals_end", "no_class_day", "holiday", "custom":
		return true
	}
	return false
}

func icalEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
