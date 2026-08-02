package com.lextures.android.core.routing

import android.content.Context
import android.content.Intent
import android.net.Uri
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.lms.MobileLinkPolicy
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.features.home.HomeShellState

/**
 * Single entry point for opening URLs from app UI (MB.1 FR-1).
 * Feature code MUST call this instead of `Intent.ACTION_VIEW`.
 */
object LinkOpener {
    /**
     * Optional shell registered by [com.lextures.android.features.home.HomeScreen]
     * so feature call sites without an explicit shell still present the in-app browser.
     */
    @Volatile
    var activeShell: HomeShellState? = null

    data class Request(
        val urlString: String,
        val source: String,
        val forceSystemBrowser: Boolean = false,
        val accessToken: String? = null,
        val allowReport: Boolean = false,
    )

    data class BrowserSession(
        val initialUrl: String,
        val accessToken: String? = null,
        val source: String = "unknown",
        val allowReport: Boolean = false,
    )

    @JvmStatic
    fun policyState(features: MobilePlatformFeatures): MobileLinkPolicy.State =
        MobileLinkPolicy.State(
            handling = MobileLinkPolicy.Handling.parse(features.mobileLinkHandling),
            // MB.1: in-app browser is always on; org/platform policy uses mobileLinkHandling only.
            inAppBrowserEnabled = true,
            apiHost = Uri.parse(AppConfiguration.apiBaseUrl).host.orEmpty(),
        )

    @JvmStatic
    fun open(context: Context, request: Request, shell: HomeShellState?): MobileLinkPolicy.Classification {
        val resolvedShell = shell ?: activeShell
        val features = resolvedShell?.platformFeatures ?: MobilePlatformFeatures()
        val state = policyState(features)
        var classification = MobileLinkPolicy.classify(request.urlString, state)
        if (request.forceSystemBrowser &&
            (classification == MobileLinkPolicy.Classification.IN_APP_BROWSER ||
                classification == MobileLinkPolicy.Classification.SYSTEM_BROWSER)
        ) {
            classification = MobileLinkPolicy.Classification.SYSTEM_BROWSER
        }

        when (classification) {
            MobileLinkPolicy.Classification.NATIVE -> {
                val dest = DeepLinkRouter.resolve(request.urlString)
                resolvedShell?.openDeepLink(dest)
                emit(request.source, "internal", "native")
            }
            MobileLinkPolicy.Classification.IN_APP_BROWSER -> {
                val url = resolveUrl(request.urlString) ?: run {
                    emit(request.source, "external", "error", "bad_url")
                    return MobileLinkPolicy.Classification.BLOCKED
                }
                if (resolvedShell != null) {
                    resolvedShell.presentInAppBrowser(
                        BrowserSession(
                            initialUrl = url,
                            accessToken = request.accessToken,
                            source = request.source,
                            allowReport = request.allowReport,
                        ),
                    )
                    emit(request.source, "external", "in_app")
                } else {
                    // No shell available — degrade to system browser rather than drop the tap.
                    openSystem(context, url)
                    emit(request.source, "external", "system")
                }
            }
            MobileLinkPolicy.Classification.SYSTEM_BROWSER,
            MobileLinkPolicy.Classification.EXTERNAL_APP,
            MobileLinkPolicy.Classification.AUTH_SESSION,
            -> {
                val url = resolveUrl(request.urlString) ?: run {
                    emit(request.source, "external", "error", "bad_url")
                    return MobileLinkPolicy.Classification.BLOCKED
                }
                openSystem(context, url)
                val outcome = when (classification) {
                    MobileLinkPolicy.Classification.EXTERNAL_APP -> "external_app"
                    MobileLinkPolicy.Classification.AUTH_SESSION -> "auth_session"
                    else -> "system"
                }
                emit(request.source, "external", outcome)
            }
            MobileLinkPolicy.Classification.BLOCKED -> {
                resolvedShell?.presentLinkBlockedNotice()
                emit(request.source, "external", "blocked")
            }
        }
        return classification
    }

    @JvmStatic
    fun open(context: Context, urlString: String, shell: HomeShellState?, source: String): MobileLinkPolicy.Classification =
        open(context, Request(urlString = urlString, source = source), shell)

    fun resolveUrl(urlString: String): String? {
        val t = urlString.trim()
        if (t.isEmpty()) return null
        if (t.startsWith("/")) return AppConfiguration.webUrl(t)
        return runCatching { Uri.parse(t).toString() }.getOrNull()
    }

    private fun openSystem(context: Context, url: String) {
        runCatching {
            context.startActivity(
                Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            )
        }
    }

    private fun emit(source: String, classification: String, outcome: String, errorClass: String? = null) {
        val raw = buildMap {
            put("source", source)
            put("classification", classification)
            put("outcome", outcome)
            if (errorClass != null) put("errorClass", errorClass)
        }
        InAppBrowserTelemetry.record(MobileLinkPolicy.sanitizeTelemetry(raw))
    }
}

object InAppBrowserTelemetry {
    private val counts = mutableMapOf<String, Int>()

    fun record(payload: Map<String, String>) {
        val key = listOf(
            payload["source"] ?: "unknown",
            payload["classification"] ?: "unknown",
            payload["outcome"] ?: "unknown",
        ).joinToString("|")
        counts[key] = (counts[key] ?: 0) + 1
    }

    fun resetForTests() = counts.clear()
    fun snapshotForTests(): Map<String, Int> = counts.toMap()
}
