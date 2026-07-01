package com.lextures.android.features.courses

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.AttendanceMarkStatus
import com.lextures.android.core.lms.AttendanceRecord
import com.lextures.android.core.lms.AttendanceStatusInfo
import com.lextures.android.core.lms.AttendanceSessionDetail
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateAttendanceSessionBody
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.SaveAttendanceRecordsBody
import com.lextures.android.core.lms.TakeAttendanceLogic
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.R
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TakeAttendanceScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    initialSessionId: String? = null,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val json = remember { Json { ignoreUnknownKeys = true } }

    var activeSessionId by remember { mutableStateOf(initialSessionId) }
    var sessionDetail by remember { mutableStateOf<AttendanceSessionDetail?>(null) }
    var draft by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var sections by remember { mutableStateOf<List<CourseSection>>(emptyList()) }
    var selectedSectionId by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var creating by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var reloadKey by remember { mutableStateOf(0) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, course.courseCode, initialSessionId, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        if (!course.viewerIsStaff) {
            loading = false
            return@LaunchedEffect
        }
        loading = true
        errorMessage = null
        try {
            if (course.isSectionsEnabled) {
                sections = LmsApi.fetchCourseSections(course.courseCode, token)
                if (selectedSectionId.isEmpty()) {
                    selectedSectionId = sections.firstOrNull()?.id.orEmpty()
                }
            }
            if (activeSessionId == null) {
                val sessions = LmsApi.fetchAttendanceSessions(course.courseCode, token)
                val today = TakeAttendanceLogic.findTodaysOpenRollCallSession(sessions)
                if (today != null) {
                    activeSessionId = today.id
                }
            }
            activeSessionId?.let { sessionId ->
                val detail = LmsApi.fetchAttendanceSessionDetail(course.courseCode, sessionId, token)
                sessionDetail = detail
                draft = detail.records?.let { TakeAttendanceLogic.buildDraft(it) }.orEmpty()
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    val records = sessionDetail?.records.orEmpty()
    val isOpen = sessionDetail?.status == "open"
    val counts = TakeAttendanceLogic.summaryCounts(records, draft)

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = L.text(R.string.mobile_attendance_take_title),
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            if (loading && sessionDetail == null) {
                item { LmsSkeletonList(count = 4) }
                return@LazyColumn
            }

            if (activeSessionId == null) {
                item {
                    StartSessionCard(
                        course = course,
                        sections = sections,
                        selectedSectionId = selectedSectionId,
                        onSectionChange = { selectedSectionId = it },
                        creating = creating,
                        onStart = {
                            val token = accessToken ?: return@StartSessionCard
                            scope.launch {
                                creating = true
                                errorMessage = null
                                val today = TakeAttendanceLogic.todayDateString()
                                runCatching {
                                    val created = LmsApi.createAttendanceSession(
                                        courseCode = course.courseCode,
                                        body = CreateAttendanceSessionBody(
                                            collectionMethod = "roll_call",
                                            title = "Roll call — $today",
                                            sessionDate = today,
                                            sectionId = selectedSectionId.takeIf { it.isNotEmpty() },
                                        ),
                                        accessToken = token,
                                    )
                                    activeSessionId = created.id
                                    reloadKey++
                                }.onFailure { errorMessage = session.mapError(it) }
                                creating = false
                            }
                        },
                    )
                }
                return@LazyColumn
            }

            item {
                LmsCard {
                    Text(
                        text = L.format(
                            R.string.mobile_attendance_take_summary,
                            counts.present,
                            counts.absent,
                            counts.tardy,
                            counts.excused,
                        ),
                        style = LexturesType.display(16),
                        color = textPrimary(),
                    )
                }
            }

            if (isOpen) {
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        ActionChip(
                            label = L.text(R.string.mobile_attendance_take_markAllPresent),
                            filled = false,
                            onClick = { draft = TakeAttendanceLogic.markAllPresent(records) },
                        )
                        val saveLabel = L.text(R.string.mobile_attendance_take_save)
                        val closeLabel = L.text(R.string.mobile_attendance_take_close)
                        ActionChip(
                            label = if (saving) "…" else saveLabel,
                            filled = true,
                            enabled = !saving && records.isNotEmpty(),
                            onClick = {
                                val token = accessToken ?: return@ActionChip
                                val sessionId = activeSessionId ?: return@ActionChip
                                scope.launch {
                                    saving = true
                                    errorMessage = null
                                    val payload = TakeAttendanceLogic.recordsPayload(records, draft)
                                    val bodyJson = json.encodeToString(SaveAttendanceRecordsBody(payload))
                                    runCatching {
                                        offline.enqueueMutation(
                                            method = "PUT",
                                            path = "/api/v1/courses/${course.courseCode}/attendance/sessions/$sessionId/records",
                                            bodyJson = bodyJson,
                                            label = saveLabel,
                                            accessToken = token,
                                        )
                                        reloadKey++
                                    }.onFailure { errorMessage = session.mapError(it) }
                                    saving = false
                                }
                            },
                        )
                        ActionChip(
                            label = closeLabel,
                            filled = false,
                            enabled = !saving,
                            onClick = {
                                val token = accessToken ?: return@ActionChip
                                val sessionId = activeSessionId ?: return@ActionChip
                                scope.launch {
                                    saving = true
                                    errorMessage = null
                                    runCatching {
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = "/api/v1/courses/${course.courseCode}/attendance/sessions/$sessionId/close",
                                            bodyJson = """{"finalizeMissingAsAbsent":true}""",
                                            label = closeLabel,
                                            accessToken = token,
                                        )
                                        reloadKey++
                                    }.onFailure { errorMessage = session.mapError(it) }
                                    saving = false
                                }
                            },
                        )
                    }
                }
            }

            if (records.isEmpty()) {
                item {
                    LmsEmptyState(
                        icon = Icons.Default.FactCheck,
                        title = L.text(R.string.mobile_attendance_take_noRoster),
                        message = "",
                    )
                }
            } else {
                item {
                    LmsCard {
                        Text(
                            text = sessionDetail?.title ?: L.text(R.string.mobile_attendance_take_title),
                            style = LexturesType.display(17),
                            color = textPrimary(),
                        )
                        records.forEachIndexed { index, record ->
                            if (index > 0) HorizontalDivider()
                            StudentAttendanceRow(
                                record = record,
                                status = draft[record.studentUserId] ?: record.status,
                                editable = isOpen,
                                onStatusChange = { draft = draft + (record.studentUserId to it) },
                            )
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun StartSessionCard(
    course: CourseSummary,
    sections: List<CourseSection>,
    selectedSectionId: String,
    onSectionChange: (String) -> Unit,
    creating: Boolean,
    onStart: () -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    val selectedSection = sections.firstOrNull { it.id == selectedSectionId }

    LmsCard {
        Text(
            text = L.text(R.string.mobile_attendance_take_newSessionHint),
            fontSize = 14.sp,
            color = textSecondary(),
        )

        if (course.isSectionsEnabled && sections.isNotEmpty()) {
            ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                TextField(
                    value = selectedSection?.displayName.orEmpty(),
                    onValueChange = {},
                    readOnly = true,
                    label = { Text(L.text(R.string.mobile_attendance_take_section)) },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
                    modifier = Modifier
                        .menuAnchor()
                        .fillMaxWidth(),
                )
                ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                    sections.forEach { section ->
                        DropdownMenuItem(
                            text = { Text(section.displayName) },
                            onClick = {
                                onSectionChange(section.id)
                                expanded = false
                            },
                        )
                    }
                }
            }
        }

        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(12.dp))
                .background(accentColor())
                .clickable(enabled = !creating, onClick = onStart)
                .padding(vertical = 11.dp),
            contentAlignment = Alignment.Center,
        ) {
            if (creating) {
                CircularProgressIndicator(
                    color = if (isDarkTheme()) LexturesColors.PrimaryDeep else Color.White,
                    modifier = Modifier.padding(2.dp),
                    strokeWidth = 2.dp,
                )
            } else {
                Text(
                    text = L.text(R.string.mobile_attendance_take_start),
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = if (isDarkTheme()) LexturesColors.PrimaryDeep else Color.White,
                )
            }
        }
    }
}

