import Foundation

/// Presentation helpers for the Homeschool subscribe paywall (App Store 3.1.2(c)).
enum HomeschoolPaywallLogic {
    enum PlanKind: Equatable {
        case monthly
        case annual
    }

    enum PeriodUnit: Equatable {
        case day
        case week
        case month
        case year
    }

    struct Period: Equatable {
        var value: Int
        var unit: PeriodUnit
    }

    /// Public marketing legal pages — not the API origin — so App Review
    /// always gets a live document even when the app is pointed at a tenant host.
    static let privacyPolicyURL = URL(string: "https://lextures.com/privacy")!
    static let termsOfUseURL = URL(string: "https://lextures.com/terms")!

    /// Yearly first when both plans exist (higher LTV, lower cognitive load).
    static func defaultPlan(hasMonthly: Bool, hasAnnual: Bool) -> PlanKind? {
        if hasAnnual { return .annual }
        if hasMonthly { return .monthly }
        return nil
    }

    static func weeksInPeriod(_ period: Period) -> Decimal {
        guard period.value > 0 else { return 0 }
        switch period.unit {
        case .day:
            return Decimal(period.value) / 7
        case .week:
            return Decimal(period.value)
        case .month:
            return Decimal(period.value) * Decimal(string: "4.345")!
        case .year:
            return Decimal(period.value) * 52
        }
    }

    /// Price per week for Apple's "price per unit" disclosure.
    static func weeklyPrice(price: Decimal, period: Period) -> Decimal? {
        let weeks = weeksInPeriod(period)
        guard weeks > 0 else { return nil }
        return price / weeks
    }

    /// Whole-percent savings of annual vs twelve monthly charges. Nil when not cheaper.
    static func annualSavingsPercent(monthlyPrice: Decimal, annualPrice: Decimal) -> Int? {
        let yearOfMonthly = monthlyPrice * 12
        guard yearOfMonthly > 0, yearOfMonthly > annualPrice else { return nil }
        let fraction = (yearOfMonthly - annualPrice) / yearOfMonthly
        let percent = NSDecimalNumber(decimal: fraction * 100).rounding(
            accordingToBehavior: NSDecimalNumberHandler(
                roundingMode: .down,
                scale: 0,
                raiseOnExactness: false,
                raiseOnOverflow: false,
                raiseOnUnderflow: false,
                raiseOnDivideByZero: false
            )
        ).intValue
        return percent > 0 ? percent : nil
    }

    static func periodLabelKey(for unit: PeriodUnit) -> String {
        switch unit {
        case .day: return "mobile.billing.paywall.everyDays"
        case .week: return "mobile.billing.paywall.everyWeeks"
        case .month: return "mobile.billing.paywall.everyMonths"
        case .year: return "mobile.billing.paywall.everyYears"
        }
    }
}
