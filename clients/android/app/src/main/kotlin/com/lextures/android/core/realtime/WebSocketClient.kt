package com.lextures.android.core.realtime

import com.lextures.android.core.config.AppConfiguration
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener

/**
 * Reconnecting JSON WebSocket client for the server's realtime hubs.
 *
 * Mirrors the web client's handshake (`clients/web/src/context/inbox-unread-provider.tsx`):
 * on open, sends `{"authToken":"<jwt>"}` as the first text message (the server reads exactly
 * one auth message before subscribing — see `handleCommWS` in
 * `server/internal/httpserver/communication.go`), then treats every later frame as an event
 * payload. Reconnects after a fixed 2s delay, matching the web app's `scheduleReconnect`.
 */
class WebSocketClient(
    private val path: String,
    private val http: OkHttpClient = OkHttpClient(),
    private val accessTokenProvider: () -> String?,
    private val onMessage: (String) -> Unit,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var socket: WebSocket? = null
    private var connectedToken: String? = null
    private var isStopped = true
    private var reconnectJob: Job? = null

    /** Connects if not already connected with the current token; safe to call repeatedly. */
    fun connect() {
        isStopped = false
        reconnectJob?.cancel()
        reconnectJob = null
        val token = accessTokenProvider()?.takeIf { it.isNotBlank() } ?: return
        if (socket != null && connectedToken == token) return
        openConnection(token)
    }

    fun disconnect() {
        isStopped = true
        reconnectJob?.cancel()
        reconnectJob = null
        connectedToken = null
        socket?.close(NORMAL_CLOSURE, null)
        socket = null
    }

    private fun openConnection(token: String) {
        socket?.close(NORMAL_CLOSURE, null)
        connectedToken = token

        val url = AppConfiguration.apiUrl(path).toString()
            .replaceFirst("https://", "wss://")
            .replaceFirst("http://", "ws://")
        val request = Request.Builder().url(url).build()

        socket = http.newWebSocket(
            request,
            object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
                    webSocket.send(Json.encodeToString(AuthHandshake(token)))
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    onMessage(text)
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    scheduleReconnect(webSocket)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    scheduleReconnect(webSocket)
                }
            },
        )
    }

    private fun scheduleReconnect(finished: WebSocket) {
        if (isStopped || socket !== finished) return
        socket = null
        connectedToken = null
        reconnectJob = scope.launch {
            delay(RECONNECT_DELAY_MS)
            connect()
        }
    }

    @Serializable
    private data class AuthHandshake(val authToken: String)

    private companion object {
        const val NORMAL_CLOSURE = 1000
        const val RECONNECT_DELAY_MS = 2000L
    }
}
