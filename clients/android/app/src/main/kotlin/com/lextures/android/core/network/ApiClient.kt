package com.lextures.android.core.network

import com.lextures.android.core.config.AppConfiguration
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

class ApiClient(
    private val http: OkHttpClient = OkHttpClient(),
    private val json: Json = Json { ignoreUnknownKeys = true },
) {
    suspend fun request(
        path: String,
        method: String = "GET",
        body: String? = null,
        accessToken: String? = null,
    ): Pair<String, Int> {
        val builder = Request.Builder()
            .url(AppConfiguration.apiUrl(path))
            .header("Accept", "application/json")

        if (body != null) {
            builder
                .method(method, body.toRequestBody("application/json".toMediaType()))
                .header("Content-Type", "application/json")
        } else {
            builder.method(method, null)
        }

        if (!accessToken.isNullOrBlank()) {
            builder.header("Authorization", "Bearer $accessToken")
        }

        val response = try {
            http.newCall(builder.build()).execute()
        } catch (e: Exception) {
            throw ApiError.Transport(e)
        }

        response.use {
            val responseBody = it.body?.string().orEmpty()
            val code = it.code
            if (code !in 200..299) {
                throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            }
            return responseBody to code
        }
    }

    fun <T> encodeBody(value: T, serializer: kotlinx.serialization.KSerializer<T>): String =
        json.encodeToString(serializer, value)
}
