package com.lextures.android.core.lms

import java.net.URI

/**
 * Pure MB.1 link classification — mirrored on iOS. No networking / UI.
 * Uses [java.net.URI] so unit tests run on the JVM without Android stubs.
 */
object MobileLinkPolicy {
    enum class Handling(val wire: String) {
        IN_APP("in_app"),
        SYSTEM("system"),
        BLOCKED("blocked");

        companion object {
            fun parse(raw: String?): Handling = when (raw?.trim()?.lowercase()) {
                "system" -> SYSTEM
                "blocked" -> BLOCKED
                else -> IN_APP
            }
        }
    }

    enum class Classification(val wire: String) {
        NATIVE("native"),
        IN_APP_BROWSER("in_app_browser"),
        SYSTEM_BROWSER("system_browser"),
        EXTERNAL_APP("external_app"),
        AUTH_SESSION("auth_session"),
        BLOCKED("blocked"),
    }

    data class State(
        val handling: Handling = Handling.IN_APP,
        /** When false, external http(s) fall through to the system browser.
         *  Production always leaves this true; tests may force false. */
        val inAppBrowserEnabled: Boolean = true,
        /** Host of the configured API base (may include port). */
        val apiHost: String = "",
        val firstPartyHosts: List<String> = listOf("lextures.com", "localhost", "127.0.0.1"),
    )

    private val externalSchemes = setOf("mailto", "tel", "sms", "geo", "maps", "itms-apps", "market")
    private val webSchemes = setOf("http", "https")

    private val nativeAppHostSuffixes = listOf(
        "zoom.us", "zoom.com", "meet.google.com", "teams.microsoft.com", "teams.live.com",
    )

    private val checkoutHostSuffixes = listOf(
        "checkout.stripe.com", "billing.stripe.com", "pay.stripe.com",
    )

    private val ssoPathMarkers = listOf(
        "/oauth", "/oidc", "/saml", "/sso", "/auth/callback", "/login/oauth",
    )

    /** Pure classification of a URL string (absolute, relative `/…`, or `lextures://…`). */
    fun classify(urlString: String, state: State): Classification {
        val trimmed = urlString.trim()
        if (trimmed.isEmpty()) return Classification.BLOCKED

        if (trimmed.startsWith("/")) return Classification.NATIVE

        val lower = trimmed.lowercase()
        // Scheme without full parse for known custom schemes (URI may reject some).
        val scheme = schemeOf(trimmed)?.lowercase()
        if (scheme == null && !lower.startsWith("http://") && !lower.startsWith("https://")) {
            // bare relative already handled; unknown
            return Classification.BLOCKED
        }

        if (scheme == "lextures") return Classification.NATIVE
        if (scheme != null && scheme in externalSchemes) return Classification.EXTERNAL_APP
        if (scheme != null && scheme !in webSchemes) return Classification.BLOCKED

        val uri = parseUri(trimmed) ?: return Classification.BLOCKED
        val host = (uri.host ?: "").lowercase()
        if (host.isEmpty()) return Classification.BLOCKED

        if (isFirstParty(host, state)) return Classification.NATIVE
        if (isNativeAppHost(host)) return Classification.EXTERNAL_APP
        if (isCheckoutHost(host)) return Classification.SYSTEM_BROWSER

        val path = (uri.path ?: "").lowercase()
        if (isSSOPath(path)) return Classification.AUTH_SESSION

        return when (state.handling) {
            Handling.BLOCKED -> Classification.BLOCKED
            Handling.SYSTEM -> Classification.SYSTEM_BROWSER
            Handling.IN_APP ->
                if (state.inAppBrowserEnabled) Classification.IN_APP_BROWSER
                else Classification.SYSTEM_BROWSER
        }
    }

    /**
     * True when an Authorization bearer header may be attached for this request host.
     * Parsed-host equality / subdomain — never string-prefix on full URLs (FR-20).
     */
    fun shouldAttachBearer(requestHost: String?, apiHost: String): Boolean {
        val req = stripPort(requestHost.orEmpty()).lowercase()
        val api = stripPort(apiHost).lowercase()
        if (req.isEmpty() || api.isEmpty()) return false
        if (req == api) return true
        if (req.endsWith(".$api")) return true
        if (isLoopback(req) && isLoopback(api)) return true
        return false
    }

    fun displayHost(url: String): String {
        val host = (parseUri(url)?.host ?: "").lowercase()
        return if (host.startsWith("www.")) host.removePrefix("www.") else host
    }

    fun isSecure(url: String): Boolean =
        schemeOf(url)?.lowercase() == "https"

    /** Content-free telemetry payload (FR-25 / AC-20). */
    fun sanitizeTelemetry(raw: Map<String, String>): Map<String, String> {
        val allowed = setOf("source", "classification", "outcome", "errorClass")
        return raw.filterKeys { it in allowed }
    }

    private fun schemeOf(raw: String): String? {
        val idx = raw.indexOf(':')
        if (idx <= 0) return null
        val scheme = raw.substring(0, idx)
        if (!scheme.all { it.isLetter() || it == '+' || it == '.' || it == '-' }) return null
        return scheme
    }

    private fun parseUri(raw: String): URI? {
        return try {
            // java.net.URI is case-sensitive on scheme for some ops; normalize scheme casing.
            val scheme = schemeOf(raw)
            val normalized = if (scheme != null && scheme != scheme.lowercase()) {
                scheme.lowercase() + raw.substring(scheme.length)
            } else {
                raw
            }
            URI(normalized)
        } catch (_: Exception) {
            null
        }
    }

    private fun isFirstParty(host: String, state: State): Boolean {
        val h = host.lowercase()
        for (candidate in state.firstPartyHosts) {
            val c = candidate.lowercase()
            if (h == c || h.endsWith(".$c")) return true
        }
        val api = stripPort(state.apiHost).lowercase()
        if (api.isNotEmpty() && (h == api || h.endsWith(".$api"))) return true
        return false
    }

    private fun isNativeAppHost(host: String): Boolean {
        val h = host.lowercase()
        return nativeAppHostSuffixes.any { suffix -> h == suffix || h.endsWith(".$suffix") }
    }

    private fun isCheckoutHost(host: String): Boolean {
        val h = host.lowercase()
        return checkoutHostSuffixes.any { suffix -> h == suffix || h.endsWith(".$suffix") }
    }

    private fun isSSOPath(path: String): Boolean =
        ssoPathMarkers.any { path.contains(it) }

    private fun stripPort(host: String): String {
        val idx = host.indexOf(':')
        return if (idx >= 0) host.substring(0, idx) else host
    }

    private fun isLoopback(host: String): Boolean {
        val h = stripPort(host).lowercase()
        return h == "localhost" || h == "127.0.0.1" || h == "::1"
    }
}
