package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

object CourseReviewLogic {
    const val MAX_REVIEW_TEXT_LENGTH = 2000

    fun reviewsEnabled(features: MobilePlatformFeatures): Boolean = features.ffCourseReviews

    fun starLabel(rating: Int): String = when (rating) {
        1 -> "1 star – Poor"
        2 -> "2 stars – Fair"
        3 -> "3 stars – Average"
        4 -> "4 stars – Good"
        5 -> "5 stars – Excellent"
        else -> "Select a rating"
    }

    fun shouldShowComposer(eligibility: ReviewEligibility): Boolean =
        eligibility.eligible && (eligibility.canEdit || !eligibility.hasReview)

    fun validateRating(rating: Int): String? =
        if (rating !in 1..5) "Please select a star rating." else null

    fun validateReviewText(text: String): String? =
        if (text.length > MAX_REVIEW_TEXT_LENGTH) "Review must be $MAX_REVIEW_TEXT_LENGTH characters or fewer." else null
}