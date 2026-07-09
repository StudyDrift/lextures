package com.lextures.android.core.lms

import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseGradingAgentsLogicTest {
    @Test
    fun gradableOptions_excludesExistingAgents() {
        val structure = listOf(
            CourseStructureItem(id = "a1", kind = "assignment", title = "Essay"),
            CourseStructureItem(id = "q1", kind = "quiz", title = "Quiz 1"),
        )
        val options = CourseGradingAgentsLogic.gradableOptions(structure, setOf("a1"))
        assertEquals(1, options.size)
        assertEquals("q1", options.first().id)
    }

    @Test
    fun isDirty_detectsPromptChange() {
        val baseline = CourseGradingAgentsLogic.draft(null)
        val current = baseline.copy(prompt = "Grade for clarity")
        assertFalse(CourseGradingAgentsLogic.isDirty(baseline, baseline))
        assertTrue(CourseGradingAgentsLogic.isDirty(current, baseline))
    }

    @Test
    fun validateDraft_requiresPrompt() {
        assertEquals(
            CourseGradingAgentsLogic.ValidationError.PromptRequired,
            CourseGradingAgentsLogic.validateDraft(CourseGradingAgentsLogic.AgentDraft(prompt = "   ")),
        )
        assertNull(CourseGradingAgentsLogic.validateDraft(CourseGradingAgentsLogic.AgentDraft(prompt = "Use rubric")))
    }

    @Test
    fun buildPutBody_trimsPrompt() {
        val draft = CourseGradingAgentsLogic.AgentDraft(
            prompt = "  Grade carefully  ",
            includeAssignmentContent = true,
            status = "accepted",
            autoGradeNew = true,
        )
        val body = CourseGradingAgentsLogic.buildPutBody(draft, "assignment")
        assertEquals("Grade carefully", body.prompt)
        assertEquals("accepted", body.status)
        assertTrue(body.autoGradeNew)
    }

    @Test
    fun defaultWorkflowGraph_includesOutputNode() {
        val graph = CourseGradingAgentsLogic.defaultWorkflowGraph("assignment")
        val nodes = graph.jsonObject["nodes"]?.jsonArray
        assertNotNull(nodes)
        assertEquals(1, nodes!!.size)
        assertEquals("output", nodes[0].jsonObject["type"]?.jsonPrimitive?.content)
    }

    @Test
    fun graderAgentPath_usesQuizCollection() {
        assertTrue(
            CourseGradingAgentsLogic.graderAgentPath("C-1", "item", "quiz").contains("/quizzes/"),
        )
        assertTrue(
            CourseGradingAgentsLogic.graderAgentPath("C-1", "item", "assignment").contains("/assignments/"),
        )
    }
}
