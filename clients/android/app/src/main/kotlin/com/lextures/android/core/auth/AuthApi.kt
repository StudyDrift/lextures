package com.lextures.android.core.auth

import com.lextures.android.core.network.ApiClient
import kotlinx.serialization.json.Json

object AuthApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    suspend fun fetchPasswordPolicy(): PasswordPolicy {
        return try {
            val (body, _) = client.request("/api/v1/auth/password-policy")
            json.decodeFromString<PasswordPolicy>(body)
        } catch (_: Exception) {
            PasswordPolicy.fallback
        }
    }

    suspend fun login(email: String, password: String): AuthTokenResponse {
        val body = client.encodeBody(LoginRequest(email = email, password = password), LoginRequest.serializer())
        val (response, _) = client.request(
            path = "/api/v1/auth/login",
            method = "POST",
            body = body,
        )
        return json.decodeFromString(response)
    }

    suspend fun signup(
        email: String,
        password: String,
        displayName: String?,
        registerAsParent: Boolean,
        timezone: String?,
    ): AuthTokenResponse {
        val request = SignupRequest(
            email = email,
            password = password,
            displayName = displayName?.takeIf { it.isNotBlank() },
            accountType = if (registerAsParent) "parent" else null,
            timezone = timezone?.takeIf { it.isNotBlank() },
        )
        val body = client.encodeBody(request, SignupRequest.serializer())
        val (response, _) = client.request(
            path = "/api/v1/auth/signup",
            method = "POST",
            body = body,
        )
        return json.decodeFromString(response)
    }
}
