package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	botsrepo "github.com/lextures/lextures/server/internal/repos/bots"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
)

func sweepBotDueSoonReminders(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if pool == nil || !cfg.FFWebhooks {
		return
	}
	if !cfg.FFBotSlack && !cfg.FFBotTeams && !cfg.FFBotDiscord {
		return
	}
	rows, err := pool.Query(ctx, `
SELECT csi.id, csi.course_id, csi.title, csi.due_at, c.org_id, COALESCE(c.course_code, ''), ce.user_id
FROM course.course_structure_items csi
INNER JOIN course.courses c ON c.id = csi.course_id
INNER JOIN course.course_enrollments ce ON ce.course_id = c.id AND ce.status = 'active'
INNER JOIN integrations.bot_connections bc ON bc.org_id = c.org_id
WHERE csi.due_at IS NOT NULL
  AND csi.due_at > $1
  AND csi.due_at <= $1 + interval '1 hour' * COALESCE((bc.settings->>'dueSoonHours')::int, 24)
`, now)
	if err != nil {
		slog.Warn("bots.due_soon.list", "err", err)
		return
	}
	defer rows.Close()
	webOrigin := cfg.PublicWebOrigin
	if webOrigin == "" {
		webOrigin = "http://localhost:5173"
	}
	for rows.Next() {
		var structureID, courseID, orgID, userID uuid.UUID
		var title, courseCode string
		var dueAt time.Time
		if err := rows.Scan(&structureID, &courseID, &title, &dueAt, &orgID, &courseCode, &userID); err != nil {
			continue
		}
		sent, err := botsrepo.WasDueSoonSent(ctx, pool, structureID, userID)
		if err != nil || sent {
			continue
		}
		webhooksvc.EmitAssignmentDueSoon(pool, cfg, orgID, webhooksvc.AssignmentDueSoonData{
			CourseID:        courseID.String(),
			CourseCode:      courseCode,
			StructureItemID: structureID.String(),
			Title:           title,
			DueAt:           dueAt.UTC().Format(time.RFC3339),
			StudentUserID:   userID.String(),
			URL:             webOrigin + "/courses/" + courseID.String(),
		})
		_ = botsrepo.MarkDueSoonSent(ctx, pool, structureID, userID)
	}
}
