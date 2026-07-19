package com.lextures.android.features.live

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Videocam
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CourseWhiteboard
import com.lextures.android.core.lms.LiveMeetingsLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MeetingAttendanceRecord
import com.lextures.android.core.lms.VirtualMeeting
import com.lextures.android.core.lms.WhiteboardLogic
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.courses.CourseDestinationPlaceholder
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.R
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun CourseLiveSection(
    session: AuthSession,
    course: CourseSummary,
    platformFeatures: MobilePlatformFeatures = MobilePlatformFeatures(),
) {
    if (!platformFeatures.ffMobileLiveMeetings) {
        CourseDestinationPlaceholder(section = com.lextures.android.core.navigation.CourseWorkspaceSection.Live)
        return
    }
    MeetingListScreen(session = session, course = course, platformFeatures = platformFeatures)
}

@Composable
fun MeetingListScreen(
    session: AuthSession,
    course: CourseSummary,
    platformFeatures: MobilePlatformFeatures = MobilePlatformFeatures(),
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var meetings by remember { mutableStateOf<List<VirtualMeeting>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var selectedMeeting by remember { mutableStateOf<VirtualMeeting?>(null) }
    var openWhiteboard by remember { mutableStateOf<CourseWhiteboard?>(null) }

    suspend fun load(force: Boolean = false) {
        val token = accessToken ?: return
        if (!force && meetings.isNotEmpty()) return
        loading = meetings.isEmpty()
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.liveMeetings(course.courseCode),
                accessToken = token,
                serializer = kotlinx.serialization.serializer<List<VirtualMeeting>>(),
            ) { LmsApi.fetchCourseMeetings(course.courseCode, token) }
            meetings = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = context.getString(R.string.mobile_live_error_load)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, course.courseCode) { load() }

    LaunchedEffect(meetings) {
        val grouped = LiveMeetingsLogic.groupMeetings(meetings)
        if (grouped.live.isEmpty() && grouped.upcoming.none { LiveMeetingsLogic.isLiveOrSoon(it) }) return@LaunchedEffect
        while (true) {
            delay(30_000)
            load(force = true)
        }
    }

    val grouped = LiveMeetingsLogic.groupMeetings(meetings)

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(it) }
        errorMessage?.let { LmsErrorBanner(it) }

        if (!grouped.live.isEmpty()) {
            LmsCard(accent = accentColor()) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        if (grouped.live.size > 1) liveBannerMultiple(grouped.live.size) else liveBannerSingle(),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Button(onClick = {
                        scope.launch { joinMeeting(session, course, grouped.live.first(), context) { errorMessage = it } }
                    }) { Text(liveJoinNow()) }
                }
            }
        }

        when {
            loading -> LmsSkeletonList(count = 3)
            meetings.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Videocam,
                title = liveEmptyTitle(),
                message = if (course.viewerIsStaff) liveEmptyStaffMessage() else liveEmptyMessage(),
            )
            else -> {
                meetingSection(liveSectionLiveNow(), grouped.live, session, course, context, onError = { errorMessage = it }) { selectedMeeting = it }
                meetingSection(liveSectionUpcoming(), grouped.upcoming, session, course, context, onError = { errorMessage = it }) { selectedMeeting = it }
                meetingSection(liveSectionPast(), grouped.past, session, course, context, onError = { errorMessage = it }) { selectedMeeting = it }
            }
        }

        if (course.viewerIsStaff) {
            Text(liveManageOnWeb(), fontSize = 12.sp, color = textSecondary())
        }
    }

    selectedMeeting?.let { meeting ->
        MeetingDetailDialog(
            session = session,
            course = course,
            meeting = meeting,
            platformFeatures = platformFeatures,
            onDismiss = { selectedMeeting = null },
            onUpdated = { updated ->
                meetings = meetings.map { if (it.id == updated.id) updated else it }
                    .filterNot { it.status == "cancelled" }
                if (updated.status == "cancelled") selectedMeeting = null
            },
            onOpenWhiteboard = { board ->
                selectedMeeting = null
                openWhiteboard = board
            },
        )
    }

    openWhiteboard?.let { board ->
        WhiteboardDialog(
            session = session,
            course = course,
            board = board,
            canEdit = WhiteboardLogic.canEdit(course.viewerIsStaff, platformFeatures),
            onDismiss = { openWhiteboard = null },
            onDeleted = { openWhiteboard = null },
        )
    }
}

