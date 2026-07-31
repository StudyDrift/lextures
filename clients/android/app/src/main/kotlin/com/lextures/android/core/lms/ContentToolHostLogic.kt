package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import java.util.UUID

/**
 * Pure CT.M3 host decisions — fence mapping, debounce, conflict, read-only reasons,
 * contract gating, and render gates. No networking.
 */
object ContentToolHostLogic {
    const val DEFAULT_DEBOUNCE_MS = 1500
    const val MIN_DEBOUNCE_MS = 500
    const val MAX_DEBOUNCE_MS = 10_000
    const val RUNTIME_CONTRACT_VERSION = 1

    enum class ConflictPolicy {
        SERVER_WINS,
        CLIENT_WINS,
        MERGE,
        ;

        companion object {
            fun from(raw: String?): ConflictPolicy = when (raw?.lowercase()) {
                "client_wins" -> CLIENT_WINS
                "merge" -> MERGE
                else -> SERVER_WINS
            }
        }
    }

    enum class ReadOnlyReason {
        TOMBSTONE,
        BREAKER,
        ARCHIVED,
        PAST_DUE,
        OBSERVER,
        QUARANTINED,
        FLAG_OFF,
    }

    enum class SyncStatus {
        IDLE,
        SAVING,
        SAVED,
        UNSYNCED,
        ERROR,
    }

    enum class FenceRenderMode {
        /** Client dark-launch flag off — keep CT.M1 placeholder. */
        LEGACY_PLACEHOLDER,
        /** Course contentToolsEnabled false — render nothing (parity with web). */
        HIDDEN,
        /** Mount the CT.M3 host. */
        HOST,
    }

    data class ReadOnlyInput(
        val tombstone: Boolean = false,
        val breakerOpen: Boolean = false,
        val status: String = "published",
        val pastDue: Boolean = false,
        val respectsDueDate: Boolean = false,
        val observer: Boolean = false,
        val quarantined: Boolean = false,
        val courseFlagOffAfterLoad: Boolean = false,
    )

    fun clampDebounceMs(ms: Int?): Int {
        if (ms == null) return DEFAULT_DEBOUNCE_MS
        return ms.coerceIn(MIN_DEBOUNCE_MS, MAX_DEBOUNCE_MS)
    }

    fun clampDebounceMs(ms: Double?): Int {
        if (ms == null || ms.isNaN() || ms.isInfinite()) return DEFAULT_DEBOUNCE_MS
        return clampDebounceMs(kotlin.math.round(ms).toInt())
    }

    fun instanceMap(instances: List<ToolInstance>): Map<String, ToolInstance> =
        instances.associateBy { it.id }

    fun resolveInstance(instances: Map<String, ToolInstance>, instanceId: String): ToolInstance? =
        instances[instanceId]

    /** Missing / invisible fences render nothing — never an error (FR-3). */
    fun shouldMountFence(instance: ToolInstance?): Boolean = instance != null

    fun fenceRenderMode(
        mobileContentToolsEnabled: Boolean,
        contentToolsEnabled: Boolean,
    ): FenceRenderMode = when {
        !mobileContentToolsEnabled -> FenceRenderMode.LEGACY_PLACEHOLDER
        !contentToolsEnabled -> FenceRenderMode.HIDDEN
        else -> FenceRenderMode.HOST
    }

    fun shouldFetchInstances(
        mobileContentToolsEnabled: Boolean,
        contentToolsEnabled: Boolean,
        courseCode: String?,
        itemId: String?,
    ): Boolean =
        mobileContentToolsEnabled &&
            contentToolsEnabled &&
            !courseCode.isNullOrBlank() &&
            !itemId.isNullOrBlank()

    fun contractSupported(contract: Int, supported: Int = RUNTIME_CONTRACT_VERSION): Boolean =
        contract == supported

    fun conflictPolicyForTool(toolId: String, manifestPolicy: String? = null): ConflictPolicy {
        if (toolId == "highlight_annotate" || toolId == "flashcards") return ConflictPolicy.MERGE
        return ConflictPolicy.from(manifestPolicy)
    }

    fun defaultMerge(client: Map<String, JsonElement>, server: Map<String, JsonElement>): Map<String, JsonElement> =
        server + client

    fun resolveConflictState(
        policy: ConflictPolicy,
        client: Map<String, JsonElement>,
        server: Map<String, JsonElement>,
        mergeFn: (Map<String, JsonElement>, Map<String, JsonElement>) -> Map<String, JsonElement> = ::defaultMerge,
    ): Map<String, JsonElement> = when (policy) {
        ConflictPolicy.SERVER_WINS -> server.toMap()
        ConflictPolicy.CLIENT_WINS -> client.toMap()
        ConflictPolicy.MERGE -> mergeFn(client, server)
    }

