package background

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/devicepushtokens"
	"github.com/lextures/lextures/server/internal/repos/pushjobs"
	"github.com/lextures/lextures/server/internal/repos/pushsubscriptions"
	"github.com/lextures/lextures/server/internal/service/push"
)

var pushRetryDelays = []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute}

func sweepPushJobs(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.PushNotificationsEnabled || pool == nil {
		return
	}
	jobs, err := pushjobs.ListDue(ctx, pool, 50, now)
	if err != nil {
		slog.Warn("push_jobs.list", "err", err)
		return
	}
	native := push.NewNativeDispatcher(cfg)
	for _, job := range jobs {
		if err := deliverPushJob(ctx, pool, cfg, native, job, now); err != nil {
			slog.Warn("push_jobs.deliver", "job_id", job.ID, "err", err)
		}
	}
}

func deliverPushJob(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, native *push.NativeDispatcher, job pushjobs.Job, now time.Time) error {
	payload := buildPushPayload(job.Title, job.Body, job.ActionURL)

	subs, err := pushsubscriptions.ListAllForUser(ctx, pool, job.UserID)
	if err != nil {
		return err
	}
	devices, err := devicepushtokens.ListActiveForUser(ctx, pool, job.UserID)
	if err != nil {
		return err
	}
	if len(subs) == 0 && len(devices) == 0 {
		return pushjobs.MarkSent(ctx, pool, job.ID, now)
	}

	allGone := true
	var firstErr error

	if cfg.VAPIDPublicKey != "" && cfg.VAPIDPrivateKey != "" {
		for _, sub := range subs {
			resp, err := sendWebPush(payload, sub, cfg)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				slog.Warn("push_jobs.web_send", "job_id", job.ID, "endpoint_prefix", sub.Endpoint[:min(len(sub.Endpoint), 30)], "err", err)
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusGone {
				_ = pushsubscriptions.DeleteByEndpoint(ctx, pool, sub.Endpoint)
				continue
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				_ = pushsubscriptions.MarkUsed(ctx, pool, sub.ID)
				allGone = false
				firstErr = nil
			} else {
				if firstErr == nil {
					firstErr = fmt.Errorf("push endpoint returned %d", resp.StatusCode)
				}
				allGone = false
			}
		}
	}

	for _, device := range devices {
		result, err := native.Send(ctx, device, job.Title, job.Body, job.ActionURL)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if result.Retryable {
				allGone = false
			}
			slog.Warn("push_jobs.native_send", "job_id", job.ID, "platform", device.Platform, "err", err)
			continue
		}
		if result.InvalidToken {
			_ = devicepushtokens.MarkInactive(ctx, pool, device.ID)
			continue
		}
		_ = devicepushtokens.MarkUsed(ctx, pool, device.ID)
		allGone = false
		firstErr = nil
	}

	if allGone || firstErr == nil {
		return pushjobs.MarkSent(ctx, pool, job.ID, now)
	}

	attempts := job.Attempts + 1
	dead := attempts >= len(pushRetryDelays)
	var next time.Time
	if !dead {
		next = now.Add(pushRetryDelays[attempts-1])
	}
	return pushjobs.MarkRetry(ctx, pool, job.ID, attempts, next, dead)
}

func sendWebPush(payload []byte, sub pushsubscriptions.Row, cfg config.Config) (*http.Response, error) {
	return webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256DHKey,
			Auth:   sub.AuthSecret,
		},
	}, &webpush.Options{
		VAPIDPublicKey:  cfg.VAPIDPublicKey,
		VAPIDPrivateKey: cfg.VAPIDPrivateKey,
		Subscriber:      cfg.VAPIDSubject,
		TTL:             86400,
	})
}

func buildPushPayload(title, body, actionURL string) []byte {
	escaped := func(s string) string {
		b := make([]byte, 0, len(s)+2)
		for _, c := range s {
			switch c {
			case '"':
				b = append(b, '\\', '"')
			case '\\':
				b = append(b, '\\', '\\')
			case '\n':
				b = append(b, '\\', 'n')
			case '\r':
				b = append(b, '\\', 'r')
			default:
				b = append(b, []byte(string(c))...)
			}
		}
		return string(b)
	}
	if actionURL == "" {
		return []byte(`{"title":"` + escaped(title) + `","body":"` + escaped(body) + `"}`)
	}
	return []byte(`{"title":"` + escaped(title) + `","body":"` + escaped(body) + `","url":"` + escaped(actionURL) + `"}`)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
