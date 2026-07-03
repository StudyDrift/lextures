import Foundation

/// Course review composer validation and UI state (M9.3).
enum CourseReviewLogic {
    static let maxReviewTextLength = 2000

    static func reviewsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCourseReviews
    }

    static func starLabel(rating: Int) -> String {
        switch rating {
        case 1: return L.text("mobile.reviews.star1")
        case 2: return L.text("mobile.reviews.star2")
        case 3: return L.text("mobile.reviews.star3")
        case 4: return L.text("mobile.reviews.star4")
        case 5: return L.text("mobile.reviews.star5")
        default: return L.text("mobile.reviews.selectRating")
        }
    }

    static func shouldShowComposer(_ eligibility: ReviewEligibility) -> Bool {
        eligibility.eligible && (eligibility.canEdit || !eligibility.hasReview)
    }

    static func composerTitle(hasReview: Bool, canEdit: Bool) -> String {
        if hasReview && canEdit {
            return L.text("mobile.reviews.editTitle")
        }
        return L.text("mobile.reviews.writeTitle")
    }

    static func validateRating(_ rating: Int) -> String? {
        guard (1 ... 5).contains(rating) else {
            return L.text("mobile.reviews.ratingRequired")
        }
        return nil
    }

    static func validateReviewText(_ text: String) -> String? {
        guard text.count <= maxReviewTextLength else {
            return L.format("mobile.reviews.textTooLong", maxReviewTextLength)
        }
        return nil
    }

    static func progressHint(_ percent: Int) -> String {
        L.format("mobile.reviews.progressHint", percent)
    }
}