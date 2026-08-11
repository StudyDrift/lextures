package billing

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/coupons"
)

func couponTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("short")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedCouponUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	em := "mktc1-" + uuid.NewString()[:8] + "@example.test"
	ph, err := auth.HashPassword("Mktc1-test-" + uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, err := uuid.Parse(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user".users WHERE id = $1`, id)
	})
	return id
}

func seedCouponCourse(t *testing.T, pool *pgxpool.Pool, priceCents int, currency string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	courseID := uuid.New()
	code := "C-" + strings.ToUpper(strings.ReplaceAll(courseID.String(), "-", "")[:6])
	if currency == "" {
		currency = "usd"
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, price_cents, price_currency, marketplace_listed)
VALUES ($1, $2, 'MKTC.1 test', $3, $4, TRUE)
`, courseID, code, priceCents, currency)
	if err != nil {
		t.Fatalf("seed course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})
	return courseID
}

func TestCreateListGetCoupon_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	userID := seedCouponUser(t, pool)

	// AC-11: lower-case + spaces → LAUNCH25
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "  launch25  ",
		DiscountType: "percent",
		PercentOff:   ptrF(25),
		CreatedBy:    &userID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.Code != "LAUNCH25" {
		t.Fatalf("normalized code: got %q", c.Code)
	}

	// Duplicate normalized code violates unique index among non-archived.
	_, err = CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "Launch25",
		DiscountType: "percent",
		PercentOff:   ptrF(10),
	})
	if err == nil {
		t.Fatal("expected unique violation for duplicate code")
	}

	got, err := GetCouponByCourseAndCode(ctx, pool, courseID, " launch25 ")
	if err != nil || got == nil {
		t.Fatalf("get by code: %v %#v", err, got)
	}
	if got.ID != c.ID {
		t.Fatal("id mismatch")
	}

	list, err := ListCouponsByCourse(ctx, pool, courseID, false)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: err=%v n=%d", err, len(list))
	}

	// Archive frees the code for re-issue.
	if err := SetCouponStatus(ctx, pool, c.ID, CouponStatusArchived); err != nil {
		t.Fatal(err)
	}
	c2, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "LAUNCH25",
		DiscountType: "percent",
		PercentOff:   ptrF(15),
	})
	if err != nil {
		t.Fatalf("re-issue after archive: %v", err)
	}
	if c2.ID == c.ID {
		t.Fatal("expected new coupon row")
	}
}

func TestReserveConcurrency_Pg(t *testing.T) {
	// AC-5: max_redemptions=1, two concurrent Reserves → exactly one ok.
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	max := 1
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:       courseID,
		Code:           "ONLYONE",
		DiscountType:   "percent",
		PercentOff:     ptrF(50),
		MaxRedemptions: &max,
	})
	if err != nil {
		t.Fatal(err)
	}
	u1 := seedCouponUser(t, pool)
	u2 := seedCouponUser(t, pool)

	type result struct {
		reason coupons.Reason
		err    error
	}
	var wg sync.WaitGroup
	ch := make(chan result, 2)
	start := make(chan struct{})
	for _, uid := range []uuid.UUID{u1, u2} {
		wg.Add(1)
		go func(userID uuid.UUID) {
			defer wg.Done()
			<-start
			_, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
				CouponID:         c.ID,
				UserID:           userID,
				CoursePriceCents: 4000,
				CourseCurrency:   "usd",
			})
			ch <- result{reason: reason, err: err}
		}(uid)
	}
	close(start)
	wg.Wait()
	close(ch)

	var ok, exhausted int
	for r := range ch {
		if r.err != nil {
			t.Fatalf("reserve err: %v", r.err)
		}
		switch r.reason {
		case coupons.ReasonOK:
			ok++
		case coupons.ReasonExhausted:
			exhausted++
		default:
			t.Fatalf("unexpected reason %s", r.reason)
		}
	}
	if ok != 1 || exhausted != 1 {
		t.Fatalf("AC-5: ok=%d exhausted=%d want 1/1", ok, exhausted)
	}
}

func TestReserveExpiredSeatFreesWithoutSweeper_Pg(t *testing.T) {
	// AC-6: expired reservation frees seat without sweeper.
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	max := 1
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:       courseID,
		Code:           "TTLTEST",
		DiscountType:   "percent",
		PercentOff:     ptrF(10),
		MaxRedemptions: &max,
	})
	if err != nil {
		t.Fatal(err)
	}
	u1 := seedCouponUser(t, pool)
	u2 := seedCouponUser(t, pool)

	// Insert an already-expired reservation directly.
	past := time.Now().UTC().Add(-time.Hour)
	_, err = pool.Exec(ctx, `
INSERT INTO billing.coupon_redemptions (
    coupon_id, course_id, user_id, status,
    list_price_cents, discount_cents, charged_cents, currency,
    reserved_at, expires_at
) VALUES ($1,$2,$3,'reserved',4000,400,3600,'usd',$4,$4)
`, c.ID, courseID, u1, past)
	if err != nil {
		t.Fatal(err)
	}

	// New learner can reserve — seat available because expired rows don't count.
	r, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
		CouponID:         c.ID,
		UserID:           u2,
		CoursePriceCents: 4000,
		CourseCurrency:   "usd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reason != coupons.ReasonOK || r == nil {
		t.Fatalf("AC-6: reason=%s r=%v", reason, r)
	}
}

func TestRedeemIdempotent_Pg(t *testing.T) {
	// AC-7
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "REDEEM1",
		DiscountType: "percent",
		PercentOff:   ptrF(25),
	})
	if err != nil {
		t.Fatal(err)
	}
	userID := seedCouponUser(t, pool)
	res, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
		CouponID:         c.ID,
		UserID:           userID,
		CoursePriceCents: 4000,
		CourseCurrency:   "usd",
	})
	if err != nil || reason != coupons.ReasonOK {
		t.Fatalf("reserve: %v %s", err, reason)
	}
	eventID := "evt_coupon_" + uuid.NewString()
	r1, created1, err := RedeemCoupon(ctx, pool, RedeemInput{
		RedemptionID:    res.ID,
		ProviderEventID: eventID,
	})
	if err != nil || !created1 {
		t.Fatalf("first redeem: err=%v created=%v", err, created1)
	}
	r2, created2, err := RedeemCoupon(ctx, pool, RedeemInput{
		RedemptionID:    res.ID,
		ProviderEventID: eventID,
	})
	if err != nil || created2 {
		t.Fatalf("second redeem: err=%v created=%v", err, created2)
	}
	if r1.ID != r2.ID {
		t.Fatal("id mismatch on idempotent redeem")
	}

	got, err := GetCouponByID(ctx, pool, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RedeemedCount != 1 {
		t.Fatalf("redeemed_count=%d want 1", got.RedeemedCount)
	}

	// Exactly one redeemed row.
	var n int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM billing.coupon_redemptions WHERE coupon_id = $1 AND status = 'redeemed'
`, c.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("redeemed rows=%d", n)
	}
}

