package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// ePortfolio (M12.1)

@Serializable
data class PortfolioSummary(
    val id: String,
    val title: String,
    val introText: String,
    val isPublic: Boolean,
    val publicSlug: String? = null,
    val order: List<String> = emptyList(),
    val createdAt: String,
    val updatedAt: String,
)

@Serializable
data class PortfolioArtifact(
    val id: String,
    val portfolioId: String,
    val artifactType: String,
    val title: String,
    val description: String,
    val sourceSubmissionId: String? = null,
    val sourceCourseId: String? = null,
    val fileName: String = "",
    val fileMime: String = "",
    val textContent: String = "",
    val externalUrl: String = "",
    val outcomeIds: List<String> = emptyList(),
    val isPublic: Boolean = false,
    val sortOrder: Int = 0,
    val createdAt: String,
    val updatedAt: String,
)

@Serializable
data class PortfolioDetailResponse(
    val portfolio: PortfolioSummary,
    val artifacts: List<PortfolioArtifact> = emptyList(),
)

@Serializable
data class PortfoliosListResponse(
    val portfolios: List<PortfolioSummary>? = null,
)

@Serializable
data class CreatePortfolioRequest(
    val title: String,
    val introText: String,
)

@Serializable
data class PatchPortfolioRequest(
    val title: String? = null,
    val introText: String? = null,
    val isPublic: Boolean? = null,
    val order: List<String>? = null,
)

@Serializable
data class CreateArtifactRequest(
    val artifactType: String,
    val title: String,
    val description: String? = null,
    val sourceSubmissionId: String? = null,
    val textContent: String? = null,
    val externalUrl: String? = null,
    val outcomeIds: List<String>? = null,
    val isPublic: Boolean? = null,
)

@Serializable
data class PatchArtifactRequest(
    val title: String? = null,
    val description: String? = null,
    val textContent: String? = null,
    val externalUrl: String? = null,
    val outcomeIds: List<String>? = null,
    val isPublic: Boolean? = null,
)