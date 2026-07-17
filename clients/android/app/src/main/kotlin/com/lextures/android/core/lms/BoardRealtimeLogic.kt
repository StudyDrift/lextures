package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** Connection chip states for board realtime (VC.M4 FR-8). */
enum class BoardSyncState {
    Connecting,
    Live,
    Reconnecting,
    Offline,
}

/** Parsed `board.changed` text frame from the board WebSocket relay. */
data class BoardChangedEvent(
    val reason: String,
    val postId: String? = null,
)

data class BoardRefetchPlan(
    val full: Boolean,
    val postId: String? = null,
    val createdCount: Int = 0,
)

/**
 * Pure helpers for board realtime (VC.M4) — unit-tested without a live socket.
 */
object BoardRealtimeLogic {
    private val json = Json { ignoreUnknownKeys = true }

    /** After this many consecutive failures without a successful open, stop retrying. */
    const val MAX_TRANSIENT_FAILURES_BEFORE_OFFLINE = 8

    /** Debounce window for coalescing `board.changed` bursts (milliseconds). */
    const val REFETCH_COALESCE_MS = 250L

    /** HTTP statuses that mean the upgrade was refused permanently. */
    fun isPermanentWsRefusal(statusCode: Int?): Boolean =
        statusCode == 401 || statusCode == 403 || statusCode == 404

    fun shouldStopRetrying(consecutiveFailures: Int, lastHttpStatus: Int?): Boolean {
        if (isPermanentWsRefusal(lastHttpStatus)) return true
        return consecutiveFailures >= MAX_TRANSIENT_FAILURES_BEFORE_OFFLINE
    }

    /** Parses a JSON text frame into a `board.changed` event. Binary / non-JSON / other types → null. */
    fun parseBoardChangedEvent(raw: String): BoardChangedEvent? {
        val msg = runCatching { json.decodeFromString<BoardChangedFrame>(raw) }.getOrNull() ?: return null
        if (msg.type != "board.changed" || msg.reason.isNullOrBlank()) return null
        val postId = msg.postId?.trim()?.takeIf { it.isNotEmpty() }
        return BoardChangedEvent(reason = msg.reason, postId = postId)
    }

    /** Detects `{"error":"board_locked_or_frozen"}` notice frames. */
    fun isBoardLockedOrFrozenError(raw: String): Boolean {
        val msg = runCatching { json.decodeFromString<BoardErrorFrame>(raw) }.getOrNull() ?: return false
        return msg.error == "board_locked_or_frozen"
    }

    /**
     * Coalesces a burst of change events into one refetch plan (FR-5 / AC-7).
     * A general bump (no postId) or multiple distinct postIds → full list refetch.
     */
    fun coalesceRefetchPlan(events: List<BoardChangedEvent>): BoardRefetchPlan {
        val postIds = linkedSetOf<String>()
        var hasGeneral = false
        for (event in events) {
            val postId = event.postId
            if (!postId.isNullOrBlank()) {
                postIds.add(postId)
            } else {
                hasGeneral = true
            }
        }
        val createdCount = events.count { it.reason == "post.created" }
        if (hasGeneral || postIds.size != 1) {
            return BoardRefetchPlan(full = true, postId = null, createdCount = createdCount)
        }
        return BoardRefetchPlan(full = false, postId = postIds.first(), createdCount = createdCount)
    }

    @Serializable
    private data class BoardChangedFrame(
        val type: String? = null,
        val reason: String? = null,
        val postId: String? = null,
    )

    @Serializable
    private data class BoardErrorFrame(val error: String? = null)
}
