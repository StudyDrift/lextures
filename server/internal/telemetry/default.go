package telemetry

import "sync/atomic"

// defaultMetrics holds the process-wide Metrics instance set by Init, so service
// code can emit business and AI metrics via the package-level helpers below
// without threading a *Metrics through every call site. It is nil until Init
// runs (and in unit tests that never call Init), in which case the helpers are
// safe no-ops.
var defaultMetrics atomic.Pointer[Metrics]

// setDefault installs m as the process-wide metrics instance (called by Init).
func setDefault(m *Metrics) { defaultMetrics.Store(m) }

// SetDefaultForTest installs or clears the process-wide metrics instance for tests.
func SetDefaultForTest(m *Metrics) { setDefault(m) }

// Default returns the process-wide Metrics, or nil if telemetry is not started.
func Default() *Metrics { return defaultMetrics.Load() }

// RecordBusinessEvent increments a business metric on the default instance
// (plan 17.7 FR-5e). No-op when telemetry is not initialised.
func RecordBusinessEvent(event string) {
	if m := defaultMetrics.Load(); m != nil {
		m.IncBusinessEvent(event)
	}
}

// SetMarketplaceFlagState records the course marketplace flag on the default
// metrics instance (plan MKT1). No-op when telemetry is not initialised.
func SetMarketplaceFlagState(enabled bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.SetMarketplaceFlagState(enabled)
	}
}

// RecordMarketplaceListingSaved records a marketplace listing save on the default
// metrics instance (plan MKT2). No-op when telemetry is not initialised.
func RecordMarketplaceListingSaved(listed, free bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceListingSaved(listed, free)
	}
}

// RecordMarketplaceStorefrontView records a storefront list view (plan MKT3).
func RecordMarketplaceStorefrontView() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceStorefrontView()
	}
}

// RecordMarketplaceDetailView records a marketplace detail view (plan MKT3).
func RecordMarketplaceDetailView(owned bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceDetailView(owned)
	}
}

// RecordMarketplaceFacetUsage records a filtered storefront search (plan MKT3).
func RecordMarketplaceFacetUsage() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceFacetUsage()
	}
}

// RecordMarketplaceClaim records a free-claim attempt result (plan MKT4).
func RecordMarketplaceClaim(result string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceClaim(result)
	}
}

// RecordMarketplaceCheckoutCreated records a paid checkout session (plan MKT4).
func RecordMarketplaceCheckoutCreated() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceCheckoutCreated()
	}
}

// RecordMarketplacePurchaseCompleted records a paid purchase via webhook (plan MKT4).
func RecordMarketplacePurchaseCompleted() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplacePurchaseCompleted()
	}
}

// RecordMarketplaceRefund records a marketplace course refund (plan MKT4).
func RecordMarketplaceRefund() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceRefund()
	}
}

// RecordMyPurchasesView records a My purchases list view (plan MKT5).
func RecordMyPurchasesView() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMyPurchasesView()
	}
}

// RecordPurchasedBadgeRender records that a courses list included purchased badges (plan MKT5).
func RecordPurchasedBadgeRender() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordPurchasedBadgeRender()
	}
}

// RecordPublicMarketplaceList records an unauthenticated public marketplace list view (plan MKT7).
func RecordPublicMarketplaceList() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordPublicMarketplaceList()
	}
}

// RecordFeedbackSubmitted records a product feedback submission (plan FB0).
func RecordFeedbackSubmitted(source, category string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordFeedbackSubmitted(source, category)
	}
}

// RecordFeedbackSubmitError records a failed feedback submission (plan FB0).
func RecordFeedbackSubmitError() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordFeedbackSubmitError()
	}
}

// ObserveFeedbackAdminList records admin feedback list latency (plan FB0).
func ObserveFeedbackAdminList(seconds float64) {
	if m := defaultMetrics.Load(); m != nil {
		m.ObserveFeedbackAdminList(seconds)
	}
}

// RecordOnboardingEventInsertFailed records a failed onboarding_events insert (plan HS.5).
func RecordOnboardingEventInsertFailed(program string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordOnboardingEventInsertFailed(program)
	}
}

// RecordPinnedSettingsWrite records a successful pinned-settings PUT (plan PS.2).
func RecordPinnedSettingsWrite(surface string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordPinnedSettingsWrite(surface)
	}
}

// ObservePinnedSettingsPinCount records pin-list length on successful PUT (plan PS.4).
func ObservePinnedSettingsPinCount(count int) {
	if m := defaultMetrics.Load(); m != nil {
		m.ObservePinnedSettingsPinCount(count)
	}
}

// RecordPinnedSettingsReject records a rejected pinned-settings write (plan PS.2).
func RecordPinnedSettingsReject(reason string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordPinnedSettingsReject(reason)
	}
}

// RecordAIProvider records an AI provider call on the default instance
// (plan 16.7 / 17.7 §11). No-op when telemetry is not initialised.
func RecordAIProvider(provider, model, outcome string, seconds, costDollars float64) {
	if m := defaultMetrics.Load(); m != nil {
		m.ObserveAIProvider(provider, model, outcome, seconds, costDollars)
	}
}

// RecordCouponReserve records a coupon reservation attempt (plan MKTC.1).
func RecordCouponReserve(result string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponReserve(result)
	}
}

// RecordCouponRedeem records a coupon redeem attempt (plan MKTC.1).
func RecordCouponRedeem(result string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponRedeem(result)
	}
}

// RecordCouponRelease records a coupon release (plan MKTC.1).
func RecordCouponRelease(reason string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponRelease(reason)
	}
}

// RecordCouponReservationExpired records expired reservation sweeps (plan MKTC.1).
func RecordCouponReservationExpired(n int) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponReservationExpired(n)
	}
}

// RecordCouponAdminRequest records a creator coupon admin API request (plan MKTC.2).
func RecordCouponAdminRequest(route, result string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponAdminRequest(route, result)
	}
}

// RecordCouponCreated records a coupon create (plan MKTC.2).
func RecordCouponCreated(discountType string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponCreated(discountType)
	}
}

// RecordCouponStatusChanged records a coupon status transition (plan MKTC.2).
func RecordCouponStatusChanged(to string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponStatusChanged(to)
	}
}

// RecordCouponApply records a learner coupon preview/apply attempt (plan MKTC.3).
func RecordCouponApply(result string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponApply(result)
	}
}

// RecordCouponCheckoutCreated records a discounted marketplace checkout session (plan MKTC.3).
func RecordCouponCheckoutCreated(discounted bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponCheckoutCreated(discounted)
	}
}

// RecordCouponRedeemed records a completed coupon redemption (plan MKTC.3).
func RecordCouponRedeemed() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponRedeemed()
	}
}

// RecordCouponDiscountCents records discount cents granted (plan MKTC.3).
func RecordCouponDiscountCents(n int) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponDiscountCents(n)
	}
}

// RecordCouponFreeGrant records a 100%-off / clamp-to-free grant (plan MKTC.3).
func RecordCouponFreeGrant() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponFreeGrant()
	}
}

// RecordCouponClampedToFree records a residual charge clamped to free (plan MKTC.3).
func RecordCouponClampedToFree() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponClampedToFree()
	}
}

// RecordCouponApplyCooldown records a cool-down rejection on coupon apply (plan MKTC.7).
func RecordCouponApplyCooldown() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponApplyCooldown()
	}
}

// RecordCouponWebRedirect records a mobile→web coupon checkout redirect (plan MKTC.7).
func RecordCouponWebRedirect(platform string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordCouponWebRedirect(platform)
	}
}
