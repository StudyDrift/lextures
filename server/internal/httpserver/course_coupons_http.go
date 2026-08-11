// Creator coupon management API (plan MKTC.2).
package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/coupons"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const couponWriteRateLimitPerMinute = 30

var (
	couponRateMu    sync.Mutex
	couponRateByUID = map[uuid.UUID]billingRateEntry{}
)

func (d Deps) checkCouponWriteRateLimit(userID uuid.UUID) bool {
	couponRateMu.Lock()
	defer couponRateMu.Unlock()
	now := time.Now()
	e, ok := couponRateByUID[userID]
	if !ok || now.Sub(e.windowStart) >= time.Minute {
		couponRateByUID[userID] = billingRateEntry{windowStart: now, count: 1}
		return true
	}
	if e.count >= couponWriteRateLimitPerMinute {
		return false
	}
	e.count++
	couponRateByUID[userID] = e
	return true
}

// --- JSON DTOs ----------------------------------------------------------------

type couponSeatsJSON struct {
	Consumed  int  `json:"consumed"`
	Reserved  int  `json:"reserved"`
	Redeemed  int  `json:"redeemed"`
	Remaining *int `json:"remaining"` // null = unlimited
}

type courseCouponJSON struct {
	ID                    string          `json:"id"`
	CourseID              string          `json:"courseId"`
	Code                  string          `json:"code"`
	DiscountType          string          `json:"discountType"`
	PercentOff            *float64        `json:"percentOff"`
	AmountOffCents        *int            `json:"amountOffCents"`
	Currency              *string         `json:"currency"`
	StartsAt              *string         `json:"startsAt"`
	EndsAt                *string         `json:"endsAt"`
	MaxRedemptions        *int            `json:"maxRedemptions"`
	MaxRedemptionsPerUser int             `json:"maxRedemptionsPerUser"`
	Seats                 couponSeatsJSON `json:"seats"`
	Status                string          `json:"status"`
	Note                  *string         `json:"note"`
	ShareURL              string          `json:"shareUrl"`
	PublicShareURL        *string         `json:"publicShareUrl"`
	CreatedBy             *string         `json:"createdBy"`
	CreatedAt             string          `json:"createdAt"`
	UpdatedAt             string          `json:"updatedAt"`
	Warnings              []string        `json:"warnings,omitempty"`
}

type createCouponBody struct {
	Code                  string   `json:"code"`
	DiscountType          string   `json:"discountType"`
	PercentOff            *float64 `json:"percentOff"`
	AmountOffCents        *int     `json:"amountOffCents"`
	Currency              *string  `json:"currency"`
	StartsAt              *string  `json:"startsAt"`
	EndsAt                *string  `json:"endsAt"`
	MaxRedemptions        *int     `json:"maxRedemptions"`
	MaxRedemptionsPerUser *int     `json:"maxRedemptionsPerUser"`
	Note                  *string  `json:"note"`
}

type updateCouponBody struct {
	StartsAt              *string `json:"startsAt"`
	EndsAt                *string `json:"endsAt"`
	MaxRedemptions        *int    `json:"maxRedemptions"`
	MaxRedemptionsPerUser *int    `json:"maxRedemptionsPerUser"`
	Note                  *string `json:"note"`
	Status                *string `json:"status"`
	// clear* helpers: JSON null for optional time/cap/note clears the field.
	// Detected via raw map in the handler.
}

type couponRedemptionJSON struct {
	ID             string  `json:"id"`
	UserID         string  `json:"userId"`
	UserName       *string `json:"userName"`
	UserEmail      *string `json:"userEmail"`
	Status         string  `json:"status"`
	ListPriceCents int     `json:"listPriceCents"`
	DiscountCents  int     `json:"discountCents"`
	ChargedCents   int     `json:"chargedCents"`
	Currency       string  `json:"currency"`
	ReservedAt     string  `json:"reservedAt"`
	RedeemedAt     *string `json:"redeemedAt"`
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func parseOptionalRFC3339(label string, raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Accept RFC3339Nano too.
		t, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, errors.New(label + " must be an RFC 3339 UTC timestamp.")
		}
	}
	u := t.UTC()
	return &u, nil
}

