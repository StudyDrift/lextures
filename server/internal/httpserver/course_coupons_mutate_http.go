// Creator coupon management mutate/list-redemptions handlers (plan MKTC.2).
package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// handlePatchCourseCoupon is PATCH /api/v1/courses/{course_code}/coupons/{coupon_id}.
func (d Deps) handlePatchCourseCoupon() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, listing, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("patch", "denied")
			return
		}
		if !d.checkCouponWriteRateLimit(viewer) {
			telemetry.RecordCouponAdminRequest("patch", "rate_limited")
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Coupon write rate limit exceeded. Try again in a minute.")
			return
		}
		couponID, err := uuid.Parse(chi.URLParam(r, "coupon_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid coupon id.")
			return
		}
		cur, err := repoBilling.GetCouponByCourseAndID(r.Context(), d.Pool, courseID, couponID)
		if err != nil {
			telemetry.RecordCouponAdminRequest("patch", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load coupon.")
			return
		}
		if cur == nil {
			telemetry.RecordCouponAdminRequest("patch", "not_found")
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Coupon not found.")
			return
		}
		if cur.Status == repoBilling.CouponStatusArchived {
			telemetry.RecordCouponAdminRequest("patch", "unprocessable")
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
				"Archived coupons cannot be updated. Create a new coupon instead.")
			return
		}

		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		// FR-4: immutable value fields after create.
		for _, key := range []string{"code", "discountType", "percentOff", "amountOffCents", "currency"} {
			if _, present := raw[key]; present {
				telemetry.RecordCouponAdminRequest("patch", "unprocessable")
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
					"Field \""+key+"\" cannot be changed after creation. Archive this coupon and create a new one.")
				return
			}
		}

		var body updateCouponBody
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		in := repoBilling.UpdateCouponInput{ID: couponID}
		changed := map[string]any{}

		if _, has := raw["startsAt"]; has {
			if body.StartsAt == nil || strings.TrimSpace(*body.StartsAt) == "" {
				in.ClearStartsAt = true
				changed["startsAt"] = nil
			} else {
				t, err := parseOptionalRFC3339("startsAt", body.StartsAt)
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
					return
				}
				in.StartsAt = t
				changed["startsAt"] = rfc3339Ptr(t)
			}
		}
		if _, has := raw["endsAt"]; has {
			if body.EndsAt == nil || strings.TrimSpace(*body.EndsAt) == "" {
				in.ClearEndsAt = true
				changed["endsAt"] = nil
			} else {
				t, err := parseOptionalRFC3339("endsAt", body.EndsAt)
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
					return
				}
				in.EndsAt = t
				changed["endsAt"] = rfc3339Ptr(t)
			}
		}

		// Resolve effective window for endsAt > startsAt check.
		effStarts := cur.StartsAt
		effEnds := cur.EndsAt
		if in.ClearStartsAt {
			effStarts = nil
		} else if in.StartsAt != nil {
			effStarts = in.StartsAt
		}
		if in.ClearEndsAt {
			effEnds = nil
		} else if in.EndsAt != nil {
			effEnds = in.EndsAt
		}
		if effStarts != nil && effEnds != nil && !effEnds.After(*effStarts) {
			telemetry.RecordCouponAdminRequest("patch", "unprocessable")
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
				"endsAt must be after startsAt.")
			return
		}

		if _, has := raw["maxRedemptions"]; has {
			if body.MaxRedemptions == nil {
				in.ClearMaxRedemptions = true
				changed["maxRedemptions"] = nil
			} else {
				if *body.MaxRedemptions <= 0 {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
						"maxRedemptions must be greater than 0 when set.")
					return
				}
				// FR-12: reject cap below current consumed seats.
				seatMap, err := repoBilling.CouponSeatCounts(r.Context(), d.Pool, []uuid.UUID{couponID})
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load seat counts.")
					return
				}
				consumed := seatMap[couponID].Consumed
				if *body.MaxRedemptions < consumed {
					telemetry.RecordCouponAdminRequest("patch", "unprocessable")
					apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
						"maxRedemptions cannot be lower than the current consumed seat count ("+strconv.Itoa(consumed)+").")
					return
				}
				in.MaxRedemptions = body.MaxRedemptions
				changed["maxRedemptions"] = *body.MaxRedemptions
			}
		}
		if _, has := raw["maxRedemptionsPerUser"]; has {
			if body.MaxRedemptionsPerUser == nil || *body.MaxRedemptionsPerUser < 1 || *body.MaxRedemptionsPerUser > 100 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"maxRedemptionsPerUser must be between 1 and 100.")
				return
			}
			in.MaxRedemptionsPerUser = body.MaxRedemptionsPerUser
			changed["maxRedemptionsPerUser"] = *body.MaxRedemptionsPerUser
		}
		if _, has := raw["note"]; has {
			if body.Note == nil || strings.TrimSpace(*body.Note) == "" {
				in.ClearNote = true
				changed["note"] = nil
			} else {
				n := strings.TrimSpace(*body.Note)
				in.Note = &n
				changed["note"] = n
			}
		}

		// Status transition (active|disabled only via PATCH; archive via DELETE).
		var newStatus string
		if body.Status != nil {
			s := strings.ToLower(strings.TrimSpace(*body.Status))
			switch s {
			case repoBilling.CouponStatusActive, repoBilling.CouponStatusDisabled:
				newStatus = s
			case repoBilling.CouponStatusArchived:
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
					"Use DELETE to archive a coupon.")
				return
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"status must be \"active\" or \"disabled\".")
				return
			}
		}

		// Apply field updates first (if any mutable fields present besides status).
		updated := cur
		hasFieldPatch := len(raw) > 0 && (len(raw) != 1 || body.Status == nil)
		if hasFieldPatch {
			// Only call Update when something other than status is present.
			fieldKeys := 0
			for k := range raw {
				if k != "status" {
					fieldKeys++
				}
			}
			if fieldKeys > 0 {
				u, err := repoBilling.UpdateCoupon(r.Context(), d.Pool, in)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Coupon not found.")
						return
					}
					telemetry.RecordCouponAdminRequest("patch", "error")
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update coupon.")
					return
				}
				updated = u
			}
		}
		if newStatus != "" && newStatus != updated.Status {
			if err := repoBilling.SetCouponStatus(r.Context(), d.Pool, couponID, newStatus); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Coupon not found.")
					return
				}
				telemetry.RecordCouponAdminRequest("patch", "error")
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update coupon status.")
				return
			}
			updated.Status = newStatus
			changed["status"] = newStatus
			telemetry.RecordCouponStatusChanged(newStatus)
		}

		// Reload for consistent timestamps.
		reloaded, err := repoBilling.GetCouponByCourseAndID(r.Context(), d.Pool, courseID, couponID)
		if err != nil || reloaded == nil {
			reloaded = updated
		}
		seatMap, _ := repoBilling.CouponSeatCounts(r.Context(), d.Pool, []uuid.UUID{couponID})
		seats := seatMap[couponID]
		var warnings []string
		if couponClampsToFree(listing, reloaded) {
			warnings = append(warnings, "clamps_to_free")
		}
		cfg := d.effectiveConfig()
		share, pub := couponShareURLs(cfg.PublicWebOrigin, cfg.MarketingSiteOrigin, courseCode, listing.Slug, listing.IsPublic, reloaded.Code)
		dto := couponToJSON(reloaded, seats, share, pub, warnings)

		if len(changed) > 0 {
			after := map[string]any{
				"courseCode": courseCode,
				"couponId":   couponID.String(),
				"changed":    changed,
			}
			d.auditCoupon(r, viewer, "course.coupon.updated", couponID, after)
		}
		telemetry.RecordCouponAdminRequest("patch", "ok")
		log.Printf("coupon patch course=%s coupon=%s actor=%s", courseCode, couponID, viewer)
		writeJSON(w, http.StatusOK, map[string]any{"coupon": dto})
	}
}

