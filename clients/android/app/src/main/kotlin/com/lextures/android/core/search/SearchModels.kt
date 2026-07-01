package com.lextures.android.core.search

enum class SearchResultGroup {
    RecentSearch,
    RecentDestination,
    Action,
    Course,
    Content,
    Person,
}

data class SearchListItem(
    val id: String,
    val group: SearchResultGroup,
    val title: String,
    val subtitle: String,
    val path: String,
    val haystack: String = "$title $subtitle ${group.name}".lowercase(),
)

data class SearchResultSection(
    val group: SearchResultGroup,
    val items: List<SearchListItem>,
)

object SearchQueryEngine {
    const val DEBOUNCE_MS = 280L
    const val MIN_QUERY_LENGTH = 2
    const val MAX_RECENTS = 10

    fun shouldQuery(query: String): Boolean =
        query.trim().length >= MIN_QUERY_LENGTH
}