func couponShareURLs(publicWebOrigin, marketingOrigin, courseCode, slug string, isPublic bool, code string) (shareURL string, publicShareURL *string) {
	hint := marketplaceCheckoutHint(slug, courseCode)
	origin := strings.TrimRight(strings.TrimSpace(publicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	shareURL = origin + hint + "?coupon=" + url.QueryEscape(code)
	if isPublic {
		mOrigin := strings.TrimRight(strings.TrimSpace(marketingOrigin), "/")
		if mOrigin == "" {
			mOrigin = "https://lextures.com"
		}
		s := strings.TrimSpace(slug)
		if s == "" {
			s = strings.TrimSpace(courseCode)
		}
		pub := mOrigin + "/courses/" + url.PathEscape(s) + "?coupon=" + url.QueryEscape(code)
		publicShareURL = &pub
	}
	return shareURL, publicShareURL
}

func couponClampsToFree(listing *course.CatalogListing, c *repoBilling.Coupon) bool {
	if listing == nil || c == nil || listing.PriceCents <= 0 {
		return false
	}
	domain := c.ToDomain()
	q := coupons.ApplyDiscount(listing.PriceCents, listing.PriceCurrency, domain)
	return q.ClampedToFree
}

func couponToJSON(
	c *repoBilling.Coupon,
	seats repoBilling.SeatCount,
	shareURL string,
	publicShareURL *string,
	warnings []string,
) courseCouponJSON {
	var remaining *int
	if c.MaxRedemptions != nil {
		r := *c.MaxRedemptions - seats.Consumed
		if r < 0 {
			r = 0
		}
		remaining = &r
	}
	var createdBy *string
	if c.CreatedBy != nil {
		s := c.CreatedBy.String()
		createdBy = &s
	}
	out := courseCouponJSON{
		ID:                    c.ID.String(),
		CourseID:              c.CourseID.String(),
		Code:                  c.Code,
		DiscountType:          c.DiscountType,
		PercentOff:            c.PercentOff,
		AmountOffCents:        c.AmountOffCents,
		Currency:              c.Currency,
		StartsAt:              rfc3339Ptr(c.StartsAt),
		EndsAt:                rfc3339Ptr(c.EndsAt),
		MaxRedemptions:        c.MaxRedemptions,
		MaxRedemptionsPerUser: c.MaxRedemptionsPerUser,
		Seats: couponSeatsJSON{
			Consumed:  seats.Consumed,
			Reserved:  seats.Reserved,
			Redeemed:  seats.Redeemed,
			Remaining: remaining,
		},
		Status:         c.Status,
		Note:           c.Note,
		ShareURL:       shareURL,
		PublicShareURL: publicShareURL,
		CreatedBy:      createdBy,
		CreatedAt:      c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      c.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if len(warnings) > 0 {
		out.Warnings = warnings
	}
	return out
}

// --- Auth helpers -------------------------------------------------------------

func (d Deps) requireCouponManager(w http.ResponseWriter, r *http.Request) (
	courseCode string, viewer uuid.UUID, courseID uuid.UUID, listing *course.CatalogListing, ok bool,
) {
	if d.couponsFeatureOff(w) {
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage coupons for this course.")
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	listing, err = course.GetCatalogListing(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course listing.")
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	if listing == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, uuid.UUID{}, nil, false
	}
	return courseCode, viewer, *cid, listing, true
}

func (d Deps) auditCoupon(r *http.Request, actor uuid.UUID, eventType string, couponID uuid.UUID, after map[string]any) {
	if d.Pool == nil {
		return
	}
	targetType := "course_coupon"
	var afterBytes []byte
	if after != nil {
		afterBytes, _ = json.Marshal(after)
	}
	if _, err := adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
		EventType:  eventType,
		ActorID:    actor,
		TargetType: &targetType,
		TargetID:   &couponID,
		AfterValue: afterBytes,
	}); err != nil {
		log.Printf("coupon audit: event=%s coupon=%s err=%v", eventType, couponID, err)
	}
}

// --- Handlers -----------------------------------------------------------------

// handleListCourseCoupons is GET /api/v1/courses/{course_code}/coupons.
func (d Deps) handleListCourseCoupons() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, courseID, listing, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("list", "denied")
			return
		}
		includeArchived := strings.EqualFold(r.URL.Query().Get("includeArchived"), "true") ||
			r.URL.Query().Get("includeArchived") == "1"
		rows, err := repoBilling.ListCouponsByCourse(r.Context(), d.Pool, courseID, includeArchived)
		if err != nil {
			telemetry.RecordCouponAdminRequest("list", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list coupons.")
			return
		}
		ids := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].ID)
		}
		seatMap, err := repoBilling.CouponSeatCounts(r.Context(), d.Pool, ids)
		if err != nil {
			telemetry.RecordCouponAdminRequest("list", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load seat counts.")
			return
		}
		cfg := d.effectiveConfig()
		out := make([]courseCouponJSON, 0, len(rows))
		for i := range rows {
			c := &rows[i]
			seats := seatMap[c.ID]
			share, pub := couponShareURLs(cfg.PublicWebOrigin, cfg.MarketingSiteOrigin, courseCode, listing.Slug, listing.IsPublic, c.Code)
			out = append(out, couponToJSON(c, seats, share, pub, nil))
		}
		telemetry.RecordCouponAdminRequest("list", "ok")
		writeJSON(w, http.StatusOK, map[string]any{"coupons": out})
	}
}

