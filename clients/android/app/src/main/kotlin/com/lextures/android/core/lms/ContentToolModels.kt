package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

/** CT.M3 — Content Tool instance / state shapes mirroring server contenttools models. */

@Serializable
data class ToolScore(
    val raw: Double,
    val max: Double,
)

@Serializable
data class ToolStateEnvelope(
    val instanceId: String,
    val revision: Long = 0,
    val status: String = "not_started",
    val state: JsonElement = JsonObject(emptyMap()),
    val stateJson: JsonElement? = null,
    val score: ToolScore? = null,
    val updatedAt: String? = null,
    val resetCount: Int = 0,
    val lastResetAt: String? = null,
    val scope: String? = null,
    val stateSchemaVersion: Int = 0,
    val quarantined: Boolean = false,
) {
    fun document(): JsonElement = stateJson ?: state
}

@Serializable
data class ToolInstance(
    val id: String,
    val toolId: String,
    val toolVersion: String = "1.0.0",
    val hostKind: String = "content_page",
    val structureItemId: String? = null,
    val sectionKey: String? = null,
    val title: String? = null,
    val config: JsonElement = JsonObject(emptyMap()),
    val status: String = "published",
    val updatedAt: String = "",
    val state: ToolStateEnvelope? = null,
    val sandboxMode: String? = null,
    val contract: Int = 0,
    val breakerOpen: Boolean = false,
    val deprecated: Boolean = false,
    val sunsetAt: String? = null,
    val capabilities: List<String> = emptyList(),
    val tombstone: Boolean = false,
)

@Serializable
data class ToolInstancesListResponse(
    val instances: List<ToolInstance> = emptyList(),
)

@Serializable
data class ContentToolSettings(
    val allowedToolIds: List<String> = emptyList(),
    val studentResetAllowed: Boolean = false,
    val maxInstancesPerItem: Int = 0,
    val monthlyAiTokenBudget: Long = 0,
    val dailyAiCallsPerUser: Int = 0,
    val linkIngestionMode: String = "off",
    val linkHostAllowlist: List<String> = emptyList(),
    val gradeLinksAllowed: Boolean = false,
    val updatedAt: String? = null,
)

@Serializable
data class SaveToolStateBody(
    val revision: Long,
    val state: JsonElement,
    val stateJson: JsonElement? = null,
)

@Serializable
data class RunToolActionBody(
    val input: JsonElement = JsonObject(emptyMap()),
    val idempotencyKey: String? = null,
)

@Serializable
data class RunToolActionResponse(
    val result: JsonElement? = null,
    val state: ToolStateEnvelope? = null,
)

@Serializable
data class RevisionConflictBody(
    val error: String = "revision_conflict",
    val current: ToolStateEnvelope,
)

@Serializable
data class StateTooLargeBody(
    val error: String = "state_too_large",
    val maxBytes: Long = 0,
)

fun emptyToolState(instanceId: String): ToolStateEnvelope = ToolStateEnvelope(
    instanceId = instanceId,
    revision = 0,
    status = "not_started",
    state = buildJsonObject { },
)

fun jsonObjectOf(vararg pairs: Pair<String, String>): JsonObject = buildJsonObject {
    for ((k, v) in pairs) put(k, JsonPrimitive(v))
}

/** CT.M6 / CT.8 — AI consent response for content-tool composers. */
@Serializable
data class ContentToolAIConsent(
    val aiDisclosureMode: String = "acknowledge",
    val decision: String? = null,
    val decidedAt: String? = null,
)

@Serializable
data class ContentToolAIConsentBody(
    val toolId: String? = null,
    val decision: String,
)
