import XCTest
@testable import Lextures

final class MarketplaceLogicTests: XCTestCase {
    override func setUp() {
        super.setUp()
        MarketplaceObservability.resetForTests()
    }

    func testIsPaidAndFree() {
        XCTAssertTrue(MarketplaceLogic.isPaid(priceCents: 1999))
        XCTAssertFalse(MarketplaceLogic.isPaid(priceCents: 0))
        XCTAssertTrue(MarketplaceLogic.isFree(priceCents: 0))
        XCTAssertFalse(MarketplaceLogic.isFree(priceCents: 500))
    }

    func testCardAccessibleName() {
        XCTAssertEqual(
            MarketplaceLogic.cardAccessibleName(
                title: "Spanish",
                priceLabel: "Free",
                owned: true,
                ownedLabel: "Owned"
            ),
            "Spanish, Owned, Free"
        )
        XCTAssertEqual(
            MarketplaceLogic.cardAccessibleName(
                title: "Spanish",
                priceLabel: "$19.99",
                owned: false,
                ownedLabel: "Owned"
            ),
            "Spanish, $19.99"
        )
    }

    func testShouldShowPurchasedBadge() {
        var course = CourseSummary(
            id: "1",
            courseCode: "SPAN101",
            title: "Spanish",
            description: "",
            acquiredViaMarketplace: true
        )
        XCTAssertFalse(
            MarketplaceLogic.shouldShowPurchasedBadge(
                features: MobilePlatformFeatures(),
                course: course
            )
        )
        XCTAssertTrue(
            MarketplaceLogic.shouldShowPurchasedBadge(
                features: MobilePlatformFeatures(ffCourseMarketplace: true),
                course: course
            )
        )
        course.acquiredViaMarketplace = false
        XCTAssertFalse(
            MarketplaceLogic.shouldShowPurchasedBadge(
                features: MobilePlatformFeatures(ffCourseMarketplace: true),
                course: course
            )
        )
    }

    func testMajorUnitsAndValidation() {
        XCTAssertEqual(MarketplaceLogic.majorUnitsToPriceCents(""), 0)
        XCTAssertEqual(MarketplaceLogic.majorUnitsToPriceCents("19.99"), 1999)
        XCTAssertEqual(MarketplaceLogic.majorUnitsToPriceCents("1000", currency: "jpy"), 1000)
        XCTAssertNil(MarketplaceLogic.majorUnitsToPriceCents("abc"))
        XCTAssertNil(MarketplaceLogic.majorUnitsToPriceCents("1000.50", currency: "jpy"))
        XCTAssertNil(MarketplaceLogic.validateAmount(""))
        XCTAssertEqual(MarketplaceLogic.validateAmount("12.345"), "invalid")
        XCTAssertEqual(MarketplaceLogic.validateAmount("0.10"), "min")
        XCTAssertNil(MarketplaceLogic.validateAmount("19.99"))
    }

    func testCtaAndWebPath() {
        XCTAssertEqual(MarketplaceLogic.ctaLabelKey(owned: true, priceCents: 0), "goToCourse")
        XCTAssertEqual(MarketplaceLogic.ctaLabelKey(owned: false, priceCents: 0), "enrollFree")
        XCTAssertEqual(MarketplaceLogic.ctaLabelKey(owned: false, priceCents: 500), "buyOnWeb")
        XCTAssertEqual(
            MarketplaceLogic.ctaLabelKey(owned: false, priceCents: 500, purchaseEnabled: true),
            "buy"
        )
        XCTAssertEqual(MarketplaceLogic.marketplaceWebPath(slug: "spanish-a1"), "/marketplace/spanish-a1")
    }

    func testPurchaseEnabledRequiresBothFlags() {
        var features = MobilePlatformFeatures()
        features.ffMobileMarketplacePurchase = false
        XCTAssertFalse(MarketplaceLogic.purchaseEnabled(features))
        features.ffCourseMarketplace = true
        XCTAssertFalse(MarketplaceLogic.purchaseEnabled(features))
        features.ffMobileMarketplacePurchase = true
        XCTAssertTrue(MarketplaceLogic.purchaseEnabled(features))
    }

    func testPurchaseSourceAndAcquiredFormatting() {
        XCTAssertEqual(
            MarketplaceLogic.purchaseSourceLabelKey("free"),
            "mobile.marketplace.purchases.source.free"
        )
        XCTAssertEqual(
            MarketplaceLogic.purchaseSourceLabelKey("stripe"),
            "mobile.marketplace.purchases.source.stripe"
        )
        XCTAssertEqual(MarketplaceLogic.formatAcquiredAt("2026-07-19T12:00:00Z"), "2026-07-19")
    }

