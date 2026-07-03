import Foundation

// MARK: - Course reviews submit (M9.3)

struct ReviewEligibility: Codable {
    var eligible: Bool
    var progressPercent: Int
    var hasReview: Bool
    var canEdit: Bool
    var reviewId: String?
}

struct SubmitCourseReviewRequest: Encodable {
    var rating: Int
    var reviewText: String?
}

struct SubmittedCourseReview: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var reviewerId: String
    var rating: Int
    var reviewText: String?
    var reviewerDisplayName: String
    var createdAt: String
    var updatedAt: String
}