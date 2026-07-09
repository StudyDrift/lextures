import Foundation

/// Global archived-course admin helpers (M14.10).
enum ArchivedCoursesAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings
    }

    static func canManageArchivedCourses(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features) && canManageArchivedCourses(permissions: permissions)
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        shouldShowEntry(features: features, permissions: permissions)
    }

    static func filterRows(_ rows: [ArchivedCourseRow], query: String) -> [ArchivedCourseRow] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return rows }
        let needle = trimmed.lowercased()
        return rows.filter { row in
            row.title.lowercased().contains(needle)
                || row.courseCode.lowercased().contains(needle)
                || archivedByLabel(row).lowercased().contains(needle)
        }
    }

    static func archivedByLabel(_ row: ArchivedCourseRow) -> String {
        let name = row.archivedByName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        let email = row.archivedByEmail?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !email.isEmpty { return email }
        return L.text("mobile.emDash")
    }

    static func formatArchivedAt(_ raw: String?) -> String {
        let formatted = DateFormatting.formatDateTime(raw)
        return formatted.isEmpty ? L.text("mobile.emDash") : formatted
    }

    static func deleteConfirmPhrase(for row: ArchivedCourseRow) -> String {
        row.courseCode.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func deleteConfirmMatches(typed: String, row: ArchivedCourseRow) -> Bool {
        typed.trimmingCharacters(in: .whitespacesAndNewlines)
            .caseInsensitiveCompare(deleteConfirmPhrase(for: row)) == .orderedSame
    }

    static func rowsAfterRestore(_ rows: [ArchivedCourseRow], courseCode: String) -> [ArchivedCourseRow] {
        rows.filter { $0.courseCode != courseCode }
    }

    static func rowsAfterDelete(_ rows: [ArchivedCourseRow], courseCode: String) -> [ArchivedCourseRow] {
        rows.filter { $0.courseCode != courseCode }
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty {
            return message
        }
        return L.text("mobile.admin.archivedCourses.error")
    }
}