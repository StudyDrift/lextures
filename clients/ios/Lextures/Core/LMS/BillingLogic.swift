import Foundation

/// Checkout and billing helpers (M9.2). iOS Path A uses StoreKit IAP for digital goods;
/// web/Android continue to use Stripe browser checkout.
enum BillingLogic {
    static let entitlementPollAttempts = 10
    static let entitlementPollIntervalSeconds: UInt64 = 1

    static func billingEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffStripeBilling || features.ffPaymentsEnabled
    }

    /// iOS always purchases digital goods via StoreKit when the server maps a product.
    static var prefersInAppPurchase: Bool { true }

    /// Prefer a course non-consumable product; fall back to first product id.
    static func preferredAppleProduct(
        from products: [AppleIAPProductInfo],
        courseId: String?
    ) -> AppleIAPProductInfo? {
        if let courseId, !courseId.isEmpty {
            if let match = products.first(where: { $0.isCoursePurchase && $0.courseId == courseId }) {
                return match
            }
            if let anyCourse = products.first(where: \.isCoursePurchase) {
                return anyCourse
            }
        }
        return products.first
    }

    static func appStoreSubscriptionsURL() -> URL {
        URL(string: "https://apps.apple.com/account/subscriptions")!
    }

    static func checkoutSuccessPath(courseId: String) -> String {
        let encoded = courseId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? courseId
        return "/checkout/success?course_id=\(encoded)"
    }

    static func checkoutCancelPath() -> String { "/checkout/cancel" }

    static func checkoutSuccessURL(courseId: String) -> URL {
        AppConfiguration.webURL(path: checkoutSuccessPath(courseId: courseId))
    }

    static func checkoutCancelURL() -> URL {
        AppConfiguration.webURL(path: checkoutCancelPath())
    }

    static func billingReturnPath() -> String { "/me/billing" }

    static func billingReturnURL() -> URL {
        AppConfiguration.webURL(path: billingReturnPath())
    }

    static func formatMoney(cents: Int, currency: String = "USD") -> String {
        CatalogLogic.formatPrice(cents: cents, currency: currency)
    }

    static func entitlementLabel(_ type: String) -> String {
        switch type {
        case "course_purchase":
            return L.text("mobile.billing.entitlement.coursePurchase")
        case "subscription_monthly":
            return L.text("mobile.billing.entitlement.subscriptionMonthly")
        case "subscription_annual":
            return L.text("mobile.billing.entitlement.subscriptionAnnual")
        default:
            return type
        }
    }

    static func isSubscriptionEntitlement(_ type: String) -> Bool {
        type.hasPrefix("subscription")
    }

    static func activeSubscription(_ entitlements: [BillingEntitlement]) -> BillingEntitlement? {
        entitlements.first { isSubscriptionEntitlement($0.entitlementType) && $0.status == "active" }
    }

    static func quoteLineItems(_ quote: CheckoutTaxQuote) -> [(label: String, cents: Int)] {
        if !quote.lines.isEmpty {
            return quote.lines.map { ($0.label, $0.amountCents) }
        }
        var rows: [(String, Int)] = [
            (L.text("mobile.billing.subtotal"), quote.subtotalCents),
        ]
        if quote.taxAmountCents > 0 {
            rows.append((L.text("mobile.billing.tax"), quote.taxAmountCents))
        }
        return rows
    }

    static func checkoutEndpoint(usePaymentsAbstraction: Bool) -> String {
        usePaymentsAbstraction ? "/api/v1/checkout" : "/api/v1/billing/checkout"
    }
}
