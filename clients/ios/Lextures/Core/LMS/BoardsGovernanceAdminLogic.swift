import Foundation

/// MOB.8 admin boards governance — menu gating + policy helpers.
enum BoardsGovernanceAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    static func adminConsoleEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminConsole
    }

    static func canManage(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func canView(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        adminConsoleEnabled(features)
            && features.ffMobileBoardsAdvanced
            && canManage(permissions: permissions)
    }

    static func shouldShowInMenu(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        canView(features: features, permissions: permissions)
    }

    static func webPath() -> String { "/admin/boards" }
}
