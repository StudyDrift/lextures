import Foundation

// MARK: - Authenticated course marketplace (MKT6)

struct MarketplaceCourse: Codable, Identifiable, Hashable {
    var id: String
    var slug: String
    var courseCode: String
    var title: String
    var description: String
    var heroImageUrl: String?
    var category: String?
    var level: String?
    var language: String
    var priceCents: Int
    var priceCurrency: String
    var listPriceCents: Int?
    var enrollmentCount: Int
    var averageRating: Double?
    var ratingCount: Int
    var instructorName: String?
    var createdAt: String
    var owned: Bool
}

struct MarketplaceSearchResponse: Codable {
    var courses: [MarketplaceCourse]?
    var total: Int?
    var nextCursor: String?
}

struct MarketplaceCategory: Codable, Identifiable, Hashable {
    var category: String
    var count: Int

    var id: String { category }
}

struct MarketplaceCategoriesResponse: Codable {
    var categories: [MarketplaceCategory]?
}

struct MarketplaceWhatsIncluded: Codable, Hashable {
    var moduleCount: Int
    var itemCount: Int
    var estimatedDurationMinutes: Int?
}

struct MarketplaceRating: Codable, Hashable {
    var average: Double?
    var count: Int
}

struct MarketplaceCourseDetail: Codable {
    var course: MarketplaceCourse
    var owned: Bool
    var priceCents: Int
    var priceCurrency: String
    var listPriceCents: Int?
    var whatsIncluded: MarketplaceWhatsIncluded
    var rating: MarketplaceRating
}

struct MarketplaceClaimResult: Codable {
    var enrolled: Bool
    var entitlementId: String
    var alreadyOwned: Bool?
    var firstItemId: String?
    var courseCode: String
}

struct MarketplaceCheckoutResult: Codable {
    var sessionId: String?
    var checkoutUrl: String?
    var alreadyOwned: Bool?
    var courseCode: String?
    var courseId: String?
}

struct CoursePurchase: Codable, Identifiable, Hashable {
    var courseCode: String
    var courseId: String
    var title: String
    var priceCents: Int
    var currency: String
    var source: String
    var acquiredAt: String
    var receiptUrl: String?
    var entitlementId: String

    var id: String { entitlementId }
}

struct CoursePurchasesResponse: Codable {
    var purchases: [CoursePurchase]?
}

struct CourseCatalogListing: Codable, Hashable {
    var isPublic: Bool
    var category: String?
    var difficultyLevel: String?
    var language: String
    var priceCents: Int
    var priceCurrency: String
    var slug: String
    var marketplaceListed: Bool
    var publishState: String
    var activePurchaseCount: Int
}

struct CourseCatalogListingResponse: Codable {
    var listing: CourseCatalogListing
}

struct CourseCatalogListingPutBody: Codable {
    var isPublic: Bool
    var category: String?
    var difficultyLevel: String?
    var language: String
    var priceCents: Int
    var priceCurrency: String
    var slug: String
    var marketplaceListed: Bool?
}
