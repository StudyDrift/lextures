package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Book
import androidx.compose.material.icons.filled.EditNote
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.CoursesDashboardStats
import com.lextures.android.core.lms.CoursesListFilter
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PaginatedPlatformCourses
import com.lextures.android.core.lms.PlatformCourseRow
import com.lextures.android.core.lms.PlatformCoursesAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CoursesAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var searchText by remember { mutableStateOf("") }
    var submittedQuery by remember { mutableStateOf("") }
    var page by remember { mutableIntStateOf(1) }
    var results by remember { mutableStateOf<PaginatedPlatformCourses?>(null) }
    var loading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    var stats by remember { mutableStateOf<CoursesDashboardStats?>(null) }
    var statsLoading by remember { mutableStateOf(true) }
    var statsError by remember { mutableStateOf<String?>(null) }

    var selectedFilter by remember { mutableStateOf<CoursesListFilter?>(null) }
    var filterPage by remember { mutableIntStateOf(1) }
    var filterResults by remember { mutableStateOf<PaginatedPlatformCourses?>(null) }
    var filterLoading by remember { mutableStateOf(false) }
    var filterError by remember { mutableStateOf<String?>(null) }
    var openingCourseId by remember { mutableStateOf<String?>(null) }

    val canView = PlatformCoursesAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val genericError = L.text(context, localePrefs, R.string.mobile_admin_courses_error)

    fun stringByName(name: String): String {
        val id = context.resources.getIdentifier(name, "string", context.packageName)
        return if (id == 0) name else L.text(context, localePrefs, id)
    }

    suspend fun loadStats(token: String) {
        statsLoading = true
        statsError = null
        runCatching { stats = LmsApi.fetchCoursesStats(token) }
            .onFailure { statsError = PlatformCoursesAdminLogic.userFacingError(it, genericError) }
        statsLoading = false
    }

    suspend fun loadFilter(token: String) {
        val filter = selectedFilter ?: return
        filterLoading = true
        filterError = null
        runCatching {
            filterResults = LmsApi.searchPlatformCourses(
                filter = filter,
                page = filterPage,
                perPage = PlatformCoursesAdminLogic.DEFAULT_PER_PAGE,
                accessToken = token,
            )
        }.onFailure {
            filterError = PlatformCoursesAdminLogic.userFacingError(it, genericError)
            filterResults = null
        }
        filterLoading = false
    }

    suspend fun search(token: String) {
        if (!PlatformCoursesAdminLogic.shouldSearch(submittedQuery)) {
            results = null
            return
        }
        loading = true
        errorMessage = null
        runCatching {
            results = LmsApi.searchPlatformCourses(
                query = submittedQuery,
                page = page,
                perPage = PlatformCoursesAdminLogic.DEFAULT_PER_PAGE,
                accessToken = token,
            )
        }.onFailure {
            errorMessage = PlatformCoursesAdminLogic.userFacingError(it, genericError)
            results = null
        }
        loading = false
    }

    fun openCourse(course: PlatformCourseRow) {
        val token = accessToken ?: return
        openingCourseId = course.id
        scope.launch {
            val code = runCatching {
                LmsApi.ensurePlatformCourseAdminAccess(course.id, token).courseCode
            }.getOrDefault(course.courseCode)
            context.startActivity(
                Intent(Intent.ACTION_VIEW, Uri.parse(AppConfiguration.webUrl(PlatformCoursesAdminLogic.courseWebPath(code)))),
            )
            openingCourseId = null
        }
    }

    LaunchedEffect(accessToken, canView) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) loadStats(token)
    }
    LaunchedEffect(accessToken, submittedQuery, page) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView && submittedQuery.isNotEmpty()) search(token)
    }
    LaunchedEffect(accessToken, selectedFilter, filterPage) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView && selectedFilter != null) loadFilter(token)
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_courses_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        if (!canView) {
            LmsEmptyState(
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_courses_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_courses_accessDeniedMessage),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
            return@Scaffold
        }

        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_admin_courses_description), color = textSecondary())

            LmsCard {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            context.startActivity(
                                Intent(
                                    Intent.ACTION_VIEW,
                                    Uri.parse(AppConfiguration.webUrl(PlatformCoursesAdminLogic.webSettingsPath())),
                                ),
                            )
                        }
                        .padding(12.dp),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Icon(Icons.Default.OpenInBrowser, contentDescription = null)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_courses_webTitle),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_courses_webHint),
                            color = textSecondary(),
                        )
                    }
                }
            }

            statsError?.let { LmsErrorBanner(it) }
            AdminMetricCardsGrid(
                cards = PlatformCoursesAdminLogic.METRIC_DEFINITIONS.map { def ->
                    AdminMetricCardModel(
                        id = def.filter.apiValue,
                        title = stringByName(def.titleResName),
                        hint = def.hintResName?.let(::stringByName),
                        value = stats?.let { PlatformCoursesAdminLogic.value(def.filter, it) },
                        icon = courseMetricIcon(def.filter),
                        selected = selectedFilter == def.filter,
                    )
                },
                loading = statsLoading,
                hintLine = L.text(context, localePrefs, R.string.mobile_admin_courses_metric_hint),
                viewListLabel = L.text(context, localePrefs, R.string.mobile_admin_metric_viewList),
                hideListLabel = L.text(context, localePrefs, R.string.mobile_admin_metric_hideList),
                onSelect = { id ->
                    val tapped = CoursesListFilter.entries.first { it.apiValue == id }
                    selectedFilter = PlatformCoursesAdminLogic.toggleFilter(selectedFilter, tapped)
                    filterPage = 1
                    filterResults = null
                    filterError = null
                },
                formatCount = PlatformCoursesAdminLogic::formatCount,
            )

            selectedFilter?.let { filter ->
                val metric = PlatformCoursesAdminLogic.metric(filter)
                if (metric != null) {
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(stringByName(metric.tableTitleResName), style = LexturesType.display(17), color = textPrimary())
                                    Text(stringByName(metric.tableDescriptionResName), color = textSecondary())
                                    filterResults?.takeIf { !filterLoading }?.let {
                                        Text(
                                            L.format(context, localePrefs, R.string.mobile_admin_courses_resultsCount, it.total.toInt()),
                                            fontWeight = FontWeight.SemiBold,
                                            color = textPrimary(),
                                        )
                                    }
                                }
                                TextButton(onClick = {
                                    selectedFilter = null
                                    filterResults = null
                                    filterError = null
                                }) {
                                    Text(L.text(context, localePrefs, R.string.mobile_admin_metric_close))
                                }
                            }
                            filterError?.let { LmsErrorBanner(it) }
                            when {
                                filterLoading && filterResults == null -> LmsSkeletonList(count = 3)
                                filterResults != null -> courseResults(
                                    context, localePrefs, filterResults!!, filterLoading, openingCourseId,
                                    onOpen = ::openCourse,
                                    onPrevious = { if (filterPage > 1) filterPage -= 1 },
                                    onNext = {
                                        val total = filterResults?.totalPages ?: 1
                                        if (filterPage < total) filterPage += 1
                                    },
                                    emptyTitle = R.string.mobile_admin_courses_metric_emptyTitle,
                                    emptyMessage = R.string.mobile_admin_courses_metric_emptyMessage,
                                )
                            }
                        }
                    }
                }
            }

            LmsSectionHeader(
                title = L.text(context, localePrefs, R.string.mobile_admin_courses_searchSection),
                icon = Icons.Default.Search,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = searchText,
                    onValueChange = { searchText = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_courses_search)) },
                    modifier = Modifier.weight(1f),
                    singleLine = true,
                )
                Button(onClick = {
                    submittedQuery = PlatformCoursesAdminLogic.normalizedSearchQuery(searchText)
                    page = 1
                }) {
                    Icon(Icons.Default.Search, contentDescription = null)
                }
            }

            errorMessage?.let { LmsErrorBanner(it) }
            when {
                loading && results == null -> LmsSkeletonList(count = 3)
                results != null -> courseResults(
                    context, localePrefs, results!!, loading, openingCourseId,
                    onOpen = ::openCourse,
                    onPrevious = { if (page > 1) page -= 1 },
                    onNext = {
                        val total = results?.totalPages ?: 1
                        if (page < total) page += 1
                    },
                    emptyTitle = R.string.mobile_admin_courses_emptyTitle,
                    emptyMessage = R.string.mobile_admin_courses_emptySearch,
                    showCount = true,
                )
                !PlatformCoursesAdminLogic.shouldSearch(submittedQuery) -> LmsEmptyState(
                    icon = Icons.Default.MenuBook,
                    title = L.text(context, localePrefs, R.string.mobile_admin_courses_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_courses_emptyMessage),
                )
            }
        }
    }
}

