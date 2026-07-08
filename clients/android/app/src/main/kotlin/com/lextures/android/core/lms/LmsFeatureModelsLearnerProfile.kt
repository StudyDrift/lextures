package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

/** Learner profile models (LP10). */
@Serializable
data class LearnerProfile(
    val status: String = "insufficient_data",
    val lastComputedAt: String? = null,
    val facets: List<LearnerProfileFacetSummary> = emptyList(),
)

@Serializable
data class LearnerProfileFacetSummary(
    val facetKey: String,
    val state: String,
    val summary: JsonObject = JsonObject(emptyMap()),
    val confidence: Double = 0.0,
    val computedVersion: Int = 0,
    val updatedAt: String = "",
)

@Serializable
data class LearnerProfileInsight(
    val insightKey: String,
    val label: String,
    val value: JsonObject = JsonObject(emptyMap()),
    val confidence: Double = 0.0,
    val salience: Double = 0.0,
    val evidence: List<LearnerProfileEvidenceRow>? = null,
)

@Serializable
data class LearnerProfileEvidenceRow(
    val sourceKind: String,
    val sourceTable: String,
    val observationCount: Int,
    val courseId: String? = null,
    val windowStart: String? = null,
    val windowEnd: String? = null,
    val contribution: Double? = null,
)

@Serializable
data class LearnerProfileResponse(
    val profile: LearnerProfile? = null,
)

@Serializable
data class LearnerProfileFacetDetailResponse(
    val facet: LearnerProfileFacetSummary? = null,
    val insights: List<LearnerProfileInsight>? = null,
)

@Serializable
data class LearnerProfileControlResponse(
    val status: String? = null,
)