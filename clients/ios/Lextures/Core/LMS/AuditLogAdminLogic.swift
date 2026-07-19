import Foundation

/// Admin console audit log helpers (MOB.3 Phase 1) — read-only parity with web AdminAuditLog.
enum AuditLogAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    static func shouldShowInMenu(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        features.ffMobileAdminConsole
            && features.adminConsoleEnabled
            && features.adminAuditLogEnabled
            && permissions.contains(rbacManagePermission)
    }

    static func canView(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        shouldShowInMenu(features: features, permissions: permissions)
    }

    static func normalizedActionFilter(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    static func targetLabel(type: String?, id: String?) -> String {
        let t = (type ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let i = (id ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        if t.isEmpty && i.isEmpty { return "—" }
        if i.isEmpty { return t }
        if t.isEmpty { return i }
        return "\(t) / \(i)"
    }

    static func webPath() -> String { "/org-admin/audit-log" }
}
