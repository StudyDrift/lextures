package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/** Course grading agents list/edit helpers (M13.6). */
object CourseGradingAgentsLogic {
    data class GradableOption(
        val id: String,
        val label: String,
        val kind: String,
    )

    data class AgentDraft(
        val prompt: String = "",
        val includeAssignmentContent: Boolean = false,
        val includeRubric: Boolean = false,
        val status: String = AgentStatus.Draft.name.lowercase(),
        val autoGradeNew: Boolean = false,
        val postPolicy: String = "draft",
        val confidenceFloor: Double? = null,
        val workflowGraph: JsonElement? = null,
    )

    enum class AgentStatus {
        Draft,
        Accepted,
        Archived,
        ;

        val apiValue: String
            get() = name.lowercase()

        val labelRes: Int
            get() = when (this) {
                Draft -> com.lextures.android.R.string.mobile_courseSettings_gradingAgents_status_draft
                Accepted -> com.lextures.android.R.string.mobile_courseSettings_gradingAgents_status_accepted
                Archived -> com.lextures.android.R.string.mobile_courseSettings_gradingAgents_status_archived
            }

        companion object {
            fun fromApi(value: String): AgentStatus =
                entries.firstOrNull { it.apiValue == value } ?: Draft
        }
    }

    enum class ValidationError {
        PromptRequired,
    }

    private const val WORKFLOW_VERSION = 1

    fun cacheKeyAgents(courseCode: String): String = "course:$courseCode:grading-agents"

    fun saveIdempotencyKey(courseCode: String, itemId: String, itemKind: String): String =
        "course-grading-agents:$courseCode:$itemKind:$itemId:save"

    fun deleteIdempotencyKey(courseCode: String, itemId: String, itemKind: String): String =
        "course-grading-agents:$courseCode:$itemKind:$itemId:delete"

    fun graderAgentPath(courseCode: String, itemId: String, itemKind: String): String {
        val collection = if (itemKind == "quiz") "quizzes" else "assignments"
        return "/api/v1/courses/$courseCode/$collection/$itemId/grader-agent"
    }

    fun normalizedItemKind(itemKind: String?): String =
        if (itemKind == "quiz") "quiz" else "assignment"

    fun gradableOptions(
        structure: List<CourseStructureItem>,
        existingItemIds: Set<String>,
    ): List<GradableOption> {
        val byId = structure.associateBy { it.id }
        return structure
            .filter { item ->
                (item.kind == "assignment" || item.kind == "quiz") &&
                    item.archived != true &&
                    item.id !in existingItemIds
            }
            .map { item ->
                var moduleTitle = ""
                var parent = item.parentId?.let { byId[it] }
                val guardIds = mutableSetOf<String>()
                while (parent != null && parent.id !in guardIds) {
                    guardIds += parent.id
                    if (parent.kind == "module") {
                        moduleTitle = parent.title
                        break
                    }
                    parent = parent.parentId?.let { byId[it] }
                }
                val label = if (moduleTitle.isEmpty()) item.title else "$moduleTitle — ${item.title}"
                GradableOption(id = item.id, label = label, kind = item.kind)
            }
            .sortedBy { it.label.lowercase() }
    }

    fun draft(config: GraderAgentConfig?): AgentDraft =
        AgentDraft(
            prompt = config?.prompt.orEmpty(),
            includeAssignmentContent = config?.includeAssignmentContent == true,
            includeRubric = config?.includeRubric == true,
            status = config?.status ?: AgentStatus.Draft.apiValue,
            autoGradeNew = config?.autoGradeNew == true,
            postPolicy = config?.postPolicy ?: "draft",
            confidenceFloor = config?.confidenceFloor,
            workflowGraph = config?.workflowGraph,
        )

    fun draft(template: GraderAgentTemplateDetail): AgentDraft =
        AgentDraft(
            prompt = template.prompt,
            includeAssignmentContent = template.includeAssignmentContent,
            includeRubric = template.includeRubric,
            status = AgentStatus.Draft.apiValue,
            autoGradeNew = false,
            postPolicy = "draft",
            confidenceFloor = null,
            workflowGraph = template.workflowGraph,
        )

    fun isDirty(current: AgentDraft, baseline: AgentDraft): Boolean = current != baseline