@Composable
private fun meetingSection(
    title: String,
    meetings: List<VirtualMeeting>,
    session: AuthSession,
    course: CourseSummary,
    context: android.content.Context,
    onError: (String) -> Unit,
    onSelect: (VirtualMeeting) -> Unit,
) {
    if (meetings.isEmpty()) return
    val scope = rememberCoroutineScope()
    Text(title.uppercase(), fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = accentColor())
    meetings.forEach { meeting ->
        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(meeting.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                Text(LiveMeetingsLogic.formatMeetingTime(meeting), fontSize = 12.sp, color = textSecondary())
                Text(meeting.provider, fontSize = 11.sp, color = textSecondary())
                meeting.scheduledStart?.let { LiveMeetingsLogic.countdownText(it) }?.let {
                    Text(it, fontSize = 11.sp, color = accentColor())
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (LiveMeetingsLogic.canJoin(meeting)) {
                        Button(onClick = {
                            scope.launch { joinMeeting(session, course, meeting, context, onError) }
                        }) { Text(if (meeting.status == "live") liveJoinNow() else liveJoin()) }
                    }
                    OutlinedButton(onClick = { onSelect(meeting) }) { Text(liveDetails()) }
                    OutlinedButton(onClick = {
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(LiveMeetingsLogic.meetingIcalUrl(meeting.id))))
                    }) { Text(liveAddToCalendar()) }
                }
            }
        }
    }
}

private suspend fun joinMeeting(
    session: AuthSession,
    course: CourseSummary,
    meeting: VirtualMeeting,
    context: android.content.Context,
    onError: (String) -> Unit,
) {
    val token = session.accessToken.value ?: return
    val info = runCatching { LmsApi.fetchMeetingJoinInfo(meeting.id, token) }.getOrNull()
    val url = (if (course.viewerIsStaff) info?.hostUrl else null) ?: info?.joinUrl
    if (url.isNullOrBlank()) {
        onError(context.getString(R.string.mobile_live_error_join))
        return
    }
    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
}

