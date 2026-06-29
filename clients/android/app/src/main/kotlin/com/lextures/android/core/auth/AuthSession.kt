package com.lextures.android.core.auth

import android.app.Application
import android.util.Base64
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.offline.OfflineService
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

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

    // Declared before init: the splash coroutine below can call refreshIfNeeded()
    // synchronously during construction (Main.immediate), and Kotlin runs property
    // initializers and init blocks strictly in declaration order.
    private val refreshMutex = Mutex()

    init {
        viewModelScope.launch {
            // Refresh in parallel with the splash so a stale 15-minute access
            // token is replaced before the app shows.
            val splash = launch { delay(SPLASH_DURATION_MS) }
            refreshIfNeeded()
            splash.join()
            finishSplash()
        }
        viewModelScope.launch {
            // Keep the access token fresh while the app process is alive.
            while (true) {
                delay(REFRESH_INTERVAL_MS)
                if (_phase.value == AuthPhase.Authenticated) {
                    refreshIfNeeded()
                }
            }
        }
    }

    fun finishSplash() {
        _phase.value = if (isSignedIn) AuthPhase.Authenticated else AuthPhase.Unauthenticated
    }

    /** Hook for ON_RESUME: the token has likely expired while backgrounded. */
    fun onAppResumed() {
        if (_phase.value == AuthPhase.Authenticated) {
            viewModelScope.launch { refreshIfNeeded() }
        }
    }

    // Token refresh
    //
    // Backend access tokens last 15 minutes by design; refresh tokens last 30 days
    // and rotate on every exchange. Refreshing keeps the mobile session alive for
    // weeks without weakening the backend's short-lived bearer tokens.

    /** Refreshes the access token when it is missing, expired, or expiring soon. */
    suspend fun refreshIfNeeded(force: Boolean = false) {
        refreshMutex.withLock {
            val refreshToken = tokenStore.readRefreshToken() ?: return
            if (!force) {
                val expiry = jwtExpiryEpochSeconds(_accessToken.value)
                if (expiry != null && expiry - System.currentTimeMillis() / 1000 > 120) return
            }
            try {
                val response = AuthApi.refresh(refreshToken)
                val token = response.accessToken?.takeIf { it.isNotBlank() } ?: return
                tokenStore.saveTokens(token, response.refreshToken ?: refreshToken)
                _accessToken.value = token
                response.user?.email?.let { _userEmail.value = it }
            } catch (e: ApiError.HttpStatus) {
                // Only an explicit rejection ends the session; network blips keep it.
                if (e.code == 401 || e.code == 403) {
                    signOut()
                }
            } catch (_: Exception) {
                // Transport/decoding errors: keep the session and retry later.
            }
        }
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
        OfflineService.get(getApplication()).clearAllOnLogout()
        tokenStore.clearAll()
        _accessToken.value = null
        _userEmail.value = null
        _phase.value = AuthPhase.Unauthenticated
    }

    fun serverUnreachableMessage(): String =
        "Could not reach the server at ${AppConfiguration.apiBaseUrl}. Is the API running?"

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
        private const val REFRESH_INTERVAL_MS = 10L * 60 * 1000

        /** Decodes the `exp` claim from a JWT without verifying the signature. */
        fun jwtExpiryEpochSeconds(jwt: String?): Long? {
            if (jwt.isNullOrBlank()) return null
            val segments = jwt.split(".")
            if (segments.size < 2) return null
            return runCatching {
                val payload = Base64.decode(
                    segments[1],
                    Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP,
                ).decodeToString()
                Regex(""""exp"\s*:\s*(\d+)""").find(payload)?.groupValues?.get(1)?.toLong()
            }.getOrNull()
        }
    }
}
