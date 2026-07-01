package com.lextures.android.features.courses

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material.icons.filled.Help
import androidx.compose.material.icons.filled.HourglassEmpty
import androidx.compose.material.icons.filled.PanTool
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Verified
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.AttendanceRecord
import com.lextures.android.core.lms.AttendanceSession
import com.lextures.android.core.lms.AttendanceSessionDetail
import com.lextures.android.core.lms.AttendanceStatusInfo
import com.lextures.android.core.lms.CourseSummary
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Science
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.lms.GradeCalculator
import com.lextures.android.core.lms.GradeColumn
import com.lextures.android.core.lms.GradeFeedbackRoute
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.grades.GradesDisplayLogic
import com.lextures.android.features.grades.GradesSection
import com.lextures.android.features.grades.WhatIfState
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.MyGradesResponse
import com.lextures.android.core.lms.SyllabusPayload
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsProgressRing
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

// region Syllabus ("Overview")

/** Syllabus sections rendered as markdown; falls back to the course description. */
@Composable
fun CourseSyllabusSection(
    session: AuthSession,
    course: CourseSummary,
) {
    val accessToken by session.accessToken.collectAsState()
    var syllabus by remember { mutableStateOf<SyllabusPayload?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        // A missing syllabus is expected for many courses — fall back quietly.
        syllabus = runCatching { LmsApi.fetchSyllabus(course.courseCode, token) }.getOrNull()
        loading = false
    }

    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        val payload = syllabus
        when {
            loading -> LmsSkeletonList(count = 2)
            payload != null && payload.hasContent -> {
                if (payload.syllabusAcceptancePending == true) {
                    Text(
                        text = "This course asks you to review and accept the syllabus. You can accept it from the web app.",
                        fontSize = 12.sp,
                        color = LexturesColors.Amber,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(14.dp))
                            .background(LexturesColors.Amber.copy(alpha = 0.11f))
                            .padding(12.dp),
                    )
                }
                payload.sections.forEach { section ->
                    LmsCard {
                        if (section.heading.isNotEmpty()) {
                            Text(
                                text = section.heading,
                                style = LexturesType.display(18),
                                color = textPrimary(),
                            )
                        }
                        MarkdownText(section.markdown)
                    }
                }
                LmsDates.parse(payload.updatedAt)?.let {
                    Text(
                        text = "Updated ${LmsDates.shortDate(payload.updatedAt)}",
                        fontSize = 11.sp,
                        color = textSecondary(),
                        modifier = Modifier.align(Alignment.End),
                    )
                }
            }
            course.description.isNotEmpty() -> {
                LmsCard {
                    Text(text = "About this course", style = LexturesType.display(18), color = textPrimary())
                    Text(
                        text = course.description,
                        fontSize = 14.sp,
                        color = textPrimary(),
                    )
                }
            }
            else -> LmsEmptyState(
                icon = Icons.Default.Description,
                title = "No syllabus yet",
                message = "The course overview will appear here once the instructor adds it.",
            )
        }
    }
}

// endregion

// region Grades

