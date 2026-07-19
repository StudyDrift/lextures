package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

/** MOB.3 Settings/Admin menu registry — groups/items mirror web admin nav inventory. */
object SettingsMenuLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    enum class GroupId {
        Platform,
        SchoolOperations,
        StudentRecords,
        Integrations,
        Compliance,
        ;

        val titleResName: String
            get() = when (this) {
                Platform -> "mobile_settings_menu_group_platform"
                SchoolOperations -> "mobile_settings_menu_group_schoolOperations"
                StudentRecords -> "mobile_settings_menu_group_studentRecords"
                Integrations -> "mobile_settings_menu_group_integrations"
                Compliance -> "mobile_settings_menu_group_compliance"
            }
    }

    enum class ItemId {
        PlatformSettings,
        OrgStructure,
        OrgBranding,
        RolesPermissions,
        People,
        ArchivedCourses,
        AiAdmin,
        TranscriptsAdvising,
        Integrations,
        AuditLog,
        ;

        val group: GroupId
            get() = when (this) {
                PlatformSettings, OrgStructure, OrgBranding, RolesPermissions, People, ArchivedCourses, AiAdmin ->
                    GroupId.Platform
                TranscriptsAdvising -> GroupId.StudentRecords
                Integrations -> GroupId.Integrations
                AuditLog -> GroupId.Compliance
            }

        val titleResName: String
            get() = when (this) {
                PlatformSettings -> "mobile_admin_platform_title"
                OrgStructure -> "mobile_admin_orgStructure_title"
                OrgBranding -> "mobile_admin_orgBranding_title"
                RolesPermissions -> "mobile_admin_roles_title"
                People -> "mobile_admin_people_title"
                ArchivedCourses -> "mobile_admin_archivedCourses_title"
                AiAdmin -> "mobile_admin_ai_hub_title"
                TranscriptsAdvising -> "mobile_admin_transcriptsAdvising_hub_title"
                Integrations -> "mobile_admin_integrations_hub_title"
                AuditLog -> "mobile_admin_auditLog_title"
            }

        val subtitleResName: String
            get() = when (this) {
                PlatformSettings -> "mobile_admin_platform_entry_subtitle"
                OrgStructure -> "mobile_admin_orgStructure_entry_subtitle"
                OrgBranding -> "mobile_admin_orgBranding_entry_subtitle"
                RolesPermissions -> "mobile_admin_roles_entry_subtitle"
                People -> "mobile_admin_people_entry_subtitle"
                ArchivedCourses -> "mobile_admin_archivedCourses_entry_subtitle"
                AiAdmin -> "mobile_admin_ai_hub_entry_subtitle"
                TranscriptsAdvising -> "mobile_admin_transcriptsAdvising_hub_entry_subtitle"
                Integrations -> "mobile_admin_integrations_hub_entry_subtitle"
                AuditLog -> "mobile_admin_auditLog_entry_subtitle"
            }
    }

    data class MenuItem(
        val id: ItemId,
        val group: GroupId,
        val titleResName: String,
        val subtitleResName: String,
    )

    data class MenuGroup(
        val id: GroupId,
        val titleResName: String,
        val items: List<MenuItem>,
    )

    fun adminConsoleEnabled(features: MobilePlatformFeatures): Boolean = features.ffMobileAdminConsole

    fun canManageRbac(permissions: Collection<String>): Boolean =
        RBAC_MANAGE_PERMISSION in permissions

    fun shouldShowHubEntry(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean {
        if (!adminConsoleEnabled(features)) return false
        return canManageRbac(permissions) ||
            AuditLogAdminLogic.shouldShowInMenu(features, permissions)
    }

    fun isItemVisible(
        item: ItemId,
        features: MobilePlatformFeatures,
        permissions: Collection<String>,
    ): Boolean {
        if (!adminConsoleEnabled(features)) return false
        return when (item) {
            ItemId.PlatformSettings -> PlatformSettingsAdminLogic.canView(features, permissions.toList())
            ItemId.OrgStructure -> OrgStructureAdminLogic.canView(features, permissions.toList())
            ItemId.OrgBranding -> OrgBrandingAdminLogic.canView(features, permissions.toList())
            ItemId.RolesPermissions -> RolesPermissionsAdminLogic.canView(features, permissions.toList())
            ItemId.People -> PeopleAdminLogic.canView(features, permissions.toList())
            ItemId.ArchivedCourses -> ArchivedCoursesAdminLogic.canView(features, permissions.toList())
            ItemId.AiAdmin -> AiModelsAdminLogic.canView(features, permissions)
            ItemId.TranscriptsAdvising ->
                TranscriptsAdvisingAdminLogic.canViewTranscripts(features, permissions) ||
                    TranscriptsAdvisingAdminLogic.canViewAdvising(features, permissions)
            ItemId.Integrations -> IntegrationsAdminLogic.canView(features, permissions.toList())
            ItemId.AuditLog -> AuditLogAdminLogic.shouldShowInMenu(features, permissions)
        }
    }

    fun visibleGroups(
        features: MobilePlatformFeatures,
        permissions: Collection<String>,
        query: String = "",
        titleResolver: (String) -> String = { it },
    ): List<MenuGroup> {
        val needle = query.trim().lowercase()
        return GroupId.entries.mapNotNull { group ->
            val items = ItemId.entries
                .filter { it.group == group }
                .filter { isItemVisible(it, features, permissions) }
                .filter { item ->
                    if (needle.isEmpty()) return@filter true
                    item.name.lowercase().contains(needle) ||
                        titleResolver(item.titleResName).lowercase().contains(needle) ||
                        titleResolver(item.subtitleResName).lowercase().contains(needle)
                }
                .map {
                    MenuItem(
                        id = it,
                        group = it.group,
                        titleResName = it.titleResName,
                        subtitleResName = it.subtitleResName,
                    )
                }
            if (items.isEmpty()) null
            else MenuGroup(id = group, titleResName = group.titleResName, items = items)
        }
    }

    /** Snapshot of group → item ids for parity tests (Phase 1 surface). */
    val phase1Registry: List<Pair<GroupId, List<ItemId>>>
        get() = GroupId.entries.map { group ->
            group to ItemId.entries.filter { it.group == group }
        }
}