@Composable
private fun ActionChip(
    label: String,
    filled: Boolean,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    val shape = RoundedCornerShape(50)
    val modifier = if (filled) {
        Modifier
            .clip(shape)
            .background(accentColor())
    } else {
        Modifier
            .clip(shape)
            .border(1.dp, textSecondary().copy(alpha = 0.35f), shape)
    }
    Box(
        modifier = modifier
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 8.dp),
    ) {
        Text(
            text = label,
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
            color = if (filled) {
                if (isDarkTheme()) LexturesColors.PrimaryDeep else Color.White
            } else {
                textSecondary()
            },
        )
    }
}

@Composable
private fun StudentAttendanceRow(
    record: AttendanceRecord,
    status: String,
    editable: Boolean,
    onStatusChange: (String) -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            text = TakeAttendanceLogic.studentLabel(record),
            fontSize = 14.sp,
            fontWeight = FontWeight.Medium,
            color = textPrimary(),
        )
        if (editable) {
            val selected = AttendanceMarkStatus.entries.firstOrNull { it.raw == status }
                ?: AttendanceMarkStatus.Present
            LmsSegmentedChips(
                options = AttendanceMarkStatus.markable.map { it.raw to AttendanceStatusInfo.label(it.raw) },
                selectedId = selected.raw,
                onSelect = onStatusChange,
            )
        } else {
            Text(
                text = AttendanceStatusInfo.label(status),
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
            )
        }
    }
}