@Composable
private fun MeetingDetailDialog(
    session: AuthSession,
    course: CourseSummary,
    meeting: VirtualMeeting,
    platformFeatures: MobilePlatformFeatures,
    onDismiss: () -> Unit,
    onUpdated: (VirtualMeeting) -> Unit,
    onOpenWhiteboard: (CourseWhiteboard) -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var current by remember(meeting.id) { mutableStateOf(meeting) }
    var attendance by remember { mutableStateOf<List<MeetingAttendanceRecord>>(emptyList()) }
    var whiteboards by remember { mutableStateOf<List<CourseWhiteboard>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var updating by remember { mutableStateOf(false) }
    var creatingWhiteboard by remember { mutableStateOf(false) }
    val canEditWhiteboard = WhiteboardLogic.canEdit(course.viewerIsStaff, platformFeatures)

    LaunchedEffect(accessToken, current.id) {
        val token = accessToken ?: return@LaunchedEffect
        if (!course.viewerIsStaff) return@LaunchedEffect
        attendance = runCatching { LmsApi.fetchMeetingAttendance(current.id, token) }.getOrDefault(emptyList())
        whiteboards = runCatching { LmsApi.fetchCourseWhiteboards(course.courseCode, token) }.getOrDefault(emptyList())
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(current.title, fontWeight = FontWeight.SemiBold) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(LiveMeetingsLogic.formatMeetingTime(current), color = textSecondary())
                errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red, fontSize = 13.sp) }
                if (LiveMeetingsLogic.canJoin(current)) {
                    Button(onClick = {
                        scope.launch {
                            joinMeeting(session, course, current, context) { errorMessage = it }
                        }
                    }) { Text(liveJoinNow()) }
                }
                if (course.viewerIsStaff) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        if (current.status == "scheduled") {
                            OutlinedButton(
                                enabled = !updating,
                                onClick = {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        updating = true
                                        runCatching {
                                            onUpdated(LmsApi.patchMeeting(current.id, "live", token))
                                        }.onFailure { errorMessage = it.message }
                                        updating = false
                                    }
                                },
                            ) { Text(liveStartSession()) }
                        }
                        if (current.status == "live") {
                            OutlinedButton(
                                enabled = !updating,
                                onClick = {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        updating = true
                                        runCatching {
                                            onUpdated(LmsApi.patchMeeting(current.id, "ended", token))
                                        }.onFailure { errorMessage = it.message }
                                        updating = false
                                    }
                                },
                            ) { Text(liveEndSession()) }
                        }
                    }
                    Text(liveAttendanceCount(attendance.size), fontSize = 12.sp, color = textSecondary())
                    Text(liveWhiteboardTitle(), fontWeight = FontWeight.SemiBold, color = textPrimary())
                    if (canEditWhiteboard) {
                        OutlinedButton(
                            enabled = !creatingWhiteboard,
                            onClick = {
                                scope.launch {
                                    val token = accessToken ?: return@launch
                                    creatingWhiteboard = true
                                    runCatching {
                                        LmsApi.createCourseWhiteboard(
                                            course.courseCode,
                                            WhiteboardLogic.defaultTitle(whiteboards.size),
                                            emptyList(),
                                            token,
                                        )
                                    }.onSuccess { created ->
                                        whiteboards = listOf(created) + whiteboards
                                        onOpenWhiteboard(created)
                                    }.onFailure {
                                        errorMessage = context.getString(R.string.mobile_whiteboard_error_create)
                                    }
                                    creatingWhiteboard = false
                                }
                            },
                        ) { Text(L.text(R.string.mobile_whiteboard_create)) }
                    }
                    whiteboards.forEach { board ->
                        OutlinedButton(onClick = { onOpenWhiteboard(board) }) {
                            Text(board.title)
                        }
                    }
                    if (!canEditWhiteboard) {
                        Text(liveWhiteboardWebHint(), fontSize = 11.sp, color = textSecondary())
                    }
                }
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text(liveClose()) } },
    )
}

@Composable
fun LiveMeetingsRail(
    items: List<LiveMeetingsLogic.LiveUpcomingItem>,
    courses: List<CourseSummary>,
    session: AuthSession,
    onOpenCourse: (CourseSummary) -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(liveRailTitle(), fontWeight = FontWeight.SemiBold, color = textPrimary())
        Row(
            modifier = Modifier.horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            items.forEach { item ->
                LmsCard(modifier = Modifier.width(240.dp)) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(item.meeting.title, fontWeight = FontWeight.SemiBold, maxLines = 2)
                        Text(item.courseTitle, fontSize = 12.sp, color = textSecondary())
                        Text(LiveMeetingsLogic.formatMeetingTime(item.meeting), fontSize = 11.sp, color = textSecondary())
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Button(onClick = {
                                scope.launch {
                                    joinMeeting(session, courses.first { it.courseCode == item.courseCode }, item.meeting, context) {}
                                }
                            }) { Text(liveJoin()) }
                            OutlinedButton(onClick = {
                                courses.firstOrNull { it.courseCode == item.courseCode }?.let(onOpenCourse)
                            }) { Text(liveOpenCourse()) }
                        }
                    }
                }
            }
        }
    }
}