package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
)

func (d Deps) revenueShareFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFRevenueShare {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Revenue share is not enabled.")
		return true
	}
	if !d.effectiveConfig().FFStripeBilling {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Billing is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerRevenueShareRoutes(r chi.Router) {
	r.Get("/api/v1/creator/earnings", d.handleCreatorEarningsSummary())
	r.Get("/api/v1/creator/earnings/ledger", d.handleCreatorEarningsLedger())
	r.Post("/api/v1/creator/affiliate-codes", d.handleCreateAffiliateCode())
	r.Get("/api/v1/creator/affiliate-codes", d.handleListAffiliateCodes())
	r.Post("/api/v1/creator/connect/onboarding", d.handleConnectOnboarding())
	r.Get("/api/v1/creator/connect/status", d.handleConnectStatus())
	r.Post("/api/v1/affiliate/track-click", d.handleAffiliateTrackClick())
	r.Get("/api/v1/admin/revenue/summary", d.handleAdminRevenueSummary())
	r.Post("/api/v1/admin/revenue/trigger-payout", d.handleAdminTriggerPayout())
}

func ledgerToJSON(e repoBilling.LedgerEntry) map[string]any {
	out := map[string]any{
		"id":          e.ID.String(),
		"entryType":   e.EntryType,
		"amountCents": e.AmountCents,
		"currency":    e.Currency,
		"status":      e.Status,
		"createdAt":   e.CreatedAt.UTC().Format(time.RFC3339),
	}
	if e.CourseID != nil {
		out["courseId"] = e.CourseID.String()
	}
	if e.AffiliateCode != nil {
		out["affiliateCode"] = *e.AffiliateCode
	}
	return out
}

func (d Deps) handleCreatorEarningsSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		summary, err := repoBilling.EarningsSummaryForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load earnings.")
			return
		}
		connectID, _ := repoBilling.StripeConnectID(r.Context(), d.Pool, userID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pendingCents":      summary.PendingCents,
			"paidCents":         summary.PaidCents,
			"currency":          summary.Currency,
			"connectConfigured": connectID != "",
		})
	}
}

func (d Deps) handleCreatorEarningsLedger() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		limit := 20
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		var before *time.Time
		if v := strings.TrimSpace(r.URL.Query().Get("before")); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				before = &t
			}
		}
		rows, err := repoBilling.ListLedgerForUser(r.Context(), d.Pool, userID, limit, before)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load ledger.")
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, ledgerToJSON(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": items})
	}
}

func (d Deps) handleCreateAffiliateCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		var body struct {
			CourseID *string `json:"courseId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var courseID *uuid.UUID
		if body.CourseID != nil && strings.TrimSpace(*body.CourseID) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.CourseID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			courseID = &id
		}
		ac, err := repoBilling.CreateAffiliateCode(r.Context(), d.Pool, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create affiliate code.")
			return
		}
		origin := strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/")
		url := origin + "/explore"
		if courseID != nil {
			url += "/course/" + courseID.String()
		}
		url += "?ref=" + ac.Code
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        ac.ID.String(),
			"code":      ac.Code,
			"courseId":  nullableUUID(ac.CourseID),
			"url":       url,
			"createdAt": ac.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleListAffiliateCodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		codes, conversions, err := repoBilling.ListAffiliateCodesForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load affiliate codes.")
			return
		}
		origin := strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/")
		items := make([]map[string]any, 0, len(codes))
		for _, ac := range codes {
			url := origin + "/explore?ref=" + ac.Code
			items = append(items, map[string]any{
				"id":           ac.ID.String(),
				"code":         ac.Code,
				"courseId":     nullableUUID(ac.CourseID),
				"url":          url,
				"clickCount":   ac.ClickCount,
				"conversions":  conversions[ac.Code],
				"createdAt":    ac.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"codes": items})
	}
}

func (d Deps) handleConnectOnboarding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		email, err := svcBilling.LookupUserEmail(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "User not found.")
			return
		}
		cfg := svcBilling.ConnectConfig{
			SecretKey:       d.effectiveConfig().StripeSecretKey,
			PublicWebOrigin: d.effectiveConfig().PublicWebOrigin,
		}
		url, err := svcBilling.CreateConnectOnboardingLink(r.Context(), d.Pool, cfg, userID, email)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not start Connect onboarding.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"onboardingUrl": url})
	}
}

func (d Deps) handleConnectStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		connectID, err := repoBilling.StripeConnectID(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load Connect status.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connectConfigured": connectID != "",
			"connectAccountId":  connectID,
		})
	}
}

func (d Deps) handleAffiliateTrackClick() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.revenueShareFeatureOff(w) {
			return
		}
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" || d.Pool == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = repoBilling.IncrementAffiliateClickCount(r.Context(), d.Pool, code)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminRevenueSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		summary, err := repoBilling.PlatformRevenueOverview(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load revenue summary.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"totalSalesCents":     summary.TotalSalesCents,
			"totalCreatorCents":   summary.TotalCreatorCents,
			"totalAffiliateCents": summary.TotalAffiliateCents,
			"pendingPayoutCents":  summary.PendingPayoutCents,
		})
	}
}

func (d Deps) handleAdminTriggerPayout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.revenueShareFeatureOff(w) {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		cfg := svcBilling.ConnectConfig{
			SecretKey: d.effectiveConfig().StripeSecretKey,
		}
		result, err := svcBilling.RunMonthlyPayouts(r.Context(), d.Pool, cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Payout batch failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"processed": result.Processed,
			"failed":    result.Failed,
			"errors":    result.Errors,
		})
	}
}

func nullableUUID(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return id.String()
}
