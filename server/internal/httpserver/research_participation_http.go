package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/researchparticipation"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
)

type researchParticipationJSON struct {
	OrgID         string              `json:"orgId"`
	Participation *repo.Participation `json:"participation"`
	Resolved      bool                `json:"resolved"`
	UpdatedAt     *string             `json:"updatedAt,omitempty"`
}

func researchParticipationResponse(orgID uuid.UUID, s *repo.Setting) researchParticipationJSON {
	out := researchParticipationJSON{OrgID: orgID.String()}
	if s != nil {
		p := s.Participation
		at := s.UpdatedAt.UTC().Format(time.RFC3339)
		out.Participation = &p
		out.Resolved = true
		out.UpdatedAt = &at
	}
	return out
}

func (d Deps) handleResearchParticipationGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
		if err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, ok := d.orgReadAccess(w, r, orgID); !ok {
			return
		}
		s, err := repo.Get(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, 500, apierr.CodeInternal, "Failed to load research participation.")
			return
		}
		writeJSON(w, http.StatusOK, researchParticipationResponse(orgID, s))
	}
}

func (d Deps) handleResearchParticipationPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
		if err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		actor, _, ok := d.adminOrgOrUnitAccess(w, r, orgID)
		if !ok {
			return
		}
		var body struct {
			Participation repo.Participation `json:"participation"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || !body.Participation.Valid() {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "participation must be opt_in or opt_out.")
			return
		}
		before, err := repo.Get(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, 500, apierr.CodeInternal, "Failed to load research participation.")
			return
		}
		updated, err := repo.Upsert(r.Context(), d.Pool, orgID, actor, body.Participation)
		if err != nil {
			apierr.WriteJSON(w, 500, apierr.CodeInternal, "Failed to save research participation.")
			return
		}
		beforeJSON, _ := json.Marshal(researchParticipationResponse(orgID, before))
		afterJSON, _ := json.Marshal(researchParticipationResponse(orgID, updated))
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventOrgSettingsChange, "research_participation", &orgID, beforeJSON, afterJSON)
		writeJSON(w, http.StatusOK, researchParticipationResponse(orgID, updated))
	}
}
