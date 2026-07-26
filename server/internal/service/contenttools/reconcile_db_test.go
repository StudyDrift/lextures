package contenttools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

func reconcileTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("short")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
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
	t.Cleanup(pool.Close)
	return pool
}

func TestDB_ReconcileMarkdownFencesStripsCrossCourse(t *testing.T) {
	pool := reconcileTestPool(t)
	ctx := context.Background()

	courseA := uuid.New()
	courseB := uuid.New()
	codeA := "C-" + strings.ToUpper(strings.ReplaceAll(courseA.String(), "-", "")[:6])
	codeB := "C-" + strings.ToUpper(strings.ReplaceAll(courseB.String(), "-", "")[:6])
	for _, row := range []struct {
		id   uuid.UUID
		code string
	}{{courseA, codeA}, {courseB, codeB}} {
		if _, err := pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, content_tools_enabled)
VALUES ($1, $2, 'CT.2 reconcile', TRUE)
`, row.id, row.code); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, row.id)
		})
	}

	inA, err := ctrepo.CreateInstance(ctx, pool, ctrepo.InstanceRow{
		CourseID: courseA, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"a"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	inB, err := ctrepo.CreateInstance(ctx, pool, ctrepo.InstanceRow{
		CourseID: courseB, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"b"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := ctrepo.CreateInstance(ctx, pool, ctrepo.InstanceRow{
		CourseID: courseA, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"orphan"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	md := SerializeLexToolFence(LexToolFencePayload{InstanceID: inA.ID.String(), ToolID: "noop_probe", V: 1}) +
		"\n" +
		SerializeLexToolFence(LexToolFencePayload{InstanceID: inB.ID.String(), ToolID: "noop_probe", V: 1})
	cleaned, err := ReconcileMarkdownFences(ctx, pool, courseA, nil, md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cleaned, inA.ID.String()) {
		t.Fatalf("kept in-course fence missing: %s", cleaned)
	}
	if strings.Contains(cleaned, inB.ID.String()) {
		t.Fatalf("cross-course fence not stripped: %s", cleaned)
	}

	rows, err := ctrepo.ListInstances(ctx, pool, courseA, nil, "", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID == orphan.ID && r.Status != "archived" {
			t.Fatalf("unreferenced instance should be archived, got %q", r.Status)
		}
		if r.ID == inA.ID && r.Status != "active" {
			t.Fatalf("referenced instance should stay active, got %q", r.Status)
		}
	}
}