/** Student grades: categories, totals, what-if, and feedback detail (M6.1). */
@Composable
fun CourseGradesSection(
    session: AuthSession,
    course: CourseSummary,
    onOpenFeedback: (GradeFeedbackRoute) -> Unit = {},
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var grades by remember { mutableStateOf<MyGradesResponse?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    val whatIf = remember { WhatIfState() }
    var showWhatIfFeature by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            showWhatIfFeature = runCatching {
                LmsApi.fetchPlatformFeatures(token).ffWhatifGrades
            }.getOrNull() == true
            val result = offline.cachedFetch(
                key = OfflineCacheKey.myGrades(course.courseCode),
                accessToken = token,
                serializer = MyGradesResponse.serializer(),
            ) {
                LmsApi.fetchMyGrades(course.courseCode, token)
            }
            grades = result.first
            val cached = result.second
            cacheLabel = if (cached != null && cached.isStale(isOnline)) cached.lastUpdatedLabel() else null
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(it) }
        val response = grades
        when {
            loading -> LmsSkeletonList(count = 3)
            response == null -> {}
            response.columns.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Verified,
                title = "No graded work yet",
                message = "Grades will appear here as assignments are graded.",
            )
            else -> {
                GradesSummaryCard(response, whatIf)
                if (showWhatIfFeature) {
                    WhatIfPanel(whatIf)
                }
                val dropped = whatIf.activeDropped(response)
                for (section in GradesDisplayLogic.buildSections(response)) {
                    GradesCategoryHeader(section)
                    for (column in section.columns) {
                        GradeRow(
                            column = column,
                            response = response,
                            dropped = dropped,
                            whatIf = whatIf,
                            showWhatIfFeature = showWhatIfFeature,
                            onOpenFeedback = onOpenFeedback,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun GradesSummaryCard(response: MyGradesResponse, whatIf: WhatIfState) {
    val actual = whatIf.actualPercent(response)
    val projected = whatIf.projectedPercent(response)
    val display = if (whatIf.mode && whatIf.hasOverrides) projected else actual

    LmsCard {
        Row(
            horizontalArrangement = Arrangement.spacedBy(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (display != null) {
                LmsProgressRing(progress = (display / 100).toFloat(), size = 56)
            } else {
                Box(
                    modifier = Modifier
                        .size(56.dp)
                        .clip(CircleShape)
                        .background(LexturesColors.BrandTeal.copy(alpha = 0.1f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Default.HourglassEmpty, contentDescription = null, tint = textSecondary())
                }
            }
            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                if (whatIf.mode && whatIf.hasOverrides) {
                    Text(
                        text = "Hypothetical: ${GradeCalculator.formatFinalPercent(projected)}",
                        style = LexturesType.display(20, FontWeight.Bold),
                        color = accentColor(),
                    )
                    Text(
                        text = "Actual: ${GradeCalculator.formatFinalPercent(actual)}",
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                } else {
                    Text(
                        text = GradeCalculator.formatFinalPercent(actual),
                        style = LexturesType.display(22, FontWeight.Bold),
                        color = textPrimary(),
                    )
                    Text(
                        text = if (actual == null) {
                            "Your overall grade appears once work is graded."
                        } else {
                            "Current overall grade"
                        },
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}

@Composable
private fun WhatIfPanel(whatIf: WhatIfState) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.Science, contentDescription = null, tint = accentColor())
                    Text("What-if grades", fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                }
                Switch(checked = whatIf.mode, onCheckedChange = { whatIf.toggleMode() })
            }
            if (whatIf.mode) {
                Text(
                    "Enter hypothetical scores below. These projections are not saved and do not change your real grades.",
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
                if (whatIf.hasOverrides) {
                    Text(
                        "Reset hypothetical scores",
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = accentColor(),
                        modifier = Modifier.clickable { whatIf.reset() },
                    )
                }
            }
        }
    }
}

@Composable
private fun GradesCategoryHeader(section: GradesSection) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(section.title, fontSize = 12.sp, fontWeight = FontWeight.Bold, color = textSecondary())
        section.weightPercent?.let {
            Text("${it.toInt()}%", fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = accentColor())
        }
    }
}

@Composable
private fun GradeRow(
    column: GradeColumn,
    response: MyGradesResponse,
    dropped: Map<String, Boolean>,
    whatIf: WhatIfState,
    showWhatIfFeature: Boolean,
    onOpenFeedback: (GradeFeedbackRoute) -> Unit,
) {
    val isDropped = dropped[column.id] == true
    val isHeld = response.heldGradeItemIds.contains(column.id)
    val isExcused = response.gradeStatuses[column.id] == "excused"
    val score = response.grades[column.id]
    val display = response.displayGrades[column.id]
    val hasOverride = whatIf.overrides[column.id]?.trim()?.isNotEmpty() == true
    val isHypothetical = whatIf.mode && hasOverride
    val badges = GradesDisplayLogic.statusBadges(column, response, dropped)
    val canOpen = !isHeld && column.kind == "assignment" &&
        (!score.isNullOrEmpty() || isExcused || response.gradeStatuses[column.id] == "graded")

    LmsCard(onClick = { if (canOpen) onOpenFeedback(GradeFeedbackRoute(column)) }) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(32.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.16f else 0.13f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(ItemKind.icon(column.kind), contentDescription = null, tint = accentColor(), modifier = Modifier.size(16.dp))
            }
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                Text(
                    text = column.title,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = if (isDropped) textSecondary() else textPrimary(),
                    textDecoration = if (isDropped) TextDecoration.LineThrough else null,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    LmsDates.parse(column.dueAt)?.let {
                        Text("Due ${LmsDates.shortDate(column.dueAt)}", fontSize = 11.sp, color = textSecondary())
                    }
                    badges.forEach { GradeBadge(it, badgeTint(it)) }
                    if (isHypothetical) GradeBadge("Hypothetical", accentColor())
                }
            }
            if (whatIf.mode && showWhatIfFeature && !isExcused && !isHeld && (column.maxPoints ?: 0.0) > 0) {
                OutlinedTextField(
                    value = whatIf.overrides[column.id].orEmpty(),
                    onValueChange = { whatIf.setOverride(column.id, it) },
                    modifier = Modifier.size(width = 72.dp, height = 48.dp),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    singleLine = true,
                    label = null,
                )
            } else {
                GradeScoreColumn(isExcused, isHeld, score, display, column.maxPoints)
            }
            if (canOpen) {
                Icon(Icons.Default.ChevronRight, contentDescription = null, tint = textSecondary(), modifier = Modifier.size(16.dp))
            }
        }
    }
}

@Composable
private fun GradeScoreColumn(
    isExcused: Boolean,
    isHeld: Boolean,
    score: String?,
    display: String?,
    maxPoints: Double?,
) {
    Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(2.dp)) {
        when {
            isExcused -> Text("—", style = LexturesType.display(16, FontWeight.Bold), color = textSecondary())
            isHeld -> Icon(Icons.Default.VisibilityOff, contentDescription = "Grade held", tint = LexturesColors.Amber, modifier = Modifier.size(17.dp))
            !score.isNullOrEmpty() -> {
                Text(
                    text = maxPoints?.let { "$score / ${formatPts(it)}" } ?: score,
                    style = LexturesType.display(16, FontWeight.Bold),
                    color = textPrimary(),
                )
                if (!display.isNullOrEmpty() && display != score) {
                    Text(display, fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = accentColor())
                }
                maxPoints?.takeIf { it > 0 }?.let { max ->
                    score.replace(",", "").toDoubleOrNull()?.let { earned ->
                        Text("${"%.1f".format(earned / max * 100)}%", fontSize = 11.sp, color = textSecondary())
                    }
                }
            }
            else -> Text("Not graded", fontSize = 12.sp, color = textSecondary())
        }
    }
}

@Composable
private fun badgeTint(badge: String): Color = when (badge) {
    "Dropped" -> textSecondary()
    "Pending", "Late" -> LexturesColors.Amber
    "Excused" -> accentColor()
    else -> textSecondary()
}

@Composable
private fun GradeBadge(text: String, tint: Color) {
    Text(
        text = text,
        fontSize = 10.sp,
        fontWeight = FontWeight.SemiBold,
        color = tint,
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .background(tint.copy(alpha = 0.13f))
            .padding(horizontal = 6.dp, vertical = 2.dp),
    )
}

private fun formatPts(points: Double): String =
    if (points % 1.0 == 0.0) points.toLong().toString() else points.toString()

// endregion

// region Attendance

/** Session list; students self-report from the detail screen. */
@Composable
fun CourseAttendanceSection(
    session: AuthSession,
    course: CourseSummary,
    onOpenSession: (AttendanceSession) -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    var sessions by remember { mutableStateOf<List<AttendanceSession>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            sessions = LmsApi.fetchAttendanceSessions(course.courseCode, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        errorMessage?.let { LmsErrorBanner(it) }
        when {
            loading && sessions.isEmpty() -> LmsSkeletonList(count = 3)
            sessions.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.FactCheck,
                title = "No attendance sessions",
                message = "Attendance sessions will appear here when your instructor opens one.",
            )
            else -> sessions.forEach { attendanceSession ->
                LmsCard(
                    accent = if (attendanceSession.isOpen) LexturesColors.BrandTeal else null,
                    onClick = { onOpenSession(attendanceSession) },
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier
                                .size(32.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(
                                    LexturesColors.BrandTeal.copy(alpha = if (isDarkTheme()) 0.16f else 0.13f),
                                ),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                if (attendanceSession.isSelfReport) Icons.Default.PanTool else Icons.Default.FactCheck,
                                contentDescription = null,
                                tint = accentColor(),
                                modifier = Modifier.size(16.dp),
                            )
                        }
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(
                                text = attendanceSession.displayTitle,
                                fontSize = 14.sp,
                                fontWeight = FontWeight.Medium,
                                color = textPrimary(),
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                attendanceSession.sessionDate?.takeIf { it.isNotEmpty() }?.let {
                                    Text(
                                        text = LmsDates.shortDate(it).ifEmpty { it },
                                        fontSize = 11.sp,
                                        color = textSecondary(),
                                    )
                                }
                                Text(
                                    text = if (attendanceSession.isSelfReport) "Self report" else "Roll call",
                                    fontSize = 11.sp,
                                    color = textSecondary(),
                                )
                            }
                        }
                        Text(
                            text = if (attendanceSession.isOpen) "Open" else "Closed",
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = if (attendanceSession.isOpen) accentColor() else textSecondary(),
                            modifier = Modifier
                                .clip(RoundedCornerShape(50))
                                .background(
                                    if (attendanceSession.isOpen) {
                                        LexturesColors.BrandTeal.copy(alpha = 0.16f)
                                    } else {
                                        textSecondary().copy(alpha = 0.12f)
                                    },
                                )
                                .padding(horizontal = 8.dp, vertical = 3.dp),
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = textSecondary().copy(alpha = 0.6f),
                            modifier = Modifier.size(16.dp),
                        )
                    }
                }
            }
        }
    }
}

