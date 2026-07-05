package com.lextures.android.core.lms

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class EvaluationLogicTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

    @Test
    fun evaluationsEnabled_requiresBothFlags() {
        assertFalse(EvaluationLogic.evaluationsEnabled(MobilePlatformFeatures()))
        val on = MobilePlatformFeatures(ffCourseEvaluations = true, ffMobileCourseEvaluations = true)
        assertTrue(EvaluationLogic.evaluationsEnabled(on))
        val rolloutOff = MobilePlatformFeatures(ffCourseEvaluations = true, ffMobileCourseEvaluations = false)
        assertFalse(EvaluationLogic.evaluationsEnabled(rolloutOff))
    }

    @Test
    fun missingRequiredIndices_flagsBlankRequiredAnswers() {
        val questions = listOf(
            EvaluationQuestion(EvaluationQuestionType.Rating, "Q1", required = true),
            EvaluationQuestion(EvaluationQuestionType.OpenText, "Q2", required = false),
            EvaluationQuestion(EvaluationQuestionType.MultipleChoice, "Q3", options = listOf("A"), required = true),
        )
        val missing = EvaluationLogic.missingRequiredIndices(questions, mapOf("1" to "note"))
        assertEquals(listOf(0, 2), missing)
    }

    @Test
    fun isSubmitBlocked_respectsSubmittedAndClosedStates() {
        assertTrue(EvaluationLogic.isSubmitBlocked(null))
        assertTrue(
            EvaluationLogic.isSubmitBlocked(
                EvaluationStatus(windowOpen = true, windowId = "w1", hasSubmitted = true),
            ),
        )
        assertFalse(
            EvaluationLogic.isSubmitBlocked(
                EvaluationStatus(windowOpen = true, windowId = "w1", hasSubmitted = false),
            ),
        )
    }

    @Test
    fun draftRoundTrip() {
        EvaluationLogic.saveDraft(context, "CS101", "win-1", mapOf("0" to "5"))
        assertEquals(mapOf("0" to "5"), EvaluationLogic.loadDraft(context, "CS101", "win-1"))
        EvaluationLogic.clearDraft(context, "CS101", "win-1")
        assertTrue(EvaluationLogic.loadDraft(context, "CS101", "win-1").isEmpty())
    }
}