// handleDeleteCourseCoupon is DELETE /api/v1/courses/{course_code}/coupons/{coupon_id} (soft archive).
func (d Deps) handleDeleteCourseCoupon() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, listing, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("delete", "denied")
			return
		}
		if !d.checkCouponWriteRateLimit(viewer) {
			telemetry.RecordCouponAdminRequest("delete", "rate_limited")
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Coupon write rate limit exceeded. Try again in a minute.")
			return
		}
		couponID, err := uuid.Parse(chi.URLParam(r, "coupon_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid coupon id.")
			return
		}
		cur, err := repoBilling.GetCouponByCourseAndID(r.Context(), d.Pool, courseID, couponID)
		if err != nil {
			telemetry.RecordCouponAdminRequest("delete", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load coupon.")
			return
		}
		if cur == nil {
			telemetry.RecordCouponAdminRequest("delete", "not_found")
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Coupon not found.")
			return
		}
		if cur.Status != repoBilling.CouponStatusArchived {
			if err := repoBilling.SetCouponStatus(r.Context(), d.Pool, couponID, repoBilling.CouponStatusArchived); err != nil {
				telemetry.RecordCouponAdminRequest("delete", "error")
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to archive coupon.")
				return
			}
			telemetry.RecordCouponStatusChanged(repoBilling.CouponStatusArchived)
			d.auditCoupon(r, viewer, "course.coupon.archived", couponID, map[string]any{
				"courseCode": courseCode,
				"couponId":   couponID.String(),
				"code":       cur.Code,
			})
		}
		cur.Status = repoBilling.CouponStatusArchived
		seatMap, _ := repoBilling.CouponSeatCounts(r.Context(), d.Pool, []uuid.UUID{couponID})
		cfg := d.effectiveConfig()
		share, pub := couponShareURLs(cfg.PublicWebOrigin, cfg.MarketingSiteOrigin, courseCode, listing.Slug, listing.IsPublic, cur.Code)
		dto := couponToJSON(cur, seatMap[couponID], share, pub, nil)
		telemetry.RecordCouponAdminRequest("delete", "ok")
		log.Printf("coupon archive course=%s coupon=%s actor=%s", courseCode, couponID, viewer)
		writeJSON(w, http.StatusOK, map[string]any{"coupon": dto})
	}
}

