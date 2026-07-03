import Foundation

// MARK: - Public course catalog (M9.1)

struct PublicCatalogCourse: Codable, Identifiable, Hashable {
    var id: String
    var slug: String
    var courseCode: String
    var title: String
    var description: String
    var heroImageUrl: String?
    var category: String?
    var difficultyLevel: String?
    var language: String
    var priceCents: Int
    var enrollmentCount: Int
    var averageRating: Double?
    var ratingCount: Int?
    var instructorName: String?
    var createdAt: String
}

struct PublicCatalogSearchResponse: Codable {
    var courses: [PublicCatalogCourse]?
    var total: Int?
    var nextCursor: String?
}

struct CatalogCategory: Codable, Identifiable, Hashable {
    var category: String
    var count: Int

    var id: String { category }
}

struct CatalogCategoriesResponse: Codable {
    var categories: [CatalogCategory]?
}

struct PublicCatalogCourseDetailResponse: Codable {
    var course: PublicCatalogCourse
}

struct CourseReviewSummary: Codable, Hashable {
    var averageRating: Double?
    var ratingCount: Int
}

struct CourseReview: Codable, Identifiable, Hashable {
    var id: String
    var rating: Int
    var reviewText: String?
    var reviewerDisplayName: String
    var createdAt: String
}

struct CourseReviewsListResponse: Codable {
    var summary: CourseReviewSummary
    var reviews: [CourseReview]?
    var nextCursor: String?
}

struct CourseSelfEnrollResponse: Codable {
    var enrolled: Bool
    var enrollmentId: String
    var firstItemId: String?
}