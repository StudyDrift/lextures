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
data class RefreshRequest(
    @SerialName("refresh_token") val refreshToken: String,
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

@Serializable
data class SamlStatusResponse(
    val enabled: Boolean = false,
    val idp: SamlIdpInfo? = null,
) {
    @Serializable
    data class SamlIdpInfo(
        val id: String,
        val label: String,
        @SerialName("forceSaml") val forceSaml: Boolean = false,
    )
}

@Serializable
data class OidcStatusResponse(
    val enabled: Boolean = false,
    @SerialName("cleverEnabled") val cleverEnabled: Boolean? = null,
    @SerialName("classlinkEnabled") val classlinkEnabled: Boolean? = null,
    val clever: Boolean? = null,
    val classlink: Boolean? = null,
    val google: Boolean? = null,
    val microsoft: Boolean? = null,
    val apple: Boolean? = null,
    val custom: List<OidcCustomProvider>? = null,
) {
    @Serializable
    data class OidcCustomProvider(
        val id: String,
        @SerialName("displayName") val displayName: String,
    )

    val showsClever: Boolean get() = cleverEnabled == true || clever == true
    val showsClassLink: Boolean get() = classlinkEnabled == true || classlink == true
}

@Serializable
data class MagicLinkRequest(
    val email: String,
    @SerialName("redirect_to") val redirectTo: String? = null,
)

@Serializable
data class MagicLinkRequestResponse(
    val message: String? = null,
)

@Serializable
data class MagicLinkConsumeRequest(
    val token: String,
)

@Serializable
data class MfaTotpChallengeRequest(
    val code: String,
)

@Serializable
data class MfaTotpEnrolVerifyRequest(
    @SerialName("credential_id") val credentialId: String,
    val code: String,
)

@Serializable
data class MfaBackupChallengeRequest(
    val code: String,
)

@Serializable
data class MfaTotpEnrolResponse(
    @SerialName("credential_id") val credentialId: String? = null,
    @SerialName("otpauth_uri") val otpauthUri: String? = null,
)

@Serializable
data class MfaWebAuthnBeginResponse(
    @SerialName("session_id") val sessionId: String? = null,
    val options: kotlinx.serialization.json.JsonObject? = null,
)
