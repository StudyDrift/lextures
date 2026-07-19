package com.lextures.android.features.livequiz

import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.lms.LiveGameAnswerAck
import com.lextures.android.core.lms.LiveGameInboundMessage
import com.lextures.android.core.lms.LiveGameLogic
import com.lextures.android.core.lms.LiveGameStateFrame
import com.lextures.android.core.lms.LiveQuizLogic
import com.lextures.android.core.lms.LiveQuizObservability
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject
import java.time.Instant

/**
 * Live-quiz WebSocket with role + playerToken handshake (IQ.3 / MOB.5).
 * Dedicated client — shared [com.lextures.android.core.realtime.WebSocketClient] only sends `{authToken}`.
 */
class LiveGameSocket(
    private val courseCode: String,
    private val gameId: String,
    private val role: LiveGameLogic.Role = LiveGameLogic.Role.Player,
    private val playerToken: String?,
    private val accessTokenProvider: () -> String?,
    private val onState: (LiveGameStateFrame) -> Unit,
    private val onAck: (LiveGameAnswerAck) -> Unit,
    private val onKicked: () -> Unit,
    private val onConn: (LiveGameLogic.ConnStatus) -> Unit,
    private val http: OkHttpClient = OkHttpClient(),
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var socket: WebSocket? = null
    private var isStopped = true
    private var reconnectJob: Job? = null
    private var retry = 0
    private var seq = 0
    private var kicked = false

    fun connect() {
        if (kicked) return
        isStopped = false
        reconnectJob?.cancel()
        reconnectJob = null
        val auth = accessTokenProvider()?.takeIf { it.isNotBlank() }
        if (auth == null && playerToken.isNullOrBlank()) {
            scheduleReconnect()
            return
        }
        openConnection(auth)
    }

    fun disconnect() {
        isStopped = true
        reconnectJob?.cancel()
        reconnectJob = null
        socket?.close(1000, null)
        socket = null
    }

    fun send(payload: JSONObject) {
        socket?.send(payload.toString())
    }

    fun submitAnswer(
        questionIndex: Int,
        answer: LiveGameLogic.AnswerPayload,
        powerUp: String? = null,
    ) {
        send(
            LiveGameLogic.answerMessage(
                questionIndex = questionIndex,
                answer = answer,
                clientSentAt = Instant.now().toString(),
                powerUp = powerUp,
            ),
        )
    }

    private fun openConnection(authToken: String?) {
        socket?.close(1000, null)
        onConn(if (retry > 0) LiveGameLogic.ConnStatus.Reconnecting else LiveGameLogic.ConnStatus.Connecting)

        val path = LiveQuizLogic.webSocketPath(courseCode, gameId)
        val url = AppConfiguration.apiUrl(path).toString()
            .replaceFirst("https://", "wss://")
            .replaceFirst("http://", "ws://")
        val request = Request.Builder().url(url).build()

        socket = http.newWebSocket(
            request,
            object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
                    webSocket.send(
                        LiveGameLogic.authHandshake(authToken, role, playerToken).toString(),
                    )
                    if (seq > 0) {
                        webSocket.send(LiveGameLogic.catchupMessage(seq).toString())
                    } else if (role == LiveGameLogic.Role.Player) {
                        webSocket.send(LiveGameLogic.helloMessage(0).toString())
                    }
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    when (val msg = LiveGameLogic.parseInbound(text)) {
                        LiveGameInboundMessage.Kicked -> {
                            kicked = true
                            isStopped = true
                            onKicked()
                            onConn(LiveGameLogic.ConnStatus.Kicked)
                            webSocket.close(1000, null)
                        }
                        is LiveGameInboundMessage.AnswerAck -> onAck(msg.ack)
                        is LiveGameInboundMessage.State -> {
                            seq = msg.frame.seq
                            onState(msg.frame)
                            onConn(
                                if (msg.frame.phase == LiveGameLogic.Phase.Ended.wire) {
                                    LiveGameLogic.ConnStatus.Ended
                                } else {
                                    LiveGameLogic.ConnStatus.Connected
                                },
                            )
                            retry = 0
                            LiveQuizObservability.record("live_quiz_reconnect", mapOf("ok" to "1"))
                        }
                        LiveGameInboundMessage.Unknown -> Unit
                    }
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    handleDisconnect(webSocket)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    handleDisconnect(webSocket)
                }
            },
        )
    }

    private fun handleDisconnect(finished: WebSocket) {
        if (isStopped || kicked || socket !== finished) return
        socket = null
        retry += 1
        LiveQuizObservability.record("live_quiz_reconnect", mapOf("ok" to "0"))
        scheduleReconnect()
    }

    private fun scheduleReconnect() {
        onConn(LiveGameLogic.ConnStatus.Reconnecting)
        val delayMs = LiveGameLogic.reconnectDelayMs(retry).toLong()
        reconnectJob = scope.launch {
            delay(delayMs)
            connect()
        }
    }
}
