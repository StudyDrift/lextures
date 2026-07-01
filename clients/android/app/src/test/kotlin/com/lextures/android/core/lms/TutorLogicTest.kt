package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TutorLogicTest {
    @Test
    fun parseStreamContentEvent() {
        val event = TutorLogic.parseStreamEvent("""{"type":"content","text":"Hello"}""")
        assertEquals(TutorStreamEvent.Content("Hello"), event)
    }

    @Test
    fun parseStreamDoneEvent() {
        val event = TutorLogic.parseStreamEvent("""{"type":"done","sessionId":"s1","citations":[]}""")
        assertEquals(TutorStreamEvent.Done(sessionId = "s1"), event)
    }

    @Test
    fun messageWithContextPrefixesFirstMessage() {
        val result = TutorLogic.messageWithContext(
            text = "Explain step 3",
            itemTitle = "Photosynthesis",
            itemKind = "content_page",
            includeContext = true,
        )
        assertTrue(result.contains("Photosynthesis"))
        assertTrue(result.contains("Explain step 3"))
    }

    @Test
    fun shouldShowFabWhenCourseEnablesTutor() {
        val disabled = CourseSummary(
            id = "1",
            courseCode = "BIO-101",
            title = "Biology",
            aiTutorEnabled = false,
        )
        assertFalse(TutorLogic.shouldShowFab(disabled))
        assertTrue(TutorLogic.shouldShowFab(disabled.copy(aiTutorEnabled = true)))
    }

    @Test
    fun gracefulHttpMessageBudgetExceeded() {
        assertEquals("BUDGET_EXCEEDED", TutorLogic.gracefulHttpMessage(402, "BUDGET_EXCEEDED"))
    }
}