// handleCreateCourseCoupon is POST /api/v1/courses/{course_code}/coupons.
func (d Deps) handleCreateCourseCoupon() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, listing, ok := d.requireCouponManager(w, r)
		if !ok {
			telemetry.RecordCouponAdminRequest("create", "denied")
			return
		}
		if !d.checkCouponWriteRateLimit(viewer) {
			telemetry.RecordCouponAdminRequest("create", "rate_limited")
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Coupon write rate limit exceeded. Try again in a minute.")
			return
		}
		if listing.PriceCents == 0 {
			telemetry.RecordCouponAdminRequest("create", "unprocessable")
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
				"This course is free; coupons apply to paid courses only.")
			return
		}

		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		var body createCouponBody
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		code := coupons.NormalizeCode(body.Code)
		if err := coupons.ValidateCode(code); err != nil {
			telemetry.RecordCouponAdminRequest("create", "invalid_input")
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"Coupon code must be 4–32 characters matching [A-Z0-9][A-Z0-9_-]* after normalization (letters, digits, underscore, hyphen; cannot start with a separator).")
			return
		}

		dtype := strings.ToLower(strings.TrimSpace(body.DiscountType))
		switch dtype {
		case "percent":
			if body.PercentOff == nil || *body.PercentOff <= 0 || *body.PercentOff > 100 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"percentOff is required for percent coupons and must be greater than 0 and at most 100.")
				return
			}
			// FR-5 (MKTC.7): platform discount ceiling.
			ceiling := d.effectiveConfig().CouponMaxPercentOff
			if ceiling <= 0 {
				ceiling = 100
			}
			if *body.PercentOff > ceiling {
				telemetry.RecordCouponAdminRequest("create", "unprocessable")
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
					"percentOff exceeds the platform maximum coupon discount ("+formatPercentCeiling(ceiling)+"%).")
				return
			}
			if body.AmountOffCents != nil || (body.Currency != nil && strings.TrimSpace(*body.Currency) != "") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"amountOffCents and currency must not be set on percent coupons.")
				return
			}
		case "fixed":
			if body.AmountOffCents == nil || *body.AmountOffCents <= 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"amountOffCents is required for fixed coupons and must be greater than 0.")
				return
			}
			if body.PercentOff != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"percentOff must not be set on fixed coupons.")
				return
			}
			curr := ""
			if body.Currency != nil {
				curr = course.NormalizePriceCurrency(*body.Currency)
			}
			if curr == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"currency is required for fixed coupons.")
				return
			}
			if !course.ValidPriceCurrency(curr) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported currency.")
				return
			}
			if curr != course.NormalizePriceCurrency(listing.PriceCurrency) {
				telemetry.RecordCouponAdminRequest("create", "unprocessable")
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
					"Fixed coupon currency must match the course price currency ("+course.NormalizePriceCurrency(listing.PriceCurrency)+").")
				return
			}
			body.Currency = &curr
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"discountType must be \"percent\" or \"fixed\".")
			return
		}

		startsAt, err := parseOptionalRFC3339("startsAt", body.StartsAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		endsAt, err := parseOptionalRFC3339("endsAt", body.EndsAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if startsAt != nil && endsAt != nil && !endsAt.After(*startsAt) {
			telemetry.RecordCouponAdminRequest("create", "unprocessable")
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity,
				"endsAt must be after startsAt.")
			return
		}
		if body.MaxRedemptions != nil && *body.MaxRedemptions <= 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"maxRedemptions must be greater than 0 when set.")
			return
		}
		perUser := 1
		if body.MaxRedemptionsPerUser != nil {
			if *body.MaxRedemptionsPerUser < 1 || *body.MaxRedemptionsPerUser > 100 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"maxRedemptionsPerUser must be between 1 and 100.")
				return
			}
			perUser = *body.MaxRedemptionsPerUser
		}
		var note *string
		if body.Note != nil {
			n := strings.TrimSpace(*body.Note)
			if n != "" {
				note = &n
			}
		}

		// Pre-check unique non-archived code for a clear 409 (FR-10).
		existing, err := repoBilling.GetCouponByCourseAndCode(r.Context(), d.Pool, courseID, code)
		if err != nil {
			telemetry.RecordCouponAdminRequest("create", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check existing coupons.")
			return
		}
		if existing != nil {
			telemetry.RecordCouponAdminRequest("create", "conflict")
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict,
				"A coupon with code \""+existing.Code+"\" already exists on this course.")
			return
		}

		created, err := repoBilling.CreateCoupon(r.Context(), d.Pool, repoBilling.CreateCouponInput{
			CourseID:              courseID,
			Code:                  code,
			DiscountType:          dtype,
			PercentOff:            body.PercentOff,
			AmountOffCents:        body.AmountOffCents,
			Currency:              body.Currency,
			StartsAt:              startsAt,
			EndsAt:                endsAt,
			MaxRedemptions:        body.MaxRedemptions,
			MaxRedemptionsPerUser: perUser,
			Note:                  note,
			CreatedBy:             &viewer,
			Status:                repoBilling.CouponStatusActive,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				telemetry.RecordCouponAdminRequest("create", "conflict")
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict,
					"A coupon with code \""+code+"\" already exists on this course.")
				return
			}
			// CHECK constraint violations → 400/422 with a clear sentence.
			if errors.As(err, &pgErr) && pgErr.Code == "23514" {
				telemetry.RecordCouponAdminRequest("create", "invalid_input")
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
					"Coupon failed validation: check discount fields, window, and caps.")
				return
			}
			telemetry.RecordCouponAdminRequest("create", "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create coupon.")
			return
		}

		var warnings []string
		if couponClampsToFree(listing, created) {
			warnings = append(warnings, "clamps_to_free")
		}
		warnings = append(warnings, coupons.LowEntropyWarnings(created.Code)...)
		cfg := d.effectiveConfig()
		share, pub := couponShareURLs(cfg.PublicWebOrigin, cfg.MarketingSiteOrigin, courseCode, listing.Slug, listing.IsPublic, created.Code)
		dto := couponToJSON(created, repoBilling.SeatCount{CouponID: created.ID}, share, pub, warnings)

		d.auditCoupon(r, viewer, "course.coupon.created", created.ID, map[string]any{
			"courseCode":   courseCode,
			"couponId":     created.ID.String(),
			"code":         created.Code,
			"discountType": created.DiscountType,
		})
		telemetry.RecordCouponCreated(created.DiscountType)
		telemetry.RecordCouponAdminRequest("create", "ok")
		log.Printf("coupon create course=%s coupon=%s actor=%s", courseCode, created.ID, viewer)
		writeJSON(w, http.StatusCreated, map[string]any{"coupon": dto})
	}
}
