package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board analytics & org governance REST (MOB.8 / VC.10). */
object BoardAnalyticsApi {
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

    suspend fun fetchBoardAnalytics(
        courseCode: String,
        boardId: String,
        days: Int = 14,
        accessToken: String,
    ): BoardAnalyticsSummary = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/analytics?days=$days",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchAdminPolicies(
        orgId: String? = null,
        accessToken: String,
    ): BoardOrgPolicies = withContext(Dispatchers.IO) {
        var path = "/api/v1/admin/boards/policies"
        if (!orgId.isNullOrBlank()) path += "?orgId=${encodePath(orgId)}"
        val (body, code) = client.requestRaw(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchAdminPolicies(
        body: PatchBoardOrgPoliciesBody,
        orgId: String? = null,
        accessToken: String,
    ): BoardOrgPolicies = withContext(Dispatchers.IO) {
        var path = "/api/v1/admin/boards/policies"
        if (!orgId.isNullOrBlank()) path += "?orgId=${encodePath(orgId)}"
        val (resp, code) = client.requestRaw(
            path = path,
            method = "PATCH",
            body = client.encodeBody(body, PatchBoardOrgPoliciesBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode(resp)
    }

    suspend fun fetchAdminOverview(
        orgId: String? = null,
        activeDays: Int = 30,
        accessToken: String,
    ): BoardAdminOverview = withContext(Dispatchers.IO) {
        val items = mutableListOf("activeDays=$activeDays")
        if (!orgId.isNullOrBlank()) items += "orgId=${encodePath(orgId)}"
        val path = "/api/v1/admin/boards/overview?${items.joinToString("&")}"
        val (body, code) = client.requestRaw(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }
}