/** One attendance session: my status + self-report (students) or records (staff). */
@Composable
fun AttendanceSessionDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    attendanceSession: AttendanceSession,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var detail by remember { mutableStateOf<AttendanceSessionDetail?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var reporting by remember { mutableStateOf(false) }
    var reloadKey by remember { mutableIntStateOf(0) }

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken, attendanceSession.id, reloadKey) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            detail = LmsApi.fetchAttendanceSessionDetail(course.courseCode, attendanceSession.id, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

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
                text = attendanceSession.displayTitle,
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

            val loaded = detail
            if (loading && loaded == null) {
                item { LmsSkeletonList(count = 2) }
                return@LazyColumn
            }
            if (loaded == null) return@LazyColumn

            loaded.myRecord?.let { record ->
                item {
                    LmsCard {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(44.dp)
                                    .clip(CircleShape)
                                    .background(statusTint(record.status).copy(alpha = 0.13f)),
                                contentAlignment = Alignment.Center,
                            ) {
                                Icon(
                                    statusIcon(record.status),
                                    contentDescription = null,
                                    tint = statusTint(record.status),
                                    modifier = Modifier.size(22.dp),
                                )
                            }
                            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                Text(
                                    text = AttendanceStatusInfo.label(record.status),
                                    style = LexturesType.display(18),
                                    color = textPrimary(),
                                )
                                LmsDates.parse(record.recordedAt)?.let {
                                    Text(
                                        text = "Recorded ${LmsDates.shortDateTime(record.recordedAt)}",
                                        fontSize = 12.sp,
                                        color = textSecondary(),
                                    )
                                }
                            }
                        }
                    }
                }
            }

            if (loaded.canSelfReport == true && attendanceSession.isOpen) {
                item {
                    LmsCard {
                        Text(text = "Report your attendance", style = LexturesType.display(17), color = textPrimary())
                        Text(
                            text = "This session is open for self-reporting.",
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                            ReportButton("I'm here", LexturesColors.Primary, reporting, Modifier.weight(1f)) {
                                val token = accessToken ?: return@ReportButton
                                scope.launch {
                                    reporting = true
                                    runCatching {
                                        LmsApi.selfReportAttendance(course.courseCode, attendanceSession.id, "present", token)
                                    }.onFailure { errorMessage = session.mapError(it) }
                                    reporting = false
                                    reloadKey++
                                }
                            }
                            ReportButton("I'm late", LexturesColors.Amber, reporting, Modifier.weight(1f)) {
                                val token = accessToken ?: return@ReportButton
                                scope.launch {
                                    reporting = true
                                    runCatching {
                                        LmsApi.selfReportAttendance(course.courseCode, attendanceSession.id, "tardy", token)
                                    }.onFailure { errorMessage = session.mapError(it) }
                                    reporting = false
                                    reloadKey++
                                }
                            }
                        }
                    }
                }
            }

            val records = loaded.records.orEmpty()
            if (records.isNotEmpty()) {
                item {
                    LmsCard {
                        Text(text = "Roster", style = LexturesType.display(17), color = textPrimary())
                        records.forEachIndexed { index, record ->
                            if (index > 0) HorizontalDivider()
                            RosterRow(record)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ReportButton(
    label: String,
    tint: Color,
    busy: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    Box(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(tint)
            .clickable(enabled = !busy, onClick = onClick)
            .padding(vertical = 11.dp),
        contentAlignment = Alignment.Center,
    ) {
        if (busy) {
            CircularProgressIndicator(
                color = Color.White,
                modifier = Modifier.size(16.dp),
                strokeWidth = 2.dp,
            )
        } else {
            Text(
                text = label,
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                color = Color.White,
            )
        }
    }
}

@Composable
private fun RosterRow(record: AttendanceRecord) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = record.displayName ?: "Student",
            fontSize = 14.sp,
            color = textPrimary(),
            modifier = Modifier.weight(1f),
        )
        Text(
            text = AttendanceStatusInfo.label(record.status),
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
            color = statusTint(record.status),
            modifier = Modifier
                .clip(RoundedCornerShape(50))
                .background(statusTint(record.status).copy(alpha = 0.12f))
                .padding(horizontal = 8.dp, vertical = 3.dp),
        )
    }
}

private fun statusIcon(status: String): ImageVector = when (status) {
    "present" -> Icons.Default.CheckCircle
    "absent" -> Icons.Default.Cancel
    "tardy" -> Icons.Default.Schedule
    "excused" -> Icons.Default.Verified
    else -> Icons.Default.Help
}

private fun statusTint(status: String): Color = when (status) {
    "present" -> LexturesColors.Primary
    "absent" -> LexturesColors.Error
    "tardy" -> LexturesColors.Amber
    "excused" -> LexturesColors.BrandTeal
    else -> LexturesColors.TextSecondaryDark
}

// endregion
