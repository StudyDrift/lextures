package contenttools

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// SyncRegistryMirror upserts every in-process registry tool into content_tool_versions
// (FR backfill on first boot — no learner data touched).
func SyncRegistryMirror(ctx context.Context, pool *pgxpool.Pool, reg *Registry) error {
	if pool == nil || reg == nil {
		return nil
	}
	for _, m := range reg.List() {
		raw, err := json.Marshal(m.Manifest)
		if err != nil {
			return err
		}
		sandbox := m.Sandbox
		if sandbox == "" {
			sandbox = SandboxInProcess
		}
		status := "active"
		if m.Deprecated {
			status = "deprecated"
		}
		var sunset *time.Time
		if m.SunsetAt != "" {
			if t, err := time.Parse(time.RFC3339, m.SunsetAt); err == nil {
				sunset = &t
			}
		}
		stateVer := m.StateSchemaVersion
		if stateVer <= 0 {
			stateVer = DefaultMigrations().CurrentStateSchemaVersion(m.ID)
		}
		cfgVer := m.ConfigSchemaVersion
		if cfgVer <= 0 {
			cfgVer = 1
		}
		row := ctrepo.VersionRow{
			ToolID:              m.ID,
			Version:             m.Version,
			ManifestJSON:        raw,
			ConfigSchemaVersion: cfgVer,
			StateSchemaVersion:  stateVer,
			SandboxMode:         sandbox,
			Status:              status,
			SunsetAt:            sunset,
		}
		if _, err := ctrepo.UpsertToolVersion(ctx, pool, row); err != nil {
			return err
		}
		// Mirror breaker open from DB into in-memory breaker.
		if existing, _ := ctrepo.GetToolVersion(ctx, pool, m.ID, m.Version); existing != nil && existing.BreakerOpenAt != nil {
			DefaultBreaker().Open(m.ID, *existing.BreakerOpenAt)
		}
	}
	slog.Info("contenttools.registry_mirror_synced", "tools", reg.Size())
	return nil
}
