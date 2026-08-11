// Coupon Prometheus metrics (plan MKTC.1–MKTC.7).
package telemetry

import "github.com/prometheus/client_golang/prometheus"

func (m *Metrics) registerCouponMetrics(reg *prometheus.Registry) {
	m.couponReserveTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_reserve_total",
		Help:      "Coupon reservation attempts by result (plan MKTC.1).",
	}, []string{"result"})
	m.couponRedeemTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_redeem_total",
		Help:      "Coupon redeem attempts by result (plan MKTC.1).",
	}, []string{"result"})
	m.couponReleaseTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_release_total",
		Help:      "Coupon reservation/redemption releases by reason (plan MKTC.1).",
	}, []string{"reason"})
	m.couponReservationExpiredTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_reservation_expired_total",
		Help:      "Coupon reservations released by the expired-reservation sweeper (plan MKTC.1).",
	})
	m.couponAdminRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_admin_request_total",
		Help:      "Creator coupon admin API requests by route and result (plan MKTC.2).",
	}, []string{"route", "result"})
	m.couponCreatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_created_total",
		Help:      "Course coupons created by discount type (plan MKTC.2).",
	}, []string{"discount_type"})
	m.couponStatusChangedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_status_changed_total",
		Help:      "Course coupon status transitions by target status (plan MKTC.2).",
	}, []string{"to"})
	m.couponApplyTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_apply_total",
		Help:      "Learner coupon preview/apply attempts by result (plan MKTC.3).",
	}, []string{"result"})
	m.couponCheckoutCreatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_checkout_created_total",
		Help:      "Marketplace checkout sessions created with a first-party coupon (plan MKTC.3).",
	}, []string{"discounted"})
	m.couponRedeemedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_redeemed_total",
		Help:      "Coupon redemptions completed (paid webhook or free grant) (plan MKTC.3).",
	})
	m.couponDiscountCentsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_discount_cents_total",
		Help:      "Total discount cents granted via course coupons (plan MKTC.3).",
	})
	m.couponFreeGrantTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_free_grant_total",
		Help:      "100%-off / clamp-to-free coupon grants (plan MKTC.3).",
	})
	m.couponClampedToFreeTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_clamped_to_free_total",
		Help:      "Discounts clamped to free because residual was below provider minimum (plan MKTC.3).",
	})
	m.couponApplyCooldownTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_apply_cooldown_total",
		Help:      "Coupon apply attempts rejected by consecutive-failure cool-down (plan MKTC.7).",
	})
	m.couponWebRedirectTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "coupon_web_redirect_total",
		Help:      "Mobile coupon deep-link redirects to web checkout by platform (plan MKTC.6/MKTC.7).",
	}, []string{"platform"})

	reg.MustRegister(
		m.couponReserveTotal,
		m.couponRedeemTotal,
		m.couponReleaseTotal,
		m.couponReservationExpiredTotal,
		m.couponAdminRequestTotal,
		m.couponCreatedTotal,
		m.couponStatusChangedTotal,
		m.couponApplyTotal,
		m.couponCheckoutCreatedTotal,
		m.couponRedeemedTotal,
		m.couponDiscountCentsTotal,
		m.couponFreeGrantTotal,
		m.couponClampedToFreeTotal,
		m.couponApplyCooldownTotal,
		m.couponWebRedirectTotal,
	)
}

// RecordCouponReserve increments coupon_reserve_total{result} (plan MKTC.1).
func (m *Metrics) RecordCouponReserve(result string) {
	if m == nil || m.couponReserveTotal == nil {
		return
	}
	if result == "" {
		result = "unknown"
	}
	m.couponReserveTotal.WithLabelValues(result).Inc()
}

// RecordCouponRedeem increments coupon_redeem_total{result} (plan MKTC.1).
func (m *Metrics) RecordCouponRedeem(result string) {
	if m == nil || m.couponRedeemTotal == nil {
		return
	}
	if result == "" {
		result = "unknown"
	}
	m.couponRedeemTotal.WithLabelValues(result).Inc()
}

// RecordCouponRelease increments coupon_release_total{reason} (plan MKTC.1).
func (m *Metrics) RecordCouponRelease(reason string) {
	if m == nil || m.couponReleaseTotal == nil {
		return
	}
	if reason == "" {
		reason = "unknown"
	}
	m.couponReleaseTotal.WithLabelValues(reason).Inc()
}

