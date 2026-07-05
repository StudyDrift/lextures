import Foundation

/// Course evaluation helpers (M7.7).
enum EvaluationLogic {
    static func evaluationsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCourseEvaluations && features.ffMobileCourseEvaluations
    }

    static func evaluationTodoKey(courseCode: String, windowId: String) -> String {
        "evaluation:\(courseCode):\(windowId)"
    }

    static func draftCacheKey(courseCode: String, windowId: String) -> String {
        "evaluation:draft:\(courseCode):\(windowId)"
    }

    static func submitIdempotencyKey(courseCode: String, windowId: String) -> String {
        "evaluation-submit:\(courseCode):\(windowId)"
    }

    static func statusCacheKey(courseCode: String) -> String {
        "evaluation:status:\(courseCode)"
    }

    static func resultsCacheKey(courseCode: String) -> String {
        "evaluation:results:\(courseCode)"
    }

    static func shouldShowWorkspaceSection(
        course: CourseSummary,
        status: EvaluationStatus?,
        features: MobilePlatformFeatures
    ) -> Bool {
        guard evaluationsEnabled(features) else { return false }
        if course.viewerIsStaff { return true }
        guard let status else { return false }
        return status.windowOpen || status.hasSubmitted
    }

    static func missingRequiredIndices(
        questions: [EvaluationQuestion],
        answers: [String: String]
    ) -> [Int] {
        questions.enumerated().compactMap { index, question in
            guard question.isRequired else { return nil }
            let value = answers[String(index)]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            return value.isEmpty ? index : nil
        }
    }

    static func isSubmitBlocked(status: EvaluationStatus?) -> Bool {
        guard let status else { return true }
        if status.hasSubmitted { return true }
        return !status.windowOpen
    }

    static func parseClosesAt(_ iso: String?) -> Date? {
        guard let iso else { return nil }
        return LMSDates.parse(iso)
    }

    static func formatDeadline(_ iso: String?) -> String {
        guard let date = parseClosesAt(iso) else { return "" }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    static func ratingLabels() -> [String: String] {
        [
            "1": L.text("mobile.evaluations.rating.1"),
            "2": L.text("mobile.evaluations.rating.2"),
            "3": L.text("mobile.evaluations.rating.3"),
            "4": L.text("mobile.evaluations.rating.4"),
            "5": L.text("mobile.evaluations.rating.5"),
        ]
    }

    static func loadDraft(courseCode: String, windowId: String) -> [String: String] {
        let key = draftCacheKey(courseCode: courseCode, windowId: windowId)
        guard let data = UserDefaults.standard.data(forKey: key),
              let decoded = try? JSONDecoder().decode([String: String].self, from: data) else {
            return [:]
        }
        return decoded
    }

    static func saveDraft(courseCode: String, windowId: String, answers: [String: String]) {
        let key = draftCacheKey(courseCode: courseCode, windowId: windowId)
        guard let data = try? JSONEncoder().encode(answers) else { return }
        UserDefaults.standard.set(data, forKey: key)
    }

    static func clearDraft(courseCode: String, windowId: String) {
        UserDefaults.standard.removeObject(forKey: draftCacheKey(courseCode: courseCode, windowId: windowId))
    }
}
