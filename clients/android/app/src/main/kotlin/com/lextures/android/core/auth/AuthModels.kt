package com.lextures.android.core.auth

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class PasswordPolicy(
    @SerialName("minLength") val minLength: Int = 8,
    @SerialName("requireUpper") val requireUpper: Boolean = false,
    @SerialName("requireLower") val requireLower: Boolean = false,
    @SerialName("requireDigit") val requireDigit: Boolean = false,
    @SerialName("requireSpecial") val requireSpecial: Boolean = false,
    @SerialName("checkHibp") val checkHibp: Boolean = true,
) {
    companion object {
        val fallback = PasswordPolicy()
    }
}

@Serializable
data class AuthUser(
    val email: String? = null,
    @SerialName("ui_theme") val uiTheme: String? = null,
    val locale: String? = null,
    @SerialName("account_type") val accountType: String? = null,
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class SignupRequest(
    val email: String,
    val password: String,
    @SerialName("display_name") val displayName: String? = null,
    @SerialName("account_type") val accountType: String? = null,
    val timezone: String? = null,
)

@Serializable
data class AuthTokenResponse(
    @SerialName("access_token") val accessToken: String? = null,
    @SerialName("refresh_token") val refreshToken: String? = null,
    @SerialName("expires_in") val expiresIn: Int? = null,
    @SerialName("requires_mfa") val requiresMfa: Boolean? = null,
    @SerialName("mfa_pending_token") val mfaPendingToken: String? = null,
    @SerialName("mfa_setup_required") val mfaSetupRequired: Boolean? = null,
    val user: AuthUser? = null,
)
