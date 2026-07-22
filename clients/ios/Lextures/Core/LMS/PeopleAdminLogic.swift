import Foundation

/// People / user management admin helpers (M14.3).
enum PeopleAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let erasedEmailSuffix = "@erased.invalid"
    static let defaultPerPage = 25

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings || features.ffMobileAdminConsole
    }

    static func canManagePeople(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManagePeople(permissions: permissions)
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features) && canManagePeople(permissions: permissions)
    }

    static func webSettingsPath() -> String {
        "/settings/people"
    }

    static func personDisplayName(_ row: PersonRow) -> String {
        personDisplayName(
            displayName: row.displayName,
            firstName: row.firstName,
            lastName: row.lastName,
            email: row.email
        )
    }

    static func personDisplayName(_ report: PersonReport) -> String {
        personDisplayName(
            displayName: report.displayName,
            firstName: report.firstName,
            lastName: report.lastName,
            email: report.email
        )
    }

    static func personDisplayName(
        displayName: String?,
        firstName: String?,
        lastName: String?,
        email: String
    ) -> String {
        let dn = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !dn.isEmpty { return dn }
        let full = [firstName, lastName]
            .compactMap { $0?.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .joined(separator: " ")
        if !full.isEmpty { return full }
        return email
    }

    static func statusLabel(active: Bool) -> String {
        active
            ? L.text("mobile.admin.people.status.active")
            : L.text("mobile.admin.people.status.suspended")
    }

    static func isErased(email: String) -> Bool {
        email.lowercased().hasSuffix(erasedEmailSuffix)
    }

    static func blocksSelfSuspend(targetUserId: String, currentUserId: String?) -> Bool {
        guard let currentUserId, !currentUserId.isEmpty else { return false }
        return targetUserId == currentUserId
    }

    static func normalizedSearchQuery(_ query: String) -> String {
        query.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func shouldSearch(_ query: String) -> Bool {
        !normalizedSearchQuery(query).isEmpty
    }

    static func invitePersonRequest(
        email: String,
        firstName: String?,
        lastName: String?
    ) -> InvitePersonRequest {
        let trimmedEmail = email.trimmingCharacters(in: .whitespacesAndNewlines)
        let first = firstName?.trimmingCharacters(in: .whitespacesAndNewlines)
        let last = lastName?.trimmingCharacters(in: .whitespacesAndNewlines)
        return InvitePersonRequest(
            email: trimmedEmail,
            firstName: first?.isEmpty == false ? first : nil,
            lastName: last?.isEmpty == false ? last : nil
        )
    }

    static func patchPersonRequest(active: Bool) -> PatchPersonRequest {
        PatchPersonRequest(active: active)
    }

    static func resendInviteRequest(email: String) -> ForgotPasswordRequest {
        ForgotPasswordRequest(email: email.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    static func roleMatchesReport(_ role: RoleWithPermissions, report: PersonReport) -> Bool {
        let reportRole = report.role.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !reportRole.isEmpty else { return false }
        return role.name.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == reportRole
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty {
            return message
        }
        return L.text("mobile.admin.people.error")
    }

    struct MetricDefinition: Identifiable, Equatable {
        var id: PeopleListFilter { filter }
        let filter: PeopleListFilter
        let titleKey: String.LocalizationValue
        let hintKey: String.LocalizationValue?
        let tableTitleKey: String.LocalizationValue
        let tableDescriptionKey: String.LocalizationValue
        let systemImage: String
    }

    static let metricDefinitions: [MetricDefinition] = [
        MetricDefinition(
            filter: .signups7d,
            titleKey: "mobile.admin.people.metric.signups7d",
            hintKey: "mobile.admin.people.metric.signups7d.hint",
            tableTitleKey: "mobile.admin.people.metric.signups7d.tableTitle",
            tableDescriptionKey: "mobile.admin.people.metric.signups7d.tableDescription",
            systemImage: "person.badge.plus"
        ),
        MetricDefinition(
            filter: .active,
            titleKey: "mobile.admin.people.metric.active",
            hintKey: "mobile.admin.people.metric.active.hint",
            tableTitleKey: "mobile.admin.people.metric.active.tableTitle",
            tableDescriptionKey: "mobile.admin.people.metric.active.tableDescription",
            systemImage: "person.crop.circle.badge.checkmark"
        ),
        MetricDefinition(
            filter: .recent30d,
            titleKey: "mobile.admin.people.metric.recent30d",
            hintKey: "mobile.admin.people.metric.recent30d.hint",
            tableTitleKey: "mobile.admin.people.metric.recent30d.tableTitle",
            tableDescriptionKey: "mobile.admin.people.metric.recent30d.tableDescription",
            systemImage: "chart.line.uptrend.xyaxis"
        ),
        MetricDefinition(
            filter: .total,
            titleKey: "mobile.admin.people.metric.total",
            hintKey: nil,
            tableTitleKey: "mobile.admin.people.metric.total.tableTitle",
            tableDescriptionKey: "mobile.admin.people.metric.total.tableDescription",
            systemImage: "person.3"
        ),
        MetricDefinition(
            filter: .suspended,
            titleKey: "mobile.admin.people.metric.suspended",
            hintKey: "mobile.admin.people.metric.suspended.hint",
            tableTitleKey: "mobile.admin.people.metric.suspended.tableTitle",
            tableDescriptionKey: "mobile.admin.people.metric.suspended.tableDescription",
            systemImage: "person.crop.circle.badge.xmark"
        ),
    ]

    static func value(for filter: PeopleListFilter, in stats: PeopleDashboardStats) -> Int64 {
        switch filter {
        case .signups7d: stats.signupsLast7Days
        case .active: stats.activeAccounts
        case .recent30d: stats.recentlyActive30Days
        case .total: stats.totalAccounts
        case .suspended: stats.suspendedAccounts
        }
    }

    static func toggleFilter(current: PeopleListFilter?, tapped: PeopleListFilter) -> PeopleListFilter? {
        current == tapped ? nil : tapped
    }

    static func metric(for filter: PeopleListFilter) -> MetricDefinition? {
        metricDefinitions.first { $0.filter == filter }
    }

    static func formatCount(_ value: Int64) -> String {
        NumberFormatter.localizedString(from: NSNumber(value: value), number: .decimal)
    }
}
