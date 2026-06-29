package com.lextures.android.core.auth

import android.content.Context
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

/** Optional fingerprint/face unlock after the app has been backgrounded past a timeout. */
class BiometricGate private constructor(context: Context) {
    private val appContext = context.applicationContext
    private val prefs = appContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    private val _isLocked = MutableStateFlow(false)
    val isLocked: StateFlow<Boolean> = _isLocked.asStateFlow()

    var isEnabled: Boolean
        get() = prefs.getBoolean(KEY_ENABLED, false)
        set(value) {
            prefs.edit().putBoolean(KEY_ENABLED, value).apply()
            if (!value) {
                _isLocked.value = false
                backgroundedAtMillis = null
            }
        }

    private var backgroundedAtMillis: Long? = null

    val canEnableBiometrics: Boolean
        get() = BiometricManager.from(appContext)
            .canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG) ==
            BiometricManager.BIOMETRIC_SUCCESS

    fun biometryLabel(context: Context): String = when {
        BiometricManager.from(appContext)
            .canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG) !=
            BiometricManager.BIOMETRIC_SUCCESS -> context.getString(com.lextures.android.R.string.mobile_biometric_generic)
        else -> context.getString(com.lextures.android.R.string.mobile_biometric_generic)
    }

    fun recordBackground(atMillis: Long = System.currentTimeMillis()) {
        if (!isEnabled) return
        backgroundedAtMillis = atMillis
    }

    fun evaluateOnForeground(nowMillis: Long = System.currentTimeMillis()) {
        if (!isEnabled) return
        val backgroundedAt = backgroundedAtMillis ?: return
        backgroundedAtMillis = null
        if (shouldLock(nowMillis - backgroundedAt)) {
            _isLocked.value = true
        }
    }

    suspend fun unlock(activity: FragmentActivity): Boolean {
        if (!isEnabled) {
            _isLocked.value = false
            return true
        }
        val executor = ContextCompat.getMainExecutor(activity)
        return suspendCancellableCoroutine { continuation ->
            val prompt = BiometricPrompt(
                activity,
                executor,
                object : BiometricPrompt.AuthenticationCallback() {
                    override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                        _isLocked.value = false
                        if (continuation.isActive) continuation.resume(true)
                    }

                    override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                        if (continuation.isActive) continuation.resume(false)
                    }

                    override fun onAuthenticationFailed() {
                        // Keep waiting for another attempt.
                    }
                },
            )
            val info = BiometricPrompt.PromptInfo.Builder()
                .setTitle(activity.getString(com.lextures.android.R.string.mobile_biometric_lockedTitle))
                .setSubtitle(activity.getString(com.lextures.android.R.string.mobile_biometric_reason))
                .setAllowedAuthenticators(
                    BiometricManager.Authenticators.BIOMETRIC_STRONG or
                        BiometricManager.Authenticators.DEVICE_CREDENTIAL,
                )
                .build()
            prompt.authenticate(info)
            continuation.invokeOnCancellation { prompt.cancelAuthentication() }
        }
    }

    fun resetOnSignOut() {
        _isLocked.value = false
        backgroundedAtMillis = null
    }

    companion object {
        const val LOCK_TIMEOUT_MS = 60_000L
        private const val PREFS_NAME = "lextures_biometric"
        private const val KEY_ENABLED = "enabled"

        @Volatile
        private var instance: BiometricGate? = null

        fun get(context: Context): BiometricGate =
            instance ?: synchronized(this) {
                instance ?: BiometricGate(context.applicationContext).also { instance = it }
            }

        fun shouldLock(backgroundDurationMs: Long): Boolean =
            backgroundDurationMs >= LOCK_TIMEOUT_MS
    }
}
