package com.lextures.android.core.routing

import android.content.Context
import com.lextures.android.features.home.HomeShellState

/**
 * In-app link allowlist + deep-link resolution for markdown hrefs (CT.M1 FR-9 / MB.1).
 * Opening is delegated to [LinkOpener].
 */
object ContentLinkRouter {
    fun isAllowedHref(href: String): Boolean {
        val t = href.trim()
        if (t.isEmpty()) return false
        if (t.startsWith("/")) return true
        val lower = t.lowercase()
        return lower.startsWith("http://") ||
            lower.startsWith("https://") ||
            lower.startsWith("mailto:") ||
            lower.startsWith("tel:") ||
            lower.startsWith("sms:") ||
            lower.startsWith("lextures://")
    }

    fun isExternalHref(href: String): Boolean {
        val lower = href.trim().lowercase()
        return lower.startsWith("http://") ||
            lower.startsWith("https://") ||
            lower.startsWith("mailto:") ||
            lower.startsWith("tel:")
    }

    /** Resolve markdown href into a deep-link destination when it is an in-app path/URL. */
    fun resolveDeepLink(href: String): DeepLinkDestination? {
        val t = href.trim()
        if (!isAllowedHref(t) || t.lowercase().startsWith("mailto:") || t.lowercase().startsWith("tel:")) {
            return null
        }
        return DeepLinkRouter.resolve(t)
    }

    /** Open via LinkOpener (MB.1 single entry point). */
    fun open(context: Context, href: String, shell: HomeShellState?, source: String = "content") {
        if (!isAllowedHref(href)) return
        LinkOpener.open(context, href, shell, source)
    }
}
