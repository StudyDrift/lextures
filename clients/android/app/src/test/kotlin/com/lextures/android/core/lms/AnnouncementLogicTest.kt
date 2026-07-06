package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AnnouncementLogicTest {
    private fun course(staff: Boolean, feed: Boolean = true) = CourseSummary(
        id = "c1",
        courseCode = "BIO101",
        title = "Biology",
        orgId = "org-1",
        viewerEnrollmentRoles = if (staff) listOf("teacher") else listOf("student"),
        feedEnabled = feed,
    )

    @Test
    fun canComposeCourseAnnouncementRequiresStaffAndFeed() {
        assertTrue(AnnouncementLogic.canComposeCourseAnnouncement(course(staff = true)))
        assertFalse(AnnouncementLogic.canComposeCourseAnnouncement(course(staff = false)))
        assertFalse(AnnouncementLogic.canComposeCourseAnnouncement(course(staff = true, feed = false)))
    }

    @Test
    fun canComposeBroadcastRequiresFeatureAndPermission() {
        val off = MobilePlatformFeatures()
        assertFalse(AnnouncementLogic.canComposeBroadcast(emptyList(), off))

        val on = MobilePlatformFeatures(ffBroadcasts = true)
        assertFalse(AnnouncementLogic.canComposeBroadcast(emptyList(), on))
        assertTrue(
            AnnouncementLogic.canComposeBroadcast(
                listOf(AnnouncementLogic.ORG_BROADCAST_MANAGE_PERMISSION),
                on,
            ),
        )
        assertTrue(
            AnnouncementLogic.canComposeBroadcast(
                listOf(AnnouncementLogic.GLOBAL_ADMIN_PERMISSION),
                on,
            ),
        )
    }

    @Test
    fun announcementsChannelIdFindsAnnouncementsChannel() {
        val channels = listOf(
            FeedChannel(id = "1", name = "general"),
            FeedChannel(id = "2", name = "Announcements"),
        )
        assertEquals("2", AnnouncementLogic.announcementsChannelId(channels))
    }

    @Test
    fun formatAnnouncementBodyIncludesTitleSectionAndEveryone() {
        val body = AnnouncementLogic.formatAnnouncementBody(
            title = "Snow day",
            body = "No class tomorrow.",
            sectionName = "Period 2",
            mentionsEveryone = true,
        )
        assertTrue(body.contains("**Snow day**"))
        assertTrue(body.contains("No class tomorrow."))
        assertTrue(body.contains("Period 2"))
        assertTrue(body.contains("@everyone"))
    }

    @Test
    fun audienceLabelWholeCourseAndSection() {
        val course = course(staff = true)
        assertEquals("Biology", AnnouncementLogic.audienceLabel(course, AnnouncementAudience.WholeCourse, null))
        assertEquals(
            "Biology · Period 2",
            AnnouncementLogic.audienceLabel(course, AnnouncementAudience.Section, "Period 2"),
        )
    }
}