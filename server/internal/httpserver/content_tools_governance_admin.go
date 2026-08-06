package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) handleContentToolsDataSheets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
			return
		}
		if _, err := auth.UserFromRequest(r, d.JWTSigner); err != nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
			return
		}
		reg := ctsvc.MustDefault()
		if d.Pool != nil {
			_ = ctsvc.SyncDataSheets(r.Context(), d.Pool, reg)
			// Prefer DB mirror when present (trust-centre audits); fall back to registry.
			if rows, err := ctrepo.ListDataSheets(r.Context(), d.Pool); err == nil && len(rows) > 0 {
				items := make([]map[string]any, 0, len(rows))
				for _, row := range rows {
					item := map[string]any{
						"toolId":         row.ToolID,
						"version":        row.Version,
						"leavesPlatform": row.LeavesPlatform,
						"processors":     row.Processors,
						"visibility":     row.Visibility,
						"wcagLevel":      row.WCAGLevel,
					}
					if row.A11yLimitations != nil {
						item["a11yLimitations"] = *row.A11yLimitations
					}
					if m := reg.Get(row.ToolID); m != nil {
						item["name"] = m.Name
						item["capabilities"] = m.Capabilities
						item["collects"] = m.DataSheet.Collects
						item["aiTransparency"] = m.DataSheet.AITransparency
					}
					items = append(items, item)
				}
				writeJSON(w, http.StatusOK, map[string]any{"dataSheets": items})
				return
			}
		}
		items := make([]ctsvc.DataSheetPublic, 0, reg.Size())
		for _, m := range reg.List() {
			if m.DataSheet == nil {
				continue
			}
			procs := m.DataSheet.Processors
			if procs == nil {
				procs = []string{}
			}
			item := ctsvc.DataSheetPublic{
				ToolID:          m.ID,
				Version:         m.Version,
				Name:            m.Name,
				Collects:        m.DataSheet.Collects,
				LeavesPlatform:  m.DataSheet.LeavesPlatform,
				Processors:      procs,
				Visibility:      m.DataSheet.Visibility,
				WCAGLevel:       m.DataSheet.WCAGLevel,
				A11yLimitations: m.DataSheet.A11yLimitations,
				AITransparency:  m.DataSheet.AITransparency,
				Capabilities:    m.Capabilities,
			}
			if item.WCAGLevel == "" {
				item.WCAGLevel = "AA"
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"dataSheets": items})
	}
}

func (d Deps) handleContentToolsConformance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		rep := ctsvc.EvaluateConformance(ctsvc.MustDefault(), nil, nil)
		writeJSON(w, http.StatusOK, rep)
	}
}

func (d Deps) handleContentToolsAIConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		_ = courseCode
		if d.contentToolsConsentRateLimited(w, r, viewer) {
			return
		}
		var body ctmodel.AIConsentRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		decision := strings.TrimSpace(body.Decision)
		if decision != "acknowledged" && decision != "opted_out" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "decision must be acknowledged or opted_out.")
			return
		}
		var toolID *string
		if strings.TrimSpace(body.ToolID) != "" {
			t := strings.TrimSpace(body.ToolID)
			toolID = &t
		}
		cid := courseID
		row, err := ctrepo.UpsertAIConsent(r.Context(), d.Pool, viewer, &cid, toolID, decision)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record consent.")
			return
		}
		writeJSON(w, http.StatusOK, ctmodel.AIConsentResponse{
			Decision:  row.Decision,
			ToolID:    body.ToolID,
			DecidedAt: row.DecidedAt,
		})
	}
}

func (d Deps) handleContentToolsAIConsentGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		toolIDRaw := strings.TrimSpace(r.URL.Query().Get("toolId"))
		var toolID *string
		if toolIDRaw != "" {
			toolID = &toolIDRaw
		}
		cid := courseID
		row, err := ctrepo.GetAIConsent(r.Context(), d.Pool, viewer, &cid, toolID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load consent.")
			return
		}
		orgID, _ := course.CourseOrgID(r.Context(), d.Pool, chi.URLParam(r, "course_code"))
		mode := "banner"
		if orgID != nil {
			if pol, err := ctsvc.LoadOrgPolicy(r.Context(), d.Pool, *orgID); err == nil && pol != nil {
				mode = pol.AIDisclosureMode
			}
		}
		resp := map[string]any{
			"aiDisclosureMode": mode,
			"decision":         nil,
			"decidedAt":        nil,
		}
		if row != nil {
			resp["decision"] = row.Decision
			resp["decidedAt"] = row.DecidedAt.UTC().Format(time.RFC3339)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func (d Deps) handleContentToolsReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		if d.contentToolsReportRateLimited(w, r, viewer) {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		var body ctmodel.ModerationRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cat := strings.TrimSpace(body.Category)
		if cat == "" {
			cat = "other"
		}
		actor := viewer
		row := ctrepo.ModerationRow{
			InstanceID:  instanceID,
			Action:      "reported",
			Category:    &cat,
			Reason:      body.Reason,
			ActorUserID: &actor,
			ContentPath: body.ContentPath,
		}
		if body.SubjectUserID != nil {
			row.SubjectUserID = body.SubjectUserID
		}
		saved, err := ctrepo.InsertModeration(r.Context(), d.Pool, row)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record report.")
			return
		}
		ctsvc.IncModerationAction("reported")
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &instanceID, nil, &actor, inst.ToolID, "content_reported", map[string]any{
			"category": cat,
		})
		writeJSON(w, http.StatusCreated, moderationToAPI(saved))
	}
}

