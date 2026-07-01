package com.lextures.android.core.realtime

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Course-scoped realtime socket for `files_changed` events (uploads/deletes from web or other
 * devices). Created per screen and torn down when the folder browser leaves the screen —
 * mirrors [CourseStructureSocket].
 */
class CourseFilesSocket {
    private val json = Json { ignoreUnknownKeys = true }

    private val _revision = MutableStateFlow(0)
    val revision: StateFlow<Int> = _revision.asStateFlow()

    private var socket: WebSocketClient? = null

    /** Connects to `/api/v1/courses/{code}/files/ws`. Safe to call from a composable effect. */
    fun connect(courseCode: String, accessTokenProvider: () -> String?) {
        if (socket != null) return
        socket = WebSocketClient(
            path = "/api/v1/courses/$courseCode/files/ws",
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
        val event = runCatching { json.decodeFromString<FilesWsEvent>(text) }.getOrNull() ?: return
        if (event.type == "files_changed") {
            _revision.value += 1
        }
    }

    @Serializable
    private data class FilesWsEvent(val type: String)
}
