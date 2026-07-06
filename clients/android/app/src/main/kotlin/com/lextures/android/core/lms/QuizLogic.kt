package com.lextures.android.core.lms

import java.time.Instant
import java.time.format.DateTimeParseException

object QuizLogic {
    fun isServerLockdown(mode: String?): Boolean =
        mode == "one_at_a_time" || mode == "kiosk"

    fun isKioskMode(mode: String?): Boolean = mode == "kiosk"

    /** Pre-start consent for server-enforced delivery (web parity). */
    fun needsLockdownConsent(mode: String?): Boolean = isServerLockdown(mode)

    /** Platform lockdown (screen pinning / lock task, FLAG_SECURE, integrity signals). */
    fun requiresDeviceLockdown(lockdownMode: String?, proctoringRequired: Boolean = false): Boolean =
        isKioskMode(lockdownMode) || proctoringRequired

    fun visibleChoices(question: QuizQuestion): List<String> =
        question.choices.orEmpty()
            .map { it.trim() }
            .filter { it.isNotEmpty() }

    fun orderingItems(question: QuizQuestion): List<String> {
        val configured = question.typeConfig?.items
            ?.map { it.trim() }
            ?.filter { it.isNotEmpty() }
        if (!configured.isNullOrEmpty()) return configured
        return visibleChoices(question)
    }

    fun matchingPairs(question: QuizQuestion): List<QuizMatchingPairDraft> =
        question.typeConfig?.pairs.orEmpty().mapIndexedNotNull { index, pair ->
            val left = pair.left?.trim().orEmpty()
            val right = pair.right?.trim().orEmpty()
            if (left.isEmpty() && right.isEmpty()) return@mapIndexedNotNull null
            QuizMatchingPairDraft(
                leftId = pair.leftId ?: "left-$index",
                rightId = pair.rightId,
                left = left,
                right = pair.right,
            )
        }

    fun sortedRightOptions(pairs: List<QuizMatchingPairDraft>): List<String> =
        pairs.mapNotNull { it.right?.trim()?.takeIf { r -> r.isNotEmpty() } }.distinct().sorted()

    fun buildMatchingPairsPayload(question: QuizQuestion, answer: QuizAnswerState?): List<QuizMatchingPairResponse> {
        val pairs = matchingPairs(question)
        val out = mutableListOf<QuizMatchingPairResponse>()
        pairs.forEachIndexed { index, pair ->
            val selectedRight = answer?.matching?.get(pair.leftId)?.trim().orEmpty()
            if (selectedRight.isEmpty()) return@forEachIndexed
            val match = pairs.firstOrNull { it.right?.trim() == selectedRight }
            val rightId = match?.rightId ?: "right-$index"
            out += QuizMatchingPairResponse(leftId = pair.leftId, rightId = rightId)
        }
        return out
    }

    fun buildResponseItem(question: QuizQuestion, answer: QuizAnswerState?): QuizQuestionResponseItem {
        val kind = QuizQuestionKind.from(question.questionType)
        return when (kind) {
            QuizQuestionKind.MultipleChoice, QuizQuestionKind.TrueFalse -> {
                if (question.multipleAnswer == true) {
                    QuizQuestionResponseItem(
                        questionId = question.id,
                        selectedChoiceIndices = answer?.choices?.sorted(),
                    )
                } else {
                    QuizQuestionResponseItem(
                        questionId = question.id,
                        selectedChoiceIndex = answer?.choice,
                    )
                }
            }
            QuizQuestionKind.Numeric -> QuizQuestionResponseItem(
                questionId = question.id,
                textAnswer = answer?.text,
                numericValue = answer?.numeric,
            )
            QuizQuestionKind.Formula -> QuizQuestionResponseItem(
                questionId = question.id,
                formulaLatex = answer?.text,
            )
            QuizQuestionKind.Ordering -> QuizQuestionResponseItem(
                questionId = question.id,
                orderingSequence = answer?.ordering ?: orderingItems(question),
            )
            QuizQuestionKind.Matching -> QuizQuestionResponseItem(
                questionId = question.id,
                matchingPairs = buildMatchingPairsPayload(question, answer),
            )
            QuizQuestionKind.Hotspot -> QuizQuestionResponseItem(
                questionId = question.id,
                hotspotClick = answer?.hotspot,
            )
            QuizQuestionKind.Code -> QuizQuestionResponseItem(
                questionId = question.id,
                codeSubmission = QuizCodeSubmission(
                    language = question.typeConfig?.language ?: "text",
                    code = answer?.text.orEmpty(),
                ),
            )
            QuizQuestionKind.FileUpload, QuizQuestionKind.AudioResponse, QuizQuestionKind.VideoResponse ->
                QuizQuestionResponseItem(
                    questionId = question.id,
                    textAnswer = answer?.text,
                    fileKey = answer?.text,
                )
            else -> QuizQuestionResponseItem(
                questionId = question.id,
                textAnswer = answer?.text,
            )
        }
    }

