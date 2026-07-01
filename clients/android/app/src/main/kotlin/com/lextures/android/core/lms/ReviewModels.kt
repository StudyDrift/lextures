package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

enum class SrsGrade(val apiValue: String) {
    Again("again"),
    Hard("hard"),
    Good("good"),
    Easy("easy"),
    ;

    companion object {
        val entriesList = entries
    }
}

@Serializable
data class ReviewQueueItem(
    val stateId: String,
    val questionId: String,
    val courseId: String,
    val courseCode: String,
    val courseTitle: String,
    val nextReviewAt: String,
    val stem: String,
    val questionType: String,
    val options: JsonElement? = null,
    val correctAnswer: JsonElement? = null,
    val explanation: String? = null,
)

@Serializable
data class ReviewQueueResponse(
    val items: List<ReviewQueueItem> = emptyList(),
    val totalDue: Int = 0,
)

@Serializable
data class ReviewStats(
    val streak: Int = 0,
    val dueToday: Int = 0,
    val dueWeek: Int = 0,
    val retentionEstimate: Double = 0.0,
)

@Serializable
data class SrsReviewSubmitBody(
    val questionId: String,
    val grade: String,
    val responseMs: Int? = null,
)

@Serializable
data class SrsReviewSubmitResponse(
    val nextReviewAt: String,
    val intervalDays: Double,
)

@Serializable
data class LearnerRecommendationItem(
    val itemId: String,
    val itemType: String,
    val title: String,
    val surface: String,
    val reason: String,
    val score: Double,
)

@Serializable
data class LearnerRecommendationsResponse(
    val recommendations: List<LearnerRecommendationItem> = emptyList(),
    val degraded: Boolean = false,
)

data class ReviewCourseFilter(
    val courseCode: String,
    val courseTitle: String,
)

object ReviewLogic {
    const val PREFETCH_LIMIT = 50

    fun formatAnswerPreview(value: JsonElement?): String {
        if (value == null) return ""
        return when (value) {
            is JsonPrimitive -> value.contentOrNull ?: value.toString()
            is JsonArray -> value.mapNotNull { formatAnswerPreview(it).takeIf { text -> text.isNotEmpty() } }
                .joinToString(", ")
            is JsonObject -> value.entries.joinToString("\n") { (key, element) ->
                "$key: ${formatAnswerPreview(element)}"
            }
            else -> value.toString()
        }
    }

    fun filterQueue(items: List<ReviewQueueItem>, courseCode: String?): List<ReviewQueueItem> {
        if (courseCode.isNullOrEmpty()) return items
        return items.filter { it.courseCode == courseCode }
    }

    fun courseFilters(items: List<ReviewQueueItem>): List<ReviewCourseFilter> =
        items
            .distinctBy { it.courseCode }
            .sortedBy { it.courseTitle.lowercase() }
            .map { ReviewCourseFilter(it.courseCode, it.courseTitle) }

    fun toQuizQuestion(item: ReviewQueueItem): QuizQuestion? {
        val kind = QuizQuestionKind.from(item.questionType)
        if (!kind.supportsMobileInput) return null
        val parsed = parseOptions(item.options, item.questionType)
        return QuizQuestion(
            id = item.questionId,
            prompt = item.stem,
            questionType = item.questionType,
            choices = parsed.choices,
            choiceIds = parsed.choiceIds,
            typeConfig = parsed.typeConfig,
            multipleAnswer = parsed.multipleAnswer,
        )
    }

    fun idempotencyKey(questionId: String, ratedAtMs: Long): String =
        "srs-review:$questionId:$ratedAtMs"

    private data class ParsedOptions(
        val choices: List<String>? = null,
        val choiceIds: List<String>? = null,
        val typeConfig: QuizTypeConfig? = null,
        val multipleAnswer: Boolean? = null,
    )

    private fun parseOptions(value: JsonElement?, questionType: String): ParsedOptions {
        if (value == null) {
            return if (questionType == QuizQuestionKind.TrueFalse.wire) {
                ParsedOptions(choices = listOf("True", "False"))
            } else {
                ParsedOptions()
            }
        }
        if (value is JsonArray) {
            val choices = value.mapNotNull { jsonString(it) }
            return ParsedOptions(choices = choices.takeIf { it.isNotEmpty() })
        }
        if (value is JsonObject) {
            val choices = value["choices"]?.jsonArray?.mapNotNull { jsonString(it) }.orEmpty()
            if (choices.isNotEmpty()) {
                val ids = value["choiceIds"]?.jsonArray?.mapNotNull { jsonString(it) }
                return ParsedOptions(
                    choices = choices,
                    choiceIds = if (ids?.size == choices.size) ids else null,
                    typeConfig = parseTypeConfig(value),
                    multipleAnswer = value["multipleAnswer"]?.jsonPrimitive?.booleanOrNull,
                )
            }
            val items = value["items"]?.jsonArray?.mapNotNull { jsonString(it) }.orEmpty()
            if (items.isNotEmpty()) {
                return ParsedOptions(typeConfig = QuizTypeConfig(items = items))
            }
            val pairs = parseMatchingPairs(value["pairs"])
            if (pairs != null) {
                return ParsedOptions(typeConfig = QuizTypeConfig(pairs = pairs))
            }
        }
        return ParsedOptions()
    }

    private fun parseTypeConfig(value: JsonObject): QuizTypeConfig? {
        val items = value["items"]?.jsonArray?.mapNotNull { jsonString(it) }
        val pairs = parseMatchingPairs(value["pairs"])
        val starterCode = jsonString(value["starterCode"])
        val language = jsonString(value["language"])
        if (items == null && pairs == null && starterCode == null && language == null) return null
        return QuizTypeConfig(items = items, pairs = pairs, starterCode = starterCode, language = language)
    }

    private fun parseMatchingPairs(value: JsonElement?): List<QuizMatchingPairConfig>? {
        val array = value as? JsonArray ?: return null
        val pairs = array.mapNotNull { element ->
            val obj = element as? JsonObject ?: return@mapNotNull null
            val left = jsonString(obj["left"])
            val right = jsonString(obj["right"])
            if (left.isNullOrEmpty() && right.isNullOrEmpty()) return@mapNotNull null
            QuizMatchingPairConfig(
                leftId = jsonString(obj["leftId"]),
                rightId = jsonString(obj["rightId"]),
                left = left,
                right = right,
            )
        }
        return pairs.takeIf { it.isNotEmpty() }
    }

    private fun jsonString(value: JsonElement?): String? {
        if (value == null) return null
        return when (value) {
            is JsonPrimitive -> value.contentOrNull?.trim()?.takeIf { it.isNotEmpty() }
                ?: value.doubleOrNull?.let { number ->
                    if (number % 1.0 == 0.0) number.toLong().toString() else number.toString()
                }
            else -> null
        }
    }
}
