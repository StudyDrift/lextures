package com.lextures.android.features.search

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Bolt
import androidx.compose.material.icons.filled.History
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Description
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.search.SearchActionRegistry
import com.lextures.android.core.search.SearchListItem
import com.lextures.android.core.search.SearchQueryEngine
import com.lextures.android.core.search.SearchRecentsStore
import com.lextures.android.core.search.SearchResultGroup
import com.lextures.android.core.search.SearchResultSection
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UniversalSearchScreen(
    session: AuthSession,
    shell: HomeShellState,
    onDismiss: () -> Unit,
    courseScope: String? = null,
    isOnline: Boolean,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    var query by remember { mutableStateOf("") }
    var scopedToCourse by remember(courseScope) { mutableStateOf(courseScope != null) }
    var sections by remember { mutableStateOf<List<SearchResultSection>>(emptyList()) }
    var loading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var searchJob by remember { mutableStateOf<Job?>(null) }

    fun reloadRecents() {
        val searches = SearchRecentsStore.recentSearches(context).map { term ->
            SearchListItem(
                id = "recent-search:$term",
                group = SearchResultGroup.RecentSearch,
                title = term,
                subtitle = L.text(context, localePrefs, R.string.mobile_search_recentSearchSubtitle),
                path = "",
            )
        }
        val destinations = SearchRecentsStore.recentDestinations(context)
        sections = buildList {
            if (searches.isNotEmpty()) add(SearchResultSection(SearchResultGroup.RecentSearch, searches))
            if (destinations.isNotEmpty()) add(SearchResultSection(SearchResultGroup.RecentDestination, destinations))
        }
    }

    fun performSearch(trimmed: String, token: String) {
        scope.launch {
            loading = true
            errorMessage = null
            try {
                val effectiveScope = if (scopedToCourse) courseScope else null
                val response = LmsApi.fetchSearchQuery(trimmed, effectiveScope, token)
                val actions = SearchActionRegistry.buildActions(
                    context,
                    localePrefs,
                    shell.activeRoleContext,
                    shell.platformFeatures,
                )
                val matched = SearchActionRegistry.matchActions(trimmed, actions)
                val mapped = response.groups.mapNotNull { group ->
                    val mappedGroup = when (group.type) {
                        "course" -> SearchResultGroup.Course
                        "content" -> SearchResultGroup.Content
                        "person" -> SearchResultGroup.Person
                        else -> null
                    } ?: return@mapNotNull null
                    if (group.items.isEmpty()) return@mapNotNull null
                    SearchResultSection(
                        group = mappedGroup,
                        items = group.items.map { row ->
                            SearchListItem(
                                id = row.id,
                                group = mappedGroup,
                                title = row.title,
                                subtitle = row.subtitle,
                                path = row.path,
                            )
                        },
                    )
                }
                sections = buildList {
                    if (matched.isNotEmpty()) add(SearchResultSection(SearchResultGroup.Action, matched))
                    addAll(mapped)
                }
            } catch (_: Exception) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_search_error)
                reloadRecents()
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reloadRecents() }

    LaunchedEffect(query, scopedToCourse, isOnline) {
        searchJob?.cancel()
        val trimmed = query.trim()
        if (trimmed.isEmpty()) {
            loading = false
            errorMessage = null
            reloadRecents()
            return@LaunchedEffect
        }
        if (!isOnline) {
            reloadRecents()
            return@LaunchedEffect
        }
        if (!SearchQueryEngine.shouldQuery(trimmed)) {
            reloadRecents()
            return@LaunchedEffect
        }
        val token = accessToken ?: return@LaunchedEffect
        searchJob = scope.launch {
            delay(SearchQueryEngine.DEBOUNCE_MS)
            performSearch(trimmed, token)
        }
    }

    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_search_title)) },
                navigationIcon = {
                    TextButton(onClick = onDismiss) {
                        Text(L.text(context, localePrefs, R.string.mobile_ia_close))
                    }
                },
                actions = {
                    if (courseScope != null) {
                        FilterChip(
                            selected = scopedToCourse,
                            onClick = { scopedToCourse = !scopedToCourse },
                            label = { Text(L.text(context, localePrefs, R.string.mobile_search_inThisCourse)) },
                            modifier = Modifier.padding(end = 8.dp),
                        )
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
        ) {
            androidx.compose.material3.OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 8.dp),
                placeholder = { Text(L.text(context, localePrefs, R.string.mobile_search_prompt)) },
                singleLine = true,
            )

            if (!isOnline) {
                Text(
                    text = L.text(context, localePrefs, R.string.mobile_search_offlineNotice),
                    fontSize = 13.sp,
                    color = textSecondary(),
                    modifier = Modifier.padding(bottom = 8.dp),
                )
            }

            errorMessage?.let { LmsErrorBanner(message = it, modifier = Modifier.padding(bottom = 8.dp)) }

            when {
                loading && sections.isEmpty() -> LmsSkeletonList(count = 5)
                sections.isEmpty() && query.trim().isNotEmpty() && !loading -> {
                    LmsEmptyState(
                        icon = Icons.Default.Search,
                        title = L.text(context, localePrefs, R.string.mobile_search_noResultsTitle),
                        message = context.getString(R.string.mobile_search_noResultsMessage, query.trim()),
                    )
                }
                else -> {
                    LazyColumn(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        sections.forEach { section ->
                            item(key = "header-${section.group.name}") {
                                Text(
                                    text = groupLabel(context, localePrefs, section.group),
                                    fontSize = 13.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = textSecondary(),
                                    modifier = Modifier.padding(top = 12.dp, bottom = 4.dp),
                                )
                            }
                            items(section.items, key = { it.id }) { item ->
                                SearchResultRow(
                                    item = item,
                                    onClick = {
                                        if (item.group == SearchResultGroup.RecentSearch) {
                                            query = item.title
                                        } else if (item.path.isNotEmpty()) {
                                            SearchRecentsStore.recordDestination(context, item)
                                            val trimmed = query.trim()
                                            if (trimmed.isNotEmpty()) {
                                                SearchRecentsStore.recordSearch(context, trimmed)
                                            }
                                            shell.navigateFromSearch(item.path)
                                            onDismiss()
                                        }
                                    },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SearchResultRow(item: SearchListItem, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(
            imageVector = iconFor(item.group),
            contentDescription = null,
            tint = if (item.group == SearchResultGroup.Action) LexturesColors.Coral else LexturesColors.BrandTeal,
        )
        Column(modifier = Modifier.weight(1f)) {
            Text(text = item.title, fontSize = 15.sp, fontWeight = FontWeight.Medium, color = textPrimary())
            if (item.subtitle.isNotEmpty()) {
                Text(text = item.subtitle, fontSize = 12.sp, color = textSecondary(), maxLines = 2)
            }
        }
    }
}

private fun iconFor(group: SearchResultGroup): ImageVector = when (group) {
    SearchResultGroup.Action, SearchResultGroup.RecentDestination -> Icons.Default.Bolt
    SearchResultGroup.Course -> Icons.Default.MenuBook
    SearchResultGroup.Content -> Icons.Default.Description
    SearchResultGroup.Person -> Icons.Default.Person
    SearchResultGroup.RecentSearch -> Icons.Default.History
}

private fun groupLabel(context: android.content.Context, localePrefs: com.lextures.android.core.i18n.LocalePreferences, group: SearchResultGroup): String =
    when (group) {
        SearchResultGroup.RecentSearch -> L.text(context, localePrefs, R.string.mobile_search_group_recentSearches)
        SearchResultGroup.RecentDestination -> L.text(context, localePrefs, R.string.mobile_search_group_recentDestinations)
        SearchResultGroup.Action -> L.text(context, localePrefs, R.string.mobile_search_group_actions)
        SearchResultGroup.Course -> L.text(context, localePrefs, R.string.mobile_search_group_courses)
        SearchResultGroup.Content -> L.text(context, localePrefs, R.string.mobile_search_group_content)
        SearchResultGroup.Person -> L.text(context, localePrefs, R.string.mobile_search_group_people)
    }