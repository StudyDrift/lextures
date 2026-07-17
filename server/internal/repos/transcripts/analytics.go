package transcripts

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConsolePanels describes which registrar console panels a principal may see (T12 FR-5).
type ConsolePanels struct {
	Queue      bool `json:"queue"`
	Holds      bool `json:"holds"`
	Fees       bool `json:"fees"`
	Delivery   bool `json:"delivery"`
	Recipients bool `json:"recipients"`
	Settings   bool `json:"settings"`
	Analytics  bool `json:"analytics"`
	Finance    bool `json:"finance"`
	Export     bool `json:"export"`
}

// DailyStat is one precomputed day bucket for transcript analytics.
type DailyStat struct {
	Day             time.Time `json:"day"`
	Orders          int       `json:"orders"`
	Items           int       `json:"items"`
	Delivered       int       `json:"delivered"`
	OnHold          int       `json:"onHold"`
	Rejected        int       `json:"rejected"`
	Refunded        int       `json:"refunded"`
	NetRevenueMinor int64     `json:"netRevenueMinor"`
}

// MethodMixBucket is delivery-method volume for the dashboard.
type MethodMixBucket struct {
	Method string `json:"method"`
	Count  int    `json:"count"`
}

// DestinationBucket is a top destination for the dashboard.
type DestinationBucket struct {
	RecipientID   string `json:"recipientId,omitempty"`
	RecipientName string `json:"recipientName"`
	Count         int    `json:"count"`
}

// TurnaroundStats holds average and percentile turnaround in hours.
type TurnaroundStats struct {
	SampleSize int     `json:"sampleSize"`
	AvgHours   float64 `json:"avgHours"`
	P50Hours   float64 `json:"p50Hours"`
	P90Hours   float64 `json:"p90Hours"`
	P95Hours   float64 `json:"p95Hours"`
}

// DashboardSummary is GET /admin/transcripts/dashboard (T12 FR-3).
type DashboardSummary struct {
	OrgID              string              `json:"orgId"`
	From               string              `json:"from"`
	To                 string              `json:"to"`
	Orders             int                 `json:"orders"`
	Items              int                 `json:"items"`
	Delivered          int                 `json:"delivered"`
	OnHold             int                 `json:"onHold"`
	Rejected           int                 `json:"rejected"`
	Refunded           int                 `json:"refunded"`
	NetRevenueMinor    int64               `json:"netRevenueMinor"`
	HoldRate           float64             `json:"holdRate"`
	RejectionRate      float64             `json:"rejectionRate"`
	RefundRate         float64             `json:"refundRate"`
	Turnaround         TurnaroundStats     `json:"turnaround"`
	MethodMix          []MethodMixBucket   `json:"methodMix"`
	TopDestinations    []DestinationBucket `json:"topDestinations"`
	Daily              []DailyStat         `json:"daily"`
	LastRefreshedAt    *time.Time          `json:"lastRefreshedAt,omitempty"`
	Stale              bool                `json:"stale"`
	Panels             ConsolePanels       `json:"panels"`
	Currency           string              `json:"currency"`
}

// HealthSummary is GET /admin/transcripts/health (T12 FR-6).
type HealthSummary struct {
	OrgID                 string        `json:"orgId"`
	BacklogCount          int           `json:"backlogCount"`
	OldestPendingAgeHours float64       `json:"oldestPendingAgeHours"`
	OldestPendingOrderID  string        `json:"oldestPendingOrderId,omitempty"`
	DeliveryFailureRate   float64       `json:"deliveryFailureRate"`
	DeadLetterCount       int           `json:"deadLetterCount"`
	BacklogAlert          bool          `json:"backlogAlert"`
	AgeAlert              bool          `json:"ageAlert"`
	FailureAlert          bool          `json:"failureAlert"`
	AnyAlert              bool          `json:"anyAlert"`
	Thresholds            SLAThresholds `json:"thresholds"`
	Panels                ConsolePanels `json:"panels"`
}

// SLAThresholds are org-configured alert cutoffs.
type SLAThresholds struct {
	BacklogCount       int `json:"backlogCount"`
	OldestPendingHours int `json:"oldestPendingHours"`
	FailureRateBps     int `json:"failureRateBps"`
}

// DrillDownOrder is one order contributing to a dashboard metric.
type DrillDownOrder struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	UserEmail   string     `json:"userEmail,omitempty"`
	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	TotalAmount *int       `json:"totalAmount,omitempty"`
	Currency    string     `json:"currency,omitempty"`
}