    func testObservabilityCounters() {
        MarketplaceObservability.record("marketplace_viewed")
        MarketplaceObservability.record("marketplace_claim", attributes: ["already_owned": "0"])
        XCTAssertEqual(MarketplaceObservability.count(for: "marketplace_viewed"), 1)
        XCTAssertEqual(MarketplaceObservability.count(for: "marketplace_claim"), 1)
    }

    func testMoreDestinationsGating() {
        let on = MobileDestinations.moreDestinations(
            context: .learning,
            platform: MobilePlatformFeatures(ffCourseMarketplace: true)
        )
        XCTAssertTrue(on.contains(.marketplace))

        let off = MobileDestinations.moreDestinations(
            context: .learning,
            platform: MobilePlatformFeatures(ffCourseMarketplace: false)
        )
        XCTAssertFalse(off.contains(.marketplace))

        let k2 = MobileDestinations.moreDestinations(
            context: .learning,
            platform: MobilePlatformFeatures(ffLibrary: true, ffCourseMarketplace: true),
            uiMode: .k2
        )
        XCTAssertFalse(k2.contains(.marketplace))
    }

    // MARK: - Coupons (MKTC.6)

    func testNormalizeCouponCode() {
        XCTAssertEqual(MarketplaceLogic.normalizeCouponCode("  launch-25  "), "LAUNCH-25")
        XCTAssertEqual(MarketplaceLogic.normalizeCouponCode("ab cd"), "ABCD")
        XCTAssertEqual(MarketplaceLogic.normalizeCouponCode("hello@world!"), "HELLOWORLD")
        XCTAssertEqual(MarketplaceLogic.normalizeCouponCode("a_b-c1"), "A_B-C1")
        let long = String(repeating: "a", count: 40)
        XCTAssertEqual(MarketplaceLogic.normalizeCouponCode(long).count, 32)
        XCTAssertEqual(MarketplaceLogic.normalizeCouponCode(""), "")
    }

    func testIsValidCouponParam() {
        XCTAssertTrue(MarketplaceLogic.isValidCouponParam("LAUNCH25"))
        XCTAssertTrue(MarketplaceLogic.isValidCouponParam("launch-25"))
        XCTAssertTrue(MarketplaceLogic.isValidCouponParam("A_B-1"))
        XCTAssertFalse(MarketplaceLogic.isValidCouponParam(""))
        XCTAssertFalse(MarketplaceLogic.isValidCouponParam("bad code"))
        XCTAssertFalse(MarketplaceLogic.isValidCouponParam("hello@world"))
        XCTAssertFalse(MarketplaceLogic.isValidCouponParam(String(repeating: "x", count: 33)))
        XCTAssertTrue(MarketplaceLogic.isValidCouponParam(String(repeating: "x", count: 32)))
    }

    func testParseCouponFromQuery() {
        XCTAssertEqual(MarketplaceLogic.parseCouponFromQuery("?coupon=launch25"), "LAUNCH25")
        XCTAssertEqual(MarketplaceLogic.parseCouponFromQuery("coupon=SAVE_10&ref=1"), "SAVE_10")
        XCTAssertEqual(MarketplaceLogic.parseCouponFromQuery("COUPON=mixed-Case"), "MIXED-CASE")
        XCTAssertNil(MarketplaceLogic.parseCouponFromQuery("?ref=1"))
        XCTAssertNil(MarketplaceLogic.parseCouponFromQuery("?coupon=bad%20code"))
        XCTAssertNil(MarketplaceLogic.parseCouponFromQuery("?coupon="))
        XCTAssertNil(MarketplaceLogic.parseCouponFromQuery(""))
        // Oversized rejected before normalize
        XCTAssertNil(MarketplaceLogic.parseCouponFromQuery("?coupon=" + String(repeating: "a", count: 33)))
    }

