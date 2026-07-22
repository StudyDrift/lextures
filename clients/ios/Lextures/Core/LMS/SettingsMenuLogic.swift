import Foundation

/// MOB.3 Settings/Admin menu registry — groups/items mirror web admin nav inventory.
enum SettingsMenuLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    enum GroupId: String, CaseIterable, Identifiable {
        case platform
        case schoolOperations
        case studentRecords
        case integrations
        case compliance

        var id: String { rawValue }

        var titleKey: String.LocalizationValue {
            switch self {
            case .platform: "mobile.settings.menu.group.platform"
            case .schoolOperations: "mobile.settings.menu.group.schoolOperations"
            case .studentRecords: "mobile.settings.menu.group.studentRecords"
            case .integrations: "mobile.settings.menu.group.integrations"
            case .compliance: "mobile.settings.menu.group.compliance"
            }
        }

        var systemImage: String {
            switch self {
            case .platform: "gearshape.2"
            case .schoolOperations: "building.columns"
            case .studentRecords: "person.text.rectangle"
            case .integrations: "puzzlepiece.extension"
            case .compliance: "checkmark.shield"
            }
        }
    }

    enum ItemId: String, CaseIterable, Identifiable, Hashable {
        case platformSettings
        case orgStructure
        case orgBranding
        case rolesPermissions
        case people
        case courses
        case archivedCourses
        case aiAdmin
        case transcriptsAdvising
        case integrations
        case auditLog
        case boardsGovernance

        var id: String { rawValue }

        var group: GroupId {
            switch self {
            case .platformSettings, .orgStructure, .orgBranding, .rolesPermissions, .people, .courses, .archivedCourses, .aiAdmin, .boardsGovernance:
                .platform
            case .transcriptsAdvising:
                .studentRecords
            case .integrations:
                .integrations
            case .auditLog:
                .compliance
            }
        }

        var titleKey: String.LocalizationValue {
            switch self {
            case .platformSettings: "mobile.admin.platform.title"
            case .orgStructure: "mobile.admin.orgStructure.title"
            case .orgBranding: "mobile.admin.orgBranding.title"
            case .rolesPermissions: "mobile.admin.roles.title"
            case .people: "mobile.admin.people.title"
            case .courses: "mobile.admin.courses.title"
            case .archivedCourses: "mobile.admin.archivedCourses.title"
            case .aiAdmin: "mobile.admin.ai.hub.title"
            case .transcriptsAdvising: "mobile.admin.transcriptsAdvising.hub.title"
            case .integrations: "mobile.admin.integrations.hub.title"
            case .auditLog: "mobile.admin.auditLog.title"
            case .boardsGovernance: "mobile.boards.admin.title"
            }
        }

        var subtitleKey: String.LocalizationValue {
            switch self {
            case .platformSettings: "mobile.admin.platform.entry.subtitle"
            case .orgStructure: "mobile.admin.orgStructure.entry.subtitle"
            case .orgBranding: "mobile.admin.orgBranding.entry.subtitle"
            case .rolesPermissions: "mobile.admin.roles.entry.subtitle"
            case .people: "mobile.admin.people.entry.subtitle"
            case .courses: "mobile.admin.courses.entry.subtitle"
            case .archivedCourses: "mobile.admin.archivedCourses.entry.subtitle"
            case .aiAdmin: "mobile.admin.ai.hub.entry.subtitle"
            case .transcriptsAdvising: "mobile.admin.transcriptsAdvising.hub.entry.subtitle"
            case .integrations: "mobile.admin.integrations.hub.entry.subtitle"
            case .auditLog: "mobile.admin.auditLog.entry.subtitle"
            case .boardsGovernance: "mobile.boards.admin.entry.subtitle"
            }
        }

        var systemImage: String {
            switch self {
            case .platformSettings: "switch.2"
            case .orgStructure: "building.2"
            case .orgBranding: "paintpalette"
            case .rolesPermissions: "person.badge.key"
            case .people: "person.3"
            case .courses: "books.vertical"
            case .archivedCourses: "archivebox"
            case .aiAdmin: "sparkles"
            case .transcriptsAdvising: "doc.text"
            case .integrations: "puzzlepiece.extension"
            case .auditLog: "scroll"
            case .boardsGovernance: "rectangle.3.group"
            }
        }
    }

    struct MenuItem: Identifiable, Equatable {
        var id: ItemId
        var group: GroupId
        var titleKey: String.LocalizationValue
        var subtitleKey: String.LocalizationValue
        var systemImage: String
    }

    struct MenuGroup: Identifiable, Equatable {
        var id: GroupId
        var titleKey: String.LocalizationValue
        var items: [MenuItem]
    }

    static func adminConsoleEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminConsole
    }

    static func canManageRbac(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    /// Hub entry on Profile — gated by MOB.3 flag plus RBAC or audit-capable console access.
    static func shouldShowHubEntry(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        guard adminConsoleEnabled(features) else { return false }
        return canManageRbac(permissions: permissions)
            || AuditLogAdminLogic.shouldShowInMenu(features: features, permissions: permissions)
    }

    static func isItemVisible(
        _ item: ItemId,
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        guard adminConsoleEnabled(features) else { return false }
        switch item {
        case .platformSettings:
            return PlatformSettingsAdminLogic.canView(features: features, permissions: permissions)
        case .orgStructure:
            return OrgStructureAdminLogic.canView(features: features, permissions: permissions)
        case .orgBranding:
            return OrgBrandingAdminLogic.canView(features: features, permissions: permissions)
        case .rolesPermissions:
            return RolesPermissionsAdminLogic.canView(features: features, permissions: permissions)
        case .people:
            return PeopleAdminLogic.canView(features: features, permissions: permissions)
        case .courses:
            return PlatformCoursesAdminLogic.canView(features: features, permissions: permissions)
        case .archivedCourses:
            return ArchivedCoursesAdminLogic.canView(features: features, permissions: permissions)
        case .aiAdmin:
            return AiModelsAdminLogic.canView(features: features, permissions: permissions)
        case .transcriptsAdvising:
            return TranscriptsAdvisingAdminLogic.canViewTranscripts(features: features, permissions: permissions)
                || TranscriptsAdvisingAdminLogic.canViewAdvising(features: features, permissions: permissions)
        case .integrations:
            return IntegrationsAdminLogic.canView(features: features, permissions: permissions)
        case .auditLog:
            return AuditLogAdminLogic.shouldShowInMenu(features: features, permissions: permissions)
        case .boardsGovernance:
            return BoardsGovernanceAdminLogic.shouldShowInMenu(features: features, permissions: permissions)
        }
    }

    static func visibleGroups(
        features: MobilePlatformFeatures,
        permissions: [String],
        query: String = ""
    ) -> [MenuGroup] {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return GroupId.allCases.compactMap { group in
            let items = ItemId.allCases
                .filter { $0.group == group }
                .filter { isItemVisible($0, features: features, permissions: permissions) }
                .filter { item in
                    guard !needle.isEmpty else { return true }
                    return item.rawValue.lowercased().contains(needle)
                        || L.text(item.titleKey).lowercased().contains(needle)
                        || L.text(item.subtitleKey).lowercased().contains(needle)
                }
                .map {
                    MenuItem(
                        id: $0,
                        group: $0.group,
                        titleKey: $0.titleKey,
                        subtitleKey: $0.subtitleKey,
                        systemImage: $0.systemImage
                    )
                }
            guard !items.isEmpty else { return nil }
            return MenuGroup(id: group, titleKey: group.titleKey, items: items)
        }
    }

    /// Snapshot of group → item ids for parity tests vs web inventory (Phase 1 surface).
    static var phase1Registry: [(group: GroupId, items: [ItemId])] {
        GroupId.allCases.map { group in
            (group, ItemId.allCases.filter { $0.group == group })
        }
    }
}
