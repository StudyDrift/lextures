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
}
