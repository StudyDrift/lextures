package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board layout, sections, and arrange REST (VC.M3). Mirrors web `boards-api`. */
object BoardLayoutApi {
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

    suspend fun listSections(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): List<BoardSection> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/sections",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardSectionsListResponse>(body).sections
    }

    suspend fun createSection(
        courseCode: String,
        boardId: String,
        title: String,
        sortIndex: Double? = null,
        accessToken: String,
    ): BoardSection = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/sections",
            method = "POST",
            body = client.encodeBody(
                CreateBoardSectionBody(title = title, sortIndex = sortIndex),
                CreateBoardSectionBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchSection(
        courseCode: String,
        boardId: String,
        sectionId: String,
        title: String? = null,
        sortIndex: Double? = null,
        accessToken: String,
    ): BoardSection = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/sections/${encodePath(sectionId)}",
            method = "PATCH",
            body = client.encodeBody(
                PatchBoardSectionBody(title = title, sortIndex = sortIndex),
                PatchBoardSectionBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun deleteSection(
        courseCode: String,
        boardId: String,
        sectionId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/sections/${encodePath(sectionId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun arrangePost(
        courseCode: String,
        boardId: String,
        postId: String,
        input: ArrangeBoardPostBody,
        accessToken: String,
    ): BoardPost = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/arrange",
            method = "PATCH",
            body = client.encodeBody(input, ArrangeBoardPostBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }
}
