import Foundation

enum PlatformCoursesAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let defaultPerPage = 25

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings || features.ffMobileAdminConsole
    }

    static func canManageCourses(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManageCourses(permissions: permissions)
    }

    static func canView(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        adminSettingsEnabled(features) && canManageCourses(permissions: permissions)
    }

    static func webSettingsPath() -> String { "/settings/courses" }
    static func courseWebPath(courseCode: String) -> String { "/courses/\(courseCode)" }

    static func normalizedSearchQuery(_ query: String) -> String {
        query.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func shouldSearch(_ query: String) -> Bool { !normalizedSearchQuery(query).isEmpty }

    static func statusLabel(_ status: String) -> String {
        switch status.lowercased() {
        case "active": L.text("mobile.admin.courses.status.active")
        case "draft": L.text("mobile.admin.courses.status.draft")
        case "archived": L.text("mobile.admin.courses.status.archived")
        default: status
        }
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty { return message }
        return L.text("mobile.admin.courses.error")
    }

    struct MetricDefinition: Identifiable, Equatable {
        var id: CoursesListFilter { filter }
        let filter: CoursesListFilter
        let titleKey: String.LocalizationValue
        let hintKey: String.LocalizationValue?
        let tableTitleKey: String.LocalizationValue
        let tableDescriptionKey: String.LocalizationValue
        let systemImage: String
    }

    static let metricDefinitions: [MetricDefinition] = [
        MetricDefinition(
            filter: .created7d,
            titleKey: "mobile.admin.courses.metric.created7d",
            hintKey: "mobile.admin.courses.metric.created7d.hint",
            tableTitleKey: "mobile.admin.courses.metric.created7d.tableTitle",
            tableDescriptionKey: "mobile.admin.courses.metric.created7d.tableDescription",
            systemImage: "book.closed.fill"
        ),
        MetricDefinition(
            filter: .active,
            titleKey: "mobile.admin.courses.metric.active",
            hintKey: "mobile.admin.courses.metric.active.hint",
            tableTitleKey: "mobile.admin.courses.metric.active.tableTitle",
            tableDescriptionKey: "mobile.admin.courses.metric.active.tableDescription",
            systemImage: "book.fill"
        ),
        MetricDefinition(
            filter: .draft,
            titleKey: "mobile.admin.courses.metric.draft",
            hintKey: "mobile.admin.courses.metric.draft.hint",
            tableTitleKey: "mobile.admin.courses.metric.draft.tableTitle",
            tableDescriptionKey: "mobile.admin.courses.metric.draft.tableDescription",
            systemImage: "doc.badge.ellipsis"
        ),
        MetricDefinition(
            filter: .total,
            titleKey: "mobile.admin.courses.metric.total",
            hintKey: nil,
            tableTitleKey: "mobile.admin.courses.metric.total.tableTitle",
            tableDescriptionKey: "mobile.admin.courses.metric.total.tableDescription",
            systemImage: "books.vertical.fill"
        ),
        MetricDefinition(
            filter: .archived,
            titleKey: "mobile.admin.courses.metric.archived",
            hintKey: "mobile.admin.courses.metric.archived.hint",
            tableTitleKey: "mobile.admin.courses.metric.archived.tableTitle",
            tableDescriptionKey: "mobile.admin.courses.metric.archived.tableDescription",
            systemImage: "archivebox.fill"
        ),
    ]

    static func value(for filter: CoursesListFilter, in stats: CoursesDashboardStats) -> Int64 {
        switch filter {
        case .created7d: stats.createdLast7Days
        case .active: stats.activeCourses
        case .draft: stats.draftCourses
        case .total: stats.totalCourses
        case .archived: stats.archivedCourses
        }
    }

    static func toggleFilter(current: CoursesListFilter?, tapped: CoursesListFilter) -> CoursesListFilter? {
        current == tapped ? nil : tapped
    }

    static func metric(for filter: CoursesListFilter) -> MetricDefinition? {
        metricDefinitions.first { $0.filter == filter }
    }
}
