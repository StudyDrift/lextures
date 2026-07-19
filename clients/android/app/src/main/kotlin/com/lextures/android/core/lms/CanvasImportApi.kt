package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import com.lextures.android.core.realtime.WebSocketClient
import com.lextures.android.core.realtime.WebSocketLifecycleEvent
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.net.URLEncoder
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * Canvas course import API (MOB.2).
 * The Canvas access token is sent once per request and is never persisted by this client.
 */
object CanvasImportApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    suspend fun fetchCanvasCourses(
        canvasBaseUrl: String,
        accessToken: String,
        sessionAccessToken: String,
    ): List<CanvasCourseListItem> = withContext(Dispatchers.IO) {
        val body = CanvasListCoursesRequest(
            canvasBaseUrl = CanvasImportLogic.normalizeBaseUrl(canvasBaseUrl),
            accessToken = accessToken.trim(),
        )
        val (response, code) = client.requestRaw(
            path = "/api/v1/integrations/canvas/courses",
            method = "POST",
            body = json.encodeToString(CanvasListCoursesRequest.serializer(), body),
            accessToken = sessionAccessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        json.decodeFromString(CanvasCoursesResponse.serializer(), response).courses.orEmpty()
    }

    /**
     * Queues a Canvas import and streams progress over the job WebSocket until complete/error/cancel.
     */
    suspend fun postCourseImportCanvas(
        courseCode: String,
        body: PostCourseImportCanvasRequest,
        sessionAccessToken: String,
        onProgress: (String) -> Unit,
        isCancelled: () -> Boolean,
    ) = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/import/canvas",
            method = "POST",
            body = json.encodeToString(PostCourseImportCanvasRequest.serializer(), body),
            accessToken = sessionAccessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        val queued = json.decodeFromString(CanvasImportQueuedResponse.serializer(), response)
        val jobId = queued.jobId?.trim().orEmpty()
        if (jobId.isEmpty()) throw CanvasImportLogic.CanvasImportError.MissingJobId
        onProgress(queued.message?.trim()?.takeIf { it.isNotEmpty() } ?: "Canvas import queued.")
        waitForCanvasImportJob(
            jobId = jobId,
            sessionAccessToken = sessionAccessToken,
            onProgress = onProgress,
            isCancelled = isCancelled,
        )
    }

    private suspend fun waitForCanvasImportJob(
        jobId: String,
        sessionAccessToken: String,
        onProgress: (String) -> Unit,
        isCancelled: () -> Boolean,
    ) {
        val path = CanvasImportLogic.jobWebSocketPath(jobId)
        suspendCancellableCoroutine { cont ->
            val settled = AtomicBoolean(false)
            var socket: WebSocketClient? = null

            fun finish(result: Result<Unit>) {
                if (!settled.compareAndSet(false, true)) return
                socket?.disconnect()
                socket = null
                result.fold(
                    onSuccess = { cont.resume(Unit) },
                    onFailure = { cont.resumeWithException(it) },
                )
            }

            socket = WebSocketClient(
                path = path,
                accessTokenProvider = { sessionAccessToken },
                onMessage = { text ->
                    if (isCancelled()) {
                        finish(Result.failure(CanvasImportLogic.CanvasImportError.Cancelled))
                        return@WebSocketClient
                    }
                    val message = CanvasImportLogic.parseWSMessage(text) ?: return@WebSocketClient
                    when (message.type) {
                        CanvasImportLogic.WSMessageType.Progress -> {
                            message.message?.trim()?.takeIf { it.isNotEmpty() }?.let(onProgress)
                        }
                        CanvasImportLogic.WSMessageType.Complete,
                        CanvasImportLogic.WSMessageType.CoursesUpdated,
                        -> finish(Result.success(Unit))
                        CanvasImportLogic.WSMessageType.Error -> finish(
                            Result.failure(
                                CanvasImportLogic.CanvasImportError.Server(
                                    message.message?.trim()?.takeIf { it.isNotEmpty() }
                                        ?: "Canvas import failed.",
                                ),
                            ),
                        )
                        CanvasImportLogic.WSMessageType.Unknown -> Unit
                    }
                },
                onLifecycle = { event ->
                    if (isCancelled()) {
                        finish(Result.failure(CanvasImportLogic.CanvasImportError.Cancelled))
                        return@WebSocketClient
                    }
                    when (event) {
                        WebSocketLifecycleEvent.Opened -> Unit
                        is WebSocketLifecycleEvent.Closed -> {
                            if (!event.willReconnect && !settled.get()) {
                                finish(Result.failure(CanvasImportLogic.CanvasImportError.ConnectionClosed))
                            }
                        }
                    }
                },
                stopOnPermanentRefusal = true,
            )
            cont.invokeOnCancellation {
                finish(Result.failure(CanvasImportLogic.CanvasImportError.Cancelled))
            }
            socket?.connect()

            // Poll cancel flag while waiting.
            Thread {
                while (!settled.get()) {
                    if (isCancelled()) {
                        finish(Result.failure(CanvasImportLogic.CanvasImportError.Cancelled))
                        return@Thread
                    }
                    try {
                        Thread.sleep(200)
                    } catch (_: InterruptedException) {
                        return@Thread
                    }
                }
            }.start()
        }
    }
}