// NetRevenueMinor computes net-of-refund revenue in minor units (T12 FR-7).
func NetRevenueMinor(totalAmount, amountRefunded int64) int64 {
	net := totalAmount - amountRefunded
	if net < 0 {
		return 0
	}
	return net
}

// PercentileHours returns the percentile of a sorted ascending hours slice.
// p is in [0, 100]. Empty input returns 0.
func PercentileHours(sortedHours []float64, p float64) float64 {
	n := len(sortedHours)
	if n == 0 {
		return 0
	}
	if p <= 0 {
		return sortedHours[0]
	}
	if p >= 100 {
		return sortedHours[n-1]
	}
	rank := (p / 100) * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sortedHours[lo]
	}
	w := rank - float64(lo)
	return sortedHours[lo]*(1-w) + sortedHours[hi]*w
}

// ComputeTurnaroundStats builds average + percentile stats from hour samples.
func ComputeTurnaroundStats(hours []float64) TurnaroundStats {
	out := TurnaroundStats{SampleSize: len(hours)}
	if len(hours) == 0 {
		return out
	}
	sorted := append([]float64(nil), hours...)
	sort.Float64s(sorted)
	var sum float64
	for _, h := range sorted {
		sum += h
	}
	out.AvgHours = sum / float64(len(sorted))
	out.P50Hours = PercentileHours(sorted, 50)
	out.P90Hours = PercentileHours(sorted, 90)
	out.P95Hours = PercentileHours(sorted, 95)
	return out
}

// Rate returns count/total clamped to [0,1]; zero total → 0.
func Rate(count, total int) float64 {
	if total <= 0 || count <= 0 {
		return 0
	}
	r := float64(count) / float64(total)
	if r > 1 {
		return 1
	}
	return r
}

// GetSLAThresholds loads org SLA alert cutoffs (defaults when unset).
func GetSLAThresholds(ctx context.Context, pool *pgxpool.Pool) (SLAThresholds, error) {
	out := SLAThresholds{BacklogCount: 25, OldestPendingHours: 48, FailureRateBps: 500}
	err := pool.QueryRow(ctx, `
SELECT COALESCE(sla_backlog_threshold, 25),
       COALESCE(sla_oldest_pending_hours, 48),
       COALESCE(sla_failure_rate_bps, 500)
FROM settings.transcripts_config
WHERE id = 1
`).Scan(&out.BacklogCount, &out.OldestPendingHours, &out.FailureRateBps)
	if err != nil {
		// Table/columns may be missing in short tests; return defaults.
		return out, nil
	}
	return out, nil
}