private fun courseMetricIcon(filter: CoursesListFilter): ImageVector = when (filter) {
    CoursesListFilter.Created7d -> Icons.Default.Book
    CoursesListFilter.Active -> Icons.Default.MenuBook
    CoursesListFilter.Draft -> Icons.Default.EditNote
    CoursesListFilter.Total -> Icons.Default.MenuBook
    CoursesListFilter.Archived -> Icons.Default.Archive
}

@Composable
private fun courseResults(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    data: PaginatedPlatformCourses,
    loading: Boolean,
    openingCourseId: String?,
    onOpen: (PlatformCourseRow) -> Unit,
    onPrevious: () -> Unit,
    onNext: () -> Unit,
    emptyTitle: Int,
    emptyMessage: Int,
    showCount: Boolean = false,
) {
    if (data.items.isEmpty()) {
        LmsEmptyState(
            icon = Icons.Default.Search,
            title = L.text(context, localePrefs, emptyTitle),
            message = L.text(context, localePrefs, emptyMessage),
        )
        return
    }
    if (showCount) {
        Text(
            L.format(context, localePrefs, R.string.mobile_admin_courses_resultsCount, data.total.toInt()),
            color = textSecondary(),
        )
    }
    data.items.forEach { course ->
        LmsCard {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(enabled = openingCourseId == null) { onOpen(course) }
                    .padding(12.dp),
            ) {
                Text(course.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                Text(course.courseCode, color = textSecondary())
                Text(
                    "${course.orgName} · ${
                        PlatformCoursesAdminLogic.statusLabel(
                            course.status,
                            L.text(context, localePrefs, R.string.mobile_admin_courses_status_active),
                            L.text(context, localePrefs, R.string.mobile_admin_courses_status_draft),
                            L.text(context, localePrefs, R.string.mobile_admin_courses_status_archived),
                        )
                    } · ${
                        L.format(context, localePrefs, R.string.mobile_admin_courses_enrollments, course.enrollmentCount.toInt())
                    }",
                    color = textSecondary(),
                )
            }
        }
    }
    if (data.totalPages > 1) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            TextButton(onClick = onPrevious, enabled = data.page > 1 && !loading) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_previous))
            }
            Text(
                L.format(context, localePrefs, R.string.mobile_admin_people_pageOf, data.page, data.totalPages),
                color = textSecondary(),
            )
            TextButton(onClick = onNext, enabled = data.page < data.totalPages && !loading) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_next))
            }
        }
    }
}
