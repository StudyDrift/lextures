package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/timezonecatalog"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/validation"
)

type timezoneSettingsResponse struct {
	Timezone *string `json:"timezone"`
}

type putTimezoneBody struct {
	Timezone *string `json:"timezone"`
}

type timezonesListResponse struct {
	Timezones []timezonecatalog.Entry `json:"timezones"`
}

func (d Deps) handleGetSettingsTimezone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		tz, err := user.GetTimezone(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load timezone.")
			return
		}
		writeJSON(w, http.StatusOK, timezoneSettingsResponse{Timezone: tz})
	}
}

func (d Deps) handlePutSettingsTimezone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var req putTimezoneBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		tz := validation.NormalizeTimezone(req.Timezone)
		if tz != nil {
			valid, err := validation.ValidIANATimezone(r.Context(), d.Pool, *tz)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate timezone.")
				return
			}
			if !valid {
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity, "Invalid IANA timezone identifier.")
				return
			}
		}
		if err := user.SetTimezone(r.Context(), d.Pool, userID, tz); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update timezone.")
			return
		}
		writeJSON(w, http.StatusOK, timezoneSettingsResponse{Timezone: tz})
	}
}

func (d Deps) handleListTimezones() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		list, err := timezonecatalog.List(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list timezones.")
			return
		}
		if list == nil {
			list = []timezonecatalog.Entry{}
		}
		writeJSON(w, http.StatusOK, timezonesListResponse{Timezones: list})
	}
}
