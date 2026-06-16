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
