import Foundation

/// Staff course health, at-risk, and engagement helpers (M11.3).
enum InstructorInsightsLogic {
    enum AtRiskSeverity: String, Hashable {
        case high
        case moderate

        var labelKey: String {
            switch self {
            case .high: return "mobile.instructorInsights.severity.high"
            case .moderate: return "mobile.instructorInsights.severity.moderate"
            }
        }
    }

    static func enabled(_ features: MobilePlatformFeatures) -> Bool {
        guard features.ffMobileInstructorInsights else { return false }
        return features.atRiskAlertsEnabled
            || features.instructorInsightsEnabled
            || features.studentProgressEnabled
    }

    static func shouldShowWorkspaceSection(
        course: CourseSummary,
        features: MobilePlatformFeatures
    ) -> Bool {
        course.viewerIsStaff && enabled(features)
    }

    static func severity(score: Float) -> AtRiskSeverity {
        score >= 80 ? .high : .moderate
    }

    static func sortAlerts(_ alerts: [AtRiskAlert]) -> [AtRiskAlert] {
        alerts.sorted { lhs, rhs in
            if lhs.score != rhs.score { return lhs.score > rhs.score }
            return lhs.displayName.localizedCaseInsensitiveCompare(rhs.displayName) == .orderedAscending
        }
    }

    static func snapshot(
        atRiskCount: Int,
        ungradedCount: Int,
        workingWell: [InstructorSignalItem],
        needsAttention: [InstructorSignalItem]
    ) -> CourseHealthSnapshot {
        CourseHealthSnapshot(
            atRiskCount: atRiskCount,
            ungradedCount: ungradedCount,
            engagementHighlightCount: workingWell.count + needsAttention.count
        )
    }

    static func messageSubject(displayName: String) -> String {
        L.format("mobile.instructorInsights.message.subject", displayName)
    }

    static func messageBody(displayName: String, reason: String) -> String {
        L.format("mobile.instructorInsights.message.body", displayName, reason)
    }

    static func webReportsPath(courseCode: String) -> String {
        "/courses/\(courseCode)/at-risk"
    }

    static func webWhatsWorkingPath(courseCode: String) -> String {
        "/courses/\(courseCode)/whats-working"
    }

    static func completionPercentText(_ rate: Double) -> String {
        "\(Int((rate * 100).rounded()))%"
    }

    static func optionalPercentText(_ value: Double?) -> String? {
        guard let value else { return nil }
        return String(format: "%.1f%%", value)
    }
}

struct CourseHealthSnapshot: Equatable {
    var atRiskCount: Int
    var ungradedCount: Int
    var engagementHighlightCount: Int
}
