package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import java.net.URLEncoder

/** Board export job REST (MOB.8 / VC.9). */
object BoardExportApi {
    private val client = ApiClient()
    private val http = OkHttpClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString<T>(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    suspend fun createExport(
        courseCode: String,
        boardId: String,
        format: BoardExportFormat,
        includeModeration: Boolean = false,
        accessToken: String,
    ): BoardExportJob = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/export",
            method = "POST",
            body = client.encodeBody(
                CreateBoardExportBody(format.apiValue, includeModeration),
                CreateBoardExportBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardExportJobResponse>(body).job
    }

    suspend fun fetchExportJob(
        courseCode: String,
        boardId: String,
        jobId: String,
        accessToken: String,
    ): BoardExportJob = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/export/${encodePath(jobId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun downloadExport(
        courseCode: String,
        boardId: String,
        jobId: String,
        accessToken: String,
    ): ByteArray = withContext(Dispatchers.IO) {
        val path =
            "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/export/${encodePath(jobId)}/content"
        val request = Request.Builder()
            .url(AppConfiguration.apiUrl(path))
            .header("Authorization", "Bearer $accessToken")
            .header("X-Platform", "android")
            .get()
            .build()
        http.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                throw ApiError.HttpStatus(response.code, null)
            }
            response.body?.bytes() ?: ByteArray(0)
        }
    }
}