// RecordCouponReservationExpired increments coupon_reservation_expired_total by n (plan MKTC.1).
func (m *Metrics) RecordCouponReservationExpired(n int) {
	if m == nil || m.couponReservationExpiredTotal == nil || n <= 0 {
		return
	}
	m.couponReservationExpiredTotal.Add(float64(n))
}

// RecordCouponAdminRequest increments coupon_admin_request_total{route,result} (plan MKTC.2).
func (m *Metrics) RecordCouponAdminRequest(route, result string) {
	if m == nil || m.couponAdminRequestTotal == nil {
		return
	}
	if route == "" {
		route = "unknown"
	}
	if result == "" {
		result = "unknown"
	}
	m.couponAdminRequestTotal.WithLabelValues(route, result).Inc()
}

// RecordCouponCreated increments coupon_created_total{discount_type} (plan MKTC.2).
func (m *Metrics) RecordCouponCreated(discountType string) {
	if m == nil || m.couponCreatedTotal == nil {
		return
	}
	if discountType == "" {
		discountType = "unknown"
	}
	m.couponCreatedTotal.WithLabelValues(discountType).Inc()
}

// RecordCouponStatusChanged increments coupon_status_changed_total{to} (plan MKTC.2).
func (m *Metrics) RecordCouponStatusChanged(to string) {
	if m == nil || m.couponStatusChangedTotal == nil {
		return
	}
	if to == "" {
		to = "unknown"
	}
	m.couponStatusChangedTotal.WithLabelValues(to).Inc()
}

// RecordCouponApply increments coupon_apply_total{result} (plan MKTC.3).
func (m *Metrics) RecordCouponApply(result string) {
	if m == nil || m.couponApplyTotal == nil {
		return
	}
	if result == "" {
		result = "unknown"
	}
	m.couponApplyTotal.WithLabelValues(result).Inc()
}

// RecordCouponCheckoutCreated increments coupon_checkout_created_total{discounted} (plan MKTC.3).
func (m *Metrics) RecordCouponCheckoutCreated(discounted bool) {
	if m == nil || m.couponCheckoutCreatedTotal == nil {
		return
	}
	m.couponCheckoutCreatedTotal.WithLabelValues(boolLabel(discounted)).Inc()
}

// RecordCouponRedeemed increments coupon_redeemed_total (plan MKTC.3).
func (m *Metrics) RecordCouponRedeemed() {
	if m == nil || m.couponRedeemedTotal == nil {
		return
	}
	m.couponRedeemedTotal.Inc()
}

// RecordCouponDiscountCents adds n to coupon_discount_cents_total (plan MKTC.3).
func (m *Metrics) RecordCouponDiscountCents(n int) {
	if m == nil || m.couponDiscountCentsTotal == nil || n <= 0 {
		return
	}
	m.couponDiscountCentsTotal.Add(float64(n))
}

// RecordCouponFreeGrant increments coupon_free_grant_total (plan MKTC.3).
func (m *Metrics) RecordCouponFreeGrant() {
	if m == nil || m.couponFreeGrantTotal == nil {
		return
	}
	m.couponFreeGrantTotal.Inc()
}

// RecordCouponClampedToFree increments coupon_clamped_to_free_total (plan MKTC.3).
func (m *Metrics) RecordCouponClampedToFree() {
	if m == nil || m.couponClampedToFreeTotal == nil {
		return
	}
	m.couponClampedToFreeTotal.Inc()
}

// RecordCouponApplyCooldown increments coupon_apply_cooldown_total (plan MKTC.7).
func (m *Metrics) RecordCouponApplyCooldown() {
	if m == nil || m.couponApplyCooldownTotal == nil {
		return
	}
	m.couponApplyCooldownTotal.Inc()
}

// RecordCouponWebRedirect increments coupon_web_redirect_total{platform} (plan MKTC.7).
func (m *Metrics) RecordCouponWebRedirect(platform string) {
	if m == nil || m.couponWebRedirectTotal == nil {
		return
	}
	if platform == "" {
		platform = "unknown"
	}
	m.couponWebRedirectTotal.WithLabelValues(platform).Inc()
}

func boolLabel(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