    func testCouponReasonKey() {
        XCTAssertEqual(
            MarketplaceLogic.couponReasonKey("expired"),
            "mobile.marketplace.coupon.reason.expired"
        )
        XCTAssertEqual(
            MarketplaceLogic.couponReasonKey("NOT_FOUND"),
            "mobile.marketplace.coupon.reason.not_found"
        )
        XCTAssertEqual(
            MarketplaceLogic.couponReasonKey("unknown_token"),
            "mobile.marketplace.coupon.reason.not_found"
        )
        XCTAssertEqual(
            MarketplaceLogic.couponReasonKey(nil),
            "mobile.marketplace.coupon.reason.not_found"
        )
        for reason in [
            "ok", "not_found", "inactive", "not_started", "expired",
            "exhausted", "already_used", "currency_mismatch", "course_free", "owned",
        ] {
            let key = MarketplaceLogic.couponReasonKey(reason)
            XCTAssertEqual(key, "mobile.marketplace.coupon.reason.\(reason)")
        }
    }

    func testCouponsEnabledRequiresBothFlags() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(MarketplaceLogic.couponsEnabled(features))
        features.ffCourseMarketplace = true
        XCTAssertFalse(MarketplaceLogic.couponsEnabled(features))
        features.ffCourseCoupons = true
        XCTAssertTrue(MarketplaceLogic.couponsEnabled(features))
    }

    func testPurchaseRouteBranches() {
        var features = MobilePlatformFeatures()
        let appliedFree = CouponPreview(
            applied: true,
            code: "FREE100",
            reason: "ok",
            listPriceCents: 4000,
            discountCents: 4000,
            chargedCents: 0,
            currency: "usd",
            freeAfterDiscount: true
        )
        let appliedPartial = CouponPreview(
            applied: true,
            code: "SAVE25",
            reason: "ok",
            listPriceCents: 4000,
            discountCents: 1000,
            chargedCents: 3000,
            currency: "usd",
            freeAfterDiscount: false
        )
        let rejected = CouponPreview(
            applied: false,
            code: "NOPE",
            reason: "expired",
            listPriceCents: 4000,
            discountCents: 0,
            chargedCents: 4000,
            currency: "usd",
            freeAfterDiscount: false
        )

        // Flag off
        XCTAssertEqual(
            MarketplaceLogic.purchaseRoute(preview: appliedPartial, features: features),
            .flagOff
        )

        features.ffCourseMarketplace = true
        features.ffCourseCoupons = true

        XCTAssertEqual(
            MarketplaceLogic.purchaseRoute(preview: nil, features: features),
            .inAppFullPrice
        )
        XCTAssertEqual(
            MarketplaceLogic.purchaseRoute(preview: rejected, features: features),
            .reject
        )
        XCTAssertEqual(
            MarketplaceLogic.purchaseRoute(preview: appliedFree, features: features),
            .freeGrant
        )
        XCTAssertEqual(
            MarketplaceLogic.purchaseRoute(preview: appliedPartial, features: features),
            .webRedirect
        )

        let appliedZeroDiscount = CouponPreview(
            applied: true,
            code: "ZERO",
            reason: "ok",
            listPriceCents: 4000,
            discountCents: 0,
            chargedCents: 4000,
            currency: "usd",
            freeAfterDiscount: false
        )
        XCTAssertEqual(
            MarketplaceLogic.purchaseRoute(preview: appliedZeroDiscount, features: features),
            .inAppFullPrice
        )
    }

    func testMarketplaceWebPathWithCoupon() {
        XCTAssertEqual(
            MarketplaceLogic.marketplaceWebPath(slug: "spanish-a1"),
            "/marketplace/spanish-a1"
        )
        XCTAssertEqual(
            MarketplaceLogic.marketplaceWebPath(slug: "spanish-a1", couponCode: "launch25"),
            "/marketplace/spanish-a1?coupon=LAUNCH25"
        )
    }

    func testDeepLinkMarketplaceCoupon() {
        guard case let .marketplace(slug, coupon) = DeepLinkRouter.resolve(
            "https://lextures.com/marketplace/spanish-a1?coupon=launch25"
        ) else {
            return XCTFail("expected marketplace destination")
        }
        XCTAssertEqual(slug, "spanish-a1")
        XCTAssertEqual(coupon, "LAUNCH25")

        guard case let .marketplace(slug2, coupon2) = DeepLinkRouter.resolve(
            "lextures://marketplace/demo-course"
        ) else {
            return XCTFail("expected marketplace without coupon")
        }
        XCTAssertEqual(slug2, "demo-course")
        XCTAssertNil(coupon2)

        guard case let .marketplace(_, bad) = DeepLinkRouter.resolve(
            "/marketplace/x?coupon=not%20valid"
        ) else {
            return XCTFail("expected marketplace")
        }
        XCTAssertNil(bad)
    }
}
