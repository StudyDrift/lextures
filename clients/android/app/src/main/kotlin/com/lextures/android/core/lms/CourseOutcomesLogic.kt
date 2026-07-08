package com.lextures.android.core.lms

import androidx.annotation.StringRes
import com.lextures.android.R
import kotlin.math.roundToInt

/** Course outcomes CRUD, mapping, and class-progress helpers (M13.5). */
object CourseOutcomesLogic {
    data class GradableOption(val id: String, val label: String, val kind: String)

    data class OutcomeDraft(val title: String, val description: String)

    data class QuizQuestionOption(val id: String, val prompt: String)

    enum class MeasurementLevelId {
        diagnostic,
        formative,
        summative,
        performance,
    }

    enum class IntensityLevelId {
        low,
        medium,
        high,
    }

    sealed class ValidationError {
        data object TitleRequired : ValidationError()
        data object SelectItem : ValidationError()
        data object SelectQuestion : ValidationError()
    }

    val defaultMeasurement = MeasurementLevelId.formative
    val defaultIntensity = IntensityLevelId.medium

    fun cacheKeyOutcomes(courseCode: String): String = "course:$courseCode:outcomes-settings"
    fun saveIdempotencyKey(courseCode: String): String = "course-outcomes:$courseCode:save"
    fun createOutcomeIdempotencyKey(courseCode: String): String = "course-outcomes:$courseCode:create"
    fun deleteOutcomeIdempotencyKey(courseCode: String, outcomeId: String): String =
        "course-outcomes:$courseCode:delete:$outcomeId"
    fun addLinkIdempotencyKey(courseCode: String, outcomeId: String, itemId: String): String =
        "course-outcomes:$courseCode:link:$outcomeId:$itemId"
    fun deleteLinkIdempotencyKey(courseCode: String, outcomeId: String, linkId: String): String =
        "course-outcomes:$courseCode:unlink:$outcomeId:$linkId"

    @StringRes
    fun measurementLabelRes(level: String): Int = when (level) {
        "diagnostic" -> R.string.mobile_courseSettings_outcomes_measurement_diagnostic
        "formative" -> R.string.mobile_courseSettings_outcomes_measurement_formative
        "summative" -> R.string.mobile_courseSettings_outcomes_measurement_summative
        "performance" -> R.string.mobile_courseSettings_outcomes_measurement_performance
        else -> R.string.mobile_courseSettings_outcomes_measurement_formative
    }

    @StringRes
    fun intensityLabelRes(level: String): Int = when (level) {
        "low" -> R.string.mobile_courseSettings_outcomes_intensity_low
        "medium" -> R.string.mobile_courseSettings_outcomes_intensity_medium
        "high" -> R.string.mobile_courseSettings_outcomes_intensity_high
        else -> R.string.mobile_courseSettings_outcomes_intensity_medium
    }

    @StringRes
    fun kindLabelRes(kind: String): Int = when (kind) {
        "quiz" -> R.string.mobile_courseSettings_outcomes_kind_quiz
        else -> R.string.mobile_courseSettings_outcomes_kind_assignment
    }

    fun gradableOptions(structure: List<CourseStructureItem>): List<GradableOption> {
        val byId = structure.associateBy { it.id }
        return structure
            .filter { (it.kind == "assignment" || it.kind == "quiz") && it.archived != true }
            .map { item ->
                var moduleTitle = ""
                var parent = item.parentId?.let { byId[it] }
                val guard = mutableSetOf<String>()
                while (parent != null && parent.id !in guard) {
                    guard += parent.id
                    if (parent.kind == "module") {
                        moduleTitle = parent.title
                        break
                    }
                    parent = parent.parentId?.let { byId[it] }
                }
                val label = if (moduleTitle.isEmpty()) item.title else "$moduleTitle — ${item.title}"
                GradableOption(item.id, label, item.kind)
            }
            .sortedBy { it.label.lowercase() }
    }

    fun drafts(outcomes: List<CourseOutcome>): Map<String, OutcomeDraft> =
        outcomes.associate { it.id to OutcomeDraft(it.title, it.description) }

    fun dirtyOutcomeIds(drafts: Map<String, OutcomeDraft>, outcomes: List<CourseOutcome>): List<String> =
        drafts.mapNotNull { (id, draft) ->
            val original = outcomes.firstOrNull { it.id == id } ?: return@mapNotNull null
            val titleChanged = draft.title.trim() != original.title.trim()
            val descriptionChanged = draft.description.trim() != original.description.trim()
            if (titleChanged || descriptionChanged) id else null
        }

    fun isDirty(drafts: Map<String, OutcomeDraft>, outcomes: List<CourseOutcome>): Boolean =
        dirtyOutcomeIds(drafts, outcomes).isNotEmpty()

    fun validateDrafts(drafts: Map<String, OutcomeDraft>, outcomes: List<CourseOutcome>): ValidationError? {
        dirtyOutcomeIds(drafts, outcomes).forEach { id ->
            if (drafts[id]?.title?.trim().isNullOrEmpty()) return ValidationError.TitleRequired
        }
        return null
    }

    fun validateCreateTitle(title: String): ValidationError? =
        if (title.trim().isEmpty()) ValidationError.TitleRequired else null

    fun buildCreateBody(title: String, description: String) =
        CreateCourseOutcomeBody(title = title.trim(), description = description.trim())

    fun buildPatchBody(draft: OutcomeDraft) =
        PatchCourseOutcomeBody(title = draft.title.trim(), description = draft.description.trim())

    fun buildAddLinkBody(
        structureItemId: String,
        targetKind: String,
        quizQuestionId: String?,
        measurementLevel: String,
        intensityLevel: String,
    ) = AddCourseOutcomeLinkBody(
        structureItemId = structureItemId,
        targetKind = targetKind,
        quizQuestionId = quizQuestionId,
        measurementLevel = measurementLevel,
        intensityLevel = intensityLevel,
    )

    fun targetKind(gradableKind: String, quizScopeWhole: Boolean): String = when {
        gradableKind == "assignment" -> "assignment"
        quizScopeWhole -> "quiz"
        else -> "quiz_question"
    }

    fun questionOptions(questions: List<QuizQuestion>?): List<QuizQuestionOption> =
        (questions ?: emptyList()).map { QuizQuestionOption(it.id, truncatedPrompt(it.prompt)) }

    fun truncatedPrompt(prompt: String): String {
        val collapsed = prompt.replace(Regex("\\s+"), " ").trim()
        return if (collapsed.length <= 120) collapsed else collapsed.take(120) + "…"
    }

    fun progressPercentLabel(value: Double?): String =
        if (value != null && value.isFinite()) "${value.roundToInt()}%" else "—"

    @StringRes
    fun linkItemTitleRes(link: CourseOutcomeLink): Int? = when (link.targetKind) {
        "quiz" -> R.string.mobile_courseSettings_outcomes_linkWholeQuiz
        "quiz_question" -> R.string.mobile_courseSettings_outcomes_linkQuestion
        else -> null
    }

    fun rollupPercentLabel(value: Double?): String? =
        if (value != null && value.isFinite()) value.roundToInt().toString() else null
}