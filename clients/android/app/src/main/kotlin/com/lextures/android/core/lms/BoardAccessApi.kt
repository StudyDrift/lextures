package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board sharing, members, and public link resolve (VC.M6). Mirrors web `boards-api`. */
object BoardAccessApi {
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

    suspend fun patchBoardAccess(
        courseCode: String,
        boardId: String,
        visibility: String? = null,
        visibilityTarget: String? = null,
        attribution: String? = null,
        canPost: Boolean? = null,
        canInteract: Boolean? = null,
        canArrange: Boolean? = null,
        accessToken: String,
    ): Board = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}",
            method = "PATCH",
            body = client.encodeBody(
                PatchBoardBody(
                    visibility = visibility,
                    visibilityTarget = visibilityTarget,
                    attribution = attribution,
                    canPost = canPost,
                    canInteract = canInteract,
                    canArrange = canArrange,
                ),
                PatchBoardBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun listMembers(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): List<BoardMember> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/members",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardMembersListResponse>(body).members
    }

    suspend fun upsertMember(
        courseCode: String,
        boardId: String,
        userId: String,
        role: String,
        accessToken: String,
    ): BoardMember = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/members",
            method = "POST",
            body = client.encodeBody(
                UpsertBoardMemberBody(userId = userId, role = role),
                UpsertBoardMemberBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun removeMember(
        courseCode: String,
        boardId: String,
        userId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/members/${encodePath(userId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun listShares(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): List<BoardShare> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/shares",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardSharesListResponse>(body).shares
    }

    suspend fun createShare(
        courseCode: String,
        boardId: String,
        capability: String,
        password: String? = null,
        expiresAt: String? = null,
        accessToken: String,
    ): BoardShare = withContext(Dispatchers.IO) {
        val trimmed = password?.trim().orEmpty()
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/shares",
            method = "POST",
            body = client.encodeBody(
                CreateBoardShareBody(
                    capability = capability,
                    password = trimmed.ifEmpty { null },
                    expiresAt = expiresAt,
                ),
                CreateBoardShareBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun revokeShare(
        courseCode: String,
        boardId: String,
        shareId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/shares/${encodePath(shareId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    /** Public resolve — no auth token; password via header only (never logged). */
    suspend fun resolveBoardLink(token: String, password: String? = null): BoardLinkResolve =
        withContext(Dispatchers.IO) {
            val headers = buildMap {
                if (!password.isNullOrEmpty()) put("X-Board-Share-Password", password)
            }
            try {
                val (body, _) = client.requestRaw(
                    path = "/api/v1/board-links/${encodePath(token)}",
                    accessToken = null,
                    extraHeaders = headers,
                )
                decode(body)
            } catch (e: ApiError.HttpStatus) {
                throw e
            }
        }

    suspend fun createBoardLinkPost(
        token: String,
        displayName: String,
        text: String,
        password: String? = null,
    ): BoardPost = withContext(Dispatchers.IO) {
        val headers = buildMap {
            if (!password.isNullOrEmpty()) put("X-Board-Share-Password", password)
        }
        val trimmed = text.trim()
        val (body, code) = client.requestRaw(
            path = "/api/v1/board-links/${encodePath(token)}/posts",
            method = "POST",
            body = client.encodeBody(
                CreateBoardLinkPostBody(
                    displayName = displayName.trim(),
                    contentType = "text",
                    title = "",
                    body = BoardPostBody(html = trimmed, text = trimmed),
                ),
                CreateBoardLinkPostBody.serializer(),
            ),
            accessToken = null,
            extraHeaders = headers,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }
}
