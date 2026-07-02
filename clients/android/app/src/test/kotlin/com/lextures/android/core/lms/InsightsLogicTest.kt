package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class InsightsLogicTest {
    @Test
    fun formatHoursRoundsSmallValues() {
        assertEquals("0", InsightsLogic.formatHours(0.05))
        assertEquals("9.4", InsightsLogic.formatHours(9.4))
        assertEquals("12", InsightsLogic.formatHours(12.0))
    }

    @Test
    fun goalProgressPercentCapsAt100() {
        assertEquals(100, InsightsLogic.goalProgressPercent(5f, 4f))
        assertEquals(50, InsightsLogic.goalProgressPercent(2f, 4f))
        assertNull(InsightsLogic.goalProgressPercent(2f, null))
    }

    @Test
    fun moduleCompletionPercentCountsItems() {
        val snapshot = ModulesProgressSnapshot(
            enrollmentId = "e1",
            modules = listOf(
                ModuleLockState(
                    moduleId = "m1",
                    locked = false,
                    complete = false,
                    items = listOf(
                        ItemLockState(itemId = "i1", locked = false, complete = true),
                        ItemLockState(itemId = "i2", locked = false, complete = false),
                    ),
                ),
            ),
        )
        assertEquals(50, InsightsLogic.moduleCompletionPercent(snapshot))
    }

    @Test
    fun journalEntryValidEnforcesLength() {
        assertFalse(InsightsLogic.journalEntryValid("   "))
        assertTrue(InsightsLogic.journalEntryValid("Felt good today"))
        assertFalse(InsightsLogic.journalEntryValid("a".repeat(281)))
    }

    @Test
    fun barWidthPercentUsesMaxMinutes() {
        assertEquals(50.0, InsightsLogic.barWidthPercent(30.0, 60.0), 0.001)
        assertEquals(100.0, InsightsLogic.barWidthPercent(90.0, 60.0), 0.001)
    }
}