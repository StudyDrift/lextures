package com.lextures.android.core.config

import com.lextures.android.BuildConfig
import java.net.URL

/**
 * Runtime API configuration.
 *
 * Priority:
 * 1. Environment chosen on the get-started screen ([EnvironmentStore])
 * 2. `BuildConfig.API_BASE_URL` from `local.properties` / Gradle `-PAPI_BASE_URL`
 * 3. Emulator loopback default
 */
object AppConfiguration {
    private const val DEFAULT_API_BASE = "http://10.0.2.2:8080"

    /**
     * Optional process-wide override set from [EnvironmentStore] (or tests).
     * Prefer calling [bindEnvironment] once at app start.
     */
    @Volatile
    var runtimeBaseUrl: String? = null

    fun bindEnvironment(store: EnvironmentStore) {
        runtimeBaseUrl = store.apiBaseUrl?.trimEnd('/')
    }

    val apiBaseUrl: String
        get() {
            val runtime = runtimeBaseUrl?.trim().orEmpty()
            if (runtime.isNotEmpty()) {
                return runtime.trimEnd('/')
            }
            val configured = BuildConfig.API_BASE_URL.trim()
            if (configured.isNotEmpty()) {
                return configured.trimEnd('/')
            }
            return DEFAULT_API_BASE
        }

    fun apiUrl(path: String): URL {
        val normalized = if (path.startsWith("/")) path else "/$path"
        return URL(apiBaseUrl + normalized)
    }

    /**
     * Public web pages (privacy/trust center, accessibility statement) are served
     * from the same origin as the API in this monorepo deployment.
     */
    fun webUrl(path: String): String {
        val normalized = if (path.startsWith("/")) path else "/$path"
        return apiBaseUrl + normalized
    }
}
