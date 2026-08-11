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

// couponsFeatureOff writes 404 when course marketplace or course coupons are disabled
// (plan MKTC.2 FR-8). Requires both FFCourseMarketplace and FFCourseCoupons.
func (d Deps) couponsFeatureOff(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFCourseMarketplace || !cfg.FFCourseCoupons {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course coupons are not enabled.")
		return true
	}
	return false
}
