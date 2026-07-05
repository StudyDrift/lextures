package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AdvisingLogicTest {
    @Test
    fun advisingEnabled() {
        val off = MobilePlatformFeatures()
        assertFalse(AdvisingLogic.advisingEnabled(off))
        val on = MobilePlatformFeatures(ffAdvisingIntegration = true, ffMobileAdvising = true)
        assertTrue(AdvisingLogic.advisingEnabled(on))
        val rolloutOff = MobilePlatformFeatures(ffAdvisingIntegration = true, ffMobileAdvising = false)
        assertFalse(AdvisingLogic.advisingEnabled(rolloutOff))
    }

    @Test
    fun visibleNotesFiltersStaffOnly() {
        val notes = listOf(
            AdvisingNote(
                id = "1",
                studentId = "s",
                advisorId = "a",
                content = "visible",
                visibleToStudent = true,
                createdAt = "2026-01-02T12:00:00Z",
            ),
            AdvisingNote(
                id = "2",
                studentId = "s",
                advisorId = "a",
                content = "hidden",
                visibleToStudent = false,
                createdAt = "2026-01-03T12:00:00Z",
            ),
        )
        assertEquals(1, AdvisingLogic.visibleNotes(notes).size)
        assertEquals("1", AdvisingLogic.sortedNotes(notes).first().id)
    }

    @Test
    fun appointmentUrlPrefersProgress() {
        val progress = DegreeProgress(appointmentUrl = "https://progress.example/book")
        val config = MyAdvisingConfig(appointmentUrl = "https://config.example/book")
        assertEquals("https://progress.example/book", AdvisingLogic.appointmentUrl(progress, config))
    }

    @Test
    fun canBookAppointmentRequiresOnline() {
        assertFalse(AdvisingLogic.canBookAppointment(false, "https://example.com"))
        assertTrue(AdvisingLogic.canBookAppointment(true, "https://example.com"))
        assertFalse(AdvisingLogic.canBookAppointment(true, null))
    }
}
