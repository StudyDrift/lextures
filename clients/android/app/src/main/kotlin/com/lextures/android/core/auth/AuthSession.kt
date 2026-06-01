package com.lextures.android.core.auth

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

enum class AuthPhase {
    Splash,
    Unauthenticated,
    Authenticated,
}

class AuthSession(application: Application) : AndroidViewModel(application) {
    private val tokenStore = TokenStore(application)

    private val _phase = MutableStateFlow(AuthPhase.Splash)
    val phase: StateFlow<AuthPhase> = _phase.asStateFlow()

    private val _accessToken = MutableStateFlow(tokenStore.readAccessToken())
    val accessToken: StateFlow<String?> = _accessToken.asStateFlow()

    private val _userEmail = MutableStateFlow<String?>(null)
    val userEmail: StateFlow<String?> = _userEmail.asStateFlow()

    val isSignedIn: Boolean
        get() = !_accessToken.value.isNullOrBlank()

    init {
        viewModelScope.launch {
            delay(SPLASH_DURATION_MS)
            finishSplash()
        }
    }

    fun finishSplash() {
        _phase.value = if (isSignedIn) AuthPhase.Authenticated else AuthPhase.Unauthenticated
    }

    suspend fun applyTokenResponse(response: AuthTokenResponse) {
        if (response.requiresMfa == true && response.mfaPendingToken != null) {
            throw AuthSessionError.MfaRequired
        }

        val token = response.accessToken?.takeIf { it.isNotBlank() }
            ?: throw AuthSessionError.MissingAccessToken

        tokenStore.saveTokens(token, response.refreshToken)
        _accessToken.value = token
        _userEmail.value = response.user?.email
        _phase.value = AuthPhase.Authenticated
    }

    fun signOut() {
        tokenStore.clearAll()
        _accessToken.value = null
        _userEmail.value = null
        _phase.value = AuthPhase.Unauthenticated
    }

    fun serverUnreachableMessage(): String =
        "Could not reach the server. Is the API running?"

    fun mapError(error: Throwable): String {
        return when (error) {
            is AuthSessionError -> error.message ?: error.toString()
            is ApiError.Transport -> serverUnreachableMessage()
            is ApiError -> error.message ?: error.toString()
            else -> error.localizedMessage ?: error.toString()
        }
    }

    sealed class AuthSessionError(message: String) : Exception(message) {
        data object MfaRequired : AuthSessionError(
            "Multi-factor authentication is required. Complete sign-in on the web app for now.",
        )

        data object MissingAccessToken : AuthSessionError(
            "Unexpected sign-in response.",
        )
    }

    companion object {
        private const val SPLASH_DURATION_MS = 900L
    }
}
