package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class QuizLogicTest {
    @Test
    fun visibleChoicesTrimsEmpty() {
        val question = QuizQuestion(
            id = "q1",
            choices = listOf(" A ", "", "B"),
        )
        assertEquals(listOf("A", "B"), QuizLogic.visibleChoices(question))
    }

    @Test
    fun buildResponseItemMultipleChoice() {
        val question = QuizQuestion(
            id = "q1",
            questionType = "multiple_choice",
            multipleAnswer = false,
        )
        val item = QuizLogic.buildResponseItem(question, QuizAnswerState(choice = 1))
        assertEquals("q1", item.questionId)
        assertEquals(1, item.selectedChoiceIndex)
    }

    @Test
    fun isAnsweredShortAnswer() {
        val question = QuizQuestion(id = "q1", questionType = "short_answer")
        assertFalse(QuizLogic.isAnswered(question, null))
        assertTrue(QuizLogic.isAnswered(question, QuizAnswerState(text = "hello")))
        assertFalse(QuizLogic.isAnswered(question, QuizAnswerState(text = "   ")))
    }

    @Test
    fun serverLockdownModes() {
        assertTrue(QuizLogic.isServerLockdown("one_at_a_time"))
        assertTrue(QuizLogic.isServerLockdown("kiosk"))
        assertFalse(QuizLogic.isServerLockdown("standard"))
    }

    @Test
    fun formatTimer() {
        assertEquals("2:05", QuizLogic.formatTimer(125))
        assertEquals("0:59", QuizLogic.formatTimer(59))
    }
}
