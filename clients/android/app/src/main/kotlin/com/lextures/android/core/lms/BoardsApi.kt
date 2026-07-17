package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Visual collaboration boards REST (VC.M1). Mirrors web `boards-api`. */
object BoardsApi {
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

    suspend fun listBoards(
        courseCode: String,
        includeArchived: Boolean = false,
        accessToken: String,
    ): List<Board> = withContext(Dispatchers.IO) {
        var path = "/api/v1/courses/${encodePath(courseCode)}/boards"
        if (includeArchived) path += "?includeArchived=true"
        val (body, code) = client.requestRaw(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardsListResponse>(body).boards
    }

    suspend fun createBoard(
        courseCode: String,
        title: String,
        description: String = "",
        accessToken: String,
    ): Board = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards",
            method = "POST",
            body = client.encodeBody(CreateBoardBody(title, description), CreateBoardBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchBoard(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): Board = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchBoard(
        courseCode: String,
        boardId: String,
        title: String? = null,
        description: String? = null,
        archived: Boolean? = null,
        layout: String? = null,
        layoutLocked: Boolean? = null,
        accessToken: String,
    ): Board = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}",
            method = "PATCH",
            body = client.encodeBody(
                PatchBoardBody(
                    title = title,
                    description = description,
                    archived = archived,
                    layout = layout,
                    layoutLocked = layoutLocked,
                ),
                PatchBoardBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun deleteBoard(
        courseCode: String,
        boardId: String,
        hard: Boolean = false,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        var path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}"
        if (hard) path += "?hard=true"
        val (body, code) = client.requestRaw(
            path = path,
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }
}
