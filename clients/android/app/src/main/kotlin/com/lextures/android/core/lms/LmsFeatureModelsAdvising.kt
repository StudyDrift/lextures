package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class AdvisingNote(
    val id: String,
    val studentId: String,
    val advisorId: String,
    val content: String,
    val visibleToStudent: Boolean,
    val createdAt: String,
    val advisorEmail: String? = null,
    val advisorDisplayName: String? = null,
)

@Serializable
data class AdvisingNotesResponse(
    val notes: List<AdvisingNote>? = null,
)

@Serializable
data class AdvisingRequirementGroup(
    val group: String,
    val coursesRemaining: Int,
)

@Serializable
data class DegreeProgress(
    val configured: Boolean = false,
    val completionPercent: Int? = null,
    val remainingRequiredCount: Int? = null,
    val remainingRequirements: List<AdvisingRequirementGroup>? = null,
    val atRisk: Boolean? = null,
    val lastUpdated: String? = null,
    val stale: Boolean? = null,
    val appointmentUrl: String? = null,
    val recentNotesCount: Int? = null,
)

@Serializable
data class MyAdvisingConfig(
    val appointmentUrl: String? = null,
)

data class AdvisingAdvisorInfo(
    val displayName: String,
    val email: String? = null,
)
