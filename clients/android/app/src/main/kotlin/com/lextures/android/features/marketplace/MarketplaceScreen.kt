package com.lextures.android.features.marketplace

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MarketplaceCategory
import com.lextures.android.core.lms.MarketplaceCourse
import com.lextures.android.core.lms.MarketplaceLevelFilter
import com.lextures.android.core.lms.MarketplaceLogic
import com.lextures.android.core.lms.MarketplacePriceFilter
import com.lextures.android.core.lms.MarketplaceSearchResponse
import com.lextures.android.core.lms.MarketplaceSortMode
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.CourseHeroImage
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer

@Composable
fun MarketplaceScreen(
    session: AuthSession,
    shell: HomeShellState,
    onOpenCourse: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val offline = remember { OfflineService.get(context) }
    val scope = rememberCoroutineScope()

    var query by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("") }
    var level by remember { mutableStateOf(MarketplaceLevelFilter.Any) }
    var price by remember { mutableStateOf(MarketplacePriceFilter.Any) }
    var sort by remember { mutableStateOf(MarketplaceSortMode.Popular) }
    var categories by remember { mutableStateOf<List<MarketplaceCategory>>(emptyList()) }
    var courses by remember { mutableStateOf<List<MarketplaceCourse>>(emptyList()) }
    var nextCursor by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var loadingMore by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var hasSearched by remember { mutableStateOf(false) }

    suspend fun searchCourses(reset: Boolean) {
        val token = accessToken ?: return
        if (reset) loading = true
        errorMessage = null
        hasSearched = true
        try {
            val cacheKey = MarketplaceLogic.cacheKey(query, category, level, price, sort)
            val response: MarketplaceSearchResponse = offline.cachedFetch(
                key = OfflineCacheKey.marketplaceCourses(cacheKey),
                accessToken = token,
                serializer = serializer<MarketplaceSearchResponse>(),
            ) {
                LmsApi.fetchMarketplaceCourses(
                    query = query,
                    category = category,
                    level = level.queryValue.orEmpty(),
                    sort = sort.apiValue,
                    priceMax = price.priceMax,
                    freeOnly = price.freeOnly,
                    accessToken = token,
                )
            }.first
            var page = response.courses
            if (price == MarketplacePriceFilter.Paid) {
                page = page.filter { MarketplaceLogic.isPaid(it.priceCents) }
            }
            courses = page
            nextCursor = response.nextCursor
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_error)
            if (reset) courses = emptyList()
        } finally {
            if (reset) loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        categories = runCatching { LmsApi.fetchMarketplaceCategories(token) }.getOrDefault(emptyList())
    }

    LaunchedEffect(accessToken, query, category, level, price, sort) {
        if (accessToken != null) searchCourses(reset = true)
    }

    Column(
        modifier = modifier.fillMaxSize().padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(
            L.text(context, localePrefs, R.string.mobile_marketplace_title),
            fontSize = 20.sp,
            fontWeight = FontWeight.Bold,
            color = textPrimary(),
        )
        OutlinedTextField(
            value = query,
            onValueChange = { query = it },
            modifier = Modifier.fillMaxWidth(),
            placeholder = { Text(L.text(context, localePrefs, R.string.mobile_marketplace_search)) },
            leadingIcon = { androidx.compose.material3.Icon(Icons.Default.Search, contentDescription = null) },
            singleLine = true,
        )

        MarketplaceFilterRow(
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
                title = L.text(context, localePrefs, R.string.mobile_marketplace_emptyTitle),
                message = if (hasSearched) {
                    L.text(context, localePrefs, R.string.mobile_marketplace_emptyMessage)
                } else {
                    L.text(context, localePrefs, R.string.mobile_marketplace_prompt)
                },
            )
            else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(courses, key = { it.id }) { course ->
                    val freeLabel = L.text(context, localePrefs, R.string.mobile_marketplace_free)
                    val ownedLabel = L.text(context, localePrefs, R.string.mobile_marketplace_owned)
                    val priceLabel = MarketplaceLogic.formatPrice(course.priceCents, course.priceCurrency, freeLabel)
                    val a11y = MarketplaceLogic.cardAccessibleName(course.title, priceLabel, course.owned, ownedLabel)
                    LmsCard(
                        onClick = { onOpenCourse(course.slug) },
                        modifier = Modifier.semantics { contentDescription = a11y },
                    ) {
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
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text(priceLabel, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
                                if (course.owned) {
                                    Text(ownedLabel, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textSecondary())
                                }
                            }
                        }
                    }
                }
                if (nextCursor.isNotBlank()) {
                    item {
                        Button(
                            onClick = {
                                val token = accessToken ?: return@Button
                                scope.launch {
                                    loadingMore = true
                                    try {
                                        val response = LmsApi.fetchMarketplaceCourses(
                                            query = query,
                                            category = category,
                                            level = level.queryValue.orEmpty(),
                                            sort = sort.apiValue,
                                            priceMax = price.priceMax,
                                            freeOnly = price.freeOnly,
                                            cursor = nextCursor,
                                            accessToken = token,
                                        )
                                        var page = response.courses
                                        if (price == MarketplacePriceFilter.Paid) {
                                            page = page.filter { MarketplaceLogic.isPaid(it.priceCents) }
                                        }
                                        courses = courses + page
                                        nextCursor = response.nextCursor
                                    } catch (_: Exception) {
                                        errorMessage = L.text(context, localePrefs, R.string.mobile_marketplace_error)
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
                                    L.text(context, localePrefs, R.string.mobile_marketplace_loadingMore)
                                } else {
                                    L.text(context, localePrefs, R.string.mobile_marketplace_loadMore)
                                },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun MarketplaceFilterRow(
    categories: List<MarketplaceCategory>,
    category: String,
    onCategory: (String) -> Unit,
    level: MarketplaceLevelFilter,
    onLevel: (MarketplaceLevelFilter) -> Unit,
    price: MarketplacePriceFilter,
    onPrice: (MarketplacePriceFilter) -> Unit,
    sort: MarketplaceSortMode,
    onSort: (MarketplaceSortMode) -> Unit,
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
            Text(
                if (category.isBlank()) {
                    L.text(context, localePrefs, R.string.mobile_marketplace_filter_categoryAny)
                } else {
                    category
                },
            )
        }
        DropdownMenu(expanded = categoryMenu, onDismissRequest = { categoryMenu = false }) {
            DropdownMenuItem(
                text = { Text(L.text(context, localePrefs, R.string.mobile_marketplace_filter_categoryAny)) },
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
            Text(marketplaceLevelLabel(context, localePrefs, level))
        }
        DropdownMenu(expanded = levelMenu, onDismissRequest = { levelMenu = false }) {
            MarketplaceLevelFilter.entries.forEach { item ->
                DropdownMenuItem(
                    text = { Text(marketplaceLevelLabel(context, localePrefs, item)) },
                    onClick = { onLevel(item); levelMenu = false },
                )
            }
        }

        TextButton(onClick = { priceMenu = true }) {
            Text(marketplacePriceLabel(context, localePrefs, price))
        }
        DropdownMenu(expanded = priceMenu, onDismissRequest = { priceMenu = false }) {
            MarketplacePriceFilter.entries.forEach { item ->
                DropdownMenuItem(
                    text = { Text(marketplacePriceLabel(context, localePrefs, item)) },
                    onClick = { onPrice(item); priceMenu = false },
                )
            }
        }

        TextButton(onClick = { sortMenu = true }) {
            Text(marketplaceSortLabel(context, localePrefs, sort))
        }
        DropdownMenu(expanded = sortMenu, onDismissRequest = { sortMenu = false }) {
            MarketplaceSortMode.entries.forEach { item ->
                DropdownMenuItem(
                    text = { Text(marketplaceSortLabel(context, localePrefs, item)) },
                    onClick = { onSort(item); sortMenu = false },
                )
            }
        }
    }
}

private fun marketplaceLevelLabel(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    level: MarketplaceLevelFilter,
): String = when (level) {
    MarketplaceLevelFilter.Any -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_levelAny)
    MarketplaceLevelFilter.Beginner -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_levelBeginner)
    MarketplaceLevelFilter.Intermediate -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_levelIntermediate)
    MarketplaceLevelFilter.Advanced -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_levelAdvanced)
}

private fun marketplacePriceLabel(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    price: MarketplacePriceFilter,
): String = when (price) {
    MarketplacePriceFilter.Any -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_priceAny)
    MarketplacePriceFilter.Free -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_priceFree)
    MarketplacePriceFilter.Paid -> L.text(context, localePrefs, R.string.mobile_marketplace_filter_pricePaid)
}

private fun marketplaceSortLabel(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    sort: MarketplaceSortMode,
): String = when (sort) {
    MarketplaceSortMode.Popular -> L.text(context, localePrefs, R.string.mobile_marketplace_sort_popular)
    MarketplaceSortMode.Rating -> L.text(context, localePrefs, R.string.mobile_marketplace_sort_rating)
    MarketplaceSortMode.Newest -> L.text(context, localePrefs, R.string.mobile_marketplace_sort_newest)
    MarketplaceSortMode.Relevance -> L.text(context, localePrefs, R.string.mobile_marketplace_sort_relevance)
    MarketplaceSortMode.Price -> L.text(context, localePrefs, R.string.mobile_marketplace_sort_price)
}
