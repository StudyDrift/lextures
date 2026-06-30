package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import com.lextures.android.features.grades.GradesDisplayLogic

class GradeCalculatorTest {
    @Test
    fun straightPointsWhenNoWeights() {
        val cols = listOf(
            GradeCalculator.ColumnForFinal("a", 100.0),
            GradeCalculator.ColumnForFinal("b", 50.0, dueAt = "2000-01-01T00:00:00Z"),
        )
        val pct = GradeCalculator.computeCourseFinalPercent(cols, mapOf("a" to "80", "b" to "40"), emptyList())
        assertEquals(80.0, pct!!, 0.001)
    }

    @Test
    fun dropLowestInGroup() {
        val cols = listOf("a", "b", "c", "d").map {
            GradeCalculator.ColumnForFinal(it, 100.0, assignmentGroupId = "g", dueAt = "2000-01-01T00:00:00Z")
        }
        val pct = GradeCalculator.computeCourseFinalPercent(
            cols,
            mapOf("a" to "60", "b" to "70", "c" to "80", "d" to "90"),
            listOf(GradeCalculator.GroupWeight("g", 100.0, dropLowest = 1)),
        )
        assertEquals(80.0, pct!!, 0.001)
    }

    @Test
    fun whatIfIncludesFutureOverride() {
        val cols = listOf(
            GradeCalculator.ColumnForFinal("hw", 100.0, assignmentGroupId = "ex", dueAt = "2000-01-01T00:00:00Z"),
            GradeCalculator.ColumnForFinal("final", 100.0, assignmentGroupId = "fi", dueAt = "2099-01-01T00:00:00Z"),
        )
        val groups = listOf(
            GradeCalculator.GroupWeight("ex", 40.0),
            GradeCalculator.GroupWeight("fi", 60.0),
        )
        val projected = GradeCalculator.computeWhatIfFinalPercent(
            cols,
            mapOf("hw" to "80", "final" to ""),
            groups,
            emptyMap(),
            mapOf("final" to "90"),
            emptySet(),
        )
        assertEquals(86.0, projected!!, 0.001)
    }

    @Test
    fun heldItemsExcludedFromWhatIfMerge() {
        val held = setOf("secret")
        val merged = GradeCalculator.mergeGradesForWhatIf(mapOf("secret" to "99"), emptyMap(), held)
        assertNull(merged["secret"])
        val withOverride = GradeCalculator.mergeGradesForWhatIf(mapOf("secret" to "99"), mapOf("secret" to "70"), held)
        assertEquals("70", withOverride["secret"])
    }

    @Test
    fun buildSectionsGroupsColumns() {
        val response = MyGradesResponse(
            columns = listOf(
                GradeColumn(id = "1", title = "A1", assignmentGroupId = "hw"),
                GradeColumn(id = "2", title = "Other"),
            ),
            assignmentGroups = listOf(AssignmentGroup(id = "hw", name = "Homework", weightPercent = 20.0)),
        )
        val sections = GradesDisplayLogic.buildSections(response)
        assertEquals(2, sections.size)
        assertEquals("Homework", sections[0].title)
        assertEquals(20.0, sections[0].weightPercent)
    }
}
