package billing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
)

func TestCreateIdempotent_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var userID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	eventID := "evt_test_" + uuid.NewString()
	in := CreateInput{
		UserID:          userID,
		EntitlementType: TypeCoursePurchase,
		StripeEventID:   eventID,
		AmountPaidCents: 2900,
		Currency:        "usd",
	}
	e1, created1, err := CreateIdempotent(ctx, pool, in)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !created1 || e1 == nil {
		t.Fatal("expected created entitlement")
	}
	_, created2, err := CreateIdempotent(ctx, pool, in)
	if err != nil {
		t.Fatalf("duplicate insert: %v", err)
	}
	if created2 {
		t.Fatal("expected idempotent duplicate")
	}
}

func TestHasCourseAccess_FreeCourse_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var userID, courseID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := pool.QueryRow(ctx, `
SELECT id FROM course.courses WHERE price_cents = 0 LIMIT 1
`).Scan(&courseID); err != nil {
		t.Skip("no free course in seed data")
	}
	ok, err := HasCourseAccess(ctx, pool, userID, courseID)
	if err != nil {
		t.Fatalf("access: %v", err)
	}
	if !ok {
		t.Fatal("expected free course access")
	}
}

func TestCreateCourseGrantIdempotent_Free_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var userID, courseID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := pool.QueryRow(ctx, `
SELECT id FROM course.courses WHERE price_cents = 0 LIMIT 1
`).Scan(&courseID); err != nil {
		t.Skip("no free course in seed data")
	}

	// Clean any prior grant for this pair so the test is isolated.
	_, _ = pool.Exec(ctx, `
DELETE FROM billing.user_entitlements
WHERE user_id = $1 AND course_id = $2 AND entitlement_type = 'course_purchase'
`, userID, courseID)

	in := CourseGrantInput{
		UserID:            userID,
		CourseID:          courseID,
		AcquisitionSource: AcquisitionFree,
		AmountPaidCents:   0,
		Currency:          "usd",
	}
	e1, created1, err := CreateCourseGrantIdempotent(ctx, pool, in)
	if err != nil {
		t.Fatalf("first grant: %v", err)
	}
	if !created1 || e1 == nil {
		t.Fatal("expected created free entitlement")
	}
	if e1.AcquisitionSource != AcquisitionFree {
		t.Fatalf("acquisition_source: got %q want free", e1.AcquisitionSource)
	}
	if e1.AmountPaidCents != 0 {
		t.Fatalf("amount: got %d want 0", e1.AmountPaidCents)
	}

	e2, created2, err := CreateCourseGrantIdempotent(ctx, pool, in)
	if err != nil {
		t.Fatalf("duplicate grant: %v", err)
	}
	if created2 {
		t.Fatal("expected idempotent duplicate free claim")
	}
	if e2 == nil || e2.ID != e1.ID {
		t.Fatal("expected same entitlement row on duplicate")
	}

	var n int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM billing.user_entitlements
WHERE user_id = $1 AND course_id = $2
  AND entitlement_type = 'course_purchase' AND status = 'active'
`, userID, courseID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected exactly one active grant, got %d", n)
	}
}

func TestMarketplaceAccess_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var userID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("user: %v", err)
	}

	// Two distinct courses so ownership does not leak across ids.
	var ownedID, otherID uuid.UUID
	rows, err := pool.Query(ctx, `SELECT id FROM course.courses ORDER BY created_at LIMIT 2`)
	if err != nil {
		t.Fatalf("courses: %v", err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0, 2)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) < 2 {
		t.Skip("need at least two courses")
	}
	ownedID, otherID = ids[0], ids[1]

	_, _ = pool.Exec(ctx, `
DELETE FROM billing.user_entitlements
WHERE user_id = $1 AND course_id IN ($2, $3) AND entitlement_type = 'course_purchase'
`, userID, ownedID, otherID)

	_, created, err := CreateCourseGrantIdempotent(ctx, pool, CourseGrantInput{
		UserID:            userID,
		CourseID:          ownedID,
		AcquisitionSource: AcquisitionFree,
		AmountPaidCents:   0,
	})
	if err != nil || !created {
		t.Fatalf("grant: created=%v err=%v", created, err)
	}

	ok, err := MarketplaceAccess(ctx, pool, userID, ownedID)
	if err != nil {
		t.Fatalf("owned: %v", err)
	}
	if !ok {
		t.Fatal("expected MarketplaceAccess true for granted course")
	}
	ok, err = MarketplaceAccess(ctx, pool, userID, otherID)
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	if ok {
		t.Fatal("expected MarketplaceAccess false for unrelated course")
	}

	ownedSet, err := OwnedCourseIDs(ctx, pool, userID, []uuid.UUID{ownedID, otherID})
	if err != nil {
		t.Fatalf("OwnedCourseIDs: %v", err)
	}
	if _, has := ownedSet[ownedID]; !has {
		t.Fatal("expected owned course in OwnedCourseIDs")
	}
	if _, has := ownedSet[otherID]; has {
		t.Fatal("did not expect unrelated course in OwnedCourseIDs")
	}
	empty, err := OwnedCourseIDs(ctx, pool, userID, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty OwnedCourseIDs: %#v err=%v", empty, err)
	}
}

func TestRefundCourseEntitlement_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var userID, courseID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM course.courses LIMIT 1`).Scan(&courseID); err != nil {
		t.Skip("no course")
	}
	_, _ = pool.Exec(ctx, `
DELETE FROM billing.user_entitlements
WHERE user_id = $1 AND course_id = $2 AND entitlement_type = 'course_purchase'
`, userID, courseID)

	_, created, err := CreateCourseGrantIdempotent(ctx, pool, CourseGrantInput{
		UserID:            userID,
		CourseID:          courseID,
		AcquisitionSource: AcquisitionStripe,
		AmountPaidCents:   2000,
		Currency:          "usd",
	})
	if err != nil || !created {
		t.Fatalf("grant: created=%v err=%v", created, err)
	}

	ok, err := MarketplaceAccess(ctx, pool, userID, courseID)
	if err != nil || !ok {
		t.Fatalf("expected access before refund: ok=%v err=%v", ok, err)
	}

	refunded, err := RefundCourseEntitlement(ctx, pool, userID, courseID)
	if err != nil || !refunded {
		t.Fatalf("refund: refunded=%v err=%v", refunded, err)
	}

	ok, err = MarketplaceAccess(ctx, pool, userID, courseID)
	if err != nil {
		t.Fatalf("access after refund: %v", err)
	}
	if ok {
		t.Fatal("expected MarketplaceAccess false after refund")
	}

	// Idempotent second refund.
	refunded2, err := RefundCourseEntitlement(ctx, pool, userID, courseID)
	if err != nil {
		t.Fatalf("second refund: %v", err)
	}
	if refunded2 {
		t.Fatal("expected no row updated on second refund")
	}
}
