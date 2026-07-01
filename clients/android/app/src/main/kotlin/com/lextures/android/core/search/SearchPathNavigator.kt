package com.lextures.android.core.search

import com.lextures.android.core.navigation.MoreDestination
import com.lextures.android.core.navigation.ShellTab
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.core.routing.DeepLinkRouter

sealed class SearchNavigationTarget {
    data class ShellTabTarget(val tab: ShellTab) : SearchNavigationTarget()
    data class DeepLinkTarget(val destination: DeepLinkDestination) : SearchNavigationTarget()
    data class MoreTarget(val destination: MoreDestination) : SearchNavigationTarget()
}

object SearchPathNavigator {
    fun resolve(rawPath: String): SearchNavigationTarget? {
        val trimmed = rawPath.trim()
        if (trimmed.isEmpty()) return null

        if (trimmed.startsWith("lextures://shell/")) {
            return when (trimmed.removePrefix("lextures://shell/").lowercase()) {
                "teach" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Teach)
                "children" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Children)
                "calendar" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Calendar)
                else -> null
            }
        }

        val path = trimmed.substringBefore('?')
        val segments = path.trim('/').split('/').filter { it.isNotEmpty() }

        if (segments.isEmpty()) {
            return SearchNavigationTarget.ShellTabTarget(ShellTab.Home)
        }

        return when (segments.first().lowercase()) {
            "courses" -> {
                if (segments.size == 1) {
                    SearchNavigationTarget.ShellTabTarget(ShellTab.Courses)
                } else {
                    SearchNavigationTarget.DeepLinkTarget(DeepLinkRouter.resolve(path))
                }
            }
            "inbox" -> SearchNavigationTarget.DeepLinkTarget(DeepLinkDestination.Inbox)
            "notebooks" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Notebooks)
            "calendar" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Calendar)
            "parent" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Children)
            "todos" -> SearchNavigationTarget.MoreTarget(MoreDestination.Planner)
            "portfolios" -> SearchNavigationTarget.MoreTarget(MoreDestination.Portfolio)
            "catalog" -> SearchNavigationTarget.MoreTarget(MoreDestination.Catalog)
            "paths" -> SearchNavigationTarget.MoreTarget(MoreDestination.Paths)
            "library" -> SearchNavigationTarget.MoreTarget(MoreDestination.Library)
            "reading" -> SearchNavigationTarget.MoreTarget(MoreDestination.Reading)
            "review" -> SearchNavigationTarget.DeepLinkTarget(DeepLinkDestination.Review)
            "credentials" -> SearchNavigationTarget.MoreTarget(MoreDestination.Credentials)
            "advising" -> SearchNavigationTarget.MoreTarget(MoreDestination.Advising)
            "peer-reviews" -> SearchNavigationTarget.MoreTarget(MoreDestination.PeerReviews)
            "report-cards" -> SearchNavigationTarget.MoreTarget(MoreDestination.ReportCards)
            "settings" -> SearchNavigationTarget.ShellTabTarget(ShellTab.Profile)
            else -> {
                val deep = DeepLinkRouter.resolve(path)
                if (deep is DeepLinkDestination.Home && segments.first().lowercase() != "courses") {
                    null
                } else {
                    SearchNavigationTarget.DeepLinkTarget(deep)
                }
            }
        }
    }
}