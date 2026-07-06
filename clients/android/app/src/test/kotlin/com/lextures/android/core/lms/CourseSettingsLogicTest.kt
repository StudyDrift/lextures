package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseSettingsLogicTest {
    @Test
    fun courseItemCreatePermission() {
        assertEquals("course:C-ABC:item:create", CourseSettingsLogic.courseItemCreatePermission("C-ABC"))
    }

    @Test
    fun canManageCourse() {
        val perms = listOf("course:C-1:item:create")
        assertTrue(CourseSettingsLogic.canManageCourse("C-1", perms))
        assertFalse(CourseSettingsLogic.canManageCourse("C-2", perms))
    }

    @Test
    fun validateTitleRequired() {
        val error = CourseSettingsLogic.validateGeneralForm("", CourseSettingsLogic.CourseHomeLanding.data, "")
        assertNotNull(error?.title)
    }

    @Test
    fun validateContentPageRequired() {
        val error = CourseSettingsLogic.validateGeneralForm(
            "Course",
            CourseSettingsLogic.CourseHomeLanding.content_page,
            "",
        )
        assertNotNull(error?.courseHome)
    }

    @Test
    fun isoDurationRoundTrip() {
        val parts = CourseSettingsLogic.isoDurationToParts("P3M")
        assertEquals("3", parts.first)
        assertEquals(CourseSettingsLogic.RelativeDurationUnit.M, parts.second)
        assertEquals("P3M", CourseSettingsLogic.partsToIsoDuration("3", CourseSettingsLogic.RelativeDurationUnit.M))
    }

    @Test
    fun heroPositionCenterIsNil() {
        assertEquals(null, CourseSettingsLogic.formatHeroObjectPosition(50.0, 50.0))
        assertEquals("30% 70%", CourseSettingsLogic.formatHeroObjectPosition(30.0, 70.0))
    }

    @Test
    fun shouldShowWorkspaceSectionRequiresFlagAndPermission() {
        val course = CourseSummary(id = "1", courseCode = "C-1", title = "T", viewerEnrollmentRoles = listOf("teacher"))
        val featuresOff = MobilePlatformFeatures(ffMobileCourseSettings = false)
        val featuresOn = MobilePlatformFeatures(ffMobileCourseSettings = true)
        assertFalse(
            CourseSettingsLogic.shouldShowWorkspaceSection(course, listOf("course:C-1:item:create"), featuresOff),
        )
        assertTrue(
            CourseSettingsLogic.shouldShowWorkspaceSection(course, listOf("course:C-1:item:create"), featuresOn),
        )
    }
}
