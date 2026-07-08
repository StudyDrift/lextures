package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseOutcomesLogicTest {
    @Test
    fun gradableOptions_skipsArchivedAndModules() {
        val items = listOf(
            CourseStructureItem(id = "m1", kind = "module", title = "Week 1"),
            CourseStructureItem(id = "a1", kind = "assignment", title = "Essay", parentId = "m1"),
            CourseStructureItem(id = "q1", kind = "quiz", title = "Quiz 1", parentId = "m1", archived = true),
        )
        val options = CourseOutcomesLogic.gradableOptions(items)
        assertEquals(1, options.size)
        assertEquals("Week 1 — Essay", options.first().label)
    }

    @Test
    fun dirtyOutcomeIds_detectsTitleChange() {
        val outcomes = listOf(
            CourseOutcome(id = "o1", title = "Original"),
        )
        val drafts = CourseOutcomesLogic.drafts(outcomes).toMutableMap()
        assertTrue(CourseOutcomesLogic.dirtyOutcomeIds(drafts, outcomes).isEmpty())
        drafts["o1"] = CourseOutcomesLogic.OutcomeDraft("Updated", "")
        assertEquals(listOf("o1"), CourseOutcomesLogic.dirtyOutcomeIds(drafts, outcomes))
    }

    @Test
    fun validateCreateTitle_rejectsEmpty() {
        assertEquals(CourseOutcomesLogic.ValidationError.TitleRequired, CourseOutcomesLogic.validateCreateTitle("   "))
        assertNull(CourseOutcomesLogic.validateCreateTitle("Analyze sources"))
    }

    @Test
    fun targetKind_mapsQuizScopes() {
        assertEquals("assignment", CourseOutcomesLogic.targetKind("assignment", quizScopeWhole = true))
        assertEquals("quiz", CourseOutcomesLogic.targetKind("quiz", quizScopeWhole = true))
        assertEquals("quiz_question", CourseOutcomesLogic.targetKind("quiz", quizScopeWhole = false))
    }

    @Test
    fun truncatedPrompt_collapsesWhitespace() {
        val long = "word ".repeat(40)
        val result = CourseOutcomesLogic.truncatedPrompt("  $long  ")
        assertTrue(result.endsWith("…"))
        assertTrue(!result.contains("\n"))
    }

    @Test
    fun buildAddLinkBody_includesLevels() {
        val body = CourseOutcomesLogic.buildAddLinkBody(
            structureItemId = "item-1",
            targetKind = "quiz",
            quizQuestionId = null,
            measurementLevel = "summative",
            intensityLevel = "high",
        )
        assertEquals("item-1", body.structureItemId)
        assertEquals("summative", body.measurementLevel)
        assertEquals("high", body.intensityLevel)
    }
}