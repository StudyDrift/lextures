package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

/** Canonical intro course URL code (matches server introcourse.CourseCode). */
object IntroCourseConstants {
    const val courseCode = "C-WLCOME"
}

@Serializable
data class IntroCourseNextItem(
    val slug: String,
    val title: String,
    val route: String,
)

@Serializable
data class IntroCourseModuleProgress(
    val slug: String,
    val title: String,
    val status: String,
)

@Serializable
data class IntroCourseProgress(
    val enrolled: Boolean,
    val courseCode: String? = null,
    val modulesComplete: Int = 0,
    val modulesTotal: Int = 7,
    val percent: Int = 0,
    val runningGrade: Double? = null,
    val completedAt: String? = null,
    val credentialId: String? = null,
    val nextItem: IntroCourseNextItem? = null,
    val modules: List<IntroCourseModuleProgress>? = null,
    val welcomeBannerDismissed: Boolean? = null,
    val celebrationSeen: Boolean? = null,
)

enum class IntroCourseCardState {
    Hidden,
    Loading,
    Error,
    NotStarted,
    InProgress,
    Completed,
}