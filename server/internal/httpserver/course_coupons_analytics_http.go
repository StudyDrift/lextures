// Coupon performance summary and CSV export (plan MKTC.7).
package httpserver

import (
	"encoding/csv"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/user"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// handleCouponSummary is GET /api/v1/courses/{course_code}/coupons/summary (MKTC.7 FR-7).
func (d Deps) handleCouponSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, courseID, listing, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("summary", "denied")
			return
		}
		rows, err := repoBilling.CouponSummaryByCourse(r.Context(), d.Pool, courseID)
		if err != nil {
			telemetry.RecordCouponAdminRequest("summary", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load coupon summary.")
			return
		}
		currency := ""
		if listing != nil {
			currency = course.NormalizePriceCurrency(listing.PriceCurrency)
		}
		out := make([]map[string]any, 0, len(rows))
		for i := range rows {
			row := &rows[i]
			curr := row.Currency
			if curr == "" {
				curr = currency
			}
			item := map[string]any{
				"couponId":        row.CouponID.String(),
				"code":            row.Code,
				"redeemedCount":   row.RedeemedCount,
				"refundedCount":   row.RefundedCount,
				"grossListCents":  row.GrossListCents,
				"discountCents":   row.DiscountCents,
				"netChargedCents": row.NetChargedCents,
				"currency":        curr,
				"firstRedeemedAt": rfc3339Ptr(row.FirstRedeemedAt),
				"lastRedeemedAt":  rfc3339Ptr(row.LastRedeemedAt),
			}
			out = append(out, item)
		}
		if currency == "" && len(out) > 0 {
			if c, _ := out[0]["currency"].(string); c != "" {
				currency = c
			}
		}
		if currency == "" {
			currency = "usd"
		}
		telemetry.RecordCouponAdminRequest("summary", "ok")
		writeJSON(w, http.StatusOK, map[string]any{"rows": out, "currency": currency})
	}
}

// handleExportCouponRedemptionsCSV is GET .../coupons/{coupon_id}/redemptions.csv (MKTC.7 FR-9).
func (d Deps) handleExportCouponRedemptionsCSV() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, _, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("export_csv", "denied")
			return
		}
		if allowed, retry := svcBilling.CheckCouponExportRateLimit(viewer, time.Now()); !allowed {
			secs := int(retry.Round(time.Second) / time.Second)
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			telemetry.RecordCouponAdminRequest("export_csv", "rate_limited")
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited,
				"Coupon redemptions export rate limit exceeded (5 per hour). Try again later.")
			return
		}
		couponID, err := uuid.Parse(chi.URLParam(r, "coupon_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid coupon id.")
			return
		}
		cur, err := repoBilling.GetCouponByCourseAndID(r.Context(), d.Pool, courseID, couponID)
		if err != nil {
			telemetry.RecordCouponAdminRequest("export_csv", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load coupon.")
			return
		}
		if cur == nil {
			telemetry.RecordCouponAdminRequest("export_csv", "not_found")
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Coupon not found.")
			return
		}

		redemptions, err := repoBilling.StreamCouponRedemptionsForExport(r.Context(), d.Pool, couponID)
		if err != nil {
			telemetry.RecordCouponAdminRequest("export_csv", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load redemptions.")
			return
		}

		d.auditCoupon(r, viewer, "course.coupon.redemptions_exported", couponID, map[string]any{
			"courseCode": courseCode,
			"couponId":   couponID.String(),
			"code":       cur.Code,
			"rowCount":   len(redemptions),
		})

		filename := "coupon-" + cur.Code + "-redemptions.csv"
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.WriteHeader(http.StatusOK)

		cw := csv.NewWriter(w)
		_ = cw.Write([]string{
			"redeemed_at", "status", "learner_name", "learner_email", "code",
			"list_price_cents", "discount_cents", "charged_cents", "currency",
		})
		for i := range redemptions {
			rd := &redemptions[i]
			learnerName, learnerEmail := "", ""
			u, uerr := user.FindByID(r.Context(), d.Pool, rd.UserID)
			if uerr == nil && u != nil {
				learnerEmail = u.Email
				if u.DisplayName != nil {
					learnerName = strings.TrimSpace(*u.DisplayName)
				}
			}
			redeemedAt := ""
			if rd.RedeemedAt != nil {
				redeemedAt = rd.RedeemedAt.UTC().Format(time.RFC3339)
			} else if rd.Status == repoBilling.RedemptionReserved {
				redeemedAt = rd.ReservedAt.UTC().Format(time.RFC3339)
			}
			_ = cw.Write([]string{
				redeemedAt,
				rd.Status,
				learnerName,
				learnerEmail,
				cur.Code,
				strconv.Itoa(rd.ListPriceCents),
				strconv.Itoa(rd.DiscountCents),
				strconv.Itoa(rd.ChargedCents),
				rd.Currency,
			})
		}
		cw.Flush()
		telemetry.RecordCouponAdminRequest("export_csv", "ok")
		log.Printf("coupon redemptions export course=%s coupon=%s actor=%s rows=%d", courseCode, couponID, viewer, len(redemptions))
	}
}
