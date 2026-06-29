package com.lextures.android.core.auth

import com.lextures.android.core.network.ApiClient
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

object AuthApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    suspend fun fetchSamlStatus(): SamlStatusResponse {
        return try {
            val (body, _) = client.request("/api/v1/auth/saml/status")
            json.decodeFromString(body)
        } catch (_: Exception) {
            SamlStatusResponse()
        }
    }

    suspend fun fetchOidcStatus(): OidcStatusResponse {
        return try {
            val (body, _) = client.request("/api/v1/auth/oidc/status")
            json.decodeFromString(body)
        } catch (_: Exception) {
            OidcStatusResponse()
        }
    }

    suspend fun requestMagicLink(email: String): MagicLinkRequestResponse {
        val body = client.encodeBody(
            MagicLinkRequest(email = email, redirectTo = "/"),
            MagicLinkRequest.serializer(),
        )
        val (response, _) = client.request(
            path = "/api/v1/auth/magic-link/request",
            method = "POST",
            body = body,
        )
        return json.decodeFromString(response)
    }

    suspend fun consumeMagicLink(token: String): AuthTokenResponse {
        val body = client.encodeBody(MagicLinkConsumeRequest(token), MagicLinkConsumeRequest.serializer())
        val (response, _) = client.request(
            path = "/api/v1/auth/magic-link/consume",
            method = "POST",
            body = body,
        )
        return json.decodeFromString(response)
    }

    suspend fun mfaTotpChallenge(code: String, mfaPendingToken: String): AuthTokenResponse {
        val body = client.encodeBody(MfaTotpChallengeRequest(code), MfaTotpChallengeRequest.serializer())
        val (response, _) = client.request(
            path = "/api/v1/auth/mfa/totp/challenge",
            method = "POST",
            body = body,
            accessToken = mfaPendingToken,
        )
        return json.decodeFromString(response)
    }

    suspend fun mfaTotpEnrol(mfaPendingToken: String): MfaTotpEnrolResponse {
        val (response, _) = client.request(
            path = "/api/v1/auth/mfa/totp/enrol",
            method = "POST",
            accessToken = mfaPendingToken,
        )
        return json.decodeFromString(response)
    }

    suspend fun mfaTotpVerifyEnrol(credentialId: String, code: String, mfaPendingToken: String) {
        val body = client.encodeBody(
            MfaTotpEnrolVerifyRequest(credentialId, code),
            MfaTotpEnrolVerifyRequest.serializer(),
        )
        client.request(
            path = "/api/v1/auth/mfa/totp/verify-enrol",
            method = "POST",
            body = body,
            accessToken = mfaPendingToken,
        )
    }

    suspend fun mfaSetupComplete(mfaPendingToken: String): AuthTokenResponse {
        val (response, _) = client.request(
            path = "/api/v1/auth/mfa/setup/complete",
            method = "POST",
            accessToken = mfaPendingToken,
        )
        return json.decodeFromString(response)
    }

    suspend fun mfaBackupChallenge(code: String, mfaPendingToken: String): AuthTokenResponse {
        val body = client.encodeBody(MfaBackupChallengeRequest(code), MfaBackupChallengeRequest.serializer())
        val (response, _) = client.request(
            path = "/api/v1/auth/mfa/backup/challenge",
            method = "POST",
            body = body,
            accessToken = mfaPendingToken,
        )
        return json.decodeFromString(response)
    }

    suspend fun mfaWebAuthnBegin(setup: Boolean, mfaPendingToken: String): MfaWebAuthnBeginResponse {
        val path = if (setup) {
            "/api/v1/auth/mfa/webauthn/register/begin"
        } else {
            "/api/v1/auth/mfa/webauthn/authenticate/begin"
        }
        val (response, _) = client.request(path = path, method = "POST", accessToken = mfaPendingToken)
        return json.decodeFromString(response)
    }

    suspend fun mfaWebAuthnComplete(
        setup: Boolean,
        sessionId: String,
        credentialJson: String,
        mfaPendingToken: String,
    ): AuthTokenResponse? {
        val path = if (setup) {
            "/api/v1/auth/mfa/webauthn/register/complete"
        } else {
            "/api/v1/auth/mfa/webauthn/authenticate/complete"
        }
        val credentialElement = json.parseToJsonElement(credentialJson)
        val body = buildJsonObject {
            put("session_id", sessionId)
            put("credential", credentialElement)
            if (setup) put("display_name", "")
        }.toString()
        val (response, _) = client.request(
            path = path,
            method = "POST",
            body = body,
            accessToken = mfaPendingToken,
        )
        return if (setup) null else json.decodeFromString(response)
    }

    suspend fun fetchPasswordPolicy(): PasswordPolicy {
        return try {
            val (body, _) = client.request("/api/v1/auth/password-policy")
            json.decodeFromString<PasswordPolicy>(body)
        } catch (_: Exception) {
            PasswordPolicy.fallback
        }
    }

    /** Exchanges a refresh token for a new access token (+ rotated refresh token). */
    suspend fun refresh(refreshToken: String): AuthTokenResponse {
        val body = client.encodeBody(RefreshRequest(refreshToken), RefreshRequest.serializer())
        val (response, _) = client.request(
            path = "/api/v1/auth/refresh",
            method = "POST",
            body = body,
        )
        return json.decodeFromString(response)
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
