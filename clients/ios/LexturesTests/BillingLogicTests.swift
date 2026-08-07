import XCTest
@testable import Lextures

final class BillingLogicTests: XCTestCase {
    func testBillingEnabledRequiresFeatureFlag() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(BillingLogic.billingEnabled(features))
        features.ffStripeBilling = true
        XCTAssertTrue(BillingLogic.billingEnabled(features))
        features.ffStripeBilling = false
        features.ffPaymentsEnabled = true
        XCTAssertTrue(BillingLogic.billingEnabled(features))
    }

    func testCheckoutUrlsIncludeCourseId() {
        let path = BillingLogic.checkoutSuccessPath(courseId: "abc-123")
        XCTAssertTrue(path.contains("course_id=abc-123"))
        XCTAssertEqual(BillingLogic.checkoutCancelPath(), "/checkout/cancel")
    }

    func testCheckoutEndpointSelection() {
        XCTAssertEqual(BillingLogic.checkoutEndpoint(usePaymentsAbstraction: true), "/api/v1/checkout")
        XCTAssertEqual(BillingLogic.checkoutEndpoint(usePaymentsAbstraction: false), "/api/v1/billing/checkout")
    }

    func testEntitlementLabels() {
        XCTAssertEqual(BillingLogic.entitlementLabel("course_purchase"), L.text("mobile.billing.entitlement.coursePurchase"))
        XCTAssertTrue(BillingLogic.isSubscriptionEntitlement("subscription_monthly"))
        XCTAssertFalse(BillingLogic.isSubscriptionEntitlement("course_purchase"))
    }

    func testActiveSubscriptionPicksFirstActive() {
        let entitlements = [
            BillingEntitlement(
                id: "1",
                entitlementType: "course_purchase",
                amountPaidCents: 1000,
                currency: "USD",
                validFrom: "2026-01-01T00:00:00Z",
                status: "active"
            ),
            BillingEntitlement(
                id: "2",
                entitlementType: "subscription_monthly",
                amountPaidCents: 999,
                currency: "USD",
                validFrom: "2026-01-01T00:00:00Z",
                status: "active"
            ),
        ]
        XCTAssertEqual(BillingLogic.activeSubscription(entitlements)?.id, "2")
    }

    func testPrefersInAppPurchaseOnIOS() {
        XCTAssertTrue(BillingLogic.prefersInAppPurchase)
    }

    func testPreferredAppleProductPrefersCourseMatch() {
        let products = [
            AppleIAPProductInfo(productId: "sub.m", kind: "subscription_monthly"),
            AppleIAPProductInfo(productId: "course.a", kind: "course_purchase", courseId: "aaa"),
            AppleIAPProductInfo(productId: "course.b", kind: "course_purchase", courseId: "bbb"),
        ]
        XCTAssertEqual(
            BillingLogic.preferredAppleProduct(from: products, courseId: "bbb")?.productId,
            "course.b"
        )
        XCTAssertEqual(
            BillingLogic.preferredAppleProduct(from: products, courseId: nil)?.productId,
            "sub.m"
        )
    }

    func testAppStoreSubscriptionsURL() {
        XCTAssertEqual(
            BillingLogic.appStoreSubscriptionsURL().absoluteString,
            "https://apps.apple.com/account/subscriptions"
        )
    }
}
