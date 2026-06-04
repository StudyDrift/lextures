package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
)

// GET /api/v1/orgs/{orgId}/settings/org-type
// PUT /api/v1/orgs/{orgId}/settings/org-type
func (d Deps) handleOrgTypeItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgStr := strings.TrimSpace(chi.URLParam(r, "orgId"))
		orgID, err := uuid.Parse(orgStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, _, ok := d.adminOrgOrUnitAccess(w, r, orgID); !ok {
			return
		}
		ctx := r.Context()

		switch r.Method {
		case http.MethodGet:
			var orgType string
			err := d.Pool.QueryRow(ctx, `SELECT org_type FROM tenant.organizations WHERE id = $1 AND status <> 'deleted'`, orgID).Scan(&orgType)
			if err != nil {
				orgType = "higher-ed"
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"orgId": orgID.String(), "orgType": orgType})

		case http.MethodPut:
			var body struct {
				OrgType string `json:"orgType"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			if body.OrgType != "higher-ed" && body.OrgType != "k-12" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, `orgType must be "higher-ed" or "k-12".`)
				return
			}
			_, err := d.Pool.Exec(ctx, `UPDATE tenant.organizations SET org_type = $1, updated_at = NOW() WHERE id = $2`, body.OrgType, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update organization type.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"orgId": orgID.String(), "orgType": body.OrgType})

		default:
			w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut}, ", "))
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}
