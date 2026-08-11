import Foundation

// MARK: - Marketplace / coupon deep-link handoff (MKTC.6)

extension AppShellModel {
    func openMarketplaceDeepLink(slug: String, couponCode: String?) {
        let trimmedSlug = slug.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedSlug.isEmpty else { return }
        pendingMarketplaceSlug = trimmedSlug
        // Session-only handoff; MarketplaceDetailView no-ops when ffCourseCoupons is off (AC-11).
        if let couponCode, !couponCode.isEmpty {
            pendingCoupon = (slug: trimmedSlug, code: couponCode)
            MarketplaceObservability.record("coupon_from_deeplink", attributes: ["slug": trimmedSlug])
        }
        selectShellTab(.profile)
        pendingMoreDestination = .marketplace
    }

    func consumePendingMarketplaceSlug() -> String? {
        defer { pendingMarketplaceSlug = nil }
        return pendingMarketplaceSlug
    }

    /// Returns a pending coupon for `slug` without clearing (feature flags may still be loading).
    func peekPendingCoupon(for slug: String) -> String? {
        guard let pending = pendingCoupon, pending.slug == slug else { return nil }
        return pending.code
    }

    /// Returns and clears a pending coupon when it matches `slug`.
    func consumePendingCoupon(for slug: String) -> String? {
        guard let pending = pendingCoupon, pending.slug == slug else { return nil }
        pendingCoupon = nil
        return pending.code
    }

    func clearPendingCoupon(for slug: String? = nil) {
        if let slug {
            if pendingCoupon?.slug == slug {
                pendingCoupon = nil
            }
        } else {
            pendingCoupon = nil
        }
    }
}