func TestReleaseAndSeatCounts_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "RELEASE1",
		DiscountType: "fixed",
		AmountOffCents: ptrI(500),
		Currency:     ptrS("usd"),
	})
	if err != nil {
		t.Fatal(err)
	}
	userID := seedCouponUser(t, pool)
	res, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
		CouponID:         c.ID,
		UserID:           userID,
		CoursePriceCents: 4000,
		CourseCurrency:   "usd",
	})
	if err != nil || reason != coupons.ReasonOK {
		t.Fatalf("reserve: %v %s", err, reason)
	}

	counts, err := CouponSeatCounts(ctx, pool, []uuid.UUID{c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if counts[c.ID].Consumed != 1 || counts[c.ID].Reserved != 1 {
		t.Fatalf("counts after reserve: %+v", counts[c.ID])
	}

	if err := ReleaseCouponReservation(ctx, pool, res.ID, "abandon"); err != nil {
		t.Fatal(err)
	}
	counts, err = CouponSeatCounts(ctx, pool, []uuid.UUID{c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if counts[c.ID].Consumed != 0 {
		t.Fatalf("counts after release: %+v", counts[c.ID])
	}
}

func TestReleaseExpiredReservations_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "SWEEP1",
		DiscountType: "percent",
		PercentOff:   ptrF(10),
	})
	if err != nil {
		t.Fatal(err)
	}
	userID := seedCouponUser(t, pool)
	past := time.Now().UTC().Add(-time.Hour)
	_, err = pool.Exec(ctx, `
INSERT INTO billing.coupon_redemptions (
    coupon_id, course_id, user_id, status,
    list_price_cents, discount_cents, charged_cents, currency,
    reserved_at, expires_at
) VALUES ($1,$2,$3,'reserved',4000,400,3600,'usd',$4,$4)
`, c.ID, courseID, userID, past)
	if err != nil {
		t.Fatal(err)
	}
	n, err := ReleaseExpiredCouponReservations(ctx, pool, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected >=1 released, got %d", n)
	}
}

func TestCascadeCourseDelete_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	// Don't use cleanup-deleting course; we delete it ourselves.
	// Re-seed without cleanup for cascade proof.
	courseID2 := uuid.New()
	code := "C-" + strings.ToUpper(strings.ReplaceAll(courseID2.String(), "-", "")[:6])
	_, err := pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, price_cents, price_currency, marketplace_listed)
