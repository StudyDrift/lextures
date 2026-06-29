package com.lextures.android.core.auth

import com.lextures.android.core.network.ApiClient
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class ActiveSession(
    val id: String,
    @SerialName("createdAt") val createdAt: String,
    @SerialName("lastUsedAt") val lastUsedAt: String,
    @SerialName("deviceLabel") val deviceLabel: String,
    val location: String,
    @SerialName("authMethod") val authMethod: String,
    @SerialName("isCurrent") val isCurrent: Boolean,
)

@Serializable
private data class SessionsResponse(
    val sessions: List<ActiveSession>,
)

/** Device session list/revoke endpoints (`GET/DELETE /api/v1/me/sessions`). */
object SessionsApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    suspend fun fetchSessions(accessToken: String): List<ActiveSession> {
        val (body, _) = client.request(
            path = "/api/v1/me/sessions",
            accessToken = accessToken,
        )
        return json.decodeFromString(SessionsResponse.serializer(), body).sessions
    }

    suspend fun revokeSession(id: String, accessToken: String) {
        client.request(
            path = "/api/v1/me/sessions/$id",
            method = "DELETE",
            accessToken = accessToken,
        )
    }

    suspend fun revokeOtherSessions(accessToken: String) {
        client.request(
            path = "/api/v1/me/sessions",
            method = "DELETE",
            accessToken = accessToken,
        )
    }
}
