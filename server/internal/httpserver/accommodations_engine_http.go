package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/accommodations"
	"github.com/lextures/lextures/server/internal/repos/accommodationaudit"
	stac "github.com/lextures/lextures/server/internal/repos/studentaccommodations"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type accommodationAuditEntryJSON struct {
	ID                string          `json:"id"`
	StudentID         string          `json:"studentId"`
	AccommodationType string          `json:"accommodationType"`
	ValueApplied      json.RawMessage `json:"valueApplied"`
	Context           string          `json:"context"`
	ContextID         *string         `json:"contextId,omitempty"`
	AppliedAt         string          `json:"appliedAt"`
}

func (d Deps) handleAccommodationAuditLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAccommodationsEngine(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		if !requireAccManage(ctx, w, d, uid) {
			return
		}
		var studentFilter *uuid.UUID
		if s := strings.TrimSpace(r.URL.Query().Get("studentId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
				return
			}
			studentFilter = &id
		}
		limit := 100
		if ls := strings.TrimSpace(r.URL.Query().Get("limit")); ls != "" {
			if n, err := strconv.Atoi(ls); err == nil {
				limit = n
			}
		}
		rows, err := accommodationaudit.List(ctx, d.Pool, accommodationaudit.ListFilter{
			StudentID: studentFilter,
			Limit:     limit,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load audit log.")
			return
		}
		out := make([]accommodationAuditEntryJSON, 0, len(rows))
		for _, row := range rows {
			entry := accommodationAuditEntryJSON{
				ID:                row.ID.String(),
				StudentID:         row.StudentID.String(),
				AccommodationType: row.AccommodationType,
				ValueApplied:      row.ValueApplied,
				Context:           row.Context,
				AppliedAt:         row.AppliedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			}
			if row.ContextID != nil {
				s := row.ContextID.String()
				entry.ContextID = &s
			}
			out = append(out, entry)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}

type accommodationImportSummary struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Errors  []string `json:"errors"`
}

func (d Deps) handleAccommodationCSVImport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireAccommodationsEngine(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		if !requireAccManage(ctx, w, d, uid) {
			return
		}
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Expected multipart file upload.")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing file field.")
			return
		}
		defer file.Close()
		reader := csv.NewReader(io.LimitReader(file, 4<<20))
		reader.TrimLeadingSpace = true
		records, err := reader.ReadAll()
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid CSV.")
			return
		}
		if len(records) < 2 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "CSV must include a header row and at least one data row.")
			return
		}
		header := map[string]int{}
		for i, col := range records[0] {
			header[strings.ToLower(strings.TrimSpace(col))] = i
		}
		extIdx, okExt := header["student_external_id"]
		typeIdx, okType := header["accommodation_type"]
		valIdx, okVal := header["value"]
		if !okExt || !okType || !okVal {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "CSV headers must include student_external_id, accommodation_type, value.")
			return
		}
		summary := accommodationImportSummary{Errors: []string{}}
		for rowNum, rec := range records[1:] {
			if len(rec) == 0 {
				continue
			}
			extID := strings.TrimSpace(rec[extIdx])
			accType := strings.ToLower(strings.TrimSpace(rec[typeIdx]))
			val := strings.TrimSpace(rec[valIdx])
			if extID == "" || accType == "" {
				summary.Errors = append(summary.Errors, "row "+strconv.Itoa(rowNum+2)+": missing student_external_id or accommodation_type")
				continue
			}
			learnerID, lerr := user.LookupIDByEmailOrSID(ctx, d.Pool, extID)
			if lerr != nil || learnerID == nil {
				summary.Errors = append(summary.Errors, "row "+strconv.Itoa(rowNum+2)+": learner not found for "+extID)
				continue
			}
			created, uerr := stac.ApplyCSVRow(ctx, d.Pool, *learnerID, accType, val, uid)
			if uerr != nil {
				summary.Errors = append(summary.Errors, "row "+strconv.Itoa(rowNum+2)+": "+uerr.Error())
				continue
			}
			if created {
				summary.Created++
			} else {
				summary.Updated++
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

func (d Deps) accommodationsEngineFeatureEnabled() bool {
	return d.effectiveConfig().AccommodationsEngineEnabled
}

func (d Deps) requireAccommodationsEngine(w http.ResponseWriter) bool {
	if !d.accommodationsEngineFeatureEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Accommodations engine is not enabled.")
		return false
	}
	return true
}

var _ = acmodel.PermManage
