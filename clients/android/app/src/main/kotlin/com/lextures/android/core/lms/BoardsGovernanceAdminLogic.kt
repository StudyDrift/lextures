package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

/** MOB.8 admin boards governance — menu gating + policy helpers. */
object BoardsGovernanceAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    fun adminConsoleEnabled(features: MobilePlatformFeatures): Boolean = features.ffMobileAdminConsole

    fun canManage(permissions: Collection<String>): Boolean =
        RBAC_MANAGE_PERMISSION in permissions

    fun canView(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        adminConsoleEnabled(features) &&
            features.ffMobileBoardsAdvanced &&
            canManage(permissions)

    fun shouldShowInMenu(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        canView(features, permissions)

    fun webPath(): String = "/admin/boards"
}
