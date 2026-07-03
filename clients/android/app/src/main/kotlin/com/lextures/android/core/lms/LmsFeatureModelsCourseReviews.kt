package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class ReviewEligibility(
    val eligible: Boolean = false,
    val progressPercent: Int = 0,
    val hasReview: Boolean = false,
    val canEdit: Boolean = false,
    val reviewId: String? = null,
)

@Serializable
data class SubmitCourseReviewRequest(
    val rating: Int,
    val reviewText: String? = null,
)

@Serializable
data class SubmittedCourseReview(
    val id: String,
    val courseId: String,
    val reviewerId: String,
    val rating: Int,
    val reviewText: String? = null,
    val reviewerDisplayName: String,
    val createdAt: String,
    val updatedAt: String,
)