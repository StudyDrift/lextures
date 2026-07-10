package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

/** Authenticated in-app course marketplace models (plan MKT6). */

@Serializable
data class MarketplaceCourse(
    val id: String,
    val slug: String,
    val courseCode: String,
    val title: String,
    val description: String = "",
    val heroImageUrl: String? = null,
    val category: String? = null,
    val level: String? = null,
    val language: String = "en",
    val priceCents: Int = 0,
    val priceCurrency: String = "usd",
    val listPriceCents: Int? = null,
    val enrollmentCount: Int = 0,
    val averageRating: Double? = null,
    val ratingCount: Int = 0,
    val instructorName: String? = null,
    val createdAt: String = "",
    val owned: Boolean = false,
)

@Serializable
data class MarketplaceSearchResponse(
    val courses: List<MarketplaceCourse> = emptyList(),
    val total: Int = 0,
    val nextCursor: String = "",
)

@Serializable
data class MarketplaceCategory(
    val category: String,
    val count: Int = 0,
)

@Serializable
data class MarketplaceCategoriesResponse(
    val categories: List<MarketplaceCategory> = emptyList(),
)

@Serializable
data class MarketplaceWhatsIncluded(
    val moduleCount: Int = 0,
    val itemCount: Int = 0,
    val estimatedDurationMinutes: Int? = null,
)

@Serializable
data class MarketplaceRating(
    val average: Double? = null,
    val count: Int = 0,
)

@Serializable
data class MarketplaceCourseDetail(
    val course: MarketplaceCourse,
    val owned: Boolean = false,
    val priceCents: Int = 0,
    val priceCurrency: String = "usd",
    val listPriceCents: Int? = null,
    val whatsIncluded: MarketplaceWhatsIncluded = MarketplaceWhatsIncluded(),
    val rating: MarketplaceRating = MarketplaceRating(),
)

@Serializable
data class MarketplaceClaimResult(
    val enrolled: Boolean = false,
    val entitlementId: String = "",
    val alreadyOwned: Boolean = false,
    val firstItemId: String? = null,
    val courseCode: String = "",
)

@Serializable
data class CourseCatalogListing(
    val isPublic: Boolean = false,
    val category: String? = null,
    val difficultyLevel: String? = null,
    val language: String = "en",
    val priceCents: Int = 0,
    val priceCurrency: String = "usd",
    val slug: String = "",
    val marketplaceListed: Boolean = false,
    val publishState: String = "draft",
    val activePurchaseCount: Int = 0,
)

@Serializable
data class CourseCatalogListingResponse(
    val listing: CourseCatalogListing,
)

@Serializable
data class CourseCatalogListingPutBody(
    val isPublic: Boolean = false,
    val category: String? = null,
    val difficultyLevel: String? = null,
    val language: String = "en",
    val priceCents: Int = 0,
    val priceCurrency: String = "usd",
    val slug: String = "",
    val marketplaceListed: Boolean? = null,
)
