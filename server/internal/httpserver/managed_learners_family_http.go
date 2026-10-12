package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	managedlearners "github.com/lextures/lextures/server/internal/service/managedlearners"
)

// Family-device learner sign-in (#771). Parent management lives under
// /api/v1/me/dependents so managedLearnerGuardMiddleware blocks it for learner
// sessions; the picker + sign-in endpoints authenticate with the device secret.
const (
	familySignInPickerPath  = "/api/v1/auth/family-device/learners"
	familySignInSessionPath = "/api/v1/auth/family-device/sessions"
)

func (d Deps) registerFamilySignInRoutes(r chi.Router) {
	r.Get("/api/v1/me/dependents/family-devices", d.handleListFamilyDevices())
	r.Post("/api/v1/me/dependents/family-devices", d.handleRegisterFamilyDevice())
	r.Delete("/api/v1/me/dependents/family-devices/{deviceId}", d.handleRevokeFamilyDevice())
	r.Put("/api/v1/me/dependents/{id}/sign-in", d.handleSetDependentSignIn())
	r.Delete("/api/v1/me/dependents/{id}/sign-in", d.handleDisableDependentSignIn())
	r.Post(familySignInPickerPath, d.handleFamilyPicker())
	r.Post(familySignInSessionPath, d.handleFamilySignIn())
}

func readFamilySignInJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
		return false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
		return false
	}
	return true
}

func (d Deps) handleListFamilyDevices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		list, err := managedlearners.ListFamilyDevices(r.Context(), d.Pool, actor)
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": list})
	}
}

func (d Deps) handleRegisterFamilyDevice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			Label string `json:"label"`
		}
		if !readFamilySignInJSON(w, r, &body) {
			return
		}
		token, dev, err := managedlearners.RegisterFamilyDevice(r.Context(), d.Pool, actor, body.Label)
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"deviceToken": token, "device": dev})
	}
}

func (d Deps) handleRevokeFamilyDevice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "deviceId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid device id.")
			return
		}
		if err := managedlearners.RevokeFamilyDevice(r.Context(), d.Pool, actor, id); err != nil {
			if errors.Is(err, managedlearners.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Family device not found.")
				return
			}
			writeManagedLearnersErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleSetDependentSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dependent id.")
			return
		}
		var body struct {
			Method  string   `json:"method"`
			Pin     string   `json:"pin"`
			Picture []string `json:"picture"`
		}
		if !readFamilySignInJSON(w, r, &body) {
			return
		}
		dep, err := managedlearners.SetSignIn(r.Context(), d.Pool, managedlearners.SetSignInParams{
			ActorID:     actor,
			DependentID: id,
			Method:      body.Method,
			Pin:         body.Pin,
			Picture:     body.Picture,
		})
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, dep)
	}
}

func (d Deps) handleDisableDependentSignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dependent id.")
			return
		}
		dep, err := managedlearners.DisableSignIn(r.Context(), d.Pool, actor, id)
		if err != nil {
			writeManagedLearnersErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, dep)
	}
}

func (d Deps) handleFamilyPicker() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		var body struct {
			DeviceToken string `json:"deviceToken"`
		}
		if !readFamilySignInJSON(w, r, &body) {
			return
		}
		list, err := managedlearners.FamilyPicker(r.Context(), d.Pool, body.DeviceToken)
		if err != nil {
			writeFamilySignInErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"learners":       list,
			"pictureChoices": managedlearners.PictureChoices,
		})
	}
}

func (d Deps) handleFamilySignIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.managedLearnersEnabled(w) {
			return
		}
		var body struct {
			DeviceToken string   `json:"deviceToken"`
			LearnerID   string   `json:"learnerId"`
			Pin         string   `json:"pin"`
			Picture     []string `json:"picture"`
		}
		if !readFamilySignInJSON(w, r, &body) {
			return
		}
		learnerID, err := uuid.Parse(strings.TrimSpace(body.LearnerID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid learner id.")
			return
		}
		result, err := managedlearners.FamilySignIn(r.Context(), d.Pool, d.JWTSigner, managedlearners.FamilySignInParams{
			DeviceToken: body.DeviceToken,
			LearnerID:   learnerID,
			Pin:         body.Pin,
			Picture:     body.Picture,
		})
		if err != nil {
			writeFamilySignInErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"accessToken": result.Token,
			"expiresAt":   result.ExpiresAt,
			"target":      result.Target,
		})
	}
}

func writeFamilySignInErr(w http.ResponseWriter, err error) {
	var wrong *managedlearners.WrongSecretError
	var locked *managedlearners.LockedError
	switch {
	case errors.As(err, &wrong):
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error":             map[string]string{"code": apierr.CodeInvalidCredentials, "message": "That didn't match. Try again."},
			"attemptsRemaining": wrong.AttemptsRemaining,
		})
	case errors.As(err, &locked):
		secs := int(time.Until(locked.Until).Seconds())
		if secs < 1 {
			secs = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":       map[string]string{"code": apierr.CodeRateLimited, "message": "Too many tries. Ask a grown-up, or try again later."},
			"lockedUntil": locked.Until,
		})
	case errors.Is(err, managedlearners.ErrDeviceNotFamily):
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "This device is not set up for learner sign-in.")
	case errors.Is(err, managedlearners.ErrSignInNotSet):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Ask a parent to set up sign-in for this learner.")
	default:
		writeManagedLearnersErr(w, err)
	}
}
