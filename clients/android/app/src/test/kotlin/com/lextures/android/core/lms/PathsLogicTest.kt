package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PathsLogicTest {
    @Test
    fun nextCourseReturnsFirstIncomplete() {
        val progress = PathProgress(
            pathId = "p1",
            pathTitle = "Path",
            courses = listOf(
                PathCourseProgress("c1", 1, "A", "A", completed = true, recommended = true),
                PathCourseProgress("c2", 2, "B", "B", completed = false, recommended = true),
                PathCourseProgress("c3", 3, "C", "C", completed = false, recommended = false),
            ),
        )
        assertEquals("c2", PathsLogic.nextCourse(progress)?.courseId)
    }

    @Test
    fun isLockedWhenNotRecommendedAndIncomplete() {
        val course = PathCourseProgress("c3", 3, "C", "C", completed = false, recommended = false)
        assertTrue(PathsLogic.isLocked(course))
    }

    @Test
    fun mergeRecommendationsSortsByScore() {
        val merged = PathsLogic.mergeRecommendations(
            listOf(
                LearnerRecommendationsResponse(
                    recommendations = listOf(
                        LearnerRecommendationItem("1", "quiz", "Low", "continue", "r", 1.0),
                    ),
                ),
                LearnerRecommendationsResponse(
                    recommendations = listOf(
                        LearnerRecommendationItem("2", "assignment", "High", "strengthen", "r", 9.0),
                    ),
                    degraded = true,
                ),
            ),
        )
        assertEquals("2", merged.primary?.itemId)
        assertEquals(1, merged.chips.size)
        assertTrue(merged.degraded)
    }

    @Test
    fun structureItemKindMapping() {
        assertEquals("quiz", PathsLogic.structureItemKind("quiz"))
        assertNull(PathsLogic.structureItemKind("unknown"))
    }
}