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
    fun resolvesParentConferencesPath() {
        val destination = DeepLinkRouter.resolve("/parent/conferences")
        assertTrue(destination is DeepLinkDestination.Parent)
        assertEquals(ParentDeepLinkSection.Conferences, (destination as DeepLinkDestination.Parent).section)
    }
}
