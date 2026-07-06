package com.lextures.android.features.parent

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.FamilyRestroom
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.DateFormatting
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ParentAssignmentRow
import com.lextures.android.core.lms.ParentAttendanceRecord
import com.lextures.android.core.lms.ParentBehaviorResponse
import com.lextures.android.core.lms.ParentChildSummary
import com.lextures.android.core.lms.ParentCourseGradesRow
import com.lextures.android.core.lms.ParentLogic
import com.lextures.android.core.lms.ParentWeeklySummaryResponse
import com.lextures.android.core.navigation.MobileIaPreferences
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList

enum class ParentSubRoute {
    Dashboard,
    Grades,
    Attendance,
    NotificationPrefs,
    Conferences,
}

@Composable
fun ParentDashboardScreen(
    session: AuthSession,
    shell: HomeShellState,
    initialStudentId: String? = null,
    initialRoute: ParentSubRoute? = null,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current

    var subRoute by remember(initialRoute) { mutableStateOf(initialRoute ?: ParentSubRoute.Dashboard) }
    var children by remember { mutableStateOf<List<ParentChildSummary>>(emptyList()) }
    var selectedStudentId by remember { mutableStateOf<String?>(null) }
    var grades by remember { mutableStateOf<List<ParentCourseGradesRow>>(emptyList()) }
    var assignments by remember { mutableStateOf<List<ParentAssignmentRow>>(emptyList()) }
    var attendance by remember { mutableStateOf<List<ParentAttendanceRecord>>(emptyList()) }
    var behavior by remember { mutableStateOf<ParentBehaviorResponse?>(null) }
    var weeklySummary by remember { mutableStateOf<ParentWeeklySummaryResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var detailLoading by remember { mutableStateOf(false) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var detailError by remember { mutableStateOf<String?>(null) }

    val selectedChild = children.firstOrNull { it.studentUserId == selectedStudentId }
    val childName = selectedChild?.let { ParentLogic.childLabel(it) }.orEmpty()

    suspend fun loadChildren() {
        val token = accessToken ?: return
        loading = true
        loadError = null
        try {
            val list = LmsApi.fetchParentChildren(token)
            children = list
            val stored = MobileIaPreferences.loadSelectedChildId(context)
            selectedStudentId = ParentLogic.resolveSelectedChildId(list, initialStudentId ?: stored)
        } catch (e: Exception) {
            loadError = e.message
        } finally {
            loading = false
        }
    }

    suspend fun loadChildDetails(studentId: String) {
        val token = accessToken ?: return
        detailLoading = true
        detailError = null
        try {
            grades = LmsApi.fetchParentStudentGrades(studentId, token)
            assignments = LmsApi.fetchParentStudentAssignments(studentId, token)
            attendance = LmsApi.fetchParentStudentAttendance(studentId, token)
            behavior = LmsApi.fetchParentStudentBehavior(studentId, token)
            weeklySummary = LmsApi.fetchParentWeeklySummary(token)
        } catch (e: Exception) {
            detailError = e.message
        } finally {
            detailLoading = false
        }
    }

    LaunchedEffect(accessToken) { loadChildren() }

    LaunchedEffect(selectedStudentId, accessToken) {
        val studentId = selectedStudentId ?: return@LaunchedEffect
        MobileIaPreferences.saveSelectedChildId(context, studentId)
        loadChildDetails(studentId)
    }

    when (subRoute) {
        ParentSubRoute.Grades -> {
            ParentGradesDetailScreen(
                session = session,
                studentId = selectedStudentId.orEmpty(),
                childName = childName,
                onBack = { subRoute = ParentSubRoute.Dashboard },
            )
        }
        ParentSubRoute.Attendance -> {
            ParentAttendanceDetailScreen(
                session = session,
                studentId = selectedStudentId.orEmpty(),
                childName = childName,
                onBack = { subRoute = ParentSubRoute.Dashboard },
            )
        }
        ParentSubRoute.NotificationPrefs -> {
            ParentNotificationPrefsScreen(
                session = session,
                onBack = { subRoute = ParentSubRoute.Dashboard },
            )
        }
        ParentSubRoute.Conferences -> {
            ConferenceBookingScreen(
                session = session,
                studentId = selectedStudentId.orEmpty(),
                childName = childName,
                onBack = { subRoute = ParentSubRoute.Dashboard },
            )
        }
        ParentSubRoute.Dashboard -> {
            when {
                loading -> LmsSkeletonList(count = 4, modifier = modifier.fillMaxSize())
                loadError != null && children.isEmpty() -> LmsEmptyState(
                    icon = Icons.Default.FamilyRestroom,
                    title = L.text(context, localePrefs, R.string.mobile_parent_title),
                    message = loadError.orEmpty(),
                    modifier = modifier.fillMaxSize(),
                )
                else -> Column(
                    modifier = modifier
                        .fillMaxSize()
                        .verticalScroll(rememberScrollState())
                        .padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_parent_badge),
                        color = accentColor(),
                        fontWeight = FontWeight.Medium,
                        fontSize = 14.sp,
                    )
                    Text(
                        L.text(context, localePrefs, R.string.mobile_parent_subtitle),
                        color = textSecondary(),
                        fontSize = 14.sp,
                    )
                    if (children.isEmpty()) {
                        LmsCard {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_parent_no_children),
                                color = textSecondary(),
                            )
                        }
                    } else {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .horizontalScroll(rememberScrollState()),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            children.forEach { child ->
                                val active = child.studentUserId == selectedStudentId
                                FilterChip(
                                    selected = active,
                                    onClick = { selectedStudentId = child.studentUserId },
                                    label = { Text(ParentLogic.childLabel(child)) },
                                )
                            }
                        }
                        selectedChild?.let { child ->
                            LmsCard(accent = androidx.compose.ui.graphics.Color(0xFFF59E0B)) {
                                Text(
                                    localePrefs.localizedContext(context).getString(
                                        R.string.mobile_parent_read_only,
                                        ParentLogic.childLabel(child),
                                    ),
                                    color = textPrimary(),
                                )
                            }
                        }
                        detailError?.let { LmsErrorBanner(it) }
                        if (detailLoading) {
                            LmsSkeletonList(count = 3)
                        } else if (selectedStudentId != null) {
                            ParentSummarySection(
                                context = context,
                                localePrefs = localePrefs,
                                grades = grades,
                                assignments = assignments,
                                attendance = attendance,
                                behavior = behavior,
                                weeklySummary = weeklySummary,
                                childName = childName,
                            )
                            ParentActionLinks(
                                context = context,
                                localePrefs = localePrefs,
                                conferenceEnabled = shell.platformFeatures.ffConferenceScheduling,
                                onGrades = { subRoute = ParentSubRoute.Grades },
                                onAttendance = { subRoute = ParentSubRoute.Attendance },
                                onPrefs = { subRoute = ParentSubRoute.NotificationPrefs },
                                onConferences = { subRoute = ParentSubRoute.Conferences },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ParentSummarySection(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    grades: List<ParentCourseGradesRow>,
    assignments: List<ParentAssignmentRow>,
    attendance: List<ParentAttendanceRecord>,
    behavior: ParentBehaviorResponse?,
    weeklySummary: ParentWeeklySummaryResponse?,
    childName: String,
) {
    val summary = ParentLogic.attendanceSummary(attendance)
    val points = behavior?.totalPoints ?: 0
    val referrals = behavior?.referrals?.size ?: 0
    val weeklyItems = ParentLogic.weeklyItemsForChild(weeklySummary?.items.orEmpty(), childName)

    ParentSummaryCard(
        title = L.text(context, localePrefs, R.string.mobile_parent_section_grades),
        empty = L.text(context, localePrefs, R.string.mobile_parent_grades_empty),
        hasContent = grades.isNotEmpty(),
    ) {
        ParentLogic.recentGrades(grades).forEach { row ->
            Row(Modifier.fillMaxWidth()) {
                Column(Modifier.weight(1f)) {
                    Text(row.course.title, fontWeight = FontWeight.Medium)
                    Text(row.itemId.take(8) + "…", fontSize = 12.sp, color = textSecondary())
                }
                Text(row.score, fontWeight = FontWeight.SemiBold)
            }
        }
    }

    ParentSummaryCard(
        title = L.text(context, localePrefs, R.string.mobile_parent_section_attendance),
        empty = L.text(context, localePrefs, R.string.mobile_parent_attendance_empty),
        hasContent = attendance.isNotEmpty(),
    ) {
        Text(
            localePrefs.localizedContext(context).getString(
                R.string.mobile_parent_attendance_summary,
                summary.present,
                summary.absent,
                summary.tardy,
            ),
        )
        ParentLogic.recentAttendance(attendance).forEach { record ->
            Row(Modifier.fillMaxWidth()) {
                Text(record.date)
                Text(
                    ParentLogic.attendanceLabel(context, record),
                    modifier = Modifier.weight(1f),
                    color = textSecondary(),
                )
            }
        }
    }

    ParentSummaryCard(
        title = L.text(context, localePrefs, R.string.mobile_parent_section_assignments),
        empty = L.text(context, localePrefs, R.string.mobile_parent_assignments_empty),
        hasContent = assignments.isNotEmpty(),
    ) {
        ParentLogic.upcomingAssignments(assignments).forEach { item ->
            Column {
                Text(item.title, fontWeight = FontWeight.Medium)
                Text(
                    "${item.courseTitle} · ${item.kind}" +
                        (item.dueAt?.let {
                            " · ${DateFormatting.formatAbsoluteShort(it, localePrefs.effectiveLocale())}"
                        }.orEmpty()),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
            }
        }
    }

    ParentSummaryCard(
        title = L.text(context, localePrefs, R.string.mobile_parent_section_behavior),
        empty = L.text(context, localePrefs, R.string.mobile_parent_behavior_empty),
        hasContent = points > 0 || referrals > 0,
    ) {
        Text(
            localePrefs.localizedContext(context).getString(
                R.string.mobile_parent_behavior_summary,
                points,
                referrals,
            ),
        )
    }

    ParentSummaryCard(
        title = L.text(context, localePrefs, R.string.mobile_parent_section_weekly),
        empty = L.text(context, localePrefs, R.string.mobile_parent_weekly_empty),
        hasContent = weeklyItems.isNotEmpty(),
    ) {
        weeklyItems.forEach { item ->
            Column {
                Text(item.title, fontWeight = FontWeight.Medium)
                Text("${item.courseTitle} · ${item.kind}", fontSize = 12.sp, color = textSecondary())
            }
        }
    }
}

@Composable
private fun ParentSummaryCard(
    title: String,
    empty: String,
    hasContent: Boolean,
    content: @Composable () -> Unit,
) {
    LmsCard {
        Text(title, fontWeight = FontWeight.Bold, fontSize = 16.sp)
        if (hasContent) {
            Column(Modifier.padding(top = 8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                content()
            }
        } else {
            Text(empty, color = textSecondary(), modifier = Modifier.padding(top = 8.dp))
        }
    }
}

@Composable
private fun ParentActionLinks(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    conferenceEnabled: Boolean,
    onGrades: () -> Unit,
    onAttendance: () -> Unit,
    onPrefs: () -> Unit,
    onConferences: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        ParentLink(L.text(context, localePrefs, R.string.mobile_parent_view_grades), onGrades)
        ParentLink(L.text(context, localePrefs, R.string.mobile_parent_view_attendance), onAttendance)
        ParentLink(L.text(context, localePrefs, R.string.mobile_parent_notification_prefs), onPrefs)
        if (conferenceEnabled) {
            ParentLink(L.text(context, localePrefs, R.string.mobile_parent_book_conferences), onConferences)
        }
    }
}

@Composable
private fun ParentLink(label: String, onClick: () -> Unit) {
    TextButton(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(label, color = accentColor(), fontWeight = FontWeight.Medium)
            Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = accentColor())
        }
    }
}
