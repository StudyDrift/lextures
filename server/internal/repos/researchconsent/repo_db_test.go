package researchconsent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func setupPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

func makeUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ph, err := auth.HashPassword("password1230")
	if err != nil {
		t.Fatal(err)
	}
	em := "rc-" + uuid.NewString()[:8] + "@e.com"
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(u.ID)
	return uid
}

func TestConsentFlow_ExportGateAndRate_Pg(t *testing.T) {
	ctx, pool := setupPool(t)
	researcher := makeUser(t, ctx, pool)
	granter := makeUser(t, ctx, pool)
	decliner := makeUser(t, ctx, pool)
	withdrawer := makeUser(t, ctx, pool)

	orgID, err := organization.OrgIDForUser(ctx, pool, researcher)
	if err != nil {
		t.Fatalf("org: %v", err)
	}

	study, err := CreateStudy(ctx, pool, Study{
		OrgID:        orgID,
		ResearcherID: researcher,
		Title:        "Learning Analytics Study",
		IRBProtocol:  "IRB-2026-001",
		ConsentText:  "Please consent.",
		DataUseDesc:  "Analytics only.",
		Status:       StatusActive,
	})
	if err != nil {
		t.Fatalf("create study: %v", err)
	}

	insert := func(u uuid.UUID, decision string) {
		if _, err := InsertRecord(ctx, pool, Record{StudyID: study.ID, UserID: u, Decision: decision}); err != nil {
			t.Fatalf("insert %s: %v", decision, err)
		}
		time.Sleep(time.Millisecond) // ensure distinct created_at ordering
	}
	insert(granter, DecisionGranted)
	insert(decliner, DecisionDeclined)
	// withdrawer grants then withdraws (AC-3): latest decision wins.
	insert(withdrawer, DecisionGranted)
	insert(withdrawer, DecisionWithdrawn)

	// AC-2 / AC-3: only the granter appears in the export.
	parts, err := ExportConsenting(ctx, pool, study.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(parts) != 1 || parts[0].UserID != granter {
		t.Fatalf("export gate: want only granter, got %+v", parts)
	}

	rate, err := GetConsentRate(ctx, pool, study.ID)
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if rate.Granted != 1 || rate.Declined != 1 || rate.Withdrawn != 1 {
		t.Fatalf("rate: got %+v", rate)
	}

	latest, err := LatestDecision(ctx, pool, study.ID, withdrawer)
	if err != nil || latest == nil || latest.Decision != DecisionWithdrawn {
		t.Fatalf("latest decision: %+v err=%v", latest, err)
	}
}

func TestConsentRecords_AppendOnly_Pg(t *testing.T) {
	ctx, pool := setupPool(t)
	researcher := makeUser(t, ctx, pool)
	student := makeUser(t, ctx, pool)
	orgID, _ := organization.OrgIDForUser(ctx, pool, researcher)

	study, err := CreateStudy(ctx, pool, Study{
		OrgID: orgID, ResearcherID: researcher, Title: "S", IRBProtocol: "P",
		ConsentText: "c", DataUseDesc: "d", Status: StatusActive,
	})
	if err != nil {
		t.Fatalf("study: %v", err)
	}
	rec, err := InsertRecord(ctx, pool, Record{StudyID: study.ID, UserID: student, Decision: DecisionGranted})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// The append-only trigger must reject UPDATE and DELETE (NFR Security).
	if _, err := pool.Exec(ctx, `UPDATE research.consent_records SET decision = 'declined' WHERE id = $1`, rec.ID); err == nil {
		t.Fatal("expected UPDATE on consent_records to be rejected")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("unexpected update error: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM research.consent_records WHERE id = $1`, rec.ID); err == nil {
		t.Fatal("expected DELETE on consent_records to be rejected")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("unexpected delete error: %v", err)
	}
}

func TestPendingStudies_OrgWideTargeting_Pg(t *testing.T) {
	ctx, pool := setupPool(t)
	researcher := makeUser(t, ctx, pool)
	student := makeUser(t, ctx, pool)
	orgID, _ := organization.OrgIDForUser(ctx, pool, researcher)

	study, err := CreateStudy(ctx, pool, Study{
		OrgID: orgID, ResearcherID: researcher, Title: "Org-wide", IRBProtocol: "P",
		ConsentText: "c", DataUseDesc: "d", Status: StatusActive,
	})
	if err != nil {
		t.Fatalf("study: %v", err)
	}

	pending, err := PendingStudiesForUser(ctx, pool, orgID, student)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	found := false
	for _, s := range pending {
		if s.ID == study.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("org-wide active study should be pending for the student")
	}

	// After responding, it must no longer be pending.
	if _, err := InsertRecord(ctx, pool, Record{StudyID: study.ID, UserID: student, Decision: DecisionDeclined}); err != nil {
		t.Fatalf("respond: %v", err)
	}
	pending, err = PendingStudiesForUser(ctx, pool, orgID, student)
	if err != nil {
		t.Fatalf("pending2: %v", err)
	}
	for _, s := range pending {
		if s.ID == study.ID {
			t.Fatal("study should not be pending after a decision")
		}
	}
}
