package com.lextures.android.core.design

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LexturesMotionTest {
    @Test
    fun durationScaleMatchesAN1Spec() {
        assertEquals(100, LexturesMotion.InstantMs)
        assertEquals(150, LexturesMotion.FastMs)
        assertEquals(220, LexturesMotion.BaseMs)
        assertEquals(320, LexturesMotion.SlowMs)
        assertEquals(480, LexturesMotion.DeliberateMs)
    }

    @Test
    fun distanceAndStaggerTokens() {
        assertEquals(0.97f, LexturesMotion.EnterScaleFrom)
        assertEquals(0.97f, LexturesMotion.PressScale)
        assertEquals(40, LexturesMotion.StaggerStepMs)
        assertEquals(8, LexturesMotion.StaggerMaxItems)
        assertEquals(0, LexturesMotion.staggerDelayMs(0))
        assertEquals(120, LexturesMotion.staggerDelayMs(3))
        assertEquals(280, LexturesMotion.staggerDelayMs(99))
    }

    @Test
    fun shouldReduceMotionCombinesOsAndAppFlags() {
        assertFalse(LexturesMotion.shouldReduceMotion(false, false))
        assertTrue(LexturesMotion.shouldReduceMotion(true, false))
        assertTrue(LexturesMotion.shouldReduceMotion(false, true))
        assertTrue(LexturesMotion.shouldReduceMotion(true, true))
    }

    @Test
    fun navigationDurationsHonorReducedMotionAndKillSwitch() {
        assertEquals(LexturesMotion.BaseMs, LexturesMotion.navigationDurationMs(false))
        assertEquals(LexturesMotion.InstantMs, LexturesMotion.navigationDurationMs(true))
        assertEquals(0, LexturesMotion.navigationDurationMs(false, enabled = false))
        assertEquals(LexturesMotion.DeliberateMs, LexturesMotion.phaseDurationMs(false))
        assertEquals(LexturesMotion.InstantMs, LexturesMotion.phaseDurationMs(true))
    }

    /** AN.3: stagger delay caps so total choreography stays ≤ max delay + base enter. */
    @Test
    fun staggerRevealDelayCapsAtMaxItems() {
        assertEquals(0, LexturesMotion.staggerDelayMs(0))
        assertEquals(280, LexturesMotion.staggerDelayMs(7))
        assertEquals(280, LexturesMotion.staggerDelayMs(50))
        val maxDelay = LexturesMotion.staggerDelayMs(99)
        assertTrue(maxDelay + LexturesMotion.BaseMs <= 500)
    }

    /** AN.4: concurrent animation budget and kill-switch. */
    @Test
    fun listMotionShouldAnimateRespectsBudgetAndKillSwitch() {
        assertEquals(12, LIST_MOTION_MAX_CONCURRENT)
        assertEquals(1.03f, LIST_DRAG_LIFT_SCALE)
        assertTrue(shouldAnimateListItem(0, reduceMotion = false, enabled = true))
        assertFalse(shouldAnimateListItem(99, reduceMotion = false, enabled = true))
        assertFalse(shouldAnimateListItem(0, reduceMotion = false, enabled = false))
        assertTrue(shouldAnimateListItem(99, reduceMotion = true, enabled = true))
    }

    /** AN.5: sheet drag dismiss threshold. */
    @Test
    fun overlaySheetDismissThreshold() {
        assertEquals(0.28f, OVERLAY_SHEET_DISMISS_THRESHOLD)
        assertFalse(shouldDismissSheetDrag(100f, 400f))
        assertTrue(shouldDismissSheetDrag(120f, 400f))
        assertTrue(shouldDismissSheetDrag(10f, 400f, velocityPxPerMs = 0.9f))
        assertFalse(shouldDismissSheetDrag(10f, 0f))
    }

    /** AN.6: press/shake/indicator helpers and haptics mapping. */
    @Test
    fun controlMotionAndHapticsMapping() {
        assertTrue(shouldPressScale(reduceMotion = false, enabled = true))
        assertFalse(shouldPressScale(reduceMotion = true, enabled = true))
        assertFalse(shouldPressScale(reduceMotion = false, enabled = false))
        assertFalse(shouldValidationShake(reduceMotion = true, enabled = true))
        assertFalse(shouldSlideIndicator(reduceMotion = true, enabled = true))
        assertEquals(44f, indicatorOffsetPx(1, listOf(40f, 50f, 60f), gapPx = 4f))
        assertEquals("CONTEXT_CLICK", Haptics.systemName(Haptics.Kind.Tap))
        assertEquals("CLOCK_TICK", Haptics.systemName(Haptics.Kind.Selection))
        assertEquals("CONFIRM", Haptics.systemName(Haptics.Kind.Success))
        assertEquals("REJECT", Haptics.systemName(Haptics.Kind.Error))
        assertTrue(Haptics.shouldFire(enabled = true))
        assertFalse(Haptics.shouldFire(enabled = false))
        assertFalse(Haptics.shouldFire(enabled = true, systemHapticsOff = true))
    }
}
