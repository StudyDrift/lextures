package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class CourseGradingAgentSummary(
    val id: String,
    val itemId: String,
    val itemKind: String? = null,
    val assignmentTitle: String,
    val assignmentArchived: Boolean = false,
    val status: String,
    val autoGradeNew: Boolean = false,
    val hasWorkflowGraph: Boolean = false,
    val updatedAt: String,
    val reviewCount: Int? = null,
)

@Serializable
data class CourseGradingAgentsListResponse(
    val agents: List<CourseGradingAgentSummary> = emptyList(),
)

@Serializable
data class GraderAgentTemplateSummary(
    val id: String,
    val name: String,
    val isBuiltin: Boolean? = null,
    val updatedAt: String,
)

@Serializable
data class GraderAgentTemplatesListResponse(
    val templates: List<GraderAgentTemplateSummary> = emptyList(),
)

@Serializable
data class GraderAgentConfig(
    val id: String? = null,
    val prompt: String = "",
    val includeAssignmentContent: Boolean = false,
    val includeRubric: Boolean = false,
    val status: String = "draft",
    val autoGradeNew: Boolean? = null,
    val postPolicy: String? = null,
    val confidenceFloor: Double? = null,
    val modelId: String? = null,
    val updatedAt: String? = null,
    val workflowGraph: JsonElement? = null,
)

@Serializable
data class GraderAgentConfigEnvelope(
    val config: GraderAgentConfig? = null,
)

@Serializable
data class GraderAgentTemplateDetail(
    val id: String,
    val name: String,
    val prompt: String,
    val includeAssignmentContent: Boolean,
    val includeRubric: Boolean,
    val workflowGraph: JsonElement? = null,
    val createdAt: String,
    val updatedAt: String,
)

@Serializable
data class GraderAgentTemplateDetailEnvelope(
    val template: GraderAgentTemplateDetail,
)

@Serializable
data class PutGraderAgentConfigBody(
    val prompt: String,
    val includeAssignmentContent: Boolean,
    val includeRubric: Boolean,
    val status: String,
    val autoGradeNew: Boolean,
    val postPolicy: String,
    val confidenceFloor: Double? = null,
    val workflowGraph: JsonElement,
)

@Serializable
data class PutGraderAgentConfigResponse(
    val config: GraderAgentConfig,
)
