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
 * Lifecycle hooks for per-screen sockets that need connection-state UX (e.g. [BoardSocket]).
 *
 * [httpStatus] is set when the HTTP upgrade failed before the socket opened (e.g. 404).
 * [willReconnect] is false when the client stopped retrying (permanent refusal or [disconnect]).
 */
sealed class WebSocketLifecycleEvent {
    data object Opened : WebSocketLifecycleEvent()
    data class Closed(val httpStatus: Int?, val willReconnect: Boolean) : WebSocketLifecycleEvent()
}

/**
 * Reconnecting JSON WebSocket client for the server's realtime hubs.
 *
 * Mirrors the web client's handshake (`clients/web/src/context/inbox-unread-provider.tsx`):
 * on open, sends `{"authToken":"<jwt>"}` as the first text message (the server reads exactly
 * one auth message before subscribing — see `handleCommWS` in
 * `server/internal/httpserver/communication.go`), then treats every later frame as an event
 * payload. Reconnects after a fixed 2s delay, matching the web app's `scheduleReconnect`.
 *
 * Binary frames are ignored (OkHttp only delivers text to [onMessage] when the String overload
 * is used) — required for board Y.js replay frames (VC.M4).
 */
class WebSocketClient(
    private val path: String,
    private val http: OkHttpClient = OkHttpClient(),
    private val accessTokenProvider: () -> String?,
    private val onMessage: (String) -> Unit,
    private val onLifecycle: ((WebSocketLifecycleEvent) -> Unit)? = null,
    private val stopOnPermanentRefusal: Boolean = false,
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
                    onLifecycle?.invoke(WebSocketLifecycleEvent.Opened)
                    webSocket.send(Json.encodeToString(AuthHandshake(token)))
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    onMessage(text)
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    handleDisconnect(webSocket, httpStatus = null)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    handleDisconnect(webSocket, httpStatus = response?.code)
                }
            },
        )
    }

    private fun handleDisconnect(finished: WebSocket, httpStatus: Int?) {
        if (isStopped || socket !== finished) {
            onLifecycle?.invoke(WebSocketLifecycleEvent.Closed(httpStatus, willReconnect = false))
            return
        }
        socket = null
        connectedToken = null

        val permanent = stopOnPermanentRefusal && isPermanentWsRefusal(httpStatus)
        if (permanent) {
            isStopped = true
            onLifecycle?.invoke(WebSocketLifecycleEvent.Closed(httpStatus, willReconnect = false))
            return
        }

        onLifecycle?.invoke(WebSocketLifecycleEvent.Closed(httpStatus, willReconnect = true))
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

        fun isPermanentWsRefusal(statusCode: Int?): Boolean =
            statusCode == 401 || statusCode == 403 || statusCode == 404
    }
}
