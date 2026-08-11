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

/// iOS purchase path when a coupon is considered (MKTC.6).
enum MarketplaceIOSPurchaseRoute: Equatable {
    case freeGrant
    case inAppFullPrice
    case webRedirect
    case reject
    case flagOff
}

/// Marketplace helpers (MKT6 / MOB.7 / MKTC.6). Paid path: Stripe checkout handoff when flagged.
enum MarketplaceLogic {
    static let maxPriceMajor = 99_999.99
    static let minPaidCents = 50
    static let couponInputMaxLen = 32

    private static let couponReasonTokens: Set<String> = [
        "ok",
        "not_found",
        "inactive",
        "not_started",
        "expired",
        "exhausted",
        "already_used",
        "currency_mismatch",
        "course_free",
        "owned",
    ]

    static let currencies = [
        "usd", "eur", "gbp", "cad", "aud", "jpy", "chf", "sek", "nok", "dkk", "nzd", "sgd", "hkd", "mxn",
    ]

    /// In-app claim/buy + Purchased courses library (MOB.7). Default off via platform flag.
    static func purchaseEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCourseMarketplace && features.ffMobileMarketplacePurchase
    }

    /// Coupon UI + preview (MKTC.6). Requires marketplace + coupons flags.
    static func couponsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCourseMarketplace && features.ffCourseCoupons
    }

    private static let couponNormalizedCharset = CharacterSet(charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-")
    private static let couponParamCharset = CharacterSet(charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-")

    /// Upper-case, strip whitespace, keep [A-Z0-9_-], max 32 (MKTC.6 / web parity).
    static func normalizeCouponCode(_ raw: String) -> String {
        let upperNoSpace = raw
            .uppercased()
            .unicodeScalars
            .filter { !$0.properties.isWhitespace }
            .map(String.init)
            .joined()
        let filtered = upperNoSpace.unicodeScalars
            .filter { couponNormalizedCharset.contains($0) }
            .map(String.init)
            .joined()
        return String(filtered.prefix(couponInputMaxLen))
    }

    /// Map a server reason token to `mobile.marketplace.coupon.reason.*`.
    static func couponReasonKey(_ reason: String?) -> String {
        let r = (reason ?? "not_found").lowercased().trimmingCharacters(in: .whitespacesAndNewlines)
        let token = couponReasonTokens.contains(r) ? r : "not_found"
        return "mobile.marketplace.coupon.reason.\(token)"
    }

    /// Length ≤ 32 and charset [A-Za-z0-9_-] (deep-link safety before normalize).
    static func isValidCouponParam(_ raw: String) -> Bool {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, trimmed.count <= couponInputMaxLen else { return false }
        return trimmed.unicodeScalars.allSatisfy { couponParamCharset.contains($0) }
    }

    /// Read `?coupon=` from a query string (`?coupon=X`, `coupon=X`, or full query).
    static func parseCouponFromQuery(_ query: String) -> String? {
        var q = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if q.isEmpty { return nil }
        if q.hasPrefix("?") { q = String(q.dropFirst()) }
        // Support absolute URL or path?query by taking the query portion.
        if let question = q.firstIndex(of: "?") {
            q = String(q[q.index(after: question)...])
        }
        guard let components = URLComponents(string: "https://lextures.local/?\(q)"),
              let items = components.queryItems else {
            return nil
        }
        guard let raw = items.first(where: { $0.name.lowercased() == "coupon" })?.value,
              isValidCouponParam(raw) else {
            return nil
        }
        let code = normalizeCouponCode(raw)
        return code.isEmpty ? nil : code
    }

    /// Pure iOS purchase branch from preview + feature flags (MKTC.6 AC-13).
    static func purchaseRoute(
        preview: CouponPreview?,
        features: MobilePlatformFeatures
    ) -> MarketplaceIOSPurchaseRoute {
        guard couponsEnabled(features) else { return .flagOff }
        guard let preview else { return .inAppFullPrice }
        if !preview.applied {
            return .reject
        }
        if preview.freeAfterDiscount || preview.chargedCents <= 0 {
            return .freeGrant
        }
        if preview.discountCents > 0 {
            return .webRedirect
        }
        return .inAppFullPrice
    }

    /// Storefront path with optional coupon query for browser handoff.
    static func marketplaceWebPath(slug: String, couponCode: String? = nil) -> String {
        let base = "/marketplace/\(slug)"
        guard let raw = couponCode, !raw.isEmpty else { return base }
        let code = normalizeCouponCode(raw)
        guard !code.isEmpty else { return base }
        var allowed = CharacterSet.urlQueryAllowed
        allowed.remove(charactersIn: "&=?")
        let encoded = code.addingPercentEncoding(withAllowedCharacters: allowed) ?? code
        return "\(base)?coupon=\(encoded)"
    }

    static func isPaid(priceCents: Int) -> Bool { priceCents > 0 }

    static func isFree(priceCents: Int) -> Bool { priceCents <= 0 }

    static func formatPrice(cents: Int, currency: String = "usd", freeLabel: String? = nil) -> String {
        if cents <= 0 { return freeLabel ?? L.text("mobile.marketplace.free") }
        return PathsLogic.formatPrice(cents: cents, currency: currency.uppercased())
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

    static func ctaLabelKey(owned: Bool, priceCents: Int, purchaseEnabled: Bool = false) -> String {
        if owned { return "goToCourse" }
        if isFree(priceCents: priceCents) { return "enrollFree" }
        return purchaseEnabled ? "buy" : "buyOnWeb"
    }

    static func purchaseSourceLabelKey(_ source: String) -> String {
        switch source {
        case "free": return "mobile.marketplace.purchases.source.free"
        case "stripe": return "mobile.marketplace.purchases.source.stripe"
        case "apple": return "mobile.marketplace.purchases.source.apple"
        case "comp": return "mobile.marketplace.purchases.source.comp"
        default: return "mobile.marketplace.purchases.source.other"
        }
    }

    static func purchaseSourceLabel(_ source: String) -> String {
        switch source {
        case "free": return L.text("mobile.marketplace.purchases.source.free")
        case "stripe": return L.text("mobile.marketplace.purchases.source.stripe")
        case "apple": return L.text("mobile.marketplace.purchases.source.apple")
        case "comp": return L.text("mobile.marketplace.purchases.source.comp")
        default: return L.text("mobile.marketplace.purchases.source.other")
        }
    }

    static func formatAcquiredAt(_ iso: String) -> String {
        let trimmed = iso.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 10 else { return trimmed }
        return String(trimmed.prefix(10))
    }
}

enum MarketplaceObservability {
    private static var counters: [String: Int] = [:]
    private static let lock = NSLock()

    static func record(_ event: String, attributes: [String: String] = [:]) {
        lock.lock()
        defer { lock.unlock() }
        let key = attributes.isEmpty
            ? event
            : event + "|" + attributes.keys.sorted().map { "\($0)=\(attributes[$0] ?? "")" }.joined(separator: ",")
        counters[key, default: 0] += 1
    }

    static func count(for event: String) -> Int {
        lock.lock()
        defer { lock.unlock() }
        return counters.filter { $0.key == event || $0.key.hasPrefix(event + "|") }.values.reduce(0, +)
    }

    #if DEBUG
    static func resetForTests() {
        lock.lock()
        counters.removeAll()
        lock.unlock()
    }
    #endif
}
