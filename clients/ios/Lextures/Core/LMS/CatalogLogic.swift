import Foundation

enum CatalogBrowseTab: String, CaseIterable, Identifiable {
    case courses
    case paths

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .courses: return "mobile.catalog.tab.courses"
        case .paths: return "mobile.catalog.tab.paths"
        }
    }
}

enum CatalogPriceFilter: String, CaseIterable, Identifiable {
    case any
    case free
    case paid

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .any: return "mobile.catalog.filter.priceAny"
        case .free: return "mobile.catalog.filter.priceFree"
        case .paid: return "mobile.catalog.filter.pricePaid"
        }
    }

    var priceMax: Int? {
        switch self {
        case .any, .paid: return nil
        case .free: return 0
        }
    }
}

enum CatalogLevelFilter: String, CaseIterable, Identifiable {
    case any
    case beginner
    case intermediate
    case advanced

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .any: return "mobile.catalog.filter.levelAny"
        case .beginner: return "mobile.catalog.filter.levelBeginner"
        case .intermediate: return "mobile.catalog.filter.levelIntermediate"
        case .advanced: return "mobile.catalog.filter.levelAdvanced"
        }
    }

    var queryValue: String? {
        self == .any ? nil : rawValue
    }
}

enum CatalogSortMode: String, CaseIterable, Identifiable {
    case popular
    case rating
    case newest
    case relevance

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .popular: return "mobile.catalog.sort.popular"
        case .rating: return "mobile.catalog.sort.rating"
        case .newest: return "mobile.catalog.sort.newest"
        case .relevance: return "mobile.catalog.sort.relevance"
        }
    }
}

enum CatalogLogic {
    static func isPaid(priceCents: Int) -> Bool { priceCents > 0 }

    static func isFree(priceCents: Int) -> Bool { priceCents <= 0 }

    static func formatPrice(cents: Int, currency: String = "USD") -> String {
        if cents <= 0 { return L.text("mobile.catalog.free") }
        return PathsLogic.formatPrice(cents: cents, currency: currency)
    }

    static func catalogWebPath(slug: String) -> String {
        "/explore/\(slug)"
    }

    static func cacheKey(
        query: String,
        category: String,
        level: CatalogLevelFilter,
        price: CatalogPriceFilter,
        sort: CatalogSortMode
    ) -> String {
        "\(query)|\(category)|\(level.rawValue)|\(price.rawValue)|\(sort.rawValue)"
    }

    static func isEnrolled(courseCode: String, in courses: [CourseSummary]) -> Bool {
        let code = courseCode.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !code.isEmpty else { return false }
        return courses.contains {
            $0.courseCode.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == code
        }
    }

    static func enrolledCourse(
        courseCode: String,
        in courses: [CourseSummary]
    ) -> CourseSummary? {
        let code = courseCode.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return courses.first {
            $0.courseCode.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == code
        }
    }

    static func previewParagraphs(from description: String, limit: Int = 3) -> [String] {
        description
            .components(separatedBy: .newlines)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .prefix(limit)
            .map { String($0) }
    }

    static func ratingLabel(average: Double?, count: Int?) -> String {
        guard let average, (count ?? 0) > 0 else {
            return L.text("mobile.catalog.notRated")
        }
        return L.format("mobile.catalog.rating", average, count ?? 0)
    }
}