VALUES ($1, $2, 'MKTC cascade', 4000, 'usd', TRUE)
`, courseID2, code)
	if err != nil {
		t.Fatal(err)
	}
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID2,
		Code:         "CASCADE1",
		DiscountType: "percent",
		PercentOff:   ptrF(20),
	})
	if err != nil {
		t.Fatal(err)
	}
	userID := seedCouponUser(t, pool)
	_, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
		CouponID:         c.ID,
		UserID:           userID,
		CoursePriceCents: 4000,
		CourseCurrency:   "usd",
	})
	if err != nil || reason != coupons.ReasonOK {
		t.Fatalf("reserve: %v %s", err, reason)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM course.courses WHERE id = $1`, courseID2); err != nil {
		t.Fatal(err)
	}
	got, err := GetCouponByID(ctx, pool, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("coupon should cascade-delete with course")
	}
	_ = courseID // keep seed cleanup happy if used
}

func TestUserDeleteCascadesRedemptionsNotCoupon_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	creator := seedCouponUser(t, pool)
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "USERDEL1",
		DiscountType: "percent",
		PercentOff:   ptrF(20),
		CreatedBy:    &creator,
	})
	if err != nil {
		t.Fatal(err)
	}
	learner := seedCouponUser(t, pool)
	res, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
		CouponID:         c.ID,
		UserID:           learner,
		CoursePriceCents: 4000,
		CourseCurrency:   "usd",
	})
	if err != nil || reason != coupons.ReasonOK {
		t.Fatalf("reserve: %v %s", err, reason)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, learner); err != nil {
		t.Fatal(err)
	}
	// Redemption gone.
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM billing.coupon_redemptions WHERE id = $1`, res.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("redemption should cascade with user")
	}
	// Coupon remains; created_by set null when creator deleted separately.
	got, err := GetCouponByID(ctx, pool, c.ID)
	if err != nil || got == nil {
		t.Fatalf("coupon should remain: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, creator); err != nil {
		t.Fatal(err)
	}
	got, err = GetCouponByID(ctx, pool, c.ID)
	if err != nil || got == nil {
		t.Fatalf("coupon should remain after creator delete: %v", err)
	}
	if got.CreatedBy != nil {
		t.Fatal("created_by should be SET NULL")
	}
}

func TestAcquisitionSourceCoupon_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	userID := seedCouponUser(t, pool)
	e, created, err := CreateCourseGrantIdempotent(ctx, pool, CourseGrantInput{
		UserID:            userID,
		CourseID:          courseID,
		AcquisitionSource: AcquisitionCoupon,
		AmountPaidCents:   0,
		Currency:          "usd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created || e.AcquisitionSource != AcquisitionCoupon {
		t.Fatalf("grant: created=%v src=%q", created, e.AcquisitionSource)
	}
}

func TestUpdateCoupon_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "UPDATE1",
		DiscountType: "percent",
		PercentOff:   ptrF(25),
	})
	if err != nil {
		t.Fatal(err)
	}
	note := "launch promo"
	updated, err := UpdateCoupon(ctx, pool, UpdateCouponInput{
		ID:         c.ID,
		PercentOff: ptrF(30),
		Note:       &note,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.PercentOff == nil || *updated.PercentOff != 30 {
		t.Fatalf("percent: %v", updated.PercentOff)
	}
	if updated.Note == nil || *updated.Note != note {
		t.Fatalf("note: %v", updated.Note)
	}
}

func TestGetCouponByID_NotFound(t *testing.T) {
	pool := couponTestPool(t)
	got, err := GetCouponByID(context.Background(), pool, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil")
	}
}

func TestListCouponRedemptions_Pg(t *testing.T) {
	pool := couponTestPool(t)
	ctx := context.Background()
	courseID := seedCouponCourse(t, pool, 4000, "usd")
	c, err := CreateCoupon(ctx, pool, CreateCouponInput{
		CourseID:     courseID,
		Code:         "LISTRED",
		DiscountType: "percent",
		PercentOff:   ptrF(10),
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		uid := seedCouponUser(t, pool)
		_, reason, err := ReserveCoupon(ctx, pool, ReserveInput{
			CouponID:         c.ID,
			UserID:           uid,
			CoursePriceCents: 4000,
			CourseCurrency:   "usd",
		})
		if err != nil || reason != coupons.ReasonOK {
			t.Fatalf("reserve %d: %v %s", i, err, reason)
		}
	}
	list, next, err := ListCouponRedemptions(ctx, pool, c.ID, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("page size: %d", len(list))
	}
	if next == "" {
		t.Fatal("expected cursor")
	}
	list2, _, err := ListCouponRedemptions(ctx, pool, c.ID, next, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 1 {
		t.Fatalf("page2: %d", len(list2))
	}
}

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }
func ptrS(v string) *string   { return &v }
