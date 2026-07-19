package com.lextures.android.core.routing

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DeepLinkRouterTest {
    @Test
    fun resolvesInboxPath() {
        val destination = DeepLinkRouter.resolve("/inbox")
        assertTrue(destination is DeepLinkDestination.Inbox)
    }

    @Test
    fun resolvesCourseGradesPath() {
        val destination = DeepLinkRouter.resolve("/courses/cs101/grades")
        assertTrue(destination is DeepLinkDestination.Course)
        destination as DeepLinkDestination.Course
        assertEquals("cs101", destination.code)
        assertEquals(CourseDeepLinkSection.Grades, destination.section)
    }

    @Test
    fun resolvesHttpsUrl() {
        val destination = DeepLinkRouter.resolve("https://lextures.com/courses/bio200/assignments/a1")
        assertTrue(destination is DeepLinkDestination.Course)
        destination as DeepLinkDestination.Course
        assertEquals("bio200", destination.code)
        assertEquals("a1", destination.itemId)
    }

    @Test
    fun resolvesReviewPath() {
        val destination = DeepLinkRouter.resolve("/review")
        assertTrue(destination is DeepLinkDestination.Review)
    }

    @Test
    fun resolvesStudyInsightsPath() {
        val destination = DeepLinkRouter.resolve("/me/study-insights")
        assertTrue(destination is DeepLinkDestination.Insights)
    }

    @Test
    fun resolvesCheckoutSuccessWithCourseId() {
        val destination = DeepLinkRouter.resolve("/checkout/success?course_id=abc-123")
        assertTrue(destination is DeepLinkDestination.CheckoutSuccess)
        assertEquals("abc-123", (destination as DeepLinkDestination.CheckoutSuccess).courseId)
    }

    @Test
    fun resolvesCheckoutCancelAndBilling() {
        assertTrue(DeepLinkRouter.resolve("/checkout/cancel") is DeepLinkDestination.CheckoutCancel)
        assertTrue(DeepLinkRouter.resolve("/me/billing") is DeepLinkDestination.Billing)
    }

    @Test
    fun resolvesCredentialsPath() {
        assertTrue(DeepLinkRouter.resolve("/me/credentials") is DeepLinkDestination.Credentials)
    }

    @Test
    fun resolvesParentDashboardPath() {
        val destination = DeepLinkRouter.resolve("/parent?student=child-1")
        assertTrue(destination is DeepLinkDestination.Parent)
        destination as DeepLinkDestination.Parent
        assertEquals("child-1", destination.studentId)
        assertEquals(ParentDeepLinkSection.Dashboard, destination.section)
    }

    @Test
    fun resolvesIntroCourseContentItemPath() {
        val destination = DeepLinkRouter.resolve(
            "/courses/C-WLCOME/modules/quiz/a0000000-0000-4000-8000-000000000099",
        )
        assertTrue(destination is DeepLinkDestination.Course)
        destination as DeepLinkDestination.Course
        assertEquals("a0000000-0000-4000-8000-000000000099", destination.itemId)
    }

    @Test
    fun resolvesSettingsNotificationsPath() {
        val destination = DeepLinkRouter.resolve("/settings/notifications")
        assertTrue(destination is DeepLinkDestination.Settings)
        assertEquals(
            com.lextures.android.core.routing.SettingsDeepLinkSection.Notifications,
            (destination as DeepLinkDestination.Settings).section,
        )
    }

    @Test
    fun resolvesParentConferencesPath() {
        val destination = DeepLinkRouter.resolve("/parent/conferences")
        assertTrue(destination is DeepLinkDestination.Parent)
        assertEquals(ParentDeepLinkSection.Conferences, (destination as DeepLinkDestination.Parent).section)
    }

    @Test
    fun resolvesCourseBoardsPath() {
        val list = DeepLinkRouter.resolve("/courses/cs101/boards")
        assertTrue(list is DeepLinkDestination.Course)
        list as DeepLinkDestination.Course
        assertEquals("cs101", list.code)
        assertEquals(CourseDeepLinkSection.Boards, list.section)
        assertEquals(null, list.itemId)

        val detail = DeepLinkRouter.resolve("/courses/cs101/boards/board-1")
        assertTrue(detail is DeepLinkDestination.Course)
        detail as DeepLinkDestination.Course
        assertEquals(CourseDeepLinkSection.Boards, detail.section)
        assertEquals("board-1", detail.itemId)
    }

    @Test
    fun resolvesBoardLinkPath() {
        val destination = DeepLinkRouter.resolve("https://lextures.com/board-links/tok-abc")
        assertTrue(destination is DeepLinkDestination.BoardLink)
        assertEquals("tok-abc", (destination as DeepLinkDestination.BoardLink).token)
        assertTrue(DeepLinkRouter.resolve("/board-links/") is DeepLinkDestination.Home)
    }

    @Test
    fun resolvesLiveQuizPlayPath() {
        val withCode = DeepLinkRouter.resolve("https://lextures.com/play/AB12")
        assertTrue(withCode is DeepLinkDestination.LiveQuizPlay)
        assertEquals("AB12", (withCode as DeepLinkDestination.LiveQuizPlay).code)

        val blank = DeepLinkRouter.resolve("/play")
        assertTrue(blank is DeepLinkDestination.LiveQuizPlay)
        assertEquals(null, (blank as DeepLinkDestination.LiveQuizPlay).code)

        val course = DeepLinkRouter.resolve("/courses/demo/live-quizzes")
        assertTrue(course is DeepLinkDestination.Course)
        assertEquals(
            CourseDeepLinkSection.LiveQuizzes,
            (course as DeepLinkDestination.Course).section,
        )
    }
}
