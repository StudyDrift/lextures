package com.lextures.android.features.dashboard

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.AssignmentTurnedIn
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.HeroBrush
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.fieldBorder
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.Broadcast
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GradingBacklogItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.courses.ItemDetailScreen
import com.lextures.android.features.courses.ItemKind
import com.lextures.android.features.grading.GradingBacklogScreen
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.core.lms.nameFieldsFromProfile
import com.lextures.android.core.lms.resolvedInitials
import com.lextures.android.features.home.LmsAvatarChip
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.CourseHeroImage
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.features.planner.PlannerScreen
import com.lextures.android.features.planner.PlannerTab
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.features.profile.AnnouncementsScreen
import com.lextures.android.features.profile.NotificationPreferencesScreen
import com.lextures.android.features.profile.NotificationsScreen
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import androidx.compose.runtime.rememberCoroutineScope
import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.temporal.TemporalAdjusters
import java.util.Calendar

data class DueItem(
    val course: CourseSummary,
    val item: CourseStructureItem,
    val dueAt: Instant,
)

data class StaffBacklog(
    val course: CourseSummary,
    val items: List<GradingBacklogItem>,
) {
    val total: Int get() = items.sumOf { it.ungradedCount }
}

@Composable
fun DashboardTab(
    session: AuthSession,
    shell: HomeShellState,
    onOpenProfile: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var courses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var courseCounts by remember { mutableStateOf<Map<String, Pair<Int, Int>>>(emptyMap()) }
    var dueThisWeek by remember { mutableStateOf<List<DueItem>>(emptyList()) }
    var staffBacklogs by remember { mutableStateOf<List<StaffBacklog>>(emptyList()) }
    var announcements by remember { mutableStateOf<List<Broadcast>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }
    var openDueItem by remember { mutableStateOf<DueItem?>(null) }
    var openBacklog by remember { mutableStateOf<StaffBacklog?>(null) }
    var showNotifications by remember { mutableStateOf(false) }
    var showNotificationPreferences by remember { mutableStateOf(false) }
    var showAnnouncements by remember { mutableStateOf(false) }
    var showPlanner by remember { mutableStateOf(false) }
    var showReview by remember { mutableStateOf(false) }
    var reviewStats by remember { mutableStateOf<com.lextures.android.core.lms.ReviewStats?>(null) }
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            announcements = runCatching { LmsApi.fetchMyBroadcasts(token) }.getOrDefault(emptyList())
            val list = LmsApi.fetchCourses(token)
            // The list GET omits viewer roles; enrich from the single-course GET.
            val enriched = coroutineScope {
                list.map { course ->
                    async { runCatching { LmsApi.fetchCourse(course.courseCode, token) }.getOrDefault(course) }
                }.awaitAll()
            }
            courses = enriched

            val zone = ZoneId.systemDefault()
            val weekStart = LocalDate.now(zone)
                .with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
                .atStartOfDay(zone).toInstant()
            val weekEnd = weekStart.plusSeconds(7 * 86_400 - 1)

            // One structure fetch per course feeds both the due rail and the card counts.
            val structures = coroutineScope {
                enriched.map { course ->
                    async {
                        course to runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }
                            .getOrDefault(emptyList())
                    }
                }.awaitAll()
            }
            courseCounts = structures.associate { (course, items) ->
                course.courseCode to Pair(
                    items.count { it.isModule },
                    items.count { !it.isModule && it.kind != "heading" },
                )
            }
            dueThisWeek = structures
                .filter { (course, _) -> course.viewerIsStudent }
                .flatMap { (course, items) ->
                    items.mapNotNull { item ->
                        val due = LmsDates.parse(item.dueAt) ?: return@mapNotNull null
                        if (!item.isGradable || due < weekStart || due > weekEnd) return@mapNotNull null
                        DueItem(course, item, due)
                    }
                }
                .sortedBy { it.dueAt }

            val staffCourses = enriched.filter { it.viewerIsStaff }
            staffBacklogs = coroutineScope {
                staffCourses.map { course ->
                    async {
                        runCatching { LmsApi.fetchGradingBacklog(course.courseCode, token) }
                            .getOrNull()
                            ?.let { StaffBacklog(course, it) }
                    }
                }.awaitAll()
            }.filterNotNull().filter { it.total > 0 }.sortedByDescending { it.total }

            val learnerId = com.lextures.android.core.notebook.NotebookStore.jwtSubject(token)
            reviewStats = learnerId?.let { id ->
                runCatching {
                    offline.cachedFetch(
                        key = OfflineCacheKey.reviewStats(),
                        accessToken = token,
                        serializer = kotlinx.serialization.serializer<com.lextures.android.core.lms.ReviewStats>(),
                    ) { LmsApi.fetchLearnerReviewStats(id, token) }.first
                }.getOrNull()
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    openDueItem?.let { due ->
        ItemDetailScreen(
            session = session,
            course = due.course,
            item = due.item,
            onBack = { openDueItem = null },
            modifier = modifier,
        )
        return
    }

    openCourse?.let { course ->
        CourseDetailScreen(
            session = session,
            course = course,
            onBack = { openCourse = null },
            modifier = modifier,
        )
        return
    }

    openBacklog?.let { backlog ->
        GradingBacklogScreen(
            session = session,
            course = backlog.course,
            onBack = { openBacklog = null },
            modifier = modifier,
        )
        return
    }

    if (showNotificationPreferences) {
        NotificationPreferencesScreen(
            session = session,
            onBack = { showNotificationPreferences = false },
            modifier = modifier,
        )
        return
    }

    if (showNotifications) {
        NotificationsScreen(
            session = session,
            shell = shell,
            onBack = { showNotifications = false },
            onOpenPreferences = { showNotificationPreferences = true },
            modifier = modifier,
        )
        return
    }

    if (showAnnouncements) {
        AnnouncementsScreen(
            session = session,
            onBack = { showAnnouncements = false },
            modifier = modifier,
        )
        return
    }

    LaunchedEffect(shell.pendingReview) {
        if (shell.consumePendingReview()) {
            showReview = true
        }
    }

    if (showReview) {
        com.lextures.android.features.review.ReviewHomeScreen(
            session = session,
            shell = shell,
            onBack = { showReview = false },
            modifier = modifier,
        )
        return
    }

    if (showPlanner) {
        PlannerScreen(
            session = session,
            offline = offline,
            isOnline = isOnline,
            initialTab = PlannerTab.Todos,
            onBack = { showPlanner = false },
            modifier = modifier,
        )
        return
    }

        LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = 16.dp, top = 16.dp, end = 16.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            HeroPanel(
                greeting = greetingText(),
                name = shell.accountProfile?.let { nameFieldsFromProfile(it).first }
                    ?.takeIf { it.isNotEmpty() }
                    ?: shell.profile?.firstName.orEmpty(),
                dueCount = dueThisWeek.size,
                loading = loading,
                unreadNotifications = shell.unreadNotifications,
                avatarInitials = shell.accountProfile?.resolvedInitials()
                    ?: shell.profile?.initials ?: "··",
                avatarUrl = shell.accountProfile?.avatarUrl,
                onOpenNotifications = { showNotifications = true },
                onOpenProfile = onOpenProfile,
                showSearch = shell.iaRedesignEnabled && shell.universalSearchEnabled,
                onOpenSearch = { shell.showUniversalSearch = true },
            )
        }

        errorMessage?.let { message ->
            item { LmsErrorBanner(message) }
        }

        if (loading && courses.isEmpty()) {
            item { LmsSkeletonList(count = 4) }
            return@LazyColumn
        }

        announcements.firstOrNull()?.let { broadcast ->
            item {
                AnnouncementCard(
                    broadcast = broadcast,
                    showSeeAll = announcements.size > 1,
                    onAcknowledge = {
                        val token = accessToken ?: return@AnnouncementCard
                        scope.launch {
                            // Best-effort: dismiss locally even if the POST fails.
                            runCatching { LmsApi.acknowledgeBroadcast(broadcast.id, token) }
                            announcements = announcements.filterNot { it.id == broadcast.id }
                        }
                    },
                    onSeeAll = { showAnnouncements = true },
                )
            }
        }

        reviewStats?.let { stats ->
            item {
                LmsCard(onClick = { showReview = true }) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(
                                text = context.getString(R.string.mobile_review_dashboardTitle),
                                fontSize = 15.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            Text(
                                text = if (stats.dueToday > 0) {
                                    context.resources.getQuantityString(
                                        R.plurals.mobile_review_dueCount,
                                        stats.dueToday,
                                        stats.dueToday,
                                    )
                                } else {
                                    context.getString(R.string.mobile_review_caughtUpShort)
                                },
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                            if (stats.streak > 0) {
                                Text(
                                    text = context.resources.getQuantityString(
                                        R.plurals.mobile_review_streak,
                                        stats.streak,
                                        stats.streak,
                                    ),
                                    fontSize = 11.sp,
                                    color = LexturesColors.Amber,
                                )
                            }
                        }
                        Text(
                            text = context.getString(
                                if (stats.dueToday > 0) R.string.mobile_review_start else R.string.mobile_review_open,
                            ),
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = accentColor(),
                        )
                    }
                }
            }
        }

        item {
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                StatCard("${courses.size}", "Courses", Icons.AutoMirrored.Filled.MenuBook, accentColor())
                StatCard("${dueThisWeek.size}", "Due this week", Icons.Default.AssignmentTurnedIn, LexturesColors.Coral)
                StatCard("${shell.unreadInbox}", "Unread", Icons.Default.Inbox, LexturesColors.Amber)
            }
        }

        if (staffBacklogs.isNotEmpty()) {
            item { LmsSectionHeader("Needs grading", Icons.Default.FactCheck) }
            items(staffBacklogs, key = { "backlog-${it.course.id}" }) { backlog ->
                LmsCard(accent = LexturesColors.Amber, onClick = { openBacklog = backlog }) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                            Text(
                                text = backlog.course.displayTitle,
                                fontSize = 15.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            Text(
                                text = "${backlog.total} submission${if (backlog.total == 1) "" else "s"} waiting",
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                        Text(
                            text = "${backlog.total}",
                            style = LexturesType.display(18, FontWeight.Bold),
                            color = LexturesColors.Amber,
                            modifier = Modifier
                                .clip(RoundedCornerShape(50))
                                .background(LexturesColors.Amber.copy(alpha = 0.14f))
                                .padding(horizontal = 10.dp, vertical = 4.dp),
                        )
                    }
                }
            }
        }

        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                LmsSectionHeader("Due this week", Icons.Default.CalendarMonth, modifier = Modifier.weight(1f))
                Text(
                    text = context.getString(R.string.mobile_planner_viewAll),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = accentColor(),
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .clickable { showPlanner = true }
                        .padding(horizontal = 8.dp, vertical = 6.dp),
                )
            }
        }
        if (dueThisWeek.isEmpty()) {
            item {
                LmsCard {
                    Text(
                        text = "Nothing due this week. Enjoy the breathing room!",
                        fontSize = 14.sp,
                        color = textSecondary(),
                    )
                }
            }
        } else {
            items(dueThisWeek, key = { "${it.course.courseCode}/${it.item.id}" }) { due ->
                LmsCard(accent = LexturesColors.Coral, onClick = { openDueItem = due }) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier
                                .size(34.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(LexturesColors.Coral.copy(alpha = 0.12f)),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                ItemKind.icon(due.item.kind),
                                contentDescription = null,
                                tint = LexturesColors.Coral,
                                modifier = Modifier.size(17.dp),
                            )
                        }
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(
                                text = due.item.title,
                                fontSize = 15.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            Text(
                                text = due.course.displayTitle,
                                fontSize = 12.sp,
                                color = textSecondary(),
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(
                                text = weekdayLabel(due.dueAt),
                                fontSize = 11.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textSecondary(),
                            )
                            Text(
                                text = timeLabel(due.dueAt),
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Bold,
                                color = LexturesColors.Coral,
                            )
                        }
                    }
                }
            }
        }

        item { LmsSectionHeader("Your courses", Icons.AutoMirrored.Filled.MenuBook) }
        if (courses.isEmpty()) {
            item {
                LmsEmptyState(
                    icon = Icons.AutoMirrored.Filled.MenuBook,
                    title = "No courses yet",
                    message = "Courses you enroll in will show up here.",
                )
            }
        } else {
            item {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    courses.forEach { course ->
                        CourseCarouselCard(
                            course = course,
                            counts = courseCounts[course.courseCode],
                            accessToken = accessToken,
                            onClick = { openCourse = course },
                        )
                    }
                }
            }
        }
    }
}

