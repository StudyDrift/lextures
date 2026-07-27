package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/toolmarket"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	tmsvc "github.com/lextures/lextures/server/internal/service/toolmarket"
)

func (d Deps) handleDeveloperListTools() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		tools, err := toolmarket.ListToolsByOwner(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list tools.")
			return
		}
		out := make([]map[string]any, 0, len(tools))
		for _, t := range tools {
			out = append(out, toolToJSON(t))
		}
		writeJSON(w, http.StatusOK, map[string]any{"tools": out})
	}
}

type createToolBody struct {
	ToolID        string   `json:"toolId"`
	DisplayName   string   `json:"displayName"`
	Summary       string   `json:"summary"`
	DescriptionMD string   `json:"descriptionMd"`
	SubjectTags   []string `json:"subjectTags"`
	GradeTags     []string `json:"gradeTags"`
	SupportURL    *string  `json:"supportUrl"`
	PrivacyURL    *string  `json:"privacyUrl"`
	Visibility    string   `json:"visibility"`
	PricingModel  string   `json:"pricingModel"`
}

func (d Deps) handleDeveloperCreateTool() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		var body createToolBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.ToolID) == "" || strings.TrimSpace(body.DisplayName) == "" || strings.TrimSpace(body.Summary) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "toolId, displayName, and summary are required.")
			return
		}
		orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		var orgPtr *uuid.UUID
		if orgID != uuid.Nil {
			orgPtr = &orgID
		}
		t, err := d.toolMarketService().CreateTool(r.Context(), tmsvc.CreateToolInput{
			ToolID:        body.ToolID,
			OwnerUserID:   userID,
			OwnerOrgID:    orgPtr,
			DisplayName:   body.DisplayName,
			Summary:       body.Summary,
			DescriptionMD: body.DescriptionMD,
			SubjectTags:   body.SubjectTags,
			GradeTags:     body.GradeTags,
			SupportURL:    body.SupportURL,
			PrivacyURL:    body.PrivacyURL,
			Visibility:    body.Visibility,
			PricingModel:  body.PricingModel,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		after, _ := json.Marshal(map[string]any{"toolId": t.ToolID, "action": "create"})
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			OrgID:      orgPtr,
			EventType:  adminaudit.EventContentToolMarketplace,
			ActorID:    userID,
			AfterValue: after,
		})
		writeJSON(w, http.StatusCreated, toolToJSON(*t))
	}
}

type createReleaseBody struct {
	Version            string            `json:"version"`
	Manifest           json.RawMessage   `json:"manifest"`
	DataSheet          json.RawMessage   `json:"dataSheet"`
	BundleBase64       string            `json:"bundleBase64"`
	AxeStatus          string            `json:"axeStatus"`
	KeyboardTestStatus string            `json:"keyboardTestStatus"`
	I18nKeys           map[string]string `json:"i18nKeys"`
}

func (d Deps) handleDeveloperCreateRelease() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		toolID := chi.URLParam(r, "tool_id")
		var body createReleaseBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		res, err := d.toolMarketService().CreateRelease(r.Context(), tmsvc.CreateReleaseInput{
			ToolID:             toolID,
			OwnerUserID:        userID,
			Version:            body.Version,
			ManifestJSON:       body.Manifest,
			DataSheetJSON:      body.DataSheet,
			BundleBase64:       body.BundleBase64,
			AxeStatus:          body.AxeStatus,
			KeyboardTestStatus: body.KeyboardTestStatus,
			I18nKeys:           body.I18nKeys,
		})
		if err != nil {
			msg := err.Error()
			code := http.StatusBadRequest
			if msg == "tool not found" {
				code = http.StatusNotFound
			} else if msg == "forbidden" {
				code = http.StatusForbidden
			}
			apierr.WriteJSON(w, code, apierr.CodeInvalidInput, msg)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"release": releaseToJSON(res.Release),
			"checks":  res.Checks,
		})
	}
}

func (d Deps) handleDeveloperSubmitRelease() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		toolID := chi.URLParam(r, "tool_id")
		version := chi.URLParam(r, "version")
		rel, err := d.toolMarketService().SubmitRelease(r.Context(), toolID, version, userID)
		if err != nil {
			var rej *tmsvc.SubmitRejectedError
			if errors.As(err, &rej) {
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity, err.Error())
				return
			}
			msg := err.Error()
			code := http.StatusBadRequest
			if msg == "tool not found" || msg == "release not found" {
				code = http.StatusNotFound
			} else if msg == "forbidden" {
				code = http.StatusForbidden
			}
			apierr.WriteJSON(w, code, apierr.CodeInvalidInput, msg)
			return
		}
		after, _ := json.Marshal(map[string]any{"toolId": toolID, "version": version, "action": "submit"})
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			EventType:  adminaudit.EventContentToolMarketplace,
			ActorID:    userID,
			AfterValue: after,
		})
		writeJSON(w, http.StatusOK, releaseToJSON(rel))
	}
}

func (d Deps) handleDeveloperToolAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		toolID := chi.URLParam(r, "tool_id")
		a, err := d.toolMarketService().AnalyticsForTool(r.Context(), toolID, userID)
		if err != nil {
			msg := err.Error()
			code := http.StatusBadRequest
			if msg == "tool not found" {
				code = http.StatusNotFound
			} else if msg == "forbidden" {
				code = http.StatusForbidden
			}
			apierr.WriteJSON(w, code, apierr.CodeInvalidInput, msg)
			return
		}
		// AC-8: aggregate only — no student-identifiable fields.
		writeJSON(w, http.StatusOK, a)
	}
}

type grantAccessBody struct {
	OrgID string `json:"orgId"`
}

func (d Deps) handleDeveloperGrantAccess() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		var body grantAccessBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(body.OrgID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
			return
		}
		if err := d.toolMarketService().GrantUnlistedAccess(r.Context(), chi.URLParam(r, "tool_id"), userID, orgID); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type sunsetBody struct {
	SunsetAt string `json:"sunsetAt"`
}

func (d Deps) handleDeveloperSunset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		var body sunsetBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(body.SunsetAt))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sunsetAt must be RFC3339.")
			return
		}
		if err := d.toolMarketService().AnnounceSunset(r.Context(), chi.URLParam(r, "tool_id"), userID, ts); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
