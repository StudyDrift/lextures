package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class OnboardingModelsTest {
    private val json = kotlinx.serialization.json.Json { ignoreUnknownKeys = true }

    @Test
    fun decodesOnboardingStatus() {
        val status = json.decodeFromString<OnboardingStatus>(
            """{"completed":false,"step":2,"shouldShowFlow":true}""",
        )
        assertFalse(status.completed)
        assertEquals(2, status.step)
        assertTrue(status.shouldShowFlow)
    }

    @Test
    fun decodesDiagnosticQuestions() {
        val response = json.decodeFromString<DiagnosticQuestionsResponse>(
            """{"questions":[{"id":"q1","prompt":"Pick one","choices":["a","b"]}]}""",
        )
        assertEquals(1, response.questions.size)
        assertEquals("q1", response.questions[0].id)
        assertEquals(2, response.questions[0].choices.size)
    }

    @Test
    fun decodesGoalsEnvelope() {
        val envelope = json.decodeFromString<GoalsEnvelope>(
            """
            {"goals":{"id":"g1","userId":"u1","topic":"python","dailyMinutes":20,
            "priorKnowledgeLevel":"beginner","diagnosticSkipped":true,"onboardingStep":6,
            "onboardingCompleted":true,"reminderOptIn":false,"recommendedCourseCode":"PY101",
            "recommendedCourseTitle":"Python Basics"}}
            """.trimIndent(),
        )
        assertEquals("PY101", envelope.goals.recommendedCourseCode)
        assertEquals("Python Basics", envelope.goals.recommendedCourseTitle)
        assertTrue(envelope.goals.onboardingCompleted)
    }

    @Test
    fun onboardingTopicsIncludePython() {
        assertTrue(onboardingTopics.any { it.id == "python" })
    }
}
