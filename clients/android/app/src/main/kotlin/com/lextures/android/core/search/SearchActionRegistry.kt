package com.lextures.android.core.search

import android.content.Context
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.navigation.MobileRoleContext
import com.lextures.android.core.navigation.MoreDestination
import com.lextures.android.core.navigation.ShellTab


object SearchActionRegistry {
    fun buildActions(
        context: Context,
        localePrefs: LocalePreferences,
        roleContext: MobileRoleContext,
        platform: MobilePlatformFeatures,
    ): List<SearchListItem> {
        val items = mutableListOf<SearchListItem>()

        MobileDestinations.shellTabs(roleContext).forEach { tab ->
            val label = shellTabLabel(context, localePrefs, tab)
            val path = shellTabPath(tab)
            items += SearchListItem(
                id = "action:tab:${tab.name}",
                group = SearchResultGroup.Action,
                title = context.getString(R.string.mobile_search_action_openTab, label),
                subtitle = L.text(context, localePrefs, R.string.mobile_search_action_jumpTo),
                path = path,
                haystack = "open $label ${tab.name} goto jump action".lowercase(),
            )
        }

        MobileDestinations.moreDestinations(roleContext, platform).forEach { destination ->
            val label = moreLabel(context, localePrefs, destination)
            items += SearchListItem(
                id = "action:more:${destination.name}",
                group = SearchResultGroup.Action,
                title = context.getString(R.string.mobile_search_action_openDestination, label),
                subtitle = L.text(context, localePrefs, R.string.mobile_search_action_moreHub),
                path = moreDestinationPath(destination),
                haystack = "open $label ${destination.name} goto jump action more".lowercase(),
            )
        }

        items += SearchListItem(
            id = "action:calendar",
            group = SearchResultGroup.Action,
            title = L.text(context, localePrefs, R.string.mobile_search_action_openCalendar),
            subtitle = L.text(context, localePrefs, R.string.mobile_search_action_calendarSubtitle),
            path = "/calendar",
            haystack = "calendar schedule open calendar goto jump action",
        )

        if (roleContext == MobileRoleContext.Teaching) {
            items += SearchListItem(
                id = "action:attendance",
                group = SearchResultGroup.Action,
                title = L.text(context, localePrefs, R.string.mobile_search_action_takeAttendance),
                subtitle = L.text(context, localePrefs, R.string.mobile_search_action_attendanceSubtitle),
                path = "/courses",
                haystack = "take attendance roll call goto jump action teach",
            )
        }

        if (roleContext == MobileRoleContext.Learning) {
            items += SearchListItem(
                id = "action:grades",
                group = SearchResultGroup.Action,
                title = L.text(context, localePrefs, R.string.mobile_search_action_myGrades),
                subtitle = L.text(context, localePrefs, R.string.mobile_search_action_gradesSubtitle),
                path = "/courses",
                haystack = "my grades scores transcript goto jump action",
            )
        }

        return items
    }

    fun matchActions(query: String, actions: List<SearchListItem>, limit: Int = 5): List<SearchListItem> {
        val trimmed = query.trim().lowercase()
        if (trimmed.isEmpty()) return emptyList()
        val tokens = trimmed.split(Regex("\\s+")).filter { it.isNotEmpty() }
        return actions
            .filter { item -> tokens.all { token -> item.haystack.contains(token) } }
            .take(limit)
    }

    private fun shellTabPath(tab: ShellTab): String = when (tab) {
        ShellTab.Home -> "/"
        ShellTab.Courses -> "/courses"
        ShellTab.Notebooks -> "/notebooks"
        ShellTab.Inbox -> "/inbox"
        ShellTab.Profile -> "/settings/account"
        ShellTab.Teach -> "lextures://shell/teach"
        ShellTab.Children -> "/parent"
        ShellTab.Calendar -> "/calendar"
    }

    private fun moreDestinationPath(destination: MoreDestination): String = when (destination) {
        MoreDestination.Calendar -> "/calendar"
        MoreDestination.Planner -> "/todos"
        MoreDestination.Catalog -> "/catalog"
        MoreDestination.Marketplace -> "/marketplace"
        MoreDestination.Paths -> "/paths"
        MoreDestination.Library -> "/library"
        MoreDestination.Reading -> "/reading"
        MoreDestination.Portfolio -> "/portfolios"
        MoreDestination.Credentials -> "/credentials"
        MoreDestination.Wallet -> "/wallet"
        MoreDestination.Gamification -> "/gamification"
        MoreDestination.Advising -> "/advising"
        MoreDestination.Settings -> "/settings/account"
        MoreDestination.AskAi -> "/ask-ai"
        MoreDestination.PeerReviews -> "/peer-reviews"
        MoreDestination.ReportCards -> "/report-cards"
        MoreDestination.Insights -> "/me/study-insights"
    }

    private fun shellTabLabel(context: Context, localePrefs: LocalePreferences, tab: ShellTab): String =
        L.text(context, localePrefs, shellTabLabelRes(tab))

    private fun shellTabLabelRes(tab: ShellTab): Int = when (tab) {
        ShellTab.Home -> R.string.tabs_home
        ShellTab.Courses -> R.string.tabs_courses
        ShellTab.Notebooks -> R.string.tabs_notebooks
        ShellTab.Inbox -> R.string.tabs_inbox
        ShellTab.Profile -> R.string.tabs_profile
        ShellTab.Teach -> R.string.mobile_ia_tabs_teach
        ShellTab.Children -> R.string.mobile_ia_tabs_children
        ShellTab.Calendar -> R.string.mobile_ia_tabs_calendar
    }

    private fun moreLabel(context: Context, localePrefs: LocalePreferences, destination: MoreDestination): String =
        L.text(context, localePrefs, moreLabelRes(destination))

    private fun moreLabelRes(destination: MoreDestination): Int = when (destination) {
        MoreDestination.Calendar -> R.string.mobile_ia_more_calendar
        MoreDestination.Planner -> R.string.mobile_ia_more_planner
        MoreDestination.Catalog -> R.string.mobile_ia_more_catalog
        MoreDestination.Marketplace -> R.string.mobile_ia_more_marketplace
        MoreDestination.Paths -> R.string.mobile_ia_more_paths
        MoreDestination.Library -> R.string.mobile_ia_more_library
        MoreDestination.Reading -> R.string.mobile_ia_more_reading
        MoreDestination.Portfolio -> R.string.mobile_ia_more_portfolio
        MoreDestination.Credentials -> R.string.mobile_ia_more_credentials
        MoreDestination.Wallet -> R.string.mobile_ia_more_wallet
        MoreDestination.Gamification -> R.string.mobile_ia_more_gamification
        MoreDestination.Advising -> R.string.mobile_ia_more_advising
        MoreDestination.Settings -> R.string.mobile_ia_more_settings
        MoreDestination.AskAi -> R.string.mobile_tutor_askAi
        MoreDestination.PeerReviews -> R.string.mobile_peerReview_title
        MoreDestination.ReportCards -> R.string.mobile_mastery_reportCards
        MoreDestination.Insights -> R.string.mobile_ia_more_insights
    }
}