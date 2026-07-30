package com.lextures.android.core.routing

/**
 * In-app link allowlist + deep-link resolution for markdown hrefs (CT.M1 FR-9).
 * Mirrors iOS `ContentLinkRouter` scheme policy: http, https, mailto, and in-app paths.
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
            lower.startsWith("lextures://")
    }

    fun isExternalHref(href: String): Boolean {
        val lower = href.trim().lowercase()
        return lower.startsWith("http://") ||
            lower.startsWith("https://") ||
            lower.startsWith("mailto:")
    }

    /** Resolve markdown href into a deep-link destination when it is an in-app path/URL. */
    fun resolveDeepLink(href: String): DeepLinkDestination? {
        val t = href.trim()
        if (!isAllowedHref(t) || t.lowercase().startsWith("mailto:")) return null
        return DeepLinkRouter.resolve(t)
    }
}
