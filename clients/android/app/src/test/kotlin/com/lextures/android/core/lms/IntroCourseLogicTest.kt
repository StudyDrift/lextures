package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.routing.CourseDeepLinkSection
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.core.routing.DeepLinkRouter
import com.lextures.android.core.routing.SettingsDeepLinkSection
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class IntroCourseLogicTest {
    @Test
    fun introCourseEnabledRespectsPlatformFlag() {
        assertFalse(IntroCourseLogic.introCourseEnabled(MobilePlatformFeatures()))
        assertTrue(IntroCourseLogic.introCourseEnabled(MobilePlatformFeatures(introCourseEnabled = true)))
    }

    @Test
    fun cardStateMachine() {
        assertEquals(IntroCourseCardState.Loading, IntroCourseLogic.cardState(null, loading = true, error = false))
        assertEquals(IntroCourseCardState.Error, IntroCourseLogic.cardState(null, loading = false, error = true))
        assertEquals(
            IntroCourseCardState.Hidden,
            IntroCourseLogic.cardState(IntroCourseProgress(enrolled = false), loading = false, error = false),
        )
        assertEquals(
            IntroCourseCardState.NotStarted,
            IntroCourseLogic.cardState(IntroCourseProgress(enrolled = true), loading = false, error = false),
        )
        assertEquals(
            IntroCourseCardState.InProgress,
            IntroCourseLogic.cardState(
                IntroCourseProgress(enrolled = true, modulesComplete = 2, percent = 28),
                loading = false,
                error = false,
            ),
        )
        assertEquals(
            IntroCourseCardState.Completed,
            IntroCourseLogic.cardState(
                IntroCourseProgress(
                    enrolled = true,
                    modulesComplete = 7,
                    percent = 100,
                    completedAt = "2026-01-01T00:00:00Z",
                ),
                loading = false,
                error = false,
            ),
        )
    }

    @Test
    fun deepLinkResolvesIntroModuleItem() {
        val destination = DeepLinkRouter.resolve(
            "/courses/C-WLCOME/modules/content/a0000000-0000-4000-8000-000000000099",
        )
        assertTrue(destination is DeepLinkDestination.Course)
        destination as DeepLinkDestination.Course
        assertEquals("C-WLCOME", destination.code)
        assertEquals(CourseDeepLinkSection.Modules, destination.section)
        assertEquals("a0000000-0000-4000-8000-000000000099", destination.itemId)
    }

    @Test
    fun deepLinkResolvesNotificationSettings() {
        val destination = DeepLinkRouter.resolve("/settings/notifications")
        assertTrue(destination is DeepLinkDestination.Settings)
        assertEquals(
            SettingsDeepLinkSection.Notifications,
            (destination as DeepLinkDestination.Settings).section,
        )
    }
}