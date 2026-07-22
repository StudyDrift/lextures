package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError

object PlatformCoursesAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val DEFAULT_PER_PAGE = 25

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings || features.ffMobileAdminConsole

    fun canManageCourses(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun shouldShowEntry(features: MobilePlatformFeatures, permissions: List<String>): Boolean =
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManageCourses(permissions)

    fun canView(features: MobilePlatformFeatures, permissions: List<String>): Boolean =
        adminSettingsEnabled(features) && canManageCourses(permissions)

    fun webSettingsPath(): String = "/settings/courses"

    fun courseWebPath(courseCode: String): String = "/courses/$courseCode"

    fun normalizedSearchQuery(query: String): String = query.trim()

    fun shouldSearch(query: String): Boolean = normalizedSearchQuery(query).isNotEmpty()

    fun statusLabel(status: String, active: String, draft: String, archived: String): String =
        when (status.lowercase()) {
            "active" -> active
            "draft" -> draft
            "archived" -> archived
            else -> status
        }

    fun userFacingError(error: Throwable, genericMessage: String): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() } ?: genericMessage

    data class MetricDefinition(
        val filter: CoursesListFilter,
        val titleResName: String,
        val hintResName: String?,
        val tableTitleResName: String,
        val tableDescriptionResName: String,
    )

    val METRIC_DEFINITIONS: List<MetricDefinition> = listOf(
        MetricDefinition(
            CoursesListFilter.Created7d,
            "mobile_admin_courses_metric_created7d",
            "mobile_admin_courses_metric_created7d_hint",
            "mobile_admin_courses_metric_created7d_tableTitle",
            "mobile_admin_courses_metric_created7d_tableDescription",
        ),
        MetricDefinition(
            CoursesListFilter.Active,
            "mobile_admin_courses_metric_active",
            "mobile_admin_courses_metric_active_hint",
            "mobile_admin_courses_metric_active_tableTitle",
            "mobile_admin_courses_metric_active_tableDescription",
        ),
        MetricDefinition(
            CoursesListFilter.Draft,
            "mobile_admin_courses_metric_draft",
            "mobile_admin_courses_metric_draft_hint",
            "mobile_admin_courses_metric_draft_tableTitle",
            "mobile_admin_courses_metric_draft_tableDescription",
        ),
        MetricDefinition(
            CoursesListFilter.Total,
            "mobile_admin_courses_metric_total",
            null,
            "mobile_admin_courses_metric_total_tableTitle",
            "mobile_admin_courses_metric_total_tableDescription",
        ),
        MetricDefinition(
            CoursesListFilter.Archived,
            "mobile_admin_courses_metric_archived",
            "mobile_admin_courses_metric_archived_hint",
            "mobile_admin_courses_metric_archived_tableTitle",
            "mobile_admin_courses_metric_archived_tableDescription",
        ),
    )

    fun value(filter: CoursesListFilter, stats: CoursesDashboardStats): Long = when (filter) {
        CoursesListFilter.Created7d -> stats.createdLast7Days
        CoursesListFilter.Active -> stats.activeCourses
        CoursesListFilter.Draft -> stats.draftCourses
        CoursesListFilter.Total -> stats.totalCourses
        CoursesListFilter.Archived -> stats.archivedCourses
    }

    fun toggleFilter(current: CoursesListFilter?, tapped: CoursesListFilter): CoursesListFilter? =
        if (current == tapped) null else tapped

    fun metric(filter: CoursesListFilter): MetricDefinition? =
        METRIC_DEFINITIONS.firstOrNull { it.filter == filter }

    fun formatCount(value: Long): String = "%,d".format(value)
}
