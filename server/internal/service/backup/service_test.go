package backup

import (
	"os"
	"testing"
	"time"
)

func TestComputeAlerts_StaleBackup(t *testing.T) {
	old := time.Now().UTC().Add(-30 * time.Hour).Format(time.RFC3339)
	tiers := []TierStatusJSON{{
		Tier:          "postgres",
		LastSuccessAt: &old,
		Healthy:       true,
	}}
	alerts := computeAlerts(tiers)
	if len(alerts) == 0 {
		t.Fatal("expected stale backup alert")
	}
	if alerts[0].Tier != "postgres" {
		t.Errorf("tier=%q want postgres", alerts[0].Tier)
	}
}

func TestComputeAlerts_WALLag(t *testing.T) {
	recent := time.Now().UTC().Format(time.RFC3339)
	lag := 1200
	tiers := []TierStatusJSON{{
		Tier:          "postgres",
		LastSuccessAt: &recent,
		WALLagSeconds: &lag,
		Healthy:       true,
	}}
	alerts := computeAlerts(tiers)
	found := false
	for _, a := range alerts {
		if a.Tier == "postgres" && a.Reason != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected WAL lag alert")
	}
}

func TestEnvOverlay_Postgres(t *testing.T) {
	t.Setenv("BACKUP_POSTGRES_LAST_SUCCESS", "2026-05-27T12:00:00Z")
	t.Setenv("BACKUP_POSTGRES_WAL_LAG_SECONDS", "42")
	rows := []TierStatusJSON{}
	_ = rows
	var s struct {
		LastSuccess *time.Time
		WALLag      *int
	}
	if ts := envTime("POSTGRES_LAST_SUCCESS"); ts == nil {
		t.Fatal("expected parsed time")
	} else {
		s.LastSuccess = ts
	}
	if v := envInt("POSTGRES_WAL_LAG_SECONDS"); v == nil || *v != 42 {
		t.Fatalf("wal lag: %#v", v)
	}
	_ = os.Getenv // keep import
}