// RefreshAnalyticsDaily upserts daily rollups for the given UTC day.
// When orgID is nil, all orgs with activity that day are refreshed.
func RefreshAnalyticsDaily(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, day time.Time) (int64, error) {
	day = time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	next := day.AddDate(0, 0, 1)

	q := `
WITH day_orders AS (
    SELECT id, org_id, status,
           COALESCE(total_amount, 0) AS total_amount,
           COALESCE(amount_refunded, 0) AS amount_refunded
    FROM transcripts.orders
    WHERE org_id IS NOT NULL
      AND created_at >= $1 AND created_at < $2
`
	args := []any{day, next}
	if orgID != nil {
		q += ` AND org_id = $3`
		args = append(args, *orgID)
	}
	q += `
),
order_stats AS (
    SELECT org_id,
           COUNT(*)::int AS orders,
           COUNT(*) FILTER (WHERE status = 'on_hold')::int AS on_hold,
           COUNT(*) FILTER (WHERE status = 'rejected')::int AS rejected,
           COUNT(*) FILTER (WHERE amount_refunded > 0)::int AS refunded,
           COALESCE(SUM(total_amount - amount_refunded), 0)::bigint AS net_revenue_minor
    FROM day_orders
    GROUP BY org_id
),
item_stats AS (
    SELECT o.org_id,
           COUNT(oi.id)::int AS items,
           COUNT(oi.id) FILTER (WHERE oi.status = 'delivered')::int AS delivered
    FROM day_orders o
    LEFT JOIN transcripts.order_items oi ON oi.order_id = o.id
    GROUP BY o.org_id
)
INSERT INTO transcripts.analytics_daily (
    org_id, day, orders, items, delivered, on_hold, rejected, refunded, net_revenue_minor, refreshed_at
)
SELECT
    os.org_id,
    $1::date,
    os.orders,
    COALESCE(is_.items, 0),
    COALESCE(is_.delivered, 0),
    os.on_hold,
    os.rejected,
    os.refunded,
    os.net_revenue_minor,
    NOW()
FROM order_stats os
LEFT JOIN item_stats is_ ON is_.org_id = os.org_id
ON CONFLICT (org_id, day) DO UPDATE SET
    orders = EXCLUDED.orders,
    items = EXCLUDED.items,
    delivered = EXCLUDED.delivered,
    on_hold = EXCLUDED.on_hold,
    rejected = EXCLUDED.rejected,
    refunded = EXCLUDED.refunded,
    net_revenue_minor = EXCLUDED.net_revenue_minor,
    refreshed_at = NOW()
`
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// GetDashboard returns live + rollup analytics for an org and date range.
func GetDashboard(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, from, to time.Time, panels ConsolePanels) (DashboardSummary, error) {
	if to.Before(from) {
		from, to = to, from
	}
	// Inclusive end date: treat `to` as end-of-day when date-only.
	toExclusive := to
	if to.Hour() == 0 && to.Minute() == 0 && to.Second() == 0 {
		toExclusive = to.Add(24 * time.Hour)
	}
	sum := DashboardSummary{
		OrgID:           orgID.String(),
		From:            from.UTC().Format("2006-01-02"),
		To:              to.UTC().Format("2006-01-02"),
		MethodMix:       []MethodMixBucket{},
		TopDestinations: []DestinationBucket{},
		Daily:           []DailyStat{},
		Panels:          panels,
		Currency:        "usd",
	}

	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int,
       COUNT(*) FILTER (WHERE status = 'on_hold')::int,
       COUNT(*) FILTER (WHERE status = 'rejected')::int,
       COUNT(*) FILTER (WHERE COALESCE(amount_refunded, 0) > 0)::int,
       COALESCE(SUM(COALESCE(total_amount, 0) - COALESCE(amount_refunded, 0)), 0)::bigint,
       COALESCE(MAX(currency), 'usd')
FROM transcripts.orders
WHERE org_id = $1
  AND created_at >= $2 AND created_at < $3
  AND status <> 'draft'
`, orgID, from, toExclusive).Scan(
		&sum.Orders, &sum.OnHold, &sum.Rejected, &sum.Refunded, &sum.NetRevenueMinor, &sum.Currency,
	)
	if err != nil {
		return sum, err
	}
	sum.HoldRate = Rate(sum.OnHold, sum.Orders)
	sum.RejectionRate = Rate(sum.Rejected, sum.Orders)
	sum.RefundRate = Rate(sum.Refunded, sum.Orders)

	err = pool.QueryRow(ctx, `
SELECT COUNT(oi.id)::int,
       COUNT(oi.id) FILTER (WHERE oi.status = 'delivered')::int
FROM transcripts.order_items oi
INNER JOIN transcripts.orders o ON o.id = oi.order_id
WHERE o.org_id = $1
  AND o.created_at >= $2 AND o.created_at < $3
  AND o.status <> 'draft'
`, orgID, from, toExclusive).Scan(&sum.Items, &sum.Delivered)
	if err != nil {
		return sum, err
	}

	hours, err := loadTurnaroundHours(ctx, pool, orgID, from, toExclusive)
	if err != nil {
		return sum, err
	}
	sum.Turnaround = ComputeTurnaroundStats(hours)

	mrows, err := pool.Query(ctx, `
SELECT oi.delivery_method, COUNT(*)::int
FROM transcripts.order_items oi
INNER JOIN transcripts.orders o ON o.id = oi.order_id
WHERE o.org_id = $1
  AND o.created_at >= $2 AND o.created_at < $3
  AND o.status <> 'draft'
GROUP BY oi.delivery_method
ORDER BY COUNT(*) DESC, oi.delivery_method
`, orgID, from, toExclusive)
	if err != nil {
		return sum, err
	}
	for mrows.Next() {
		var b MethodMixBucket
		if err := mrows.Scan(&b.Method, &b.Count); err != nil {
			mrows.Close()
			return sum, err
		}
		sum.MethodMix = append(sum.MethodMix, b)
	}
	mrows.Close()
	if err := mrows.Err(); err != nil {
		return sum, err
	}

	drows, err := pool.Query(ctx, `
SELECT COALESCE(r.id::text, ''), COALESCE(r.name, 'Unknown'), COUNT(*)::int
FROM transcripts.order_items oi
INNER JOIN transcripts.orders o ON o.id = oi.order_id
LEFT JOIN transcripts.recipients r ON r.id = oi.recipient_id
WHERE o.org_id = $1
  AND o.created_at >= $2 AND o.created_at < $3
  AND o.status <> 'draft'
GROUP BY r.id, r.name
ORDER BY COUNT(*) DESC, COALESCE(r.name, 'Unknown')
LIMIT 10
`, orgID, from, toExclusive)
	if err != nil {
		return sum, err
	}
	for drows.Next() {
		var b DestinationBucket
		if err := drows.Scan(&b.RecipientID, &b.RecipientName, &b.Count); err != nil {
			drows.Close()
			return sum, err
		}
		sum.TopDestinations = append(sum.TopDestinations, b)
	}
	drows.Close()
	if err := drows.Err(); err != nil {
		return sum, err
	}

	// Prefer rollups for the series; fall back to live daily aggregation when empty.
	arows, err := pool.Query(ctx, `
SELECT day, orders, items, delivered, on_hold, rejected, refunded, net_revenue_minor, refreshed_at
FROM transcripts.analytics_daily
WHERE org_id = $1 AND day >= $2::date AND day < $3::date
ORDER BY day ASC
`, orgID, from, toExclusive)
	if err != nil {
		return sum, err
	}
	var lastRefresh *time.Time
	for arows.Next() {
		var d DailyStat
		var refreshed time.Time
		if err := arows.Scan(&d.Day, &d.Orders, &d.Items, &d.Delivered, &d.OnHold, &d.Rejected, &d.Refunded, &d.NetRevenueMinor, &refreshed); err != nil {
			arows.Close()
			return sum, err
		}
		sum.Daily = append(sum.Daily, d)
		if lastRefresh == nil || refreshed.After(*lastRefresh) {
			t := refreshed
			lastRefresh = &t
		}
	}
	arows.Close()
	if err := arows.Err(); err != nil {
		return sum, err
	}
	if len(sum.Daily) == 0 {
		lrows, lerr := pool.Query(ctx, `
WITH day_orders AS (
    SELECT id, status, created_at,
           COALESCE(total_amount, 0) AS total_amount,
           COALESCE(amount_refunded, 0) AS amount_refunded
    FROM transcripts.orders
    WHERE org_id = $1
      AND created_at >= $2 AND created_at < $3
      AND status <> 'draft'
),
order_stats AS (
    SELECT date_trunc('day', created_at AT TIME ZONE 'UTC')::date AS day,
           COUNT(*)::int AS orders,
           COUNT(*) FILTER (WHERE status = 'on_hold')::int AS on_hold,
           COUNT(*) FILTER (WHERE status = 'rejected')::int AS rejected,
           COUNT(*) FILTER (WHERE amount_refunded > 0)::int AS refunded,
           COALESCE(SUM(total_amount - amount_refunded), 0)::bigint AS net_revenue_minor
    FROM day_orders
    GROUP BY 1
),
item_stats AS (
    SELECT date_trunc('day', o.created_at AT TIME ZONE 'UTC')::date AS day,
           COUNT(oi.id)::int AS items,
           COUNT(oi.id) FILTER (WHERE oi.status = 'delivered')::int AS delivered
    FROM day_orders o
    LEFT JOIN transcripts.order_items oi ON oi.order_id = o.id
    GROUP BY 1
)
SELECT os.day, os.orders, COALESCE(is_.items, 0), COALESCE(is_.delivered, 0),
       os.on_hold, os.rejected, os.refunded, os.net_revenue_minor
FROM order_stats os
LEFT JOIN item_stats is_ ON is_.day = os.day
ORDER BY os.day ASC
`, orgID, from, toExclusive)
		if lerr != nil {
			return sum, lerr
		}
		for lrows.Next() {
			var d DailyStat
			if err := lrows.Scan(&d.Day, &d.Orders, &d.Items, &d.Delivered, &d.OnHold, &d.Rejected, &d.Refunded, &d.NetRevenueMinor); err != nil {
				lrows.Close()
				return sum, err
			}
			sum.Daily = append(sum.Daily, d)
		}
		lrows.Close()
		if err := lrows.Err(); err != nil {
			return sum, err
		}
	} else {
		sum.LastRefreshedAt = lastRefresh
		if lastRefresh != nil && time.Since(*lastRefresh) > 36*time.Hour {
			sum.Stale = true
		}
	}

	if !panels.Finance {
		sum.NetRevenueMinor = 0
		sum.Refunded = 0
		sum.RefundRate = 0
		for i := range sum.Daily {
			sum.Daily[i].NetRevenueMinor = 0
			sum.Daily[i].Refunded = 0
		}
	}

	return sum, nil
}

func loadTurnaroundHours(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, from, toExclusive time.Time) ([]float64, error) {
	rows, err := pool.Query(ctx, `
SELECT EXTRACT(EPOCH FROM (v.delivered_at - v.submitted_at)) / 3600.0
FROM transcripts.v_turnaround v
WHERE v.org_id = $1
  AND v.submitted_at IS NOT NULL
  AND v.delivered_at IS NOT NULL
  AND v.submitted_at >= $2 AND v.submitted_at < $3
  AND v.delivered_at >= v.submitted_at
`, orgID, from, toExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hours []float64
	for rows.Next() {
		var h float64
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		hours = append(hours, h)
	}
	return hours, rows.Err()
}

// GetHealth returns SLA/queue health for the org.
func GetHealth(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, panels ConsolePanels) (HealthSummary, error) {
	th, _ := GetSLAThresholds(ctx, pool)
	out := HealthSummary{
		OrgID:      orgID.String(),
		Thresholds: th,
		Panels:     panels,
	}

	var oldestID *uuid.UUID
	var oldestAge *float64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int,
       (
         SELECT EXTRACT(EPOCH FROM (NOW() - o2.submitted_at)) / 3600.0
         FROM transcripts.orders o2
         WHERE o2.org_id = $1
           AND o2.status IN ('in_review', 'on_hold', 'processing', 'pending_payment', 'pending_consent')
           AND o2.submitted_at IS NOT NULL
         ORDER BY o2.submitted_at ASC
         LIMIT 1
       ),
       (
         SELECT o3.id
         FROM transcripts.orders o3
         WHERE o3.org_id = $1
           AND o3.status IN ('in_review', 'on_hold', 'processing', 'pending_payment', 'pending_consent')
           AND o3.submitted_at IS NOT NULL
         ORDER BY o3.submitted_at ASC
         LIMIT 1
       )
FROM transcripts.orders o
WHERE o.org_id = $1
  AND o.status IN ('in_review', 'on_hold', 'processing', 'pending_payment', 'pending_consent')
`, orgID).Scan(&out.BacklogCount, &oldestAge, &oldestID)
	if err != nil {
		return out, err
	}
	if oldestAge != nil {
		out.OldestPendingAgeHours = *oldestAge
	}
	if oldestID != nil {
		out.OldestPendingOrderID = oldestID.String()
	}

	var failedAttempts, totalAttempts int
	lookback := time.Now().UTC().AddDate(0, 0, -7)
	err = pool.QueryRow(ctx, `
SELECT
  COUNT(*) FILTER (WHERE da.status = 'failed')::int,
  COUNT(*)::int
FROM transcripts.delivery_attempts da
INNER JOIN transcripts.order_items oi ON oi.id = da.order_item_id
INNER JOIN transcripts.orders o ON o.id = oi.order_id
WHERE o.org_id = $1
  AND da.created_at >= $2
`, orgID, lookback).Scan(&failedAttempts, &totalAttempts)
	if err != nil {
		return out, err
	}
	out.DeliveryFailureRate = Rate(failedAttempts, totalAttempts)

	err = pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM transcripts.order_items oi
INNER JOIN transcripts.orders o ON o.id = oi.order_id
WHERE o.org_id = $1
  AND oi.status = 'failed'
`, orgID).Scan(&out.DeadLetterCount)
	if err != nil {
		return out, err
	}

	out.BacklogAlert = out.BacklogCount >= th.BacklogCount
	out.AgeAlert = out.OldestPendingAgeHours >= float64(th.OldestPendingHours)
	out.FailureAlert = out.DeliveryFailureRate*10000 >= float64(th.FailureRateBps)
	out.AnyAlert = out.BacklogAlert || out.AgeAlert || out.FailureAlert
	return out, nil
}

// ListDrillDownOrders returns orders contributing to a metric (T12 FR-10).
func ListDrillDownOrders(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, metric string, from, to time.Time, limit int) ([]DrillDownOrder, error) {
	if to.Before(from) {
		from, to = to, from
	}
	toExclusive := to
	if to.Hour() == 0 && to.Minute() == 0 && to.Second() == 0 {
		toExclusive = to.Add(24 * time.Hour)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	statusFilter := ""
	switch metric {
	case "on_hold", "holds":
		statusFilter = ` AND o.status = 'on_hold'`
	case "rejected", "rejection":
		statusFilter = ` AND o.status = 'rejected'`
	case "refunded", "refunds":
		statusFilter = ` AND COALESCE(o.amount_refunded, 0) > 0`
	case "delivered":
		statusFilter = ` AND EXISTS (
			SELECT 1 FROM transcripts.order_items oi WHERE oi.order_id = o.id AND oi.status = 'delivered'
		)`
	case "orders", "volume", "":
		// all non-draft
	default:
		return nil, fmt.Errorf("unknown metric %q", metric)
	}

	q := `
SELECT o.id, o.status, COALESCE(u.email, ''), o.submitted_at, o.created_at, o.total_amount, COALESCE(o.currency, 'usd')
FROM transcripts.orders o
LEFT JOIN "user".users u ON u.id = o.user_id
WHERE o.org_id = $1
  AND o.created_at >= $2 AND o.created_at < $3
  AND o.status <> 'draft'` + statusFilter + `
ORDER BY o.created_at DESC
LIMIT $4
`
	rows, err := pool.Query(ctx, q, orgID, from, toExclusive, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DrillDownOrder
	for rows.Next() {
		var d DrillDownOrder
		var email string
		var total *int
		if err := rows.Scan(&d.ID, &d.Status, &email, &d.SubmittedAt, &d.CreatedAt, &total, &d.Currency); err != nil {
			return nil, err
		}
		d.UserEmail = email
		d.TotalAmount = total
		out = append(out, d)
	}
	if out == nil {
		out = []DrillDownOrder{}
	}
	return out, rows.Err()
}

// WriteDashboardCSV writes a CSV that reconciles with on-screen dashboard figures (T12 FR-4).
func WriteDashboardCSV(w io.Writer, sum DashboardSummary) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"section", "key", "value",
	})
	rows := [][]string{
		{"summary", "org_id", sum.OrgID},
		{"summary", "from", sum.From},
		{"summary", "to", sum.To},
		{"summary", "orders", fmt.Sprintf("%d", sum.Orders)},
		{"summary", "items", fmt.Sprintf("%d", sum.Items)},
		{"summary", "delivered", fmt.Sprintf("%d", sum.Delivered)},
		{"summary", "on_hold", fmt.Sprintf("%d", sum.OnHold)},
		{"summary", "rejected", fmt.Sprintf("%d", sum.Rejected)},
		{"summary", "refunded", fmt.Sprintf("%d", sum.Refunded)},
		{"summary", "net_revenue_minor", fmt.Sprintf("%d", sum.NetRevenueMinor)},
		{"summary", "hold_rate", fmt.Sprintf("%.6f", sum.HoldRate)},
		{"summary", "rejection_rate", fmt.Sprintf("%.6f", sum.RejectionRate)},
		{"summary", "refund_rate", fmt.Sprintf("%.6f", sum.RefundRate)},
		{"summary", "turnaround_avg_hours", fmt.Sprintf("%.4f", sum.Turnaround.AvgHours)},
		{"summary", "turnaround_p50_hours", fmt.Sprintf("%.4f", sum.Turnaround.P50Hours)},
		{"summary", "turnaround_p90_hours", fmt.Sprintf("%.4f", sum.Turnaround.P90Hours)},
		{"summary", "turnaround_p95_hours", fmt.Sprintf("%.4f", sum.Turnaround.P95Hours)},
		{"summary", "turnaround_sample_size", fmt.Sprintf("%d", sum.Turnaround.SampleSize)},
		{"summary", "currency", sum.Currency},
	}
	for _, r := range rows {
		if err := cw.Write(r); err != nil {
			return err
		}
	}
	for _, m := range sum.MethodMix {
		if err := cw.Write([]string{"method_mix", m.Method, fmt.Sprintf("%d", m.Count)}); err != nil {
			return err
		}
	}
	for _, d := range sum.TopDestinations {
		if err := cw.Write([]string{"destination", d.RecipientName, fmt.Sprintf("%d", d.Count)}); err != nil {
			return err
		}
	}
	for _, d := range sum.Daily {
		if err := cw.Write([]string{
			"daily",
			d.Day.UTC().Format("2006-01-02"),
			fmt.Sprintf("orders=%d;items=%d;delivered=%d;on_hold=%d;rejected=%d;refunded=%d;net_revenue_minor=%d",
				d.Orders, d.Items, d.Delivered, d.OnHold, d.Rejected, d.Refunded, d.NetRevenueMinor),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
