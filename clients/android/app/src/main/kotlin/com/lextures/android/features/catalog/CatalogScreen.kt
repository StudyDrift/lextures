package com.lextures.android.features.catalog

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import kotlinx.coroutines.launch
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CatalogBrowseTab
import com.lextures.android.core.lms.CatalogCategory
import com.lextures.android.core.lms.CatalogLevelFilter
import com.lextures.android.core.lms.CatalogLogic
import com.lextures.android.core.lms.CatalogPriceFilter
import com.lextures.android.core.lms.CatalogSortMode
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PublicCatalogCourse
import com.lextures.android.core.lms.PublicCatalogSearchResponse
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.CourseHeroImage
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.features.paths.PathsCatalogScreen
import kotlinx.serialization.serializer

@Composable
fun CatalogScreen(
    session: AuthSession,
    shell: HomeShellState,
    onOpenCourse: (String) -> Unit,
    onOpenPath: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }
    val scope = rememberCoroutineScope()

    var tab by remember { mutableStateOf(CatalogBrowseTab.Courses) }
    var query by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("") }
    var level by remember { mutableStateOf(CatalogLevelFilter.Any) }
    var price by remember { mutableStateOf(CatalogPriceFilter.Any) }
    var sort by remember { mutableStateOf(CatalogSortMode.Popular) }
    var categories by remember { mutableStateOf<List<CatalogCategory>>(emptyList()) }
    var courses by remember { mutableStateOf<List<PublicCatalogCourse>>(emptyList()) }
    var nextCursor by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var loadingMore by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var hasSearched by remember { mutableStateOf(false) }

    suspend fun searchCourses(reset: Boolean) {
        if (tab != CatalogBrowseTab.Courses) return
        if (reset) loading = true
        errorMessage = null
        hasSearched = true
        try {
            val cacheKey = CatalogLogic.cacheKey(query, category, level, price, sort)
            val response: PublicCatalogSearchResponse = if (accessToken != null) {
                offline.cachedFetch(
                    key = OfflineCacheKey.catalogCourses(cacheKey),
                    accessToken = accessToken!!,
                    serializer = serializer<PublicCatalogSearchResponse>(),
                ) {
                    LmsApi.fetchPublicCatalogCourses(
                        query = query,
                        category = category,
                        level = level.queryValue.orEmpty(),
                        sort = sort.apiValue,
                        priceMax = price.priceMax,
                        accessToken = accessToken,
                    )
                }.first
            } else {
                LmsApi.fetchPublicCatalogCourses(
                    query = query,
                    category = category,
                    level = level.queryValue.orEmpty(),
                    sort = sort.apiValue,
                    priceMax = price.priceMax,
                )
            }
            var page = response.courses
            if (price == CatalogPriceFilter.Paid) {
                page = page.filter { CatalogLogic.isPaid(it.priceCents) }
            }
            courses = page
            nextCursor = response.nextCursor
        } catch (e: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_catalog_error)
            if (reset) courses = emptyList()
        } finally {
            if (reset) loading = false
        }
    }

    LaunchedEffect(accessToken) {
        categories = runCatching { LmsApi.fetchPublicCatalogCategories(accessToken) }.getOrDefault(emptyList())
    }

    LaunchedEffect(accessToken, tab, query, category, level, price, sort) {
        if (tab == CatalogBrowseTab.Courses) searchCourses(reset = true)
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (shell.platformFeatures.ffLearningPaths) {
            LmsSegmentedChips(
                options = listOf(
                    "courses" to L.text(context, localePrefs, R.string.mobile_catalog_tab_courses),
                    "paths" to L.text(context, localePrefs, R.string.mobile_catalog_tab_paths),
                ),
                selectedId = tab.name.lowercase(),
                onSelect = { id ->
                    tab = if (id == "paths") CatalogBrowseTab.Paths else CatalogBrowseTab.Courses
                },
            )
        }

        when (tab) {
            CatalogBrowseTab.Paths -> PathsCatalogScreen(
                session = session,
                onOpenPath = onOpenPath,
                modifier = Modifier.fillMaxSize(),
            )
            CatalogBrowseTab.Courses -> {
                OutlinedTextField(
                    value = query,
                    onValueChange = { query = it },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text(L.text(context, localePrefs, R.string.mobile_catalog_search)) },
                    leadingIcon = { androidx.compose.material3.Icon(Icons.Default.Search, contentDescription = null) },
                    singleLine = true,
                )

                CatalogFilterRow(
                    categories = categories,
                    category = category,
                    onCategory = { category = it },
                    level = level,
                    onLevel = { level = it },
                    price = price,
                    onPrice = { price = it },
                    sort = sort,
                    onSort = { sort = it },
                )

                errorMessage?.let { LmsErrorBanner(message = it) }

                when {
                    loading && courses.isEmpty() -> LmsSkeletonList(count = 4)
                    courses.isEmpty() -> LmsEmptyState(
                        icon = Icons.Default.Search,
                        title = L.text(context, localePrefs, R.string.mobile_catalog_emptyTitle),
                        message = if (hasSearched) {
                            L.text(context, localePrefs, R.string.mobile_catalog_emptyMessage)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_catalog_prompt)
                        },
                    )
                    else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        items(courses, key = { it.id }) { course ->
                            LmsCard(onClick = { onOpenCourse(course.slug) }) {
                                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                    CourseHeroImage(
                                        url = course.heroImageUrl,
                                        fallbackKey = course.courseCode,
                                        accessToken = accessToken,
                                        height = 120.dp,
                                    )
                                    Text(course.title, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                                    course.instructorName?.takeIf { it.isNotBlank() }?.let {
                                        Text(it, fontSize = 12.sp, color = textSecondary())
                                    }
                                    Row(modifier = Modifier.fillMaxWidth()) {
                                        Text(
                                            CatalogLogic.formatPrice(course.priceCents),
                                            fontSize = 12.sp,
                                            fontWeight = FontWeight.SemiBold,
                                            color = textPrimary(),
                                        )
                                    }
                                }
                            }
                        }
                        if (nextCursor.isNotBlank()) {
                            item {
                                Button(
                                    onClick = {
                                        scope.launch {
                                            loadingMore = true
                                            try {
                                                val response = LmsApi.fetchPublicCatalogCourses(
                                                    query = query,
                                                    category = category,
                                                    level = level.queryValue.orEmpty(),
                                                    sort = sort.apiValue,
                                                    priceMax = price.priceMax,
                                                    cursor = nextCursor,
                                                    accessToken = accessToken,
                                                )
                                                var page = response.courses
                                                if (price == CatalogPriceFilter.Paid) {
                                                    page = page.filter { CatalogLogic.isPaid(it.priceCents) }
                                                }
                                                courses = courses + page
                                                nextCursor = response.nextCursor
                                            } catch (e: Exception) {
                                                errorMessage = L.text(context, localePrefs, R.string.mobile_catalog_error)
                                            } finally {
                                                loadingMore = false
                                            }
                                        }
                                    },
                                    enabled = !loadingMore,
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    Text(
                                        if (loadingMore) {
                                            L.text(context, localePrefs, R.string.mobile_catalog_loadingMore)
                                        } else {
                                            L.text(context, localePrefs, R.string.mobile_catalog_loadMore)
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
}

@Composable
private fun CatalogFilterRow(
    categories: List<CatalogCategory>,
    category: String,
    onCategory: (String) -> Unit,
    level: CatalogLevelFilter,
    onLevel: (CatalogLevelFilter) -> Unit,
    price: CatalogPriceFilter,
    onPrice: (CatalogPriceFilter) -> Unit,
    sort: CatalogSortMode,
    onSort: (CatalogSortMode) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var categoryMenu by remember { mutableStateOf(false) }
    var levelMenu by remember { mutableStateOf(false) }
    var priceMenu by remember { mutableStateOf(false) }
    var sortMenu by remember { mutableStateOf(false) }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        TextButton(onClick = { categoryMenu = true }) {
            Text(if (category.isBlank()) L.text(context, localePrefs, R.string.mobile_catalog_filter_categoryAny) else category)
        }
        DropdownMenu(expanded = categoryMenu, onDismissRequest = { categoryMenu = false }) {
            DropdownMenuItem(
                text = { Text(L.text(context, localePrefs, R.string.mobile_catalog_filter_categoryAny)) },
                onClick = { onCategory(""); categoryMenu = false },
            )
            categories.forEach { item ->
                DropdownMenuItem(
                    text = { Text(item.category) },
                    onClick = { onCategory(item.category); categoryMenu = false },
                )
            }
        }

        TextButton(onClick = { levelMenu = true }) {
            Text(levelLabel(context, localePrefs, level))
        }
        DropdownMenu(expanded = levelMenu, onDismissRequest = { levelMenu = false }) {
            CatalogLevelFilter.entries.forEach { item ->
                DropdownMenuItem(
                    text = { Text(levelLabel(context, localePrefs, item)) },
                    onClick = { onLevel(item); levelMenu = false },
                )
            }
        }

        TextButton(onClick = { priceMenu = true }) {
            Text(priceLabel(context, localePrefs, price))
        }
        DropdownMenu(expanded = priceMenu, onDismissRequest = { priceMenu = false }) {
            CatalogPriceFilter.entries.forEach { item ->
                DropdownMenuItem(
                    text = { Text(priceLabel(context, localePrefs, item)) },
                    onClick = { onPrice(item); priceMenu = false },
                )
            }
        }

        TextButton(onClick = { sortMenu = true }) {
            Text(sortLabel(context, localePrefs, sort))
        }
        DropdownMenu(expanded = sortMenu, onDismissRequest = { sortMenu = false }) {
            CatalogSortMode.entries.forEach { item ->
                DropdownMenuItem(
                    text = { Text(sortLabel(context, localePrefs, item)) },
                    onClick = { onSort(item); sortMenu = false },
                )
            }
        }
    }
}

private fun levelLabel(context: android.content.Context, localePrefs: com.lextures.android.core.i18n.LocalePreferences, level: CatalogLevelFilter): String =
    when (level) {
        CatalogLevelFilter.Any -> L.text(context, localePrefs, R.string.mobile_catalog_filter_levelAny)
        CatalogLevelFilter.Beginner -> L.text(context, localePrefs, R.string.mobile_catalog_filter_levelBeginner)
        CatalogLevelFilter.Intermediate -> L.text(context, localePrefs, R.string.mobile_catalog_filter_levelIntermediate)
        CatalogLevelFilter.Advanced -> L.text(context, localePrefs, R.string.mobile_catalog_filter_levelAdvanced)
    }

private fun priceLabel(context: android.content.Context, localePrefs: com.lextures.android.core.i18n.LocalePreferences, price: CatalogPriceFilter): String =
    when (price) {
        CatalogPriceFilter.Any -> L.text(context, localePrefs, R.string.mobile_catalog_filter_priceAny)
        CatalogPriceFilter.Free -> L.text(context, localePrefs, R.string.mobile_catalog_filter_priceFree)
        CatalogPriceFilter.Paid -> L.text(context, localePrefs, R.string.mobile_catalog_filter_pricePaid)
    }

private fun sortLabel(context: android.content.Context, localePrefs: com.lextures.android.core.i18n.LocalePreferences, sort: CatalogSortMode): String =
    when (sort) {
        CatalogSortMode.Popular -> L.text(context, localePrefs, R.string.mobile_catalog_sort_popular)
        CatalogSortMode.Rating -> L.text(context, localePrefs, R.string.mobile_catalog_sort_rating)
        CatalogSortMode.Newest -> L.text(context, localePrefs, R.string.mobile_catalog_sort_newest)
        CatalogSortMode.Relevance -> L.text(context, localePrefs, R.string.mobile_catalog_sort_relevance)
    }