package com.lextures.android.features.dashboard

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.AssignmentTurnedIn
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.features.courses.CourseDetailScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.temporal.TemporalAdjusters
import java.util.Calendar

data class DueItem(
    val courseCode: String,
    val courseTitle: String,
    val item: CourseStructureItem,
    val dueAt: Instant,
)

@Composable
fun DashboardTab(
    session: AuthSession,
    unreadInbox: Int,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val userEmail by session.userEmail.collectAsState()

    var courses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var dueThisWeek by remember { mutableStateOf<List<DueItem>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var menuOpen by remember { mutableStateOf(false) }
    var openCourse by remember { mutableStateOf<CourseSummary?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
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
            val studentCourses = enriched.filter { it.viewerIsStudent }
            dueThisWeek = coroutineScope {
                studentCourses.map { course ->
                    async {
                        val items = runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }
                            .getOrDefault(emptyList())
                        items.mapNotNull { item ->
                            val due = LmsDates.parse(item.dueAt) ?: return@mapNotNull null
                            if (!item.isGradable || due < weekStart || due > weekEnd) return@mapNotNull null
                            DueItem(course.courseCode, course.displayTitle, item, due)
                        }
                    }
                }.awaitAll().flatten().sortedBy { it.dueAt }
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
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

    Column(modifier = modifier) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 16.dp, end = 4.dp, top = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = greetingText(),
                    fontSize = 22.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                userEmail?.let {
                    Text(text = it, fontSize = 13.sp, color = textSecondary())
                }
            }
            Box {
                IconButton(onClick = { menuOpen = true }) {
                    Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = "Account", tint = textSecondary())
                }
                DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                    DropdownMenuItem(
                        text = { Text("Sign out") },
                        onClick = {
                            menuOpen = false
                            session.signOut()
                        },
                    )
                }
            }
        }

        if (loading && courses.isEmpty()) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            return
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            item {
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    StatCard("${courses.size}", "Courses", Icons.AutoMirrored.Filled.MenuBook)
                    StatCard("${dueThisWeek.size}", "Due this week", Icons.Default.AssignmentTurnedIn)
                    StatCard("$unreadInbox", "Unread", Icons.Default.Inbox)
                }
            }

            item { LmsSectionHeader("Due this week", Icons.Default.CalendarMonth) }
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
                items(dueThisWeek, key = { "${it.courseCode}/${it.item.id}" }) { due ->
                    LmsCard {
                        Text(
                            text = due.item.title,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Row(modifier = Modifier.fillMaxWidth()) {
                            Text(
                                text = due.courseTitle,
                                fontSize = 12.sp,
                                color = textSecondary(),
                                modifier = Modifier.weight(1f),
                            )
                            Text(
                                text = LmsDates.shortDateTime(due.item.dueAt),
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
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
                items(courses.take(5), key = { it.id }) { course ->
                    LmsCard(onClick = { openCourse = course }) {
                        Text(
                            text = course.displayTitle,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(text = course.courseCode, fontSize = 12.sp, color = textSecondary())
                    }
                }
            }
        }
    }
}

@Composable
private fun RowScope.StatCard(value: String, label: String, icon: ImageVector) {
    LmsCard(modifier = Modifier.weight(1f)) {
        Icon(icon, contentDescription = null, tint = LexturesColors.Primary, modifier = Modifier.size(18.dp))
        Text(text = value, fontSize = 22.sp, fontWeight = FontWeight.Bold, color = textPrimary())
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
