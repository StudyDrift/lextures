package com.lextures.android.core.lms

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.net.URLEncoder

/**
 * Pure CT.M4 sandbox decisions — bridge validation, rate/size guards, height clamp,
 * opaque participant ids, and native → sandbox → placeholder resolution. No networking.
 */
object ContentToolSandboxLogic {
    const val BRIDGE_VERSION = 1
    const val BRIDGE_MAX_MESSAGE_BYTES = 64 * 1024
    const val BRIDGE_MAX_MESSAGES_PER_SEC = 20
    const val READY_TIMEOUT_MS = 10_000
    const val MIN_HEIGHT_DP = 80.0
    const val MAX_HEIGHT_DP = 2000.0
    const val MAX_LIVE_WEBVIEWS = 3
    const val ANNOUNCE_MAX_PER_SEC = 5
    const val SUPPORTED_READY_CONTRACT = "1"

    private val fromToolTypes = setOf("ready", "save", "runAction", "resize", "announce")
    private val toToolTypes = setOf("init", "stateAccepted", "actionResult", "error")
    private val json = Json { ignoreUnknownKeys = true }

    enum class RenderPath {
        NATIVE,
        SANDBOX,
        PLACEHOLDER,
        ;

        fun wire(): String = name.lowercase()
    }

    enum class RejectionReason(val wire: String) {
        OVERSIZED("oversized"),
        RATE_LIMITED("rate_limited"),
        MALFORMED("malformed"),
        UNKNOWN_TYPE("unknown_type"),
    }

    data class ResolutionInput(
        val toolId: String,
        val contract: Int,
        val sandboxMode: String? = null,
        val sandboxEnabled: Boolean,
        val registered: Set<String>,
        val tombstone: Boolean = false,
        val breakerOpen: Boolean = false,
        val deprecated: Boolean = false,
        val killed: Boolean = false,
    )

    fun hasNativeRenderer(toolId: String, registered: Set<String>): Boolean = toolId in registered

    fun isSandboxable(input: ResolutionInput): Boolean {
        if (!input.sandboxEnabled) return false
        if (input.tombstone || input.breakerOpen || input.deprecated || input.killed) return false
        return ContentToolHostLogic.contractSupported(input.contract)
    }

    /** Pure resolution: native → sandbox → placeholder. */
    fun resolveRenderPath(input: ResolutionInput): RenderPath {
        if (input.tombstone || input.breakerOpen || input.killed || input.deprecated) {
            return RenderPath.PLACEHOLDER
        }
        if (hasNativeRenderer(input.toolId, input.registered) &&
            ContentToolHostLogic.contractSupported(input.contract)
        ) {
            return RenderPath.NATIVE
        }
        if (isSandboxable(input)) return RenderPath.SANDBOX
        return RenderPath.PLACEHOLDER
    }

    fun resolveRenderPath(
        toolId: String,
        contract: Int,
        sandboxMode: String?,
        sandboxEnabled: Boolean,
        registered: Set<String>,
        tombstone: Boolean = false,
        breakerOpen: Boolean = false,
        deprecated: Boolean = false,
        killed: Boolean = false,
    ): RenderPath = resolveRenderPath(
        ResolutionInput(
            toolId = toolId,
            contract = contract,
            sandboxMode = sandboxMode,
            sandboxEnabled = sandboxEnabled,
            registered = registered,
            tombstone = tombstone,
            breakerOpen = breakerOpen,
            deprecated = deprecated,
            killed = killed,
        ),
    )

    fun isBridgeFromTool(msg: JsonElement?): Boolean {
        val obj = msg?.asObjectOrNull() ?: return false
        if (obj["v"]?.jsonPrimitive?.intOrNull != BRIDGE_VERSION) return false
        val t = obj["t"]?.jsonPrimitive?.contentOrNull ?: return false
        return t in fromToolTypes
    }

