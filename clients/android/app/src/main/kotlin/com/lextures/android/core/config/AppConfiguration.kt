package com.lextures.android.core.config

import com.lextures.android.BuildConfig
import java.net.URL

/** Runtime API configuration. Override via `local.properties` (`API_BASE_URL`) or Gradle `-PAPI_BASE_URL`. */
object AppConfiguration {
    private const val DEFAULT_API_BASE = "http://10.0.2.2:8080"

    val apiBaseUrl: String
        get() {
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
