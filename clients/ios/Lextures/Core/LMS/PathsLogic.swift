import Foundation

enum PathsLogic {
    static let recommendationSurfaces = ["continue", "review", "strengthen", "challenge"]

    static func sortedCourses(_ courses: [PathCourseProgress]) -> [PathCourseProgress] {
        courses.sorted { $0.position < $1.position }
    }

    static func nextCourse(in progress: PathProgress) -> PathCourseProgress? {
        sortedCourses(progress.courses).first { !$0.isCompleted }
    }

    static func isLocked(_ course: PathCourseProgress) -> Bool {
        !course.isCompleted && !course.isRecommended
    }

    static func isPaid(bundlePriceCents: Int?) -> Bool {
        guard let bundlePriceCents else { return false }
        return bundlePriceCents > 0
    }

    static func catalogWebPath(slug: String) -> String {
        "/paths/\(slug)"
    }

    static func formatPrice(cents: Int, currency: String = "USD") -> String {
        let major = CurrencyExponent.minorUnitsToMajorUnits(cents, currency: currency)
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.currencyCode = currency
        return formatter.string(from: NSNumber(value: major)) ?? "\(cents)"
    }

    static func formatDuration(minutes: Int) -> String {
        if minutes < 60 { return L.format("mobile.paths.duration.minutes", minutes) }
        let hours = minutes / 60
        let rem = minutes % 60
        if rem == 0 { return L.format("mobile.paths.duration.hours", hours) }
        return L.format("mobile.paths.duration.hoursMinutes", hours, rem)
    }

    static func mergeRecommendations(
        _ responses: [LearnerRecommendationsResponse]
    ) -> (primary: LearnerRecommendationItem?, chips: [LearnerRecommendationItem], degraded: Bool) {
        var merged: [LearnerRecommendationItem] = []
        var degraded = false
        for response in responses {
            merged.append(contentsOf: response.recommendations)
            if response.degraded == true { degraded = true }
        }
        merged.sort { $0.score > $1.score }
        return (merged.first, Array(merged.dropFirst().prefix(3)), degraded)
    }

    static func structureItemKind(for itemType: String) -> String? {
        switch itemType {
        case "quiz": return "quiz"
        case "content_page": return "content_page"
        case "assignment": return "assignment"
        case "external_link": return "external_link"
        default: return nil
        }
    }

    static func surfaceLabelKey(_ surface: String) -> String {
        switch surface {
        case "continue": return "mobile.paths.surface.continue"
        case "strengthen": return "mobile.paths.surface.strengthen"
        case "challenge": return "mobile.paths.surface.challenge"
        case "review": return "mobile.paths.surface.review"
        default: return "mobile.paths.surface.other"
        }
    }

    static func structureItem(
        for recommendation: LearnerRecommendationItem,
        in items: [CourseStructureItem]
    ) -> CourseStructureItem? {
        if let existing = items.first(where: { $0.id == recommendation.itemId }) {
            return existing
        }
        guard let kind = structureItemKind(for: recommendation.itemType) else { return nil }
        return CourseStructureItem(
            id: recommendation.itemId,
            sortOrder: 0,
            kind: kind,
            title: recommendation.title,
            parentId: nil,
            published: true,
            dueAt: nil,
            pointsWorth: nil,
            pointsPossible: nil
        )
    }
}