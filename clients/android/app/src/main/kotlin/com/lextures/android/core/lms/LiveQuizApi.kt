package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Interactive live quiz API (MOB.5 Phase 1). */
object LiveQuizApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    class LiveQuizJoinError(
        val status: Int,
        val reason: LiveQuizLogic.JoinErrorReason,
    ) : Exception(reason.name)

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    @Serializable
    private data class NicknameBody(val nickname: String)

    suspend fun lookupJoinCode(code: String): LiveQuizJoinLookup = withContext(Dispatchers.IO) {
        val normalized = LiveQuizLogic.normalizeJoinCode(code)
        val (body, status) = client.requestRaw(
            path = "/api/v1/live-quizzes/join/${encodePath(normalized)}",
            method = "GET",
        )
        if (status !in 200..299) {
            throw LiveQuizJoinError(
                status,
                LiveQuizLogic.joinErrorReason(status, parseApiErrorMessage(body) ?: body),
            )
        }
        json.decodeFromString(LiveQuizJoinLookup.serializer(), body)
    }

    suspend fun joinLiveGame(
        courseCode: String,
        gameId: String,
        nickname: String,
        accessToken: String,
    ): LiveQuizJoinPlayerResult = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/live-quizzes/games/${encodePath(gameId)}/players",
            method = "POST",
            body = json.encodeToString(NicknameBody.serializer(), NicknameBody(nickname)),
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw LiveQuizJoinError(
                status,
                LiveQuizLogic.joinErrorReason(status, parseApiErrorMessage(body) ?: body),
            )
        }
        json.decodeFromString(LiveQuizJoinPlayerResult.serializer(), body)
    }

    suspend fun joinLiveGameAsGuest(
        code: String,
        nickname: String,
    ): LiveQuizJoinPlayerResult = withContext(Dispatchers.IO) {
        val normalized = LiveQuizLogic.normalizeJoinCode(code)
        val (body, status) = client.requestRaw(
            path = "/api/v1/live-quizzes/join/${encodePath(normalized)}/players",
            method = "POST",
            body = json.encodeToString(NicknameBody.serializer(), NicknameBody(nickname)),
        )
        if (status !in 200..299) {
            throw LiveQuizJoinError(
                status,
                LiveQuizLogic.joinErrorReason(status, parseApiErrorMessage(body) ?: body),
            )
        }
        json.decodeFromString(LiveQuizJoinPlayerResult.serializer(), body)
    }

    suspend fun fetchMyGameResults(
        courseCode: String,
        gameId: String,
        accessToken: String,
    ): LiveQuizMyResults = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/live-quizzes/games/${encodePath(gameId)}/my-results",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(LiveQuizMyResults.serializer(), body)
    }

    suspend fun listQuizKits(
        courseCode: String,
        accessToken: String,
        page: Int = 1,
        pageSize: Int = 50,
    ): LiveQuizKitsListResult = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/live-quizzes/kits?page=$page&pageSize=$pageSize",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(LiveQuizKitsListResult.serializer(), body)
    }
}

@Serializable
data class LiveQuizKitSummary(
    val id: String,
    val title: String = "",
    val description: String? = null,
    val questionCount: Int? = null,
    val status: String? = null,
    val archived: Boolean? = null,
)

@Serializable
data class LiveQuizKitsListResult(
    val kits: List<LiveQuizKitSummary> = emptyList(),
    val total: Int? = null,
    val page: Int? = null,
    val pageSize: Int? = null,
    val totalPages: Int? = null,
)
