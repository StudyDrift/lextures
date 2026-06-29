package com.lextures.android.core.i18n

import com.lextures.android.core.network.ApiClient
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

object LocaleApi {
    @Serializable
    data class LocaleRequest(val locale: String)

    @Serializable
    data class LocaleResponse(val locale: String? = null)

    private val json = Json { ignoreUnknownKeys = true }

    suspend fun saveLocale(tag: String, accessToken: String, client: ApiClient = ApiClient()): String =
        withContext(Dispatchers.IO) {
            val body = json.encodeToString(LocaleRequest.serializer(), LocaleRequest(tag))
            val (responseBody, _) = client.request(
                path = "/api/v1/settings/locale",
                method = "PUT",
                body = body,
                accessToken = accessToken,
            )
            val response = json.decodeFromString(LocaleResponse.serializer(), responseBody)
            response.locale?.trim()?.takeIf { it.isNotEmpty() } ?: tag
        }
}