    fun resolveConflictJson(
        policy: ConflictPolicy,
        client: JsonElement,
        server: JsonElement,
    ): JsonObject {
        val c = client.asObjectMap()
        val s = server.asObjectMap()
        return JsonObject(resolveConflictState(policy, c, s))
    }

    fun readOnlyReason(input: ReadOnlyInput): ReadOnlyReason? = when {
        input.tombstone -> ReadOnlyReason.TOMBSTONE
        input.breakerOpen -> ReadOnlyReason.BREAKER
        input.status.equals("archived", ignoreCase = true) -> ReadOnlyReason.ARCHIVED
        input.pastDue && input.respectsDueDate -> ReadOnlyReason.PAST_DUE
        input.observer -> ReadOnlyReason.OBSERVER
        input.quarantined -> ReadOnlyReason.QUARANTINED
        input.courseFlagOffAfterLoad -> ReadOnlyReason.FLAG_OFF
        else -> null
    }

    fun readOnlyReason(
        instance: ToolInstance,
        pastDue: Boolean = false,
        respectsDueDate: Boolean = false,
        observer: Boolean = false,
        courseFlagOffAfterLoad: Boolean = false,
    ): ReadOnlyReason? = readOnlyReason(
        ReadOnlyInput(
            tombstone = instance.tombstone,
            breakerOpen = instance.breakerOpen,
            status = instance.status,
            pastDue = pastDue,
            respectsDueDate = respectsDueDate,
            observer = observer,
            quarantined = instance.state?.quarantined == true,
            courseFlagOffAfterLoad = courseFlagOffAfterLoad,
        ),
    )

    fun readOnlyMessageKey(reason: ReadOnlyReason): String = when (reason) {
        ReadOnlyReason.TOMBSTONE -> "mobile.contentTools.runtime.unavailable"
        ReadOnlyReason.BREAKER -> "mobile.contentTools.runtime.unavailable"
        ReadOnlyReason.ARCHIVED -> "mobile.contentTools.runtime.readOnlyArchived"
        ReadOnlyReason.PAST_DUE -> "mobile.contentTools.runtime.readOnlyPastDue"
        ReadOnlyReason.OBSERVER -> "mobile.contentTools.runtime.readOnlyPreview"
        ReadOnlyReason.QUARANTINED -> "mobile.contentTools.runtime.unavailable"
        ReadOnlyReason.FLAG_OFF -> "mobile.contentTools.runtime.unavailable"
    }

    fun syncStatusAfterEdit(current: SyncStatus): SyncStatus =
        if (current == SyncStatus.SAVING) SyncStatus.SAVING else SyncStatus.UNSYNCED

    fun newIdempotencyKey(): String = UUID.randomUUID().toString()

    /** Actions must not be queued offline (FR-11). */
    fun canQueueActionOffline(): Boolean = false

    fun canQueueStateWriteOffline(): Boolean = true

    /**
     * Per-instance ordered replay: sort queued writes so all writes for the same
     * instance stay in arrival order, and instances are stable by key.
     */
    fun orderOutboxByInstance(items: List<Pair<String, Long>>): List<Pair<String, Long>> =
        items.sortedWith(compareBy({ it.first }, { it.second }))

    fun webActivityPath(courseCode: String, itemId: String, instanceId: String): String =
        "/courses/${courseCode.trim()}/modules/items/${itemId.trim()}#lex-tool-$instanceId"

    fun displayTitle(instance: ToolInstance?, toolId: String): String {
        val title = instance?.title?.trim().orEmpty()
        return title.ifEmpty { toolId }
    }

    fun statusChip(status: String): String = status.ifBlank { "not_started" }

    fun accessibleName(title: String, status: String): String =
        "$title, ${statusChip(status).replace('_', ' ')}"

    fun registeredNativeToolIds(): Set<String> =
        setOf("noop_probe") + ContentToolPack1Logic.allowlistedToolIds()

    fun hasNativeRenderer(toolId: String, registered: Set<String> = registeredNativeToolIds()): Boolean =
        toolId in registered

    fun shouldShowUnsupportedPlaceholder(
        toolId: String,
        contract: Int,
        registered: Set<String> = registeredNativeToolIds(),
    ): Boolean = !hasNativeRenderer(toolId, registered) || !contractSupported(contract)

    fun mergeStatePatch(base: JsonElement, patch: Map<String, JsonElement>): JsonObject {
        val map = base.asObjectMap().toMutableMap()
        map.putAll(patch)
        return JsonObject(map)
    }

    fun stringField(element: JsonElement?, key: String): String? {
        val obj = element?.asObjectMap() ?: return null
        val value = obj[key] ?: return null
        return (value as? JsonPrimitive)?.content
    }

    private fun JsonElement.asObjectMap(): Map<String, JsonElement> =
        runCatching { jsonObject.toMap() }.getOrDefault(emptyMap())
}
