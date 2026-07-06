package com.lextures.android.features.instructorinsights

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.AlertDialog
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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.AtRiskAlert
import com.lextures.android.core.lms.AtRiskListResponse
import com.lextures.android.core.lms.CourseHealthSnapshot
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.EnrollmentMessageBody
import com.lextures.android.core.lms.InstructorInsightsLogic
import com.lextures.android.core.lms.InstructorInsightsResponse
import com.lextures.android.core.lms.InstructorSignalItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.StudentProgressResponse
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun CourseInsightsSection(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    platformFeatures: MobilePlatformFeatures,
    shell: HomeShellState?,
    onOpenAtRisk: () -> Unit,
    onOpenWhatsWorking: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var snapshot by remember { mutableStateOf<CourseHealthSnapshot?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var atRiskError by remember { mutableStateOf<String?>(null) }
    var insightsError by remember { mutableStateOf<String?>(null) }
    var backlogError by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        var atRiskCount = 0
        var ungradedCount = 0
        var working = emptyList<InstructorSignalItem>()
        var attention = emptyList<InstructorSignalItem>()
        var stale: String? = null

        if (platformFeatures.atRiskAlertsEnabled) {
            runCatching {
                val result = offline.cachedFetch(
                    key = OfflineCacheKey.courseAtRisk(course.courseCode),
                    accessToken = token,
                    serializer = AtRiskListResponse.serializer(),
                ) { LmsApi.fetchCourseAtRisk(course.courseCode, token) }
                atRiskCount = result.first.alerts.size
                result.second?.takeIf { it.isStale(isOnline) }?.let { stale = it.lastUpdatedLabel() }
            }.onFailure {
                atRiskError = L.text(context, localePrefs, R.string.mobile_instructorInsights_error_atRisk)
            }
        }

        if (course.viewerIsStaff) {
            runCatching {
                ungradedCount = LmsApi.fetchGradingBacklog(course.courseCode, token).size
            }.onFailure {
                backlogError = L.text(context, localePrefs, R.string.mobile_instructorInsights_error_backlog)
            }
        }

        if (platformFeatures.instructorInsightsEnabled) {
            runCatching {
                val result = offline.cachedFetch(
                    key = OfflineCacheKey.courseInstructorInsights(course.courseCode),
                    accessToken = token,
                    serializer = InstructorInsightsResponse.serializer(),
                ) { LmsApi.fetchInstructorInsights(course.courseCode, token) }
                working = result.first.workingWell
                attention = result.first.needsAttention
                if (stale == null) {
                    result.second?.takeIf { it.isStale(isOnline) }?.let { stale = it.lastUpdatedLabel() }
                }
            }.onFailure {
                insightsError = L.text(context, localePrefs, R.string.mobile_instructorInsights_error_insights)
            }
        }

        snapshot = InstructorInsightsLogic.snapshot(atRiskCount, ungradedCount, working, attention)
        cacheLabel = stale
        loading = false
    }

    Column(modifier = modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        Text(
            L.text(context, localePrefs, R.string.mobile_instructorInsights_predictedNotice),
            color = textSecondary(),
            style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
        )
        if (loading && snapshot == null) {
            LmsSkeletonList(count = 3)
        } else {
            snapshot?.let { snap ->
                if (platformFeatures.atRiskAlertsEnabled) {
                    SnapshotCard(
                        title = L.text(context, localePrefs, R.string.mobile_instructorInsights_snapshot_atRisk),
                        value = L.plural(context, localePrefs, R.plurals.mobile_instructorInsights_snapshot_atRiskCount, snap.atRiskCount),
                        error = atRiskError,
                    )
                }
                if (course.viewerIsStaff) {
                    LmsCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { shell?.activeCourseSection = CourseWorkspaceSection.Grading },
                    ) {
                        SnapshotRow(
                            title = L.text(context, localePrefs, R.string.mobile_instructorInsights_snapshot_ungraded),
                            value = L.plural(context, localePrefs, R.plurals.mobile_instructorInsights_snapshot_ungradedCount, snap.ungradedCount),
                            error = backlogError,
                            showChevron = true,
                        )
                    }
                }
                if (platformFeatures.instructorInsightsEnabled) {
                    SnapshotCard(
                        title = L.text(context, localePrefs, R.string.mobile_instructorInsights_snapshot_engagement),
                        value = L.plural(context, localePrefs, R.plurals.mobile_instructorInsights_snapshot_engagementCount, snap.engagementHighlightCount),
                        error = insightsError,
                    )
                }
            }
            if (platformFeatures.atRiskAlertsEnabled) {
                ActionRow(
                    title = L.text(context, localePrefs, R.string.mobile_instructorInsights_atRisk_title),
                    subtitle = L.text(context, localePrefs, R.string.mobile_instructorInsights_atRisk_subtitle),
                    onClick = onOpenAtRisk,
                )
            }
            if (platformFeatures.instructorInsightsEnabled) {
                ActionRow(
                    title = L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_title),
                    subtitle = L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_subtitle),
                    onClick = onOpenWhatsWorking,
                )
            }
            LmsCard(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable {
                        val url = AppConfiguration.webUrl(InstructorInsightsLogic.webReportsPath(course.courseCode))
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    },
            ) {
                Column(Modifier.padding(12.dp)) {
                    Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_webReports), fontWeight = FontWeight.SemiBold)
                    Text(
                        L.text(context, localePrefs, R.string.mobile_instructorInsights_webReportsHint),
                        color = textSecondary(),
                        style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AtRiskListScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    platformFeatures: MobilePlatformFeatures,
    onBack: () -> Unit,
    onOpenStudent: (String, String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var alerts by remember { mutableStateOf<List<AtRiskAlert>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        runCatching {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.courseAtRisk(course.courseCode),
                accessToken = token,
                serializer = AtRiskListResponse.serializer(),
            ) { LmsApi.fetchCourseAtRisk(course.courseCode, token) }
            alerts = InstructorInsightsLogic.sortAlerts(result.first.alerts)
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure {
            errorMessage = L.text(context, localePrefs, R.string.mobile_instructorInsights_error_atRisk)
        }
        loading = false
    }

    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_atRisk_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            Modifier.padding(padding).padding(16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (!isOnline) OfflineBanner()
            cacheLabel?.let { StalenessChip(label = it) }
            errorMessage?.let { LmsErrorBanner(message = it) }
            if (loading && alerts.isEmpty()) LmsSkeletonList(count = 4)
            else if (alerts.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Default.Warning,
                    title = L.text(context, localePrefs, R.string.mobile_instructorInsights_atRisk_empty),
                    message = L.text(context, localePrefs, R.string.mobile_instructorInsights_atRisk_emptyHint),
                )
            } else {
                alerts.forEach { alert ->
                    LmsCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable(enabled = platformFeatures.studentProgressEnabled) {
                                onOpenStudent(alert.enrollmentId, alert.displayName)
                            },
                    ) {
                        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(alert.displayName, fontWeight = FontWeight.SemiBold)
                            val severity = InstructorInsightsLogic.severity(alert.score)
                            Text(
                                "${L.text(context, localePrefs, severity.labelRes)} (${alert.score.toInt()})",
                                color = accentColor(),
                                style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
                            )
                            Text(alert.topFactorLabel, color = textSecondary(), style = androidx.compose.material3.MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WhatsWorkingScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var workingWell by remember { mutableStateOf<List<InstructorSignalItem>>(emptyList()) }
    var needsAttention by remember { mutableStateOf<List<InstructorSignalItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        runCatching {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.courseInstructorInsights(course.courseCode),
                accessToken = token,
                serializer = InstructorInsightsResponse.serializer(),
            ) { LmsApi.fetchInstructorInsights(course.courseCode, token) }
            workingWell = result.first.workingWell
            needsAttention = result.first.needsAttention
        }
        loading = false
    }

    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            Modifier.padding(padding).padding(16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (loading) LmsSkeletonList(count = 3)
            else if (workingWell.isEmpty() && needsAttention.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Default.BarChart,
                    title = L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_empty),
                    message = L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_emptyHint),
                )
            } else {
                if (workingWell.isNotEmpty()) {
                    Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_working), fontWeight = FontWeight.SemiBold)
                    workingWell.forEach { SignalCard(it) }
                }
                if (needsAttention.isNotEmpty()) {
                    Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_whatsWorking_attention), fontWeight = FontWeight.SemiBold)
                    needsAttention.forEach { SignalCard(it) }
                }
            }
        }
    }
}