    fun isAnswered(question: QuizQuestion, answer: QuizAnswerState?): Boolean {
        if (answer == null) return false
        return when (QuizQuestionKind.from(question.questionType)) {
            QuizQuestionKind.MultipleChoice, QuizQuestionKind.TrueFalse ->
                if (question.multipleAnswer == true) !answer.choices.isNullOrEmpty() else answer.choice != null
            QuizQuestionKind.Numeric ->
                answer.numeric != null || !answer.text.isNullOrBlank()
            QuizQuestionKind.Formula, QuizQuestionKind.FillInBlank, QuizQuestionKind.ShortAnswer, QuizQuestionKind.Essay ->
                !answer.text.isNullOrBlank()
            QuizQuestionKind.Ordering -> !answer.ordering.isNullOrEmpty()
            QuizQuestionKind.Matching -> !answer.matching.isNullOrEmpty()
            QuizQuestionKind.Hotspot -> answer.hotspot != null
            QuizQuestionKind.FileUpload, QuizQuestionKind.AudioResponse, QuizQuestionKind.VideoResponse ->
                !answer.text.isNullOrBlank()
            QuizQuestionKind.Code -> !answer.text.isNullOrBlank()
            QuizQuestionKind.Unknown -> false
        }
    }

    fun secondsRemaining(deadlineISO: String?, now: Instant = Instant.now()): Int? {
        if (deadlineISO.isNullOrBlank()) return null
        return try {
            val deadline = Instant.parse(deadlineISO)
            maxOf(0, (deadline.epochSecond - now.epochSecond).toInt())
        } catch (_: DateTimeParseException) {
            null
        }
    }

    fun formatTimer(seconds: Int): String {
        val minutes = seconds / 60
        val secs = seconds % 60
        return "%d:%02d".format(minutes, secs)
    }

    fun retakePolicyNotice(policy: String?): String = when (policy) {
        "highest" -> "Your highest score counts toward the course grade."
        "latest" -> "Your most recent attempt counts toward the course grade."
        "first" -> "Your first submitted attempt counts toward the course grade."
        "average" -> "The average of your attempts counts toward the course grade."
        else -> "Your instructor chose how multiple attempts are scored."
    }

    const val CODE_MOBILE_MAX_BYTES = 16_384

    val codeSymbolSnippets = listOf(
        "{", "}", "(", ")", "[", "]", ";", ":", ".", ",", "=", "+", "-", "*", "/", "<", ">", "\"", "'",
        "    ", "\n",
    )

    fun starterCode(question: QuizQuestion): String =
        question.typeConfig?.starterCode?.trim().orEmpty()

    fun codeLanguageLabel(question: QuizQuestion): String =
        question.typeConfig?.language?.trim().takeUnless { it.isNullOrEmpty() } ?: "text"

    fun isCodeQuestionOversized(question: QuizQuestion): Boolean {
        if (question.typeConfig?.multiFile == true) return true
        if ((question.typeConfig?.files?.size ?: 0) > 1) return true
        return starterCode(question).toByteArray().size > CODE_MOBILE_MAX_BYTES
    }

    fun initialCodeAnswer(question: QuizQuestion, existing: QuizAnswerState?): QuizAnswerState {
        if (!existing?.text.isNullOrBlank()) return existing ?: QuizAnswerState()
        val starter = starterCode(question)
        return if (starter.isNotEmpty()) (existing ?: QuizAnswerState()).copy(text = starter) else existing ?: QuizAnswerState()
    }

    fun applyAutoIndent(text: String): String {
        if (!text.endsWith("\n")) return text
        val lastNewline = text.lastIndexOf('\n')
        if (lastNewline <= 0) return text
        val previousLine = text.substring(0, lastNewline).substringAfterLast('\n')
        val trimmed = previousLine.trim()
        val baseIndent = previousLine.takeWhile { it == ' ' || it == '\t' }
        return if (trimmed.endsWith("{") || trimmed.endsWith(":")) {
            text + baseIndent + "    "
        } else {
            text
        }
    }
}
