package com.lextures.android.core.realtime

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Course-scoped realtime socket for `structure_changed` events (module/item edits, imports
 * finishing, etc). Unlike [RealtimeManager]'s app-wide hubs, this is created per screen and
 * torn down when the course is no longer on screen — mirrors the web app's
 * `use-course-structure-ws.ts` hook.
 */
class CourseStructureSocket {
    private val json = Json { ignoreUnknownKeys = true }

    private val _revision = MutableStateFlow(0)
    val revision: StateFlow<Int> = _revision.asStateFlow()

    private var socket: WebSocketClient? = null

    /** Connects to `/api/v1/courses/{code}/structure/ws`. Safe to call from a composable effect. */
    fun connect(courseCode: String, accessTokenProvider: () -> String?) {
        if (socket != null) return
        socket = WebSocketClient(
            path = "/api/v1/courses/$courseCode/structure/ws",
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

    private fun handleMessage(text: String) {
        val event = runCatching { json.decodeFromString<StructureWsEvent>(text) }.getOrNull() ?: return
        if (event.type == "structure_changed") {
            _revision.value += 1
        }
    }

    @Serializable
    private data class StructureWsEvent(val type: String)
}
