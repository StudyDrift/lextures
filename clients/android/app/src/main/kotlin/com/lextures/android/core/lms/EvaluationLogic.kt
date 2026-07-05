package com.lextures.android.core.lms

import android.content.Context
import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

object EvaluationLogic {
    fun evaluationsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCourseEvaluations && features.ffMobileCourseEvaluations

    fun evaluationTodoKey(courseCode: String, windowId: String): String =
        "evaluation:$courseCode:$windowId"

    fun draftCacheKey(courseCode: String, windowId: String): String =
        "evaluation:draft:$courseCode:$windowId"

    fun submitIdempotencyKey(courseCode: String, windowId: String): String =
        "evaluation-submit:$courseCode:$windowId"

    fun statusCacheKey(courseCode: String): String = "evaluation:status:$courseCode"

    fun resultsCacheKey(courseCode: String): String = "evaluation:results:$courseCode"

    fun shouldShowWorkspaceSection(
        course: CourseSummary,
        status: EvaluationStatus?,
        features: MobilePlatformFeatures,
    ): Boolean {
        if (!evaluationsEnabled(features)) return false
        if (course.viewerIsStaff) return true
        val current = status ?: return false
        return current.windowOpen || current.hasSubmitted
    }

    fun missingRequiredIndices(
        questions: List<EvaluationQuestion>,
        answers: Map<String, String>,
    ): List<Int> = questions.mapIndexedNotNull { index, question ->
        if (!question.isRequired) return@mapIndexedNotNull null
        val value = answers[index.toString()]?.trim().orEmpty()
        if (value.isEmpty()) index else null
    }

    fun isSubmitBlocked(status: EvaluationStatus?): Boolean {
        val current = status ?: return true
        if (current.hasSubmitted) return true
        return !current.windowOpen
    }

    fun formatDeadline(iso: String?): String = LmsDates.shortDateTime(iso)

    fun loadDraft(context: Context, courseCode: String, windowId: String): Map<String, String> {
        val raw = context.getSharedPreferences("evaluation_drafts", Context.MODE_PRIVATE)
            .getString(draftCacheKey(courseCode, windowId), null)
            ?: return emptyMap()
        return runCatching { Json.decodeFromString<Map<String, String>>(raw) }.getOrDefault(emptyMap())
    }

    fun saveDraft(context: Context, courseCode: String, windowId: String, answers: Map<String, String>) {
        context.getSharedPreferences("evaluation_drafts", Context.MODE_PRIVATE)
            .edit()
            .putString(draftCacheKey(courseCode, windowId), Json.encodeToString(answers))
            .apply()
    }

    fun clearDraft(context: Context, courseCode: String, windowId: String) {
        context.getSharedPreferences("evaluation_drafts", Context.MODE_PRIVATE)
            .edit()
            .remove(draftCacheKey(courseCode, windowId))
            .apply()
    }
}
