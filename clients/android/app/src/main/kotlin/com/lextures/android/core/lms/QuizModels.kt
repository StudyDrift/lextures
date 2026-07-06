package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// region Quiz delivery (M4.1)

@Serializable
data class ModuleQuizPayload(
    val itemId: String? = null,
    val title: String? = null,
    val markdown: String? = null,
    val dueAt: String? = null,
    val unlimitedAttempts: Boolean? = null,
    val maxAttempts: Int? = null,
    val gradeAttemptPolicy: String? = null,
    val passingScorePercent: Int? = null,
    val pointsWorth: Int? = null,
    val timeLimitMinutes: Int? = null,
    val oneQuestionAtATime: Boolean? = null,
    val lockdownMode: String? = null,
    val shuffleQuestions: Boolean? = null,
    val shuffleChoices: Boolean? = null,
    val allowBackNavigation: Boolean? = null,
    val requiresQuizAccessCode: Boolean? = null,
    val randomQuestionPoolCount: Int? = null,
    val questions: List<QuizQuestion>? = null,
    val usesServerQuestionSampling: Boolean? = null,
    val isAdaptive: Boolean? = null,
    val showScoreTiming: String? = null,
    val reviewVisibility: String? = null,
    val reviewWhen: String? = null,
) {
    val questionCount: Int get() = questions?.size ?: 0
}

@Serializable
data class QuizQuestion(
    val id: String,
    val prompt: String = "",
    val questionType: String = "short_answer",
    val choices: List<String>? = null,
    val choiceIds: List<String>? = null,
    val typeConfig: QuizTypeConfig? = null,
    val correctChoiceIndex: Int? = null,
    val multipleAnswer: Boolean? = null,
    val answerWithImage: Boolean? = null,
    val required: Boolean? = null,
    val points: Int? = null,
    val estimatedMinutes: Int? = null,
)

@Serializable
data class QuizTypeConfig(
    val items: List<String>? = null,
    val pairs: List<QuizMatchingPairConfig>? = null,
    val starterCode: String? = null,
    val language: String? = null,
    val languageId: Int? = null,
    val multiFile: Boolean? = null,
    val files: List<QuizCodeFileConfig>? = null,
    val testCases: List<QuizCodeTestCaseConfig>? = null,
)

@Serializable
data class QuizCodeFileConfig(
    val path: String? = null,
    val content: String? = null,
)

@Serializable
data class QuizCodeTestCaseConfig(
    val id: String? = null,
    val input: String? = null,
    val expectedOutput: String? = null,
    val isHidden: Boolean? = null,
)

@Serializable
data class QuizMatchingPairConfig(
    val leftId: String? = null,
    val rightId: String? = null,
    val left: String? = null,
    val right: String? = null,
)

@Serializable
data class QuizStartRequest(
    val quizAccessCode: String? = null,
)

@Serializable
data class QuizStartResponse(
    val attemptId: String,
    val attemptNumber: Int = 1,
    val startedAt: String = "",
    val lockdownMode: String = "standard",
    val hintsDisabled: Boolean? = null,
    val backNavigationAllowed: Boolean = true,
    val currentQuestionIndex: Int = 0,
    val deadlineAt: String? = null,
    val maxAttempts: Int? = null,
    val remainingAttempts: Int? = null,
    val retakePolicy: String? = null,
)

@Serializable
data class QuizCurrentQuestionResponse(
    val question: QuizQuestion? = null,
    val questionIndex: Int = 0,
    val totalQuestions: Int = 0,
    val completed: Boolean = false,
)

@Serializable
data class QuizAdvanceResponse(
    val locked: Boolean = true,
    val currentQuestionIndex: Int = 0,
    val completed: Boolean = false,
)

@Serializable
data class QuizMatchingPairResponse(
    val leftId: String,
    val rightId: String,
)

@Serializable
data class QuizCodeSubmission(
    val language: String,
    val code: String,
)

@Serializable
data class QuizHotspotClick(
    val x: Double,
    val y: Double,
)

