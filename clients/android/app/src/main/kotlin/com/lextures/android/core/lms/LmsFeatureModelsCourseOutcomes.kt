package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class CourseOutcomeLinkProgress(
    val avgScorePercent: Double? = null,
    val gradedLearners: Int = 0,
    val enrolledLearners: Int = 0,
)

@Serializable
data class CourseOutcomeLink(
    val id: String,
    val subOutcomeId: String? = null,
    val structureItemId: String,
    val targetKind: String,
    val quizQuestionId: String = "",
    val measurementLevel: String,
    val intensityLevel: String,
    val itemTitle: String,
    val itemKind: String,
    val progress: CourseOutcomeLinkProgress,
)

@Serializable
data class CourseOutcome(
    val id: String,
    val title: String,
    val description: String = "",
    val sortOrder: Int = 0,
    val rollupAvgScorePercent: Double? = null,
    val links: List<CourseOutcomeLink> = emptyList(),
)

@Serializable
data class CourseOutcomesListResponse(
    val enrolledLearners: Int = 0,
    val outcomes: List<CourseOutcome> = emptyList(),
)

@Serializable
data class CreateCourseOutcomeBody(
    val title: String,
    val description: String = "",
)

@Serializable
data class PatchCourseOutcomeBody(
    val title: String? = null,
    val description: String? = null,
)

@Serializable
data class AddCourseOutcomeLinkBody(
    val structureItemId: String,
    val targetKind: String,
    val quizQuestionId: String? = null,
    val measurementLevel: String,
    val intensityLevel: String,
    val subOutcomeId: String? = null,
)

@Serializable
data class CourseOutcomeSubOutcome(
    val id: String,
    val outcomeId: String,
    val title: String,
    val description: String = "",
    val sortOrder: Int = 0,
)

@Serializable
data class CreateCourseOutcomeSubOutcomeBody(
    val title: String,
    val description: String = "",
)