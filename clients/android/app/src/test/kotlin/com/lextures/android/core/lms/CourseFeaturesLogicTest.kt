package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseFeaturesLogicTest {
    @Test
    fun isEnabledDefaults() {
        val course = sampleCourse()
        assertTrue(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.notebook, course))
        assertTrue(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.feed, course))
        assertTrue(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.calendar, course))
        assertFalse(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.discussions, course))
    }

    @Test
    fun applyToggleUpdatesCourse() {
        val course = sampleCourse()
        val updated = CourseFeaturesLogic.applyToggle(course, CourseFeaturesLogic.Tool.discussions, true)
        assertTrue(CourseFeaturesLogic.isEnabled(CourseFeaturesLogic.Tool.discussions, updated))
    }

    @Test
    fun buildFeaturesPatchReflectsCourse() {
        val course = sampleCourse().copy(discussionsEnabled = true, sectionsEnabled = true)
        val patch = CourseFeaturesLogic.buildFeaturesPatch(course)
        assertTrue(patch.discussionsEnabled)
        assertTrue(patch.sectionsEnabled)
        assertTrue(patch.notebookEnabled)
    }

    @Test
    fun shouldConfirmDisableOnlyWhenEnabled() {
        assertTrue(CourseFeaturesLogic.shouldConfirmDisable(currentlyEnabled = true))
        assertFalse(CourseFeaturesLogic.shouldConfirmDisable(currentlyEnabled = false))
    }

    @Test
    fun consortiumGating() {
        assertFalse(CourseFeaturesLogic.consortiumSectionEnabled(MobilePlatformFeatures(ffConsortiumSharing = false)))
        assertTrue(CourseFeaturesLogic.consortiumSectionEnabled(MobilePlatformFeatures(ffConsortiumSharing = true)))
    }

    @Test
    fun videoCaptionsGating() {
        assertFalse(CourseFeaturesLogic.videoCaptionsSectionEnabled(MobilePlatformFeatures(videoCaptionsEnabled = false)))
        assertTrue(CourseFeaturesLogic.videoCaptionsSectionEnabled(MobilePlatformFeatures(videoCaptionsEnabled = true)))
    }

    @Test
    fun filterToolsEmptyQueryReturnsAll() {
        assertEquals(
            CourseFeaturesLogic.allToolRows.size,
            CourseFeaturesLogic.filterTools(CourseFeaturesLogic.allToolRows, "").size,
        )
    }

    private fun sampleCourse() = CourseSummary(
        id = "1",
        courseCode = "C-1",
        title = "Intro",
        notebookEnabled = true,
        calendarEnabled = true,
        feedEnabled = true,
        discussionsEnabled = false,
    )
}
