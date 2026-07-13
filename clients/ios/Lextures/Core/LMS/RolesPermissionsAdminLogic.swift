import Foundation

/// Roles & permissions admin helpers (M14.2).
enum RolesPermissionsAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings
    }

    static func canManageRoles(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features) && canManageRoles(permissions: permissions)
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        shouldShowEntry(features: features, permissions: permissions)
    }

    static func webSettingsPath() -> String {
        "/settings/roles"
    }

    static func filterRoles(_ roles: [RoleWithPermissions], query: String) -> [RoleWithPermissions] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return roles }
        let needle = trimmed.lowercased()
        return roles.filter { role in
            role.name.lowercased().contains(needle)
                || (role.description?.lowercased().contains(needle) ?? false)
                || role.permissions.contains { permission in
                    permission.permissionString.lowercased().contains(needle)
                        || permission.description.lowercased().contains(needle)
                }
        }
    }

    static func filterPermissions(_ permissions: [RBACPermission], query: String) -> [RBACPermission] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return permissions }
        let needle = trimmed.lowercased()
        return permissions.filter { permission in
            permission.permissionString.lowercased().contains(needle)
                || permission.description.lowercased().contains(needle)
        }
    }

    static func filterUsers(_ users: [RBACUserBrief], query: String) -> [RBACUserBrief] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return users }
        let needle = trimmed.lowercased()
        return users.filter { user in
            userDisplayLabel(user).lowercased().contains(needle)
                || user.email.lowercased().contains(needle)
                || (user.sid?.lowercased().contains(needle) ?? false)
        }
    }

    static func userDisplayLabel(_ user: RBACUserBrief) -> String {
        let name = user.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        return user.email
    }

    static func roleGrantsRbacManage(_ role: RoleWithPermissions) -> Bool {
        role.permissions.contains { $0.permissionString == rbacManagePermission }
    }

    static func blocksSelfElevation(
        role: RoleWithPermissions,
        targetUserId: String,
        currentUserId: String?
    ) -> Bool {
        guard let currentUserId, !currentUserId.isEmpty else { return false }
        return targetUserId == currentUserId && roleGrantsRbacManage(role)
    }

    static func addRoleUserRequest(userId: String) -> AddRoleUserRequest {
        AddRoleUserRequest(userId: userId)
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty {
            return message
        }
        return L.text("mobile.admin.roles.error")
    }
}