@Serializable
data class QuizQuestionResponseItem(
    val questionId: String,
    val selectedChoiceIndex: Int? = null,
    val selectedChoiceIndices: List<Int>? = null,
    val textAnswer: String? = null,
    val matchingPairs: List<QuizMatchingPairResponse>? = null,
    val orderingSequence: List<String>? = null,
    val hotspotClick: QuizHotspotClick? = null,
    val numericValue: Double? = null,
    val formulaLatex: String? = null,
    val codeSubmission: QuizCodeSubmission? = null,
    val fileKey: String? = null,
)

@Serializable
data class QuizSubmitRequest(
    val attemptId: String,
    val responses: List<QuizQuestionResponseItem>? = null,
)

@Serializable
data class QuizSubmitResponse(
    val attemptId: String,
    val pointsEarned: Double = 0.0,
    val pointsPossible: Double = 0.0,
    val scorePercent: Double = 0.0,
)

@Serializable
data class QuizResultsScoreSummary(
    val pointsEarned: Double = 0.0,
    val pointsPossible: Double = 0.0,
    val scorePercent: Double = 0.0,
)

@Serializable
data class QuizResultsQuestionResult(
    val questionIndex: Int = 0,
    val questionId: String? = null,
    val questionType: String = "",
    val promptSnapshot: String? = null,
    val isCorrect: Boolean? = null,
    val pointsAwarded: Double? = null,
    val maxPoints: Double = 0.0,
)

@Serializable
data class QuizResultsResponse(
    val attemptId: String,
    val attemptNumber: Int = 1,
    val status: String = "",
    val submittedAt: String? = null,
    val academicIntegrityFlag: Boolean? = null,
    val needsManualGrading: Boolean? = null,
    val score: QuizResultsScoreSummary? = null,
    val questions: List<QuizResultsQuestionResult>? = null,
)

@Serializable
data class QuizFocusLossRequest(
    val eventType: String,
    val durationMs: Int? = null,
)

@Serializable
data class QuizProctoringConfig(
    val id: String,
    val quizItemId: String,
    val externalToolId: String,
    val vendor: String,
    val required: Boolean = false,
)

@Serializable
data class QuizCodeRunRequest(
    val code: String,
    val languageId: Int? = null,
)

@Serializable
data class QuizCodeRunResult(
    val status: String,
    val passed: Boolean,
    val actualOutput: String,
    val expectedOutput: String,
    val stderr: String? = null,
    val executionMs: Int? = null,
    val memoryKb: Int? = null,
)

@Serializable
data class QuizCodeRunResponse(
    val questionId: String,
    val results: List<QuizCodeRunResult> = emptyList(),
    val pointsEarned: Double = 0.0,
    val pointsPossible: Double = 0.0,
)

data class QuizAnswerState(
    var choice: Int? = null,
    var choices: Set<Int>? = null,
    var text: String? = null,
    var numeric: Double? = null,
    var ordering: List<String>? = null,
    var matching: Map<String, String>? = null,
    var hotspot: QuizHotspotClick? = null,
)

enum class QuizSaveState {
    Idle, Saving, Saved, Failed, Queued,
}

enum class QuizQuestionKind(val wire: String) {
    MultipleChoice("multiple_choice"),
    FillInBlank("fill_in_blank"),
    Essay("essay"),
    TrueFalse("true_false"),
    ShortAnswer("short_answer"),
    Matching("matching"),
    Ordering("ordering"),
    Hotspot("hotspot"),
    Numeric("numeric"),
    Formula("formula"),
    Code("code"),
    FileUpload("file_upload"),
    AudioResponse("audio_response"),
    VideoResponse("video_response"),
    Unknown("unknown");

    val supportsMobileInput: Boolean
        get() = when (this) {
            AudioResponse, VideoResponse, Hotspot, Unknown -> false
            else -> true
        }

    companion object {
        fun from(raw: String): QuizQuestionKind =
            entries.firstOrNull { it.wire == raw } ?: Unknown
    }
}

data class QuizMatchingPairDraft(
    val leftId: String,
    val rightId: String? = null,
    val left: String,
    val right: String? = null,
)

// endregion