func formatPercentCeiling(v float64) string {
	// Prefer integer display when whole number.
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// handleListCouponRedemptions is GET .../coupons/{coupon_id}/redemptions.
func (d Deps) handleListCouponRedemptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, courseID, _, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("redemptions", "denied")
			return
		}
		couponID, err := uuid.Parse(chi.URLParam(r, "coupon_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid coupon id.")
			return
		}
		cur, err := repoBilling.GetCouponByCourseAndID(r.Context(), d.Pool, courseID, couponID)
		if err != nil {
			telemetry.RecordCouponAdminRequest("redemptions", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load coupon.")
			return
		}
		if cur == nil {
			telemetry.RecordCouponAdminRequest("redemptions", "not_found")
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Coupon not found.")
			return
		}

		limit := 25
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "limit must be a positive integer.")
				return
			}
			if n > 100 {
				n = 100
			}
			limit = n
		}
		cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))

		rows, next, err := repoBilling.ListCouponRedemptions(r.Context(), d.Pool, couponID, cursor, limit)
		if err != nil {
			if strings.Contains(err.Error(), "invalid cursor") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
				return
			}
			telemetry.RecordCouponAdminRequest("redemptions", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list redemptions.")
			return
		}

		out := make([]couponRedemptionJSON, 0, len(rows))
		for i := range rows {
			rd := &rows[i]
			var userName, userEmail *string
			u, uerr := user.FindByID(r.Context(), d.Pool, rd.UserID)
			if uerr == nil && u != nil {
				userEmail = &u.Email
				if u.DisplayName != nil && strings.TrimSpace(*u.DisplayName) != "" {
					userName = u.DisplayName
				}
			}
			item := couponRedemptionJSON{
				ID:             rd.ID.String(),
				UserID:         rd.UserID.String(),
				UserName:       userName,
				UserEmail:      userEmail,
				Status:         rd.Status,
				ListPriceCents: rd.ListPriceCents,
				DiscountCents:  rd.DiscountCents,
				ChargedCents:   rd.ChargedCents,
				Currency:       rd.Currency,
				ReservedAt:     rd.ReservedAt.UTC().Format(time.RFC3339),
				RedeemedAt:     rfc3339Ptr(rd.RedeemedAt),
			}
			out = append(out, item)
		}
		telemetry.RecordCouponAdminRequest("redemptions", "ok")
		resp := map[string]any{"redemptions": out, "nextCursor": next}
		writeJSON(w, http.StatusOK, resp)
	}
}
