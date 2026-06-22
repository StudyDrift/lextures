package background

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/mail"
	"github.com/lextures/lextures/server/internal/repos/emaildigest"
	"github.com/lextures/lextures/server/internal/repos/emailjobs"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	"github.com/lextures/lextures/server/internal/repos/organization"
	digestsvc "github.com/lextures/lextures/server/internal/service/emaildigest"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

var emailRetryDelays = []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute}

func sweepEmailJobs(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.EmailNotificationsEnabled || pool == nil {
		return
	}
	jobs, err := emailjobs.ListDue(ctx, pool, 50, now)
	if err != nil {
		slog.Warn("email_jobs.list", "err", err)
		return
	}
	for _, job := range jobs {
		if err := deliverEmailJob(ctx, pool, cfg, job, now); err != nil {
			slog.Warn("email_jobs.deliver", "job_id", job.ID, "err", err)
		}
	}
}

func deliverEmailJob(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, job emailjobs.Job, now time.Time) error {
	to, err := notifications.RecipientEmail(ctx, pool, job.RecipientID)
	if err != nil {
		return err
	}
	branding := brandingForRecipient(ctx, pool, job.RecipientID)
	mailBranding := mailBrandingFromRow(branding)
	rendered, err := mail.RenderTemplate(job.Template, job.TemplateVars, mailBranding)
	if err != nil {
		return err
	}
	subject := job.Subject
	if rendered.Subject != "" {
		subject = rendered.Subject
	}
	icsContent := job.TemplateVars["icsContent"]
	icsFilename := job.TemplateVars["icsFilename"]
	var sendErr error
	if strings.TrimSpace(icsContent) != "" {
		sendErr = mail.SendMultipartWithICS(cfg, to, subject, rendered.BodyText, rendered.HTMLBody, mailBranding, icsContent, icsFilename)
	} else {
		sendErr = mail.SendMultipart(cfg, to, subject, rendered.BodyText, rendered.HTMLBody, mailBranding)
	}
	if sendErr != nil {
		attempts := job.Attempts + 1
		dead := attempts >= len(emailRetryDelays)
		var next time.Time
		if !dead {
			next = now.Add(emailRetryDelays[attempts-1])
		}
		_ = emailjobs.MarkRetry(ctx, pool, job.ID, attempts, next, dead)
		return sendErr
	}
	return emailjobs.MarkSent(ctx, pool, job.ID, now)
}

func brandingForRecipient(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) *orgbranding.Row {
	orgID, err := organization.OrgIDForUser(ctx, pool, userID)
	if err != nil {
		return nil
	}
	row, err := orgbranding.Get(ctx, pool, orgID)
	if err != nil {
		return nil
	}
	return row
}

func mailBrandingFromRow(row *orgbranding.Row) *mail.BrandingOpts {
	if row == nil {
		return nil
	}
	return &mail.BrandingOpts{
		FromDisplayName: row.CustomEmailDisplayName,
		LogoURL:         row.LogoURL,
		PrimaryColor:    row.PrimaryColor,
	}
}

func sweepDailyDigests(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.EmailNotificationsEnabled || pool == nil {
		return
	}
	candidates, err := emaildigest.ListCandidates(ctx, pool)
	if err != nil {
		slog.Warn("email_digest.users", "err", err)
		return
	}
	ns := &notifications.Service{Pool: pool, Config: cfg}
	for _, c := range candidates {
		if !digestsvc.ShouldSendDigest(now, c.Timezone) {
			continue
		}
		dayStart := digestsvc.LocalDayStartUTC(now, c.Timezone)
		already, err := emailjobs.DigestEnqueuedSince(ctx, pool, c.UserID, dayStart)
		if err != nil || already {
			continue
		}
		since := digestsvc.DigestSince(now, c.Timezone)
		items, err := emaildigest.ListAndClear(ctx, pool, c.UserID, since)
		if err != nil || len(items) == 0 {
			continue
		}
		var lines, linesHTML strings.Builder
		for _, it := range items {
			lines.WriteString("• ")
			lines.WriteString(it.SummaryLine)
			lines.WriteString("\n")
			linesHTML.WriteString("<li>")
			linesHTML.WriteString(strings.ReplaceAll(it.SummaryLine, "<", "&lt;"))
			linesHTML.WriteString("</li>")
		}
		vars := map[string]string{
			"lines":          lines.String(),
			"linesHtml":      linesHTML.String(),
			"subject":        "Your daily StudyDrift summary",
			"unsubscribeUrl": ns.UnsubscribeURL(c.UserID, notifications.EventGradePosted),
		}
		_, _ = emailjobs.Enqueue(ctx, pool, c.UserID, "daily_digest", vars["subject"], "daily_digest", vars)
	}
}
