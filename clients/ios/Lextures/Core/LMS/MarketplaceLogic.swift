import Foundation

enum MarketplacePriceFilter: String, CaseIterable, Identifiable {
    case any
    case free
    case paid

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .any: return "mobile.marketplace.filter.priceAny"
        case .free: return "mobile.marketplace.filter.priceFree"
        case .paid: return "mobile.marketplace.filter.pricePaid"
        }
    }

    var freeOnly: Bool { self == .free }

    var priceMax: Int? {
        switch self {
        case .any, .paid: return nil
        case .free: return 0
        }
    }
}

enum MarketplaceLevelFilter: String, CaseIterable, Identifiable {
    case any
    case beginner
    case intermediate
    case advanced

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .any: return "mobile.marketplace.filter.levelAny"
        case .beginner: return "mobile.marketplace.filter.levelBeginner"
        case .intermediate: return "mobile.marketplace.filter.levelIntermediate"
        case .advanced: return "mobile.marketplace.filter.levelAdvanced"
        }
    }

    var queryValue: String? { self == .any ? nil : rawValue }
}

enum MarketplaceSortMode: String, CaseIterable, Identifiable {
    case popular
    case rating
    case newest
    case relevance
    case price

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .popular: return "mobile.marketplace.sort.popular"
        case .rating: return "mobile.marketplace.sort.rating"
        case .newest: return "mobile.marketplace.sort.newest"
        case .relevance: return "mobile.marketplace.sort.relevance"
        case .price: return "mobile.marketplace.sort.price"
        }
    }
}

/// Marketplace helpers (plan MKT6). Paid purchases use Path B (web hand-off).
enum MarketplaceLogic {
    static let maxPriceMajor = 99_999.99
    static let minPaidCents = 50

    static let currencies = [
        "usd", "eur", "gbp", "cad", "aud", "jpy", "chf", "sek", "nok", "dkk", "nzd", "sgd", "hkd", "mxn",
    ]

    static func isPaid(priceCents: Int) -> Bool { priceCents > 0 }

    static func isFree(priceCents: Int) -> Bool { priceCents <= 0 }

    static func formatPrice(cents: Int, currency: String = "usd", freeLabel: String? = nil) -> String {
        if cents <= 0 { return freeLabel ?? L.text("mobile.marketplace.free") }
        return PathsLogic.formatPrice(cents: cents, currency: currency.uppercased())
    }

    static func marketplaceWebPath(slug: String) -> String {
        "/marketplace/\(slug)"
    }

    static func cacheKey(
        query: String,
        category: String,
        level: MarketplaceLevelFilter,
        price: MarketplacePriceFilter,
        sort: MarketplaceSortMode
    ) -> String {
        "\(query)|\(category)|\(level.rawValue)|\(price.rawValue)|\(sort.rawValue)"
    }

    static func cardAccessibleName(
        title: String,
        priceLabel: String,
        owned: Bool,
        ownedLabel: String
    ) -> String {
        owned ? "\(title), \(ownedLabel), \(priceLabel)" : "\(title), \(priceLabel)"
    }

    static func shouldShowPurchasedBadge(
        features: MobilePlatformFeatures,
        course: CourseSummary
    ) -> Bool {
        features.ffCourseMarketplace && course.acquiredViaMarketplace == true
    }

    static func previewParagraphs(from description: String, limit: Int = 3) -> [String] {
        description
            .components(separatedBy: .newlines)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .prefix(limit)
            .map { String($0) }
    }

    static func majorUnitsToPriceCents(_ amount: String, currency: String = "usd") -> Int? {
        let trimmed = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return 0 }
        let pattern = CurrencyExponent.isZeroDecimal(currency) ? #"^\d+$"# : #"^\d+(\.\d{1,2})?$"#
        let maxMajor = CurrencyExponent.isZeroDecimal(currency) ? CurrencyExponent.maxPriceMajorZeroDecimal : maxPriceMajor
        guard trimmed.range(of: pattern, options: .regularExpression) != nil,
              let value = Double(trimmed),
              value >= 0,
              value <= maxMajor else {
            return nil
        }
        return CurrencyExponent.majorUnitsToMinorUnits(value, currency: currency)
    }

    static func priceCentsToMajorUnits(_ priceCents: Int, currency: String = "usd") -> String {
        if priceCents <= 0 { return "" }
        let major = CurrencyExponent.minorUnitsToMajorUnits(priceCents, currency: currency)
        return CurrencyExponent.isZeroDecimal(currency)
            ? String(Int(major.rounded()))
            : String(format: "%.2f", major)
    }

    static func validateAmount(_ amount: String, currency: String = "usd") -> String? {
        let trimmed = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return nil }
        guard let cents = majorUnitsToPriceCents(trimmed, currency: currency) else { return "invalid" }
        if cents < 0 { return "negative" }
        if cents > 0 && cents < minPaidCents { return "min" }
        if cents > CurrencyExponent.maxCatalogMinorUnits(currency) { return "max" }
        return nil
    }

    static func buildListingPutBody(
        listing: CourseCatalogListing,
        marketplaceListed: Bool,
        priceCents: Int,
        priceCurrency: String
    ) -> CourseCatalogListingPutBody {
        CourseCatalogListingPutBody(
            isPublic: listing.isPublic,
            category: listing.category,
            difficultyLevel: listing.difficultyLevel,
            language: listing.language,
            priceCents: priceCents,
            priceCurrency: priceCurrency,
            slug: listing.slug,
            marketplaceListed: marketplaceListed
        )
    }

    static func ctaLabelKey(owned: Bool, priceCents: Int) -> String {
        if owned { return "goToCourse" }
        if isFree(priceCents: priceCents) { return "enrollFree" }
        return "buyOnWeb"
    }
}
