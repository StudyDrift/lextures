package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

/** Secret-free subset of the platform settings response used by mobile admin. */
@Serializable
data class PlatformSettingsSnapshot(
    val ffFeedback: Boolean = false,
    val ffPublicCatalog: Boolean = false,
    val ffCourseMarketplace: Boolean = false,
    val ffLearningPaths: Boolean = false,
    val ffPeerReview: Boolean = false,
    val ffCompletionCredentials: Boolean = false,
    val ffPersistentTutor: Boolean = false,
    val ffAiStudyBuddy: Boolean = false,
    val ffClassroomSignals: Boolean = false,
    val ffBroadcasts: Boolean = false,
    val ffCalendarFeeds: Boolean = false,
    val learnerProfileEnabled: Boolean = false,
    val samlSsoEnabled: Boolean = false,
    val samlPublicBaseUrl: String = "",
    val samlSpEntityId: String = "",
    val mfaEnabled: Boolean = false,
    val mfaEnforcement: String = "none",
    val smtpHost: String = "",
    val smtpPort: Int = 587,
    val smtpFrom: String = "",
)

@Serializable
data class PlatformFeatureStates(
    val ffFeedback: Boolean = false,
    val ffPublicCatalog: Boolean = false,
    val ffCourseMarketplace: Boolean = false,
    val ffLearningPaths: Boolean = false,
    val ffPeerReview: Boolean = false,
    val ffCompletionCredentials: Boolean = false,
    val ffPersistentTutor: Boolean = false,
    val ffAiStudyBuddy: Boolean = false,
    val ffClassroomSignals: Boolean = false,
    val ffBroadcasts: Boolean = false,
    val ffCalendarFeeds: Boolean = false,
    val learnerProfileEnabled: Boolean = false,
)

data class PlatformFeatureDefinition(
    val key: String,
    val labelResName: String,
    val descriptionResName: String,
)
