package payments

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestPendingTransactionLifecycle_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, "pay-"+time.Now().Format("20060102150405.000000")+"@e.com", ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	suffix := uuid.NewString()

	newPending := func(sess string) {
		t.Helper()
		if _, _, err := CreateIdempotent(ctx, pool, CreateTransactionInput{
			UserID: uid, Provider: ProviderStripe, ProviderTxnID: sess,
			IdempotencyKey: "checkout:" + sess, AmountCents: 1000, Currency: "usd", Status: StatusPending,
		}); err != nil {
			t.Fatalf("create pending: %v", err)
		}
	}

	paid := "cs_paid_" + suffix
	newPending(paid)
	ok, err := MarkCompletedByProviderTxn(ctx, pool, ProviderStripe, paid, 1000, "usd", nil)
	if err != nil || !ok {
		t.Fatalf("complete: ok=%v err=%v", ok, err)
	}
	got, err := GetByProviderTxn(ctx, pool, ProviderStripe, paid)
	if err != nil || got.Status != StatusCompleted {
		t.Fatalf("expected completed, got %+v err=%v", got, err)
	}
	if ok, _ := MarkCompletedByProviderTxn(ctx, pool, ProviderStripe, paid, 1000, "usd", nil); ok {
		t.Fatal("completing twice should be a no-op")
	}
	if ok, _ := CancelPendingByProviderTxn(ctx, pool, ProviderStripe, paid); ok {
		t.Fatal("completed rows must not be canceled")
	}

	abandoned := "cs_abandoned_" + suffix
	newPending(abandoned)
	ok, err = CancelPendingByProviderTxn(ctx, pool, ProviderStripe, abandoned)
	if err != nil || !ok {
		t.Fatalf("cancel: ok=%v err=%v", ok, err)
	}
	got, err = GetByProviderTxn(ctx, pool, ProviderStripe, abandoned)
	if err != nil || got.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %+v err=%v", got, err)
	}

	stale := "cs_stale_" + suffix
	newPending(stale)
	list, err := ListStalePending(ctx, pool, ProviderStripe, time.Now().Add(time.Hour), 1000)
	if err != nil {
		t.Fatalf("stale: %v", err)
	}
	found := false
	for _, tx := range list {
		if tx.ProviderTxnID == stale {
			found = true
		}
		if tx.ProviderTxnID == paid || tx.ProviderTxnID == abandoned {
			t.Fatalf("non-pending row listed as stale: %s", tx.ProviderTxnID)
		}
	}
	if !found {
		t.Fatal("pending row missing from stale list")
	}
}
