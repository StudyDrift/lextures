package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Test

class ReviewLogicTest {
    @Test
    fun filterQueueByCourse() {
        val items = listOf(
            sampleItem("q1", "BIO101"),
            sampleItem("q2", "CS101"),
        )
        assertEquals(listOf("q1"), ReviewLogic.filterQueue(items, "BIO101").map { it.questionId })
        assertEquals(2, ReviewLogic.filterQueue(items, null).size)
    }

    @Test
    fun formatAnswerPreviewString() {
        assertEquals("Paris", ReviewLogic.formatAnswerPreview(JsonPrimitive("Paris")))
    }

    @Test
    fun toQuizQuestionParsesChoices() {
        val item = sampleItem(
            questionId = "q1",
            courseCode = "BIO101",
            questionType = "multiple_choice",
            options = buildJsonObject {
                put("choices", kotlinx.serialization.json.buildJsonArray {
                    add(JsonPrimitive("One"))
                    add(JsonPrimitive("Two"))
                })
            },
        )
        val question = ReviewLogic.toQuizQuestion(item)
        assertEquals(listOf("One", "Two"), question?.choices)
    }

    @Test
    fun idempotencyKeyUsesQuestionAndTimestamp() {
        assertEquals("srs-review:abc:1700000000000", ReviewLogic.idempotencyKey("abc", 1_700_000_000_000))
    }

    private fun sampleItem(
        questionId: String,
        courseCode: String,
        questionType: String = "short_answer",
        options: kotlinx.serialization.json.JsonElement? = null,
    ) = ReviewQueueItem(
        stateId = "state-$questionId",
        questionId = questionId,
        courseId = "course-$courseCode",
        courseCode = courseCode,
        courseTitle = courseCode,
        nextReviewAt = "2026-07-01T12:00:00Z",
        stem = "Prompt",
        questionType = questionType,
        options = options,
        correctAnswer = JsonPrimitive("Answer"),
        explanation = null,
    )
}
