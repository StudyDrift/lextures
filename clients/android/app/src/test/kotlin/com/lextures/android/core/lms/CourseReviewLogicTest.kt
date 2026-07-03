package com.lextures.android.core.lms

import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseReviewLogicTest {
    @Test
    fun shouldShowComposer() {
        val eligible = ReviewEligibility(eligible = true, progressPercent = 50, hasReview = false, canEdit = true)
        assertTrue(CourseReviewLogic.shouldShowComposer(eligible))
        val ineligible = ReviewEligibility(eligible = false, progressPercent = 5)
        assertFalse(CourseReviewLogic.shouldShowComposer(ineligible))
    }

    @Test
    fun validateRating() {
        assertNull(CourseReviewLogic.validateRating(5))
        assertNotNull(CourseReviewLogic.validateRating(0))
    }

    @Test
    fun validateReviewText() {
        assertNull(CourseReviewLogic.validateReviewText("Great course"))
        assertNotNull(CourseReviewLogic.validateReviewText("a".repeat(CourseReviewLogic.MAX_REVIEW_TEXT_LENGTH + 1)))
    }
}