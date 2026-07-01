package com.lextures.android.core.realtime

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Course-scoped realtime socket for feed events (`{"type":"feed","scope":"channels"|"messages"}`).
 * Created per screen and torn down when the feed leaves the screen — mirrors
 * [CourseStructureSocket]. Exposes a per-channel revision map so an open channel only
 * re-fetches when its own messages change.
 */
class FeedSocket {
    private val json = Json { ignoreUnknownKeys = true }

    private val _channelsRevision = MutableStateFlow(0)
    val channelsRevision: StateFlow<Int> = _channelsRevision.asStateFlow()

    private val _messagesRevision = MutableStateFlow<Map<String, Int>>(emptyMap())
    val messagesRevision: StateFlow<Map<String, Int>> = _messagesRevision.asStateFlow()

    private var socket: WebSocketClient? = null

    /** Connects to `/api/v1/courses/{code}/feed/ws`. Safe to call from a composable effect. */
    fun connect(courseCode: String, accessTokenProvider: () -> String?) {
        if (socket != null) return
        socket = WebSocketClient(
            path = "/api/v1/courses/$courseCode/feed/ws",
            accessTokenProvider = accessTokenProvider,
            onMessage = ::handleMessage,
        )
        socket?.connect()
    }

    /** Call from `DisposableEffect.onDispose` so the socket doesn't outlive the screen. */
    fun disconnect() {
        socket?.disconnect()
        socket = null
    }

    fun revision(channelId: String): Int = messagesRevision.value[channelId] ?: 0

    private fun handleMessage(text: String) {
        val event = runCatching { json.decodeFromString<FeedWsEvent>(text) }.getOrNull() ?: return
        if (event.type != "feed") return
        when (event.scope) {
            "channels" -> _channelsRevision.value += 1
            "messages" -> {
                val channelId = event.channelId ?: return
                val current = _messagesRevision.value
                _messagesRevision.value = current + (channelId to (current[channelId] ?: 0) + 1)
            }
        }
    }

    @Serializable
    private data class FeedWsEvent(val type: String, val scope: String, val channelId: String? = null)
}
