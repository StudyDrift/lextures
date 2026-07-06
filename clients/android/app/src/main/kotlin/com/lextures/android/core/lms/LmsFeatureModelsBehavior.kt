package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// region Behavior / PBIS (M10.3)

@Serializable
data class BehaviorCategory(
    val id: String,
    val orgId: String,
    val name: String,
    val type: String,
    val color: String? = null,
    val active: Boolean = true,
) {
    val isPositive: Boolean get() = type.equals("positive", ignoreCase = true)
    val isNegative: Boolean get() = type.equals("negative", ignoreCase = true)
}

@Serializable
data class BehaviorCategoriesResponse(
    val categories: List<BehaviorCategory> = emptyList(),
)

@Serializable
data class PbisAwardInput(
    val studentId: String,
    val categoryId: String,
    val points: Int = 1,
    val note: String? = null,
)

@Serializable
data class PbisAwardsBody(
    val awards: List<PbisAwardInput>,
)

@Serializable
data class PbisAward(
    val id: String,
    val studentId: String,
    val awardedBy: String? = null,
    val categoryId: String,
    val categoryName: String? = null,
    val orgId: String? = null,
    val points: Int,
    val note: String? = null,
    val awardedAt: String,
)

@Serializable
data class PbisAwardsResponse(
    val saved: Int? = null,
    val awards: List<PbisAward> = emptyList(),
    val message: String? = null,
)

@Serializable
data class BehaviorReferralBody(
    val studentId: String,
    val categoryId: String,
    val schoolId: String? = null,
    val incidentAt: String? = null,
    val location: String? = null,
    val description: String,
    val response: String? = null,
)

@Serializable
data class BehaviorReferral(
    val id: String,
    val studentId: String,
    val filedBy: String? = null,
    val orgId: String? = null,
    val schoolId: String? = null,
    val categoryId: String,
    val categoryName: String? = null,
    val incidentAt: String,
    val location: String? = null,
    val description: String? = null,
    val response: String? = null,
    val createdAt: String,
)

@Serializable
data class StudentBehaviorResponse(
    val studentId: String,
    val totalPoints: Int,
    val awards: List<PbisAward> = emptyList(),
    val referrals: List<BehaviorReferral> = emptyList(),
)

@Serializable
data class HallPass(
    val id: String,
    val sectionId: String,
    val studentId: String? = null,
    val destination: String,
    val status: String,
    val estimatedMins: Int? = null,
    val requestedAt: String,
    val approvedAt: String? = null,
    val returnedAt: String? = null,
    val approvedBy: String? = null,
    val overdue: Boolean? = null,
)

@Serializable
data class HallPassResponse(
    val pass: HallPass? = null,
)

@Serializable
data class ActiveHallPassesResponse(
    val passes: List<HallPass> = emptyList(),
)

@Serializable
data class RequestHallPassBody(
    val destination: String,
    val estimatedMins: Int,
)

@Serializable
data class UpdateHallPassBody(
    val status: String,
)

// endregion
