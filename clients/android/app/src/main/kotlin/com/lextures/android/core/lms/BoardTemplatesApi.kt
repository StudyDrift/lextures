package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board templates & duplication REST (MOB.8 / VC.8). */
object BoardTemplatesApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString<T>(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    private fun encodeQuery(value: String): String =
        URLEncoder.encode(value, "UTF-8")

    suspend fun listTemplates(
        scope: String? = null,
        courseCode: String? = null,
        query: String? = null,
        accessToken: String,
    ): List<BoardTemplate> = withContext(Dispatchers.IO) {
        val items = mutableListOf<String>()
        if (!scope.isNullOrBlank()) items += "scope=${encodeQuery(scope)}"
        if (!courseCode.isNullOrBlank()) items += "courseCode=${encodeQuery(courseCode)}"
        if (!query.isNullOrBlank()) items += "q=${encodeQuery(query)}"
        var path = "/api/v1/board-templates"
        if (items.isNotEmpty()) path += "?" + items.joinToString("&")
        val (body, code) = client.requestRaw(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardTemplatesListResponse>(body).templates
    }

    suspend fun createFromTemplate(
        courseCode: String,
        templateId: String,
        title: String = "",
        description: String = "",
        accessToken: String,
    ): Board = withContext(Dispatchers.IO) {
        val qs = "from=${encodeQuery("template:$templateId")}"
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards?$qs",
            method = "POST",
            body = client.encodeBody(CreateBoardBody(title, description), CreateBoardBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun duplicateBoard(
        targetCourseCode: String,
        sourceBoardId: String,
        mode: BoardCopyMode,
        title: String = "",
        description: String = "",
        accessToken: String,
    ): BoardCreateResult = withContext(Dispatchers.IO) {
        val qs = "from=${encodeQuery("board:$sourceBoardId")}&mode=${encodeQuery(mode.apiValue)}"
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(targetCourseCode)}/boards?$qs",
            method = "POST",
            body = client.encodeBody(CreateBoardBody(title, description), CreateBoardBody.serializer()),
            accessToken = accessToken,
        )
        if (code == 202) {
            return@withContext BoardCreateResult.JobResult(decode<BoardCopyJobResponse>(body).job)
        }
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        BoardCreateResult.BoardResult(decode(body))
    }

    suspend fun fetchCopyJob(
        courseCode: String,
        jobId: String,
        accessToken: String,
    ): BoardCopyJob = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/board-copy-jobs/${encodePath(jobId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun saveAsTemplate(
        courseCode: String,
        boardId: String,
        scope: String,
        title: String = "",
        description: String = "",
        tags: List<String> = emptyList(),
        includePosts: Boolean = false,
        accessToken: String,
    ): BoardTemplate = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/save-as-template",
            method = "POST",
            body = client.encodeBody(
                SaveBoardAsTemplateBody(scope, title, description, tags, includePosts),
                SaveBoardAsTemplateBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }
}
