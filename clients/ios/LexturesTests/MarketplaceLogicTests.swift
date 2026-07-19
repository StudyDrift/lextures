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
}
