package background

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/reportschedules"
	"github.com/lextures/lextures/server/internal/service/reportpdf"
)

// sweepScheduledReports finds due schedules and emails their PDFs.
func sweepScheduledReports(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.ReportExportEnabled {
		return
	}
	due, err := reportschedules.ListDue(ctx, pool, now, 20)
	if err != nil {
		slog.Warn("scheduled_reports.list_due", "err", err)
		return
	}
	for _, sched := range due {
		if err := runScheduledReport(ctx, pool, cfg, sched, now); err != nil {
			slog.Warn("scheduled_reports.run", "schedule_id", sched.ID, "report_type", sched.ReportType, "err", err)
		}
		next := nextRunAt(sched.Cadence, now)
		if err := reportschedules.MarkRan(ctx, pool, sched.ID, now, next); err != nil {
			slog.Warn("scheduled_reports.mark_ran", "schedule_id", sched.ID, "err", err)
		}
	}
}

func runScheduledReport(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, sched reportschedules.Schedule, now time.Time) error {
	pdfBytes, err := generateScheduledPDF(sched, now)
	if err != nil {
		return fmt.Errorf("generate pdf: %w", err)
	}
	if len(sched.Recipients) == 0 {
		return nil
	}
	subject := fmt.Sprintf("%s Report — %s", friendlyReportType(sched.ReportType), now.Format("Jan 2, 2006"))
	return sendPDFEmail(cfg, sched.Recipients, subject, pdfBytes, sched.ReportType+"-report.pdf")
}

func generateScheduledPDF(sched reportschedules.Schedule, now time.Time) ([]byte, error) {
	institution := ""
	switch sched.ReportType {
	case "learning-activity":
		return reportpdf.BuildLearningActivityPDF(reportpdf.LearningActivityInput{
			InstitutionName: institution,
			GeneratedAt:     now,
			From:            now.AddDate(0, 0, -30),
			To:              now,
		})
	case "gradebook":
		return reportpdf.BuildGradebookPDF(reportpdf.GradebookInput{
			InstitutionName: institution,
			CourseName:      sched.Parameters["course_name"],
			CourseCode:      sched.Parameters["course_code"],
			GeneratedAt:     now,
		})
	case "progress":
		return reportpdf.BuildProgressPDF(reportpdf.ProgressInput{
			InstitutionName: institution,
			CourseName:      sched.Parameters["course_name"],
			CourseCode:      sched.Parameters["course_code"],
			StudentName:     sched.Parameters["student_name"],
			GeneratedAt:     now,
		})
	default:
		return reportpdf.BuildLearningActivityPDF(reportpdf.LearningActivityInput{
			InstitutionName: institution,
			GeneratedAt:     now,
			From:            now.AddDate(0, 0, -7),
			To:              now,
		})
	}
}

func sendPDFEmail(cfg config.Config, to []string, subject string, pdfBytes []byte, filename string) error {
	if cfg.SMTPHost == "" {
		return nil
	}
	from := cfg.SMTPFrom
	if from == "" {
		from = cfg.SMTPUser
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	th := make(textproto.MIMEHeader)
	th.Set("Content-Type", "text/plain; charset=utf-8")
	tw, _ := w.CreatePart(th)
	_, _ = fmt.Fprintf(tw, "Your scheduled report is attached as %s.\n\nThis report was generated on %s.",
		filename, time.Now().UTC().Format("Jan 2, 2006 15:04 UTC"))

	ah := make(textproto.MIMEHeader)
	ah.Set("Content-Type", "application/pdf")
	ah.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	ah.Set("Content-Transfer-Encoding", "base64")
	aw, _ := w.CreatePart(ah)
	enc := make([]byte, encodedLen(len(pdfBytes)))
	encodeBase64(enc, pdfBytes)
	_, _ = aw.Write(enc)

	_ = w.Close()

	headers := strings.Join([]string{
		"From: " + from,
		"To: " + strings.Join(to, ", "),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=" + w.Boundary(),
	}, "\r\n")
	msg := []byte(headers + "\r\n\r\n" + body.String())

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	var auth smtp.Auth
	if cfg.SMTPUser != "" && cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, from, to, msg)
}

// NextRunAt calculates the next execution time based on cadence.
// Exported for use in HTTP handlers.
func NextRunAt(cadence string, from time.Time) time.Time {
	return nextRunAt(cadence, from)
}

// nextRunAt calculates the next execution time based on cadence.
func nextRunAt(cadence string, from time.Time) time.Time {
	switch strings.ToLower(cadence) {
	case "daily":
		return from.AddDate(0, 0, 1)
	case "weekly":
		return from.AddDate(0, 0, 7)
	case "monthly":
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 7)
	}
}

func friendlyReportType(t string) string {
	switch t {
	case "gradebook":
		return "Gradebook"
	case "progress":
		return "Student Progress"
	case "learning-activity":
		return "Learning Activity"
	case "at-risk":
		return "At-Risk"
	case "item-analysis":
		return "Item Analysis"
	default:
		if len(t) == 0 {
			return t
		}
		return strings.ToUpper(t[:1]) + t[1:]
	}
}

// encodedLen returns the base64 output length for n input bytes.
func encodedLen(n int) int {
	return (n + 2) / 3 * 4
}

// encodeBase64 encodes src into base64 (standard encoding) writing to dst.
func encodeBase64(dst, src []byte) {
	const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	di, si := 0, 0
	n := (len(src) / 3) * 3
	for si < n {
		val := uint(src[si+0])<<16 | uint(src[si+1])<<8 | uint(src[si+2])
		dst[di+0] = b64[val>>18&0x3F]
		dst[di+1] = b64[val>>12&0x3F]
		dst[di+2] = b64[val>>6&0x3F]
		dst[di+3] = b64[val>>0&0x3F]
		si += 3
		di += 4
	}
	rem := len(src) - si
	if rem == 0 {
		return
	}
	var val uint
	if rem >= 2 {
		val |= uint(src[si+1]) << 8
	}
	val |= uint(src[si+0]) << 16
	dst[di+0] = b64[val>>18&0x3F]
	dst[di+1] = b64[val>>12&0x3F]
	if rem >= 2 {
		dst[di+2] = b64[val>>6&0x3F]
	} else {
		dst[di+2] = '='
	}
	dst[di+3] = '='
}
