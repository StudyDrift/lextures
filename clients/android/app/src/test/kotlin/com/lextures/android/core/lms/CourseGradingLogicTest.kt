package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseGradingLogicTest {
    @Test
    fun weightTotal_sumsGroups() {
        val groups = listOf(
            CourseGradingLogic.EditableAssignmentGroup("a", "1", "Exams", 0, "50"),
            CourseGradingLogic.EditableAssignmentGroup("b", "2", "Homework", 1, "20"),
        )
        assertEquals(70.0, CourseGradingLogic.weightTotal(groups), 0.001)
    }

    @Test
    fun hasWeightWarning_whenNotOneHundred() {
        assertTrue(CourseGradingLogic.hasWeightWarning(80.0))
        assertFalse(CourseGradingLogic.hasWeightWarning(100.0))
    }

    @Test
    fun validateBands_rejectsOutOfOrder() {
        val bands = listOf(
            CourseGradingLogic.GradingSchemeBand("a", "A", "50"),
            CourseGradingLogic.GradingSchemeBand("b", "B", "40"),
        )
        assertNotNull(CourseGradingLogic.validateBands(bands))
    }

    @Test
    fun validateBands_acceptsDefaultScale() {
        assertNull(CourseGradingLogic.validateBands(CourseGradingLogic.defaultBands()))
    }

    @Test
    fun gradableRows_skipsModules() {
        val items = listOf(
            CourseStructureItem(id = "m1", kind = "module", title = "Week 1"),
            CourseStructureItem(id = "a1", kind = "assignment", title = "Essay"),
        )
        val rows = CourseGradingLogic.gradableRows(items)
        assertEquals(1, rows.size)
        assertEquals("Week 1", rows.first().moduleTitle)
    }

    @Test
    fun isSettingsDirty_detectsWeightChange() {
        val baseline = CourseGradingLogic.FormBaseline(
            gradingScale = "letter_standard",
            groups = CourseGradingLogic.defaultGroups(),
            schemeType = "points",
            bands = CourseGradingLogic.defaultBands(),
            passMinPct = "60",
            completeMinPct = "50",
        )
        val current = baseline.copy(groups = baseline.groups.map { it.copy(weightPercent = "80") })
        assertTrue(CourseGradingLogic.isSettingsDirty(current, baseline))
    }

    @Test
    fun buildPutSettingsBody_usesTrimmedNames() {
        val form = CourseGradingLogic.FormBaseline(
            gradingScale = "percent",
            groups = listOf(CourseGradingLogic.EditableAssignmentGroup("g1", null, "  Labs  ", 0, "25")),
            schemeType = "points",
            bands = CourseGradingLogic.defaultBands(),
            passMinPct = "60",
            completeMinPct = "50",
        )
        val body = CourseGradingLogic.buildPutSettingsBody(form)
        assertEquals("percent", body.gradingScale)
        assertEquals("Labs", body.assignmentGroups.first().name)
        assertEquals(25.0, body.assignmentGroups.first().weightPercent, 0.001)
    }
}