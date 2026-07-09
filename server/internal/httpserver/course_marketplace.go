package httpserver

import (
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
)

// courseMarketplaceOff writes 404 when the in-app course marketplace is disabled
// (plan MKT1 FR-2). Mirrors publicCatalogOff; consumed by MKT2–MKT5 routes.
func (d Deps) courseMarketplaceOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCourseMarketplace {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Marketplace is not enabled.")
		return true
	}
	return false
}
