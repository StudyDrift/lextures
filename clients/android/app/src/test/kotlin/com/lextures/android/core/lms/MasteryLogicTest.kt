package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Test

class MasteryLogicTest {
    @Test
    fun levelThresholds() {
        assertEquals(MasteryLevel.MASTERED, MasteryLogic.level(0.9, true))
        assertEquals(MasteryLevel.MASTERED, MasteryLogic.level(0.8, true))
        assertEquals(MasteryLevel.DEVELOPING, MasteryLogic.level(0.7, true))
        assertEquals(MasteryLevel.DEVELOPING, MasteryLogic.level(0.6, true))
        assertEquals(MasteryLevel.BEGINNING, MasteryLogic.level(0.5, true))
        assertEquals(MasteryLevel.BEGINNING, MasteryLogic.level(0.4, true))
        assertEquals(MasteryLevel.AT_RISK, MasteryLogic.level(0.1, true))
        assertEquals(MasteryLevel.NOT_ASSESSED, MasteryLogic.level(0.9, false))
        assertEquals(MasteryLevel.NOT_ASSESSED, MasteryLogic.level(null, true))
    }

    @Test
    fun rowsJoinsConceptsAndCellsAndSortsUnassessedFirst() {
        val row = StudentMasteryRow(
            enrollmentId = "e1",
            userId = "u1",
            concepts = listOf(
                MasteryConcept(id = "c1", name = "Fractions"),
                MasteryConcept(id = "c2", name = "Decimals"),
                MasteryConcept(id = "c3", name = "Ratios"),
            ),
            cells = listOf(
                MasteryCell(conceptId = "c1", masteryScore = 0.9, assessed = true),
                MasteryCell(conceptId = "c2", masteryScore = 0.3, assessed = true),
            ),
        )
        val rows = MasteryLogic.rows(row)
        assertEquals(3, rows.size)
        assertEquals(listOf("c3", "c2", "c1"), rows.map { it.id })
        assertEquals(MasteryLevel.NOT_ASSESSED, rows[0].level)
        assertEquals(MasteryLevel.AT_RISK, rows[1].level)
        assertEquals(MasteryLevel.MASTERED, rows[2].level)
    }

    @Test
    fun summaryCountsMasteredAndAtRisk() {
        val rows = listOf(
            MasteryConceptRow("1", "A", 0.9, true, MasteryLevel.MASTERED),
            MasteryConceptRow("2", "B", 0.85, true, MasteryLevel.MASTERED),
            MasteryConceptRow("3", "C", 0.1, true, MasteryLevel.AT_RISK),
            MasteryConceptRow("4", "D", null, false, MasteryLevel.NOT_ASSESSED),
        )
        val summary = MasteryLogic.summary(rows)
        assertEquals(2, summary.mastered)
        assertEquals(1, summary.atRisk)
        assertEquals(4, summary.total)
    }

    @Test
    fun releasedReportCardsFiltersAndSortsDescending() {
        val cards = listOf(
            ReportCardSummary(
                id = "1", studentId = "s", courseId = "c", gradingPeriod = "Q1",
                status = "draft",
            ),
            ReportCardSummary(
                id = "2", studentId = "s", courseId = "c", gradingPeriod = "Q2",
                status = "released",
            ),
            ReportCardSummary(
                id = "3", studentId = "s", courseId = "c", gradingPeriod = "Q1",
                status = "released",
            ),
        )
        val released = MasteryLogic.releasedReportCards(cards)
        assertEquals(listOf("2", "3"), released.map { it.id })
    }
}
