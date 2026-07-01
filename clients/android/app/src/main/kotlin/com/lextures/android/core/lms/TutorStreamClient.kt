package com.lextures.android.core.lms

import com.lextures.android.BuildConfig
import com.lextures.android.core.config.AppConfiguration

import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOn
import kotlinx.coroutines.isActive
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.atomic.AtomicReference

/** Reads tutor/study-buddy SSE streams from POST endpoints. */
class TutorStreamClient(
    private val http: OkHttpClient = OkHttpClient(),
    private val json: Json = Json { ignoreUnknownKeys = true },
) {
    private val activeCall = AtomicReference<okhttp3.Call?>(null)

    fun cancel() {
        activeCall.getAndSet(null)?.cancel()
    }

    fun stream(
        path: String,
        method: String = "POST",
        body: String?,
        accessToken: String,
    ): Flow<TutorStreamEvent> = flow {
        val builder = Request.Builder()
            .url(AppConfiguration.apiUrl(path))
            .header("Accept", "text/event-stream")
            .header("Authorization", "Bearer $accessToken")
            .header("X-Platform", "android")
            .header("X-App-Version", BuildConfig.VERSION_NAME)
            .header("Accept-Language", java.util.Locale.getDefault().toLanguageTag())

        if (body != null) {
            builder
                .method(method, body.toRequestBody("application/json".toMediaType()))
                .header("Content-Type", "application/json")
        } else {
            builder.method(method, null)
        }

        val call = http.newCall(builder.build())
        activeCall.set(call)
        try {
            val response = call.execute()
            response.use {
                if (it.code !in 200..299) {
                    val responseBody = it.body?.string().orEmpty()
                    throw ApiError.HttpStatus(
                        it.code,
                        TutorLogic.gracefulHttpMessage(it.code, responseBody),
                    )
                }
                val source = it.body?.source() ?: return@flow
                while (currentCoroutineContext().isActive && !source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    TutorLogic.parseSseLine(line, json)?.let { event ->
                        emit(event)
                        if (event is TutorStreamEvent.Done || event is TutorStreamEvent.Error) return@flow
                    }
                }
            }
        } catch (e: Exception) {
            if (!call.isCanceled()) throw e
        } finally {
            activeCall.compareAndSet(call, null)
        }
    }.flowOn(Dispatchers.IO)
}