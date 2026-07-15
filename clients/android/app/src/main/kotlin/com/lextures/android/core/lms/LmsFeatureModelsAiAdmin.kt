package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// MARK: - AI models, system prompts & reports (M14.7)

@Serializable
data class AiSettingsResponse(
    val imageModelId: String = "",
    val courseSetupModelId: String = "",
    val notebookFlashcardsModelId: String = "",
    val vibeActivityModelId: String = "",
    val graderAgentModelId: String = "",
    val openRouterApiKey: String? = null,
)

@Serializable
data class PutAiSettingsRequest(
    val imageModelId: String,
    val courseSetupModelId: String,
    val notebookFlashcardsModelId: String,
    val vibeActivityModelId: String,
    val graderAgentModelId: String,
    val openRouterApiKey: String? = null,
    val clearOpenRouterApiKey: Boolean? = null,
)

@Serializable
data class AiModelOption(
    val id: String,
    val name: String? = null,
    val contextLength: Long? = null,
    val inputPricePerMillionUsd: Double? = null,
    val outputPricePerMillionUsd: Double? = null,
    val modalitiesSummary: String? = null,
)

@Serializable
data class AiModelsListResponse(
    val configured: Boolean = false,
    val models: List<AiModelOption> = emptyList(),
)

@Serializable
data class SystemPromptItem(
    val key: String = "",
    val label: String = "",
    val content: String = "",
    val updatedAt: String? = null,
)

@Serializable
data class SystemPromptsListResponse(
    val prompts: List<SystemPromptItem> = emptyList(),
)

@Serializable
data class PutSystemPromptRequest(
    val content: String,
)

@Serializable
data class AiReportsPayload(
    val range: AiReportsDateRange = AiReportsDateRange(),
    val cost: AiCostReport = AiCostReport(),
    val byUser: List<AiUserUsageRow> = emptyList(),
    val byCourse: List<AiCourseUsageRow> = emptyList(),
)

@Serializable
data class AiReportsDateRange(
    val from: String = "",
    val to: String = "",
)

@Serializable
data class AiCostReport(
    val summary: AiCostSummary = AiCostSummary(),
    val byDay: List<AiDayCostBucket> = emptyList(),
    val byFeature: List<AiFeatureCostRow> = emptyList(),
)

@Serializable
data class AiCostSummary(
    val totalCostUsd: Double = 0.0,
    val totalCalls: Long = 0,
    val totalTokens: Long = 0,
)

@Serializable
data class AiDayCostBucket(
    val day: String = "",
    val costUsd: Double = 0.0,
    val calls: Long = 0,
    val tokens: Long = 0,
)

@Serializable
data class AiFeatureCostRow(
    val feature: String = "",
    val costUsd: Double = 0.0,
    val calls: Long = 0,
    val tokens: Long = 0,
)

@Serializable
data class AiUserUsageRow(
    val userId: String = "",
    val email: String = "",
    val displayName: String = "",
    val calls: Long = 0,
    val promptTokens: Long = 0,
    val completionTokens: Long = 0,
    val totalTokens: Long = 0,
    val costUsd: Double = 0.0,
)

@Serializable
data class AiCourseUsageRow(
    val courseId: String = "",
    val courseCode: String = "",
    val title: String = "",
    val calls: Long = 0,
    val totalTokens: Long = 0,
    val costUsd: Double = 0.0,
)
