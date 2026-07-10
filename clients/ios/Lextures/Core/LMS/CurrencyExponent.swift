import Foundation

enum CurrencyExponent {
    private static let zeroDecimalCurrencies: Set<String> = ["jpy"]

    static func isZeroDecimal(_ currency: String) -> Bool {
        zeroDecimalCurrencies.contains(currency.lowercased().trimmingCharacters(in: .whitespacesAndNewlines))
    }

    static func minorUnitFactor(_ currency: String) -> Int {
        isZeroDecimal(currency) ? 1 : 100
    }

    static func minorUnitsToMajorUnits(_ minor: Int, currency: String) -> Double {
        Double(minor) / Double(minorUnitFactor(currency))
    }

    static func majorUnitsToMinorUnits(_ major: Double, currency: String) -> Int {
        Int((major * Double(minorUnitFactor(currency))).rounded())
    }

    static func maxCatalogMinorUnits(_ currency: String) -> Int {
        isZeroDecimal(currency) ? 99_999 : 9_999_999
    }

    static let maxPriceMajorZeroDecimal = 99_999.0
}