    fun validateDraft(draft: AgentDraft): ValidationError? =
        if (draft.prompt.trim().isEmpty()) ValidationError.PromptRequired else null

    fun buildPutBody(current: AgentDraft, itemKind: String): PutGraderAgentConfigBody {
        val floor = current.confidenceFloor
        return PutGraderAgentConfigBody(
            prompt = current.prompt.trim(),
            includeAssignmentContent = current.includeAssignmentContent,
            includeRubric = current.includeRubric,
            status = current.status,
            autoGradeNew = current.autoGradeNew,
            postPolicy = current.postPolicy,
            confidenceFloor = floor?.takeIf { it > 0 },
            workflowGraph = effectiveWorkflowGraph(current.workflowGraph, itemKind),
        )
    }

    fun effectiveWorkflowGraph(stored: JsonElement?, itemKind: String): JsonElement {
        if (stored != null && hasWorkflowNodes(stored)) {
            return if (itemKind == "quiz") ensureQuizResponsesNode(stored) else stored
        }
        return defaultWorkflowGraph(itemKind)
    }

    fun filteredAgents(
        agents: List<CourseGradingAgentSummary>,
        query: String,
    ): List<CourseGradingAgentSummary> {
        val trimmed = query.trim()
        val sorted = agents.sortedBy { it.assignmentTitle.lowercase() }
        if (trimmed.isEmpty()) return sorted
        val needle = trimmed.lowercase()
        return sorted.filter { agent ->
            agent.assignmentTitle.lowercase().contains(needle) ||
                agent.status.lowercase().contains(needle)
        }
    }

    fun statusLabelRes(status: String): Int =
        AgentStatus.fromApi(status).labelRes

    fun kindLabelRes(itemKind: String?): Int? =
        if (normalizedItemKind(itemKind) == "quiz") {
            com.lextures.android.R.string.mobile_courseSettings_gradingAgents_quizBadge
        } else {
            null
        }

    private fun hasWorkflowNodes(graph: JsonElement): Boolean {
        val nodes = graph.jsonObject["nodes"]?.jsonArray ?: return false
        return nodes.isNotEmpty()
    }

    private fun ensureQuizResponsesNode(graph: JsonElement): JsonElement {
        val objectGraph = graph.jsonObject
        val nodes = objectGraph["nodes"]?.jsonArray ?: return defaultWorkflowGraph("quiz")
        val hasQuizResponses = nodes.any { node ->
            node.jsonObject["type"]?.jsonPrimitive?.content == "quizResponses"
        }
        if (hasQuizResponses) return graph
        val quizNode = buildJsonObject {
            put("id", JsonPrimitive("quizResponses"))
            put("type", JsonPrimitive("quizResponses"))
            put(
                "position",
                buildJsonObject {
                    put("x", JsonPrimitive(-420))
                    put("y", JsonPrimitive(0))
                },
            )
            put("data", JsonObject(emptyMap()))
        }
        return buildJsonObject {
            objectGraph.forEach { (key, value) ->
                if (key == "nodes") {
                    put(key, JsonArray(listOf(quizNode) + nodes))
                } else {
                    put(key, value)
                }
            }
        }
    }

    fun defaultWorkflowGraph(itemKind: String): JsonElement {
        val outputNode = buildJsonObject {
            put("id", JsonPrimitive("output"))
            put("type", JsonPrimitive("output"))
            put(
                "position",
                buildJsonObject {
                    put("x", JsonPrimitive(0))
                    put("y", JsonPrimitive(0))
                },
            )
            put("data", JsonObject(emptyMap()))
        }
        val nodes = if (itemKind == "quiz") {
            val quizNode = buildJsonObject {
                put("id", JsonPrimitive("quizResponses"))
                put("type", JsonPrimitive("quizResponses"))
                put(
                    "position",
                    buildJsonObject {
                        put("x", JsonPrimitive(-420))
                        put("y", JsonPrimitive(0))
                    },
                )
                put("data", JsonObject(emptyMap()))
            }
            buildJsonArray {
                add(quizNode)
                add(outputNode)
            }
        } else {
            buildJsonArray { add(outputNode) }
        }
        return buildJsonObject {
            put("version", JsonPrimitive(WORKFLOW_VERSION))
            put("nodes", nodes)
            put("edges", JsonArray(emptyList()))
        }
    }
}
