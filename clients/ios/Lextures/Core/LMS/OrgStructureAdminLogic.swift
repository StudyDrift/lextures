import Foundation

/// Organizations, org units, and academic terms admin helpers (M14.4).
enum OrgStructureAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let orgUnitsAdminPermission = "tenant:org:units:admin"
    static let defaultTermType = "semester"
    static let orgListLimit = 200

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings || features.ffMobileAdminConsole
    }

    static func canManageOrganizations(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func canManageOrgUnitsAndTerms(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission) || permissions.contains(orgUnitsAdminPermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        !features.ffMobileAdminConsole
            && features.ffMobileAdminSettings
            && (canManageOrganizations(permissions: permissions)
                || canManageOrgUnitsAndTerms(permissions: permissions))
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features)
            && (canManageOrganizations(permissions: permissions)
                || canManageOrgUnitsAndTerms(permissions: permissions))
    }

    static func webOrganizationsPath() -> String { "/settings/organizations" }
    static func webOrgUnitsPath() -> String { "/settings/org-units" }
    static func webTermsPath() -> String { "/settings/terms" }

    static func resolveOrgId(accessToken: String?, courses: [CourseSummary]) -> String? {
        CourseCreateLogic.resolveOrgId(accessToken: accessToken, courses: courses)
    }

    static func normalizedName(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func isValidTermName(_ value: String) -> Bool {
        !normalizedName(value).isEmpty
    }

    static func isoDateString(from date: Date) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
    }

    static func date(fromIso value: String?) -> Date? {
        guard let value, !value.isEmpty else { return nil }
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.date(from: value)
    }

    static func formatDateRange(start: String?, end: String?) -> String {
        let startText = start ?? "—"
        let endText = end ?? "—"
        return "\(startText) — \(endText)"
    }

    static func isValidDateRange(start: String, end: String) -> Bool {
        guard let startDate = date(fromIso: start), let endDate = date(fromIso: end) else {
            return false
        }
        return endDate >= startDate
    }

    static func createTermRequest(
        name: String,
        termType: String,
        startDate: String,
        endDate: String
    ) -> CreateAcademicTermRequest {
        CreateAcademicTermRequest(
            name: normalizedName(name),
            termType: termType.isEmpty ? defaultTermType : termType,
            startDate: startDate,
            endDate: endDate
        )
    }

    static func patchTermDatesRequest(startDate: String, endDate: String) -> PatchAcademicTermRequest {
        PatchAcademicTermRequest(
            name: nil,
            termType: nil,
            startDate: startDate,
            endDate: endDate,
            status: nil
        )
    }

    static func patchOrgUnitNameRequest(name: String) -> PatchOrgUnitRequest {
        PatchOrgUnitRequest(name: normalizedName(name))
    }

    static func flattenTree(_ nodes: [OrgUnitTreeNode]) -> [OrgUnitTreeNode] {
        nodes.flatMap { node in
            [node] + flattenTree(node.children ?? [])
        }
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty {
            return message
        }
        return L.text("mobile.admin.orgStructure.error")
    }
}
