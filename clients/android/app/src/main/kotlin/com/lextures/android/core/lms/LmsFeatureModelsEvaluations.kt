package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// region Course evaluations (M7.7)

@Serializable
enum class EvaluationQuestionType {
    @SerialName("rating") Rating,
    @SerialName("multiple_choice") MultipleChoice,
    @SerialName("open_text") OpenText,
}

@Serializable
data class EvaluationQuestion(
    val type: EvaluationQuestionType,
    val text: String,
    val options: List<String>? = null,
    val required: Boolean? = null,
) {
    val isRequired: Boolean get() = required == true
}

@Serializable
data class EvaluationStatus(
    val windowOpen: Boolean = false,
    val windowId: String? = null,
    val hasSubmitted: Boolean = false,
    val opensAt: String? = null,
    val closesAt: String? = null,
    val questions: List<EvaluationQuestion>? = null,
)

@Serializable
data class EvaluationSubmitBody(
    val answers: Map<String, String>,
)

@Serializable
data class EvaluationSubmitResponse(
    val message: String? = null,
)

@Serializable
data class EvaluationQuestionResult(
    val index: Int,
    val type: EvaluationQuestionType,
    val text: String,
    val average: Double? = null,
    val distribution: Map<String, Int>? = null,
    val openTexts: List<String>? = null,
)

@Serializable
data class EvaluationResults(
    val windowId: String,
    val opensAt: String,
    val closesAt: String,
    val responseCount: Int,
    val enrolledCount: Int,
    val completionPct: Double,
    val meetsThreshold: Boolean,
    val questions: List<EvaluationQuestionResult> = emptyList(),
)

// endregion
