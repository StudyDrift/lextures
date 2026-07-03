package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class PublicCatalogCourse(
    val id: String,
    val slug: String,
    val courseCode: String,
    val title: String,
    val description: String = "",
    val heroImageUrl: String? = null,
    val category: String? = null,
    val difficultyLevel: String? = null,
    val language: String = "en",
    val priceCents: Int = 0,
    val enrollmentCount: Int = 0,
    val averageRating: Double? = null,
    val ratingCount: Int? = null,
    val instructorName: String? = null,
    val createdAt: String = "",
)

@Serializable
data class PublicCatalogSearchResponse(
    val courses: List<PublicCatalogCourse> = emptyList(),
    val total: Int = 0,
    val nextCursor: String = "",
)

@Serializable
data class CatalogCategory(
    val category: String,
    val count: Int = 0,
)

@Serializable
data class CatalogCategoriesResponse(
    val categories: List<CatalogCategory> = emptyList(),
)

@Serializable
data class PublicCatalogCourseDetailResponse(
    val course: PublicCatalogCourse,
)

@Serializable
data class CourseReviewSummary(
    val averageRating: Double? = null,
    val ratingCount: Int = 0,
)

@Serializable
data class CourseReview(
    val id: String,
    val rating: Int,
    val reviewText: String? = null,
    val reviewerDisplayName: String,
    val createdAt: String,
)

@Serializable
data class CourseReviewsListResponse(
    val summary: CourseReviewSummary,
    val reviews: List<CourseReview> = emptyList(),
    val nextCursor: String? = null,
)

@Serializable
data class CourseSelfEnrollResponse(
    val enrolled: Boolean = false,
    val enrollmentId: String,
    val firstItemId: String? = null,
)