    fun isBridgeToTool(msg: JsonElement?): Boolean {
        val obj = msg?.asObjectOrNull() ?: return false
        if (obj["v"]?.jsonPrimitive?.intOrNull != BRIDGE_VERSION) return false
        val t = obj["t"]?.jsonPrimitive?.contentOrNull ?: return false
        return t in toToolTypes
    }

    fun isBridgeFromToolJson(raw: String): Boolean =
        runCatching { isBridgeFromTool(json.parseToJsonElement(raw)) }.getOrDefault(false)

    fun isBridgeToToolJson(raw: String): Boolean =
        runCatching { isBridgeToTool(json.parseToJsonElement(raw)) }.getOrDefault(false)

    fun measureMessageBytes(msg: JsonElement?): Int {
        if (msg == null || msg is JsonNull) return Int.MAX_VALUE
        return runCatching {
            json.encodeToString(JsonElement.serializer(), msg).toByteArray(Charsets.UTF_8).size
        }.getOrDefault(Int.MAX_VALUE)
    }

    fun measureMessageBytes(jsonString: String): Int =
        jsonString.toByteArray(Charsets.UTF_8).size

    /** Classify a raw ingress payload before dispatch. Returns null when accepted. */
    fun rejectIngress(
        rawJSON: String,
        limiter: BridgeRateLimiter,
        nowMs: Long = System.currentTimeMillis(),
    ): RejectionReason? {
        if (!limiter.allow(nowMs)) return RejectionReason.RATE_LIMITED
        if (measureMessageBytes(rawJSON) > BRIDGE_MAX_MESSAGE_BYTES) return RejectionReason.OVERSIZED
        if (!isBridgeFromToolJson(rawJSON)) return RejectionReason.MALFORMED
        return null
    }

    fun clampHeight(height: Double): Double {
        if (height.isNaN() || height.isInfinite()) return MIN_HEIGHT_DP
        return height.coerceIn(MIN_HEIGHT_DP, MAX_HEIGHT_DP)
    }

    fun contractInSupportedRange(contract: String): Boolean =
        contract == SUPPORTED_READY_CONTRACT

    /**
     * Opaque per-instance participant id — not email, not user id (FR-7).
     * Matches web `opaqueParticipantId` (Java-style 32-bit string hash).
     */
    fun opaqueParticipantId(instanceId: String, enrollmentHint: String? = null): String {
        val raw = "$instanceId:${enrollmentHint ?: "anon"}"
        var h = 0
        for (ch in raw) {
            h = (31 * h + ch.code)
        }
        val unsigned = h.toLong() and 0xFFFF_FFFFL
        return "p_" + unsigned.toString(16)
    }

    fun documentPath(toolId: String, version: String? = null): String {
        // Use the String charset overload (API 1); Charset overload requires API 33.
        @Suppress("DEPRECATION")
        val encoded = URLEncoder.encode(toolId, "UTF-8").replace("+", "%20")
        var path = "/tool-sandbox/$encoded.html"
        if (!version.isNullOrBlank()) {
            val sep = if (path.contains("?")) "&" else "?"
            @Suppress("DEPRECATION")
            path += sep + "v=" + URLEncoder.encode(version, "UTF-8")
        }
        return path
    }

    fun poolShouldEvict(aliveCount: Int, maxAlive: Int = MAX_LIVE_WEBVIEWS): Boolean =
        aliveCount > maxAlive

    class BridgeRateLimiter(
        private val maxPerSec: Int = BRIDGE_MAX_MESSAGES_PER_SEC,
    ) {
        private val timestamps = ArrayList<Long>()

        fun allow(nowMs: Long): Boolean {
            val cut = nowMs - 1000
            timestamps.removeAll { it < cut }
            if (timestamps.size >= maxPerSec) return false
            timestamps.add(nowMs)
            return true
        }
    }

    private fun JsonElement.asObjectOrNull(): JsonObject? =
        runCatching { jsonObject }.getOrNull()
}
