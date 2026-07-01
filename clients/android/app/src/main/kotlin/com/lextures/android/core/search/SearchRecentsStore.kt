package com.lextures.android.core.search

import android.content.Context
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

object SearchRecentsStore {
    private const val PREFS = "mobile_search_prefs"
    private const val KEY_SEARCHES = "recent_queries"
    private const val KEY_DESTINATIONS = "recent_destinations"
    private val json = Json { ignoreUnknownKeys = true }

    fun recentSearches(context: Context): List<String> {
        val raw = prefs(context).getString(KEY_SEARCHES, null) ?: return emptyList()
        return raw.lineSequence().filter { it.isNotBlank() }.toList()
    }

    fun recentDestinations(context: Context): List<SearchListItem> {
        val raw = prefs(context).getString(KEY_DESTINATIONS, null) ?: return emptyList()
        return runCatching {
            json.decodeFromString<List<StoredDestination>>(raw).map { it.toItem() }
        }.getOrDefault(emptyList())
    }

    fun recordSearch(context: Context, query: String) {
        val trimmed = query.trim()
        if (trimmed.isEmpty()) return
        val items = recentSearches(context)
            .filterNot { it.equals(trimmed, ignoreCase = true) }
            .toMutableList()
        items.add(0, trimmed)
        prefs(context).edit()
            .putString(KEY_SEARCHES, items.take(SearchQueryEngine.MAX_RECENTS).joinToString("\n"))
            .apply()
    }

    fun recordDestination(context: Context, item: SearchListItem) {
        if (item.group == SearchResultGroup.RecentSearch || item.group == SearchResultGroup.RecentDestination) {
            return
        }
        val stored = StoredDestination.from(item)
        val items = recentDestinations(context)
            .filterNot { it.id == item.id }
            .map { StoredDestination.from(it) }
            .toMutableList()
        items.add(0, stored)
        prefs(context).edit()
            .putString(
                KEY_DESTINATIONS,
                json.encodeToString(items.take(SearchQueryEngine.MAX_RECENTS)),
            )
            .apply()
    }

    fun clearAll(context: Context) {
        prefs(context).edit()
            .remove(KEY_SEARCHES)
            .remove(KEY_DESTINATIONS)
            .apply()
    }

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    @Serializable
    private data class StoredDestination(
        val id: String,
        val group: String,
        val title: String,
        val subtitle: String,
        val path: String,
        val haystack: String,
    ) {
        fun toItem(): SearchListItem = SearchListItem(
            id = id,
            group = runCatching { SearchResultGroup.valueOf(group) }.getOrDefault(SearchResultGroup.RecentDestination),
            title = title,
            subtitle = subtitle,
            path = path,
            haystack = haystack,
        )

        companion object {
            fun from(item: SearchListItem) = StoredDestination(
                id = item.id,
                group = item.group.name,
                title = item.title,
                subtitle = item.subtitle,
                path = item.path,
                haystack = item.haystack,
            )
        }
    }
}