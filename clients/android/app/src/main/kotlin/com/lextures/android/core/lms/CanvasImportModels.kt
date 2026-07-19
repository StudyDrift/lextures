package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

/** Canvas import API models (MOB.2). */
@Serializable
data class CanvasCourseListItem(
    val id: Long,
    val name: String,
    val courseCode: String? = null,
    val workflowState: String? = null,
    val termName: String? = null,
)

@Serializable
data class CanvasCoursesResponse(
    val courses: List<CanvasCourseListItem>? = null,
)

@Serializable
data class CanvasListCoursesRequest(
    val canvasBaseUrl: String,
    val accessToken: String,
)

@Serializable
data class CanvasImportIncludeBody(
    val modules: Boolean,
    val assignments: Boolean,
    val quizzes: Boolean,
    val enrollments: Boolean,
    val grades: Boolean,
    val settings: Boolean,
    val files: Boolean,
    val announcements: Boolean,
) {
    companion object {
        fun from(include: CanvasImportLogic.Include): CanvasImportIncludeBody =
            CanvasImportIncludeBody(
                modules = include.modules,
                assignments = include.assignments,
                quizzes = include.quizzes,
                enrollments = include.enrollments,
                grades = include.grades,
                settings = include.settings,
                files = include.files,
                announcements = include.announcements,
            )
    }
}

@Serializable
data class PostCourseImportCanvasRequest(
    val mode: String,
    val canvasBaseUrl: String,
    val canvasCourseId: String,
    val accessToken: String,
    val include: CanvasImportIncludeBody,
    val canvasGradeSyncEnabled: Boolean? = null,
)

@Serializable
data class CanvasImportQueuedResponse(
    val jobId: String? = null,
    val message: String? = null,
)
