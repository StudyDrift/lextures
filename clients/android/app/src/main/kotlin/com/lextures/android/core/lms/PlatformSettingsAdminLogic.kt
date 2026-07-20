package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

object PlatformSettingsAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    /** Incident-safe allowlist; identity, infrastructure, billing, and self-lockout flags stay web-only. */
    val FEATURE_DEFINITIONS = listOf(
        definition("ffPublicCatalog", "publicCatalog"),
        definition("ffCourseMarketplace", "courseMarketplace"),
        definition("ffLearningPaths", "learningPaths"),
        definition("ffPeerReview", "peerReview"),
        definition("ffCompletionCredentials", "completionCredentials"),
        definition("ffPersistentTutor", "persistentTutor"),
        definition("ffAiStudyBuddy", "aiStudyBuddy"),
        definition("ffClassroomSignals", "classroomSignals"),
        definition("ffBroadcasts", "broadcasts"),
        definition("ffCalendarFeeds", "calendarFeeds"),
        definition("learnerProfileEnabled", "learnerProfile"),
    )

    private fun definition(key: String, name: String) = PlatformFeatureDefinition(
        key = key,
        labelResName = "mobile_admin_platform_feature_${name}_label",
        descriptionResName = "mobile_admin_platform_feature_${name}_description",
    )

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings || features.ffMobileAdminConsole

    fun canManage(permissions: List<String>): Boolean = RBAC_MANAGE_PERMISSION in permissions

    fun shouldShowEntry(features: MobilePlatformFeatures, permissions: List<String>): Boolean =
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManage(permissions)

    fun canView(features: MobilePlatformFeatures, permissions: List<String>): Boolean =
        adminSettingsEnabled(features) && canManage(permissions)

    fun value(key: String, settings: PlatformSettingsSnapshot): Boolean = when (key) {
        "ffPublicCatalog" -> settings.ffPublicCatalog
        "ffCourseMarketplace" -> settings.ffCourseMarketplace
        "ffLearningPaths" -> settings.ffLearningPaths
        "ffPeerReview" -> settings.ffPeerReview
        "ffCompletionCredentials" -> settings.ffCompletionCredentials
        "ffPersistentTutor" -> settings.ffPersistentTutor
        "ffAiStudyBuddy" -> settings.ffAiStudyBuddy
        "ffClassroomSignals" -> settings.ffClassroomSignals
        "ffBroadcasts" -> settings.ffBroadcasts
        "ffCalendarFeeds" -> settings.ffCalendarFeeds
        "learnerProfileEnabled" -> settings.learnerProfileEnabled
        else -> false
    }

    fun applyingEffectiveFeatures(
        features: PlatformFeatureStates,
        settings: PlatformSettingsSnapshot,
    ): PlatformSettingsSnapshot = settings.copy(
        ffPublicCatalog = features.ffPublicCatalog,
        ffCourseMarketplace = features.ffCourseMarketplace,
        ffLearningPaths = features.ffLearningPaths,
        ffPeerReview = features.ffPeerReview,
        ffCompletionCredentials = features.ffCompletionCredentials,
        ffPersistentTutor = features.ffPersistentTutor,
        ffAiStudyBuddy = features.ffAiStudyBuddy,
        ffClassroomSignals = features.ffClassroomSignals,
        ffBroadcasts = features.ffBroadcasts,
        ffCalendarFeeds = features.ffCalendarFeeds,
        learnerProfileEnabled = features.learnerProfileEnabled,
    )

    fun webSettingsPath(): String = "/settings/platform"
}
