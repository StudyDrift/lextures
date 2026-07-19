package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

/** Admin console audit log helpers (MOB.3 Phase 1) — read-only parity with web AdminAuditLog. */
object AuditLogAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    fun shouldShowInMenu(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        features.ffMobileAdminConsole &&
            features.adminConsoleEnabled &&
            features.adminAuditLogEnabled &&
            RBAC_MANAGE_PERMISSION in permissions

    fun canView(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        shouldShowInMenu(features, permissions)

    fun normalizedActionFilter(raw: String): String? {
        val trimmed = raw.trim()
        return trimmed.ifEmpty { null }
    }

    fun targetLabel(type: String?, id: String?): String {
        val t = type?.trim().orEmpty()
        val i = id?.trim().orEmpty()
        if (t.isEmpty() && i.isEmpty()) return "—"
        if (i.isEmpty()) return t
        if (t.isEmpty()) return i
        return "$t / $i"
    }

    fun webPath(): String = "/org-admin/audit-log"
}
