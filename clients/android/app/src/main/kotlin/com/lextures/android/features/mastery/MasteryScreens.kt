package com.lextures.android.features.mastery

import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Assessment
import androidx.compose.material.icons.filled.Description
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.FileDownloadManager
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MasteryConceptRow
import com.lextures.android.core.lms.MasteryLevel
import com.lextures.android.core.lms.MasteryLogic
import com.lextures.android.core.lms.ReportCardSummary
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import java.io.File

@Composable
fun CourseMasterySection(
    session: AuthSession,
    course: CourseSummary,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var rows by remember { mutableStateOf<List<MasteryConceptRow>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    val enrollmentId = course.viewerStudentEnrollmentId

    LaunchedEffect(accessToken, course.courseCode, enrollmentId) {
        val token = accessToken
        if (token == null || enrollmentId == null) {
            loading = false
            return@LaunchedEffect
        }
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = MasteryLogic.cacheKeyMastery(course.courseCode, enrollmentId),
                accessToken = token,
                serializer = com.lextures.android.core.lms.StudentMasteryRow.serializer(),
            ) { LmsApi.fetchStudentMastery(course.courseCode, enrollmentId, token) }
            rows = MasteryLogic.rows(result.first)
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

        when {
            loading && rows.isEmpty() -> LmsSkeletonList(count = 3)
            rows.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Assessment,
                title = L.text(R.string.mobile_mastery_emptyTitle),
                message = L.text(R.string.mobile_mastery_emptyMessage),
            )
            else -> {
                val summary = MasteryLogic.summary(rows)
                LmsCard {
                    Text(
                        L.format(R.string.mobile_mastery_summary, summary.mastered, summary.total),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    if (summary.atRisk > 0) {
                        Text(
                            L.format(R.string.mobile_mastery_summaryAtRisk, summary.atRisk),
                            color = LexturesColors.Coral,
                        )
                    }
                }
                for (row in rows) {
                    LmsCard(modifier = Modifier.fillMaxWidth()) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Column {
                                Text(row.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                                Text(
                                    L.text(MasteryLogic.levelLabelRes(row.level)),
                                    color = levelColor(row.level),
                                    fontWeight = FontWeight.SemiBold,
                                )
                                if (!row.assessed) {
                                    Text(L.text(R.string.mobile_mastery_practiceHint), color = textSecondary())
                                }
                            }
                            LevelDot(color = levelColor(row.level))
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun ReportCardListScreen(
    session: AuthSession,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var cards by remember { mutableStateOf<List<ReportCardSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = MasteryLogic.cacheKeyMyReportCards(),
                accessToken = token,
                serializer = kotlinx.serialization.builtins.ListSerializer(ReportCardSummary.serializer()),
            ) { LmsApi.fetchMyReportCards(token) }
            cards = MasteryLogic.releasedReportCards(result.first)
            val cached = result.second
            cacheLabel = if (cached != null && cached.isStale(isOnline)) cached.lastUpdatedLabel() else null
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(it) }

        when {
            loading && cards.isEmpty() -> LmsSkeletonList(count = 3)
            cards.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Description,
                title = L.text(R.string.mobile_mastery_reportCardEmptyTitle),
                message = L.text(R.string.mobile_mastery_reportCardEmptyMessage),
            )
            else -> {
                for (card in cards) {
                    LmsCard(modifier = Modifier.fillMaxWidth()) {
                        Text(
                            L.format(R.string.mobile_mastery_reportCardPeriod, card.gradingPeriod),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        card.letterGrade?.let { Text(it, color = textSecondary()) }
                        TextButton(onClick = {
                            val token = accessToken ?: return@TextButton
                            scope.launch {
                                try {
                                    val bytes = FileDownloadManager.fetchReportCardPdf(card.id, token)
                                    shareReportCardPdf(context, card, bytes)
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                }
                            }
                        }) {
                            Text(L.text(R.string.mobile_mastery_viewPdf))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun LevelDot(color: Color) {
    androidx.compose.foundation.layout.Box(
        modifier = Modifier
            .size(14.dp)
            .background(color, CircleShape),
    )
}

private fun levelColor(level: MasteryLevel): Color = when (level) {
    MasteryLevel.MASTERED -> LexturesColors.BrandTeal
    MasteryLevel.DEVELOPING -> LexturesColors.Amber
    MasteryLevel.BEGINNING -> LexturesColors.Coral
    MasteryLevel.AT_RISK -> LexturesColors.Error
    MasteryLevel.NOT_ASSESSED -> LexturesColors.TextSecondary
}

private fun shareReportCardPdf(
    context: android.content.Context,
    card: ReportCardSummary,
    bytes: ByteArray,
) {
    val safeName = "report-card-${card.gradingPeriod}.pdf".replace(Regex("[^a-zA-Z0-9._-]"), "_")
    val file = File(context.cacheDir, safeName)
    file.writeBytes(bytes)
    val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
    val intent = Intent(Intent.ACTION_VIEW).apply {
        setDataAndType(uri, "application/pdf")
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }
    context.startActivity(Intent.createChooser(intent, null))
}