/** Deep-teal gradient greeting panel with bell + avatar — the brand statement. */
@Composable
private fun HeroPanel(
    greeting: String,
    name: String,
    dueCount: Int,
    loading: Boolean,
    unreadNotifications: Int,
    avatarInitials: String,
    avatarUrl: String? = null,
    onOpenNotifications: () -> Unit,
    onOpenProfile: () -> Unit,
    showSearch: Boolean = false,
    onOpenSearch: () -> Unit = {},
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(24.dp))
            .background(HeroBrush),
    ) {
        // Decorative drifting circles, echoing the rocket's arc in the logo.
        Box(
            modifier = Modifier
                .size(160.dp)
                .offset(x = 250.dp, y = (-60).dp)
                .clip(CircleShape)
                .background(Color.White.copy(alpha = 0.07f)),
        )
        Box(
            modifier = Modifier
                .size(56.dp)
                .offset(x = 200.dp, y = 70.dp)
                .clip(CircleShape)
                .background(LexturesColors.BrandCoral.copy(alpha = 0.35f)),
        )

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Row(verticalAlignment = Alignment.Top) {
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(
                        text = "$greeting,",
                        style = LexturesType.display(26),
                        color = Color.White,
                    )
                    if (name.isNotEmpty()) {
                        Text(
                            text = name,
                            style = LexturesType.display(26),
                            color = LexturesColors.BrandCream,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    if (showSearch) {
                        Box(
                            modifier = Modifier
                                .size(34.dp)
                                .clip(CircleShape)
                                .background(Color.White.copy(alpha = 0.16f))
                                .clickable(onClick = onOpenSearch),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                Icons.Default.Search,
                                contentDescription = "Search",
                                tint = Color.White,
                                modifier = Modifier.size(17.dp),
                            )
                        }
                    }
                    Box(
                        modifier = Modifier
                            .size(34.dp)
                            .clip(CircleShape)
                            .background(Color.White.copy(alpha = 0.16f))
                            .clickable(onClick = onOpenNotifications),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            Icons.Default.Notifications,
                            contentDescription = "Notifications",
                            tint = Color.White,
                            modifier = Modifier.size(17.dp),
                        )
                        if (unreadNotifications > 0) {
                            Box(
                                modifier = Modifier
                                    .align(Alignment.TopEnd)
                                    .offset(x = (-3).dp, y = 3.dp)
                                    .size(9.dp)
                                    .clip(CircleShape)
                                    .background(LexturesColors.Coral),
                            )
                        }
                    }
                    LmsAvatarChip(
                        initials = avatarInitials,
                        avatarUrl = avatarUrl,
                        onClick = onOpenProfile,
                    )
                }
            }

            if (dueCount > 0) {
                Text(
                    text = "$dueCount assignment${if (dueCount == 1) "" else "s"} due this week",
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = LexturesColors.PrimaryDeep,
                    modifier = Modifier
                        .padding(top = 8.dp)
                        .clip(RoundedCornerShape(50))
                        .background(LexturesColors.BrandCream)
                        .padding(horizontal = 10.dp, vertical = 5.dp),
                )
            } else if (!loading) {
                Text(
                    text = "You're all caught up",
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = Color.White.copy(alpha = 0.9f),
                    modifier = Modifier
                        .padding(top = 8.dp)
                        .clip(RoundedCornerShape(50))
                        .background(Color.White.copy(alpha = 0.16f))
                        .padding(horizontal = 10.dp, vertical = 5.dp),
                )
            }
        }
    }
}

