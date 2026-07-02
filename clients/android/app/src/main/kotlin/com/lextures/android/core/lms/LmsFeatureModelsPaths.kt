package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class CatalogPathSummary(
    val id: String,
    val title: String,
    val description: String = "",
    val slug: String,
    val bundlePriceCents: Int? = null,
    val courseCount: Int = 0,
    val totalDurationMinutes: Int = 0,
    val individualTotalCents: Int = 0,
    val skillTags: List<String> = emptyList(),
)

@Serializable
data class CatalogPathsListResponse(
    val paths: List<CatalogPathSummary> = emptyList(),
)

@Serializable
data class PathCourseProgress(
    val courseId: String,
    val position: Int,
    val courseCode: String,
    val title: String,
    val description: String? = null,
    val listPriceCents: Int? = null,
    val durationMinutes: Int? = null,
    val skillTags: List<String>? = null,
    val completed: Boolean? = null,
    val recommended: Boolean? = null,
) {
    val isCompleted: Boolean get() = completed == true
    val isRecommended: Boolean get() = recommended == true
}

@Serializable
data class PathProgress(
    val pathId: String,
    val pathTitle: String,
    val slug: String? = null,
    val totalCourses: Int = 0,
    val completedCourses: Int = 0,
    val percent: Int = 0,
    val progressLabel: String = "",
    val completedAt: String? = null,
    val justCompleted: Boolean = false,
    val courses: List<PathCourseProgress> = emptyList(),
)

@Serializable
data class MyPathsListResponse(
    val paths: List<PathProgress> = emptyList(),
)

@Serializable
data class LearningPathDetailPath(
    val id: String,
    val title: String,
    val description: String = "",
    val slug: String? = null,
    val bundlePriceCents: Int? = null,
    val isPublic: Boolean = true,
)

@Serializable
data class LearningPathDetail(
    val path: LearningPathDetailPath,
    val courses: List<PathCourseProgress> = emptyList(),
    val totalDurationMinutes: Int = 0,
    val individualTotalCents: Int = 0,
    val skillTags: List<String> = emptyList(),
    val slug: String = "",
)

@Serializable
data class PathEnrollResponse(
    val enrollmentId: String,
    val progress: PathProgress? = null,
)

@Serializable
data class RecommendationEventBody(
    val courseId: String,
    val itemId: String? = null,
    val surface: String,
    val eventType: String,
    val rank: Int? = null,
)