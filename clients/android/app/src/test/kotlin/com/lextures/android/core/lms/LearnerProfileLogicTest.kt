package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LearnerProfileLogicTest {
    @Test
    fun learnerProfileEnabledRequiresBothFlags() {
        val off = MobilePlatformFeatures()
        assertFalse(LearnerProfileLogic.learnerProfileEnabled(off))
        val on = MobilePlatformFeatures(learnerProfileEnabled = true)
        assertTrue(LearnerProfileLogic.learnerProfileEnabled(on))
        val mobileOff = MobilePlatformFeatures(learnerProfileEnabled = true, ffMobileLearnerProfile = false)
        assertFalse(LearnerProfileLogic.learnerProfileEnabled(mobileOff))
    }

    @Test
    fun sortFacetsUsesStablePriority() {
        val facets = listOf(
            LearnerProfileFacetSummary(facetKey = "interests", state = "ok", updatedAt = "2026-01-01T00:00:00Z"),
            LearnerProfileFacetSummary(facetKey = "study_rhythm", state = "ok", updatedAt = "2026-01-01T00:00:00Z"),
            LearnerProfileFacetSummary(facetKey = "learning_approach", state = "insufficient_data", updatedAt = "2026-01-01T00:00:00Z"),
        )
        val sorted = LearnerProfileLogic.sortFacets(facets).map { it.facetKey }
        assertEquals(listOf("study_rhythm", "interests", "learning_approach"), sorted)
    }

    @Test
    fun showEmptyStateWhenInsufficient() {
        val profile = LearnerProfile(
            status = "insufficient_data",
            facets = listOf(
                LearnerProfileFacetSummary(facetKey = "study_rhythm", state = "insufficient_data"),
            ),
        )
        assertTrue(LearnerProfileLogic.showEmptyState(profile))
    }

    @Test
    fun evidenceAggregation() {
        val evidence = listOf(
            LearnerProfileEvidenceRow(
                sourceKind = "quiz_attempt",
                sourceTable = "course.quiz_attempts",
                observationCount = 5,
                courseId = "a",
            ),
            LearnerProfileEvidenceRow(
                sourceKind = "quiz_attempt",
                sourceTable = "course.quiz_attempts",
                observationCount = 7,
                courseId = "b",
            ),
        )
        assertEquals(12, LearnerProfileLogic.totalObservationCount(evidence))
        assertEquals(2, LearnerProfileLogic.uniqueCourseCount(evidence))
    }
}