/** Dashboard banner for the newest org announcement; coral treatment for emergencies. */
@Composable
fun AnnouncementCard(
    broadcast: Broadcast,
    showSeeAll: Boolean,
    onAcknowledge: () -> Unit,
    onSeeAll: () -> Unit,
) {
    val tint = if (broadcast.isEmergency) LexturesColors.Coral else LexturesColors.Amber
    LmsCard(accent = tint) {
        Row(verticalAlignment = Alignment.Top, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(
                    text = broadcast.subject,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(
                    text = broadcast.body,
                    fontSize = 12.sp,
                    color = textSecondary(),
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Text(
                text = LmsDates.relative(broadcast.sentAt ?: broadcast.createdAt),
                fontSize = 11.sp,
                color = textSecondary(),
            )
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "Got it",
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = tint,
                modifier = Modifier
                    .clip(RoundedCornerShape(50))
                    .background(tint.copy(alpha = 0.12f))
                    .clickable(onClick = onAcknowledge)
                    .padding(horizontal = 12.dp, vertical = 6.dp),
            )
            Box(modifier = Modifier.weight(1f))
            if (showSeeAll) {
                Text(
                    text = "See all",
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = accentColor(),
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .clickable(onClick = onSeeAll)
                        .padding(horizontal = 8.dp, vertical = 6.dp),
                )
            }
        }
    }
}

@Composable
private fun CourseCarouselCard(
    course: CourseSummary,
    counts: Pair<Int, Int>?,
    accessToken: String?,
    onClick: () -> Unit,
) {
    Column(
        modifier = Modifier
            .width(190.dp)
            .clip(RoundedCornerShape(18.dp))
            .background(cardBackground())
            .clickable(onClick = onClick),
    ) {
        Box(modifier = Modifier.fillMaxWidth()) {
            CourseHeroImage(
                url = course.heroImageUrl,
                fallbackKey = course.courseCode,
                accessToken = accessToken,
                height = 84.dp,
            )
            Icon(
                Icons.AutoMirrored.Filled.MenuBook,
                contentDescription = null,
                tint = Color.White.copy(alpha = 0.5f),
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(12.dp)
                    .size(22.dp),
            )
        }
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Text(
                text = course.displayTitle,
                style = LexturesType.display(15),
                color = textPrimary(),
                maxLines = 2,
                minLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = counts?.takeIf { it.second > 0 }?.let { (modules, items) ->
                    "$modules module${if (modules == 1) "" else "s"} · $items item${if (items == 1) "" else "s"}"
                } ?: course.courseCode.uppercase(),
                fontSize = 11.sp,
                fontWeight = FontWeight.Medium,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun RowScope.StatCard(value: String, label: String, icon: ImageVector, tint: Color) {
    LmsCard(modifier = Modifier.weight(1f)) {
        Box(
            modifier = Modifier
                .size(30.dp)
                .clip(RoundedCornerShape(9.dp))
                .background(tint.copy(alpha = 0.14f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(16.dp))
        }
        Text(text = value, style = LexturesType.display(24, FontWeight.Bold), color = textPrimary())
        Text(text = label, fontSize = 11.sp, color = textSecondary())
    }
}

private fun greetingText(): String {
    return when (Calendar.getInstance().get(Calendar.HOUR_OF_DAY)) {
        in 0..11 -> "Good morning"
        in 12..16 -> "Good afternoon"
        else -> "Good evening"
    }
}

private fun weekdayLabel(instant: Instant): String =
    DateTimeFormatter.ofPattern("EEE").withZone(ZoneId.systemDefault()).format(instant)

private fun timeLabel(instant: Instant): String =
    DateTimeFormatter.ofPattern("h:mm a").withZone(ZoneId.systemDefault()).format(instant)