@Composable
private fun SignalCard(item: InstructorSignalItem) {
    LmsCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(item.title, fontWeight = FontWeight.SemiBold)
            Text(item.narrative, color = textSecondary(), style = androidx.compose.material3.MaterialTheme.typography.bodySmall)
            Text(
                InstructorInsightsLogic.completionPercentText(item.completionRate) +
                    (InstructorInsightsLogic.optionalPercentText(item.avgScore)?.let { " · $it avg" } ?: ""),
                color = textSecondary(),
                style = androidx.compose.material3.MaterialTheme.typography.labelSmall,
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StudentProgressScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    enrollmentId: String,
    displayName: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val localized = remember(localePrefs) { localePrefs.localizedContext(context) }
    val scope = rememberCoroutineScope()
    var progress by remember { mutableStateOf<StudentProgressResponse?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showMessage by remember { mutableStateOf(false) }
    var subject by remember { mutableStateOf("") }
    var body by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken, enrollmentId) {
        val token = accessToken ?: return@LaunchedEffect
        runCatching {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.studentProgress(course.courseCode, enrollmentId),
                accessToken = token,
                serializer = StudentProgressResponse.serializer(),
            ) { LmsApi.fetchStudentProgress(course.courseCode, enrollmentId, token) }
            progress = result.first
        }.onFailure {
            errorMessage = L.text(context, localePrefs, R.string.mobile_instructorInsights_error_progress)
        }
    }

    if (showMessage) {
        AlertDialog(
            onDismissRequest = { showMessage = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_message_action)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(value = subject, onValueChange = { subject = it }, label = { Text(L.text(context, localePrefs, R.string.mobile_people_message_subject)) })
                    OutlinedTextField(value = body, onValueChange = { body = it }, label = { Text(L.text(context, localePrefs, R.string.mobile_people_message_body)) }, minLines = 3)
                }
            },
            confirmButton = {
                TextButton(
                    enabled = !sending && subject.isNotBlank() && body.isNotBlank(),
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            sending = true
                            runCatching {
                                LmsApi.sendEnrollmentMessage(
                                    course.courseCode,
                                    enrollmentId,
                                    EnrollmentMessageBody(subject = subject, body = body),
                                    token,
                                )
                                showMessage = false
                            }.onFailure {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_people_message_error)
                            }
                            sending = false
                        }
                    },
                ) { Text(L.text(context, localePrefs, R.string.mobile_people_message_send)) }
            },
            dismissButton = {
                TextButton(onClick = { showMessage = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_people_detail_done))
                }
            },
        )
    }

    Scaffold(
        modifier = modifier,
        topBar = {
            TopAppBar(
                title = { Text(displayName) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            Modifier.padding(padding).padding(16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            errorMessage?.let { LmsErrorBanner(message = it) }
            progress?.let { data ->
                val summary = data.summary
                val reason = if (summary.missingCount > 0) {
                    L.text(context, localePrefs, R.string.mobile_instructorInsights_progress_missingWork)
                } else {
                    L.text(context, localePrefs, R.string.mobile_instructorInsights_progress_checkIn)
                }
                LmsCard(Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_progress_summary), fontWeight = FontWeight.SemiBold)
                        Text("${L.text(context, localePrefs, R.string.mobile_instructorInsights_progress_assignments)}: ${summary.assignmentsSubmittedPct.toInt()}%")
                        Text("${L.text(context, localePrefs, R.string.mobile_instructorInsights_progress_modules)}: ${summary.modulesViewedPct.toInt()}%")
                        summary.avgGradePercent?.let {
                            Text("${L.text(context, localePrefs, R.string.mobile_instructorInsights_progress_avgGrade)}: ${"%.1f".format(it)}%")
                        }
                        if (summary.missingCount > 0) {
                            Text(L.plural(context, localePrefs, R.plurals.mobile_instructorInsights_progress_missingCount, summary.missingCount))
                        }
                    }
                }
                Button(
                    onClick = {
                        subject = localized.getString(R.string.mobile_instructorInsights_message_subject, displayName)
                        body = localized.getString(R.string.mobile_instructorInsights_message_body, displayName, reason)
                        showMessage = true
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_instructorInsights_message_action))
                }
            }
        }
    }
}

@Composable
private fun SnapshotCard(title: String, value: String, error: String?) {
    LmsCard(Modifier.fillMaxWidth()) {
        SnapshotRow(title = title, value = value, error = error)
    }
}

@Composable
private fun SnapshotRow(title: String, value: String, error: String?, showChevron: Boolean = false) {
    Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(title, color = textSecondary(), style = androidx.compose.material3.MaterialTheme.typography.bodySmall)
            Text(value, fontWeight = FontWeight.Bold, color = textPrimary())
            error?.let { Text(it, color = accentColor(), style = androidx.compose.material3.MaterialTheme.typography.labelSmall) }
        }
        if (showChevron) Icon(Icons.Default.ChevronRight, contentDescription = null)
    }
}

@Composable
private fun ActionRow(title: String, subtitle: String, onClick: () -> Unit) {
    LmsCard(Modifier.fillMaxWidth().clickable(onClick = onClick)) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(title, fontWeight = FontWeight.SemiBold)
                Text(subtitle, color = textSecondary(), style = androidx.compose.material3.MaterialTheme.typography.bodySmall)
            }
            Icon(Icons.Default.ChevronRight, contentDescription = null)
        }
    }
}
