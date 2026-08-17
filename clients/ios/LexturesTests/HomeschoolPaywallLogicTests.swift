import XCTest
@testable import Lextures

final class HomeschoolPaywallLogicTests: XCTestCase {
    func testLegalURLsAreAbsoluteHTTPSOnTheMarketingSite() {
        let privacy = HomeschoolPaywallLogic.privacyPolicyURL
        let terms = HomeschoolPaywallLogic.termsOfUseURL
        XCTAssertEqual(privacy.scheme, "https")
        XCTAssertEqual(terms.scheme, "https")
        XCTAssertEqual(privacy.host, "lextures.com")
        XCTAssertEqual(terms.host, "lextures.com")
        XCTAssertEqual(privacy.path, "/privacy")
        XCTAssertEqual(terms.path, "/terms")
    }

    func testDefaultPlanPrefersAnnual() {
        XCTAssertEqual(
            HomeschoolPaywallLogic.defaultPlan(hasMonthly: true, hasAnnual: true),
            .annual
        )
        XCTAssertEqual(
            HomeschoolPaywallLogic.defaultPlan(hasMonthly: true, hasAnnual: false),
            .monthly
        )
        XCTAssertEqual(
            HomeschoolPaywallLogic.defaultPlan(hasMonthly: false, hasAnnual: true),
            .annual
        )
        XCTAssertNil(HomeschoolPaywallLogic.defaultPlan(hasMonthly: false, hasAnnual: false))
    }

    func testWeeklyPriceForMonthlyAndAnnual() {
        let monthly = HomeschoolPaywallLogic.Period(value: 1, unit: .month)
        let weeklyFromMonth = HomeschoolPaywallLogic.weeklyPrice(price: Decimal(string: "9.99")!, period: monthly)
        XCTAssertNotNil(weeklyFromMonth)
        XCTAssertEqual(weeklyFromMonth!, Decimal(string: "9.99")! / Decimal(string: "4.345")!)

        let annual = HomeschoolPaywallLogic.Period(value: 1, unit: .year)
        let weeklyFromYear = HomeschoolPaywallLogic.weeklyPrice(price: Decimal(string: "99.99")!, period: annual)
        XCTAssertEqual(weeklyFromYear, Decimal(string: "99.99")! / 52)
    }

    func testWeeklyPriceRejectsEmptyPeriod() {
        let empty = HomeschoolPaywallLogic.Period(value: 0, unit: .month)
        XCTAssertNil(HomeschoolPaywallLogic.weeklyPrice(price: 10, period: empty))
    }

    func testAnnualSavingsPercent() {
        XCTAssertEqual(
            HomeschoolPaywallLogic.annualSavingsPercent(monthlyPrice: 10, annualPrice: 96),
            20
        )
        XCTAssertNil(HomeschoolPaywallLogic.annualSavingsPercent(monthlyPrice: 10, annualPrice: 120))
        XCTAssertNil(HomeschoolPaywallLogic.annualSavingsPercent(monthlyPrice: 0, annualPrice: 99))
    }

    func testPeriodLabelKeys() {
        XCTAssertEqual(
            HomeschoolPaywallLogic.periodLabelKey(for: .month),
            "mobile.billing.paywall.everyMonths"
        )
        XCTAssertEqual(
            HomeschoolPaywallLogic.periodLabelKey(for: .year),
            "mobile.billing.paywall.everyYears"
        )
    }
}