func (d Deps) handleContentToolsModerate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.contentToolsModerateRateLimited(w, r, viewer) {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		var body ctmodel.ModerationRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		action := strings.TrimSpace(body.Action)
		switch action {
		case "hidden", "removed", "restored", "warned":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "action must be hidden, removed, restored, or warned.")
			return
		}
		actor := viewer
		row := ctrepo.ModerationRow{
			InstanceID:    instanceID,
			Action:        action,
			Category:      body.CategoryPtr(),
			Reason:        body.Reason,
			ActorUserID:   &actor,
			SubjectUserID: body.SubjectUserID,
			ContentPath:   body.ContentPath,
			StateID:       body.StateID,
		}
		saved, err := ctrepo.InsertModeration(r.Context(), d.Pool, row)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record moderation.")
			return
		}
		ctsvc.IncModerationAction(action)
		orgID, _ := course.CourseOrgID(r.Context(), d.Pool, courseCode)
		after, _ := json.Marshal(moderationToAPI(saved))
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			OrgID:      orgID,
			EventType:  adminaudit.EventContentToolModeration,
			ActorID:    viewer,
			TargetType: ctStrPtr("content_tool_instance"),
			TargetID:   &instanceID,
			AfterValue: after,
		})
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &instanceID, nil, &actor, inst.ToolID, "content_moderated", map[string]any{
			"action": action,
		})
		out := moderationToAPI(saved)
		path := ""
		if body.ContentPath != nil {
			path = *body.ContentPath
		}
		if latest, err := ctrepo.LatestContentAction(r.Context(), d.Pool, instanceID, path); err == nil && latest != "" {
			// Echo effective visibility action for clients (hide/remove/restore).
			writeJSON(w, http.StatusCreated, map[string]any{
				"action":            out,
				"effectiveContentAction": latest,
			})
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func (d Deps) handleContentToolsModerationList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		if inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID); err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		rows, err := ctrepo.ListModeration(r.Context(), d.Pool, instanceID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list moderation.")
			return
		}
		items := make([]ctmodel.ModerationAction, 0, len(rows))
		for i := range rows {
			items = append(items, moderationToAPI(&rows[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func (d Deps) handleContentToolsFilterFlags() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		if inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID); err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		items, err := ctrepo.ListFilterFlags(r.Context(), d.Pool, instanceID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list flags.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func (d Deps) handleAdminContentToolsKill() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		var body ctmodel.KillRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		scope := strings.TrimSpace(body.Scope)
		switch scope {
		case "tool", "capability", "all_ai", "instance":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scope must be tool, capability, all_ai, or instance.")
			return
		}
		target := strings.TrimSpace(body.Target)
		if scope != "all_ai" && target == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "target is required for this scope.")
			return
		}
		if scope == "capability" {
			if _, ok := ctsvc.NormalizePolicyCapability(target); !ok {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid capability target.")
				return
			}
		}
		engaged := true
		if body.Engaged != nil {
			engaged = *body.Engaged
		}
		actorID := actor
		row, err := ctrepo.UpsertKill(r.Context(), d.Pool, scope, target, engaged, body.Reason, &actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update kill path.")
			return
		}
		ctsvc.ForceSyncDurableKillsFromDB(r.Context(), d.Pool)
		after, _ := json.Marshal(row)
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			EventType:  adminaudit.EventContentToolKill,
			ActorID:    actor,
			TargetType: ctStrPtr("content_tool_kill"),
			AfterValue: after,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"scope":     row.Scope,
			"target":    row.Target,
			"engaged":   row.Engaged,
			"reason":    row.Reason,
			"updatedAt": row.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleAdminContentToolsKillsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		ctsvc.SyncDurableKillsFromDB(r.Context(), d.Pool)
		rows, err := ctrepo.ListActiveKills(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list kills.")
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, map[string]any{
				"id":        row.ID.String(),
				"scope":     row.Scope,
				"target":    row.Target,
				"engaged":   row.Engaged,
				"reason":    row.Reason,
				"updatedAt": row.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"kills":           items,
			"envKillSwitch":   ctsvc.KillSwitchEngaged(),
			"envAIKillSwitch": ctsvc.AIKillSwitchEngaged(),
		})
	}
}

