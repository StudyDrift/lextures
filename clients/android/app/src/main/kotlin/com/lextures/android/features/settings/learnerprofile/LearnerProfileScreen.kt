package com.lextures.android.features.settings.learnerprofile

import android.content.Intent
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.DateFormatting
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LearnerProfile
import com.lextures.android.core.lms.LearnerProfileEvidenceRow
import com.lextures.android.core.lms.LearnerProfileFacetSummary
import com.lextures.android.core.lms.LearnerProfileInsight
import com.lextures.android.core.lms.LearnerProfileLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer
import java.io.File

/** Read-only learner profile with LP08 controls (LP10). */
@Composable
fun LearnerProfileScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    var profile by remember { mutableStateOf<LearnerProfile?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var controlError by remember { mutableStateOf<String?>(null) }
    var controlBusy by remember { mutableStateOf(false) }
    var confirmingPause by remember { mutableStateOf(false) }
    var confirmingResume by remember { mutableStateOf(false) }
    var confirmingReset by remember { mutableStateOf(false) }
    var resetPhrase by remember { mutableStateOf("") }

    val controlsDisabled = !isOnline || controlBusy

    suspend fun load(token: String) {
        if (profile == null) loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.learnerProfile(),
                accessToken = token,
                serializer = serializer<LearnerProfile>(),
            ) {
                LmsApi.fetchLearnerProfile(token)
            }
            profile = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (_: Exception) {
            if (profile == null) {
                errorMessage = L.text(context, localePrefs, R.string.mobile_learnerProfile_error_load)
            }
        } finally {
            loading = false
        }
    }

    fun runControl(action: suspend (String) -> Unit) {
        val token = accessToken ?: return
        if (!isOnline) return
        scope.launch {
            controlBusy = true
            controlError = null
            try {
                action(token)
                load(token)
            } catch (_: Exception) {
                controlError = L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_error)
            } finally {
                controlBusy = false
            }
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    Column(modifier = modifier.fillMaxSize()) {
        TextButton(onClick = onBack) {
            Text(L.text(context, localePrefs, R.string.mobile_ia_close))
        }

        when {
            loading && profile == null -> LmsSkeletonList(count = 3, modifier = Modifier.padding(16.dp))
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                cacheLabel?.let { StalenessChip(label = it) }
                if (!isOnline) {
                    LmsCard {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_learnerProfile_offline_banner),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }
                IntroCard(localePrefs)
                errorMessage?.let { ErrorBanner(it) }
                controlError?.let { ErrorBanner(it) }
                profile?.let { current ->
                    if (LearnerProfileLogic.isPaused(current)) {
                        PausedBanner(localePrefs)
                    }
                    if (LearnerProfileLogic.showEmptyState(current)) {
                        EmptyState(localePrefs)
                    } else {
                        LearnerProfileLogic.sortFacets(current.facets).forEach { facet ->
                            FacetCard(
                                facet = facet,
                                session = session,
                                localePrefs = localePrefs,
                                isOnline = isOnline,
                            )
                        }
                    }
                }
                ManageCard(
                    localePrefs = localePrefs,
                    profile = profile,
                    controlsDisabled = controlsDisabled,
                    controlBusy = controlBusy,
                    isOnline = isOnline,
                    onDownload = {
                        val token = accessToken ?: return@ManageCard
                        scope.launch {
                            controlBusy = true
                            controlError = null
                            try {
                                val json = LmsApi.exportLearnerProfile(token)
                                val file = File(context.cacheDir, "learner-profile-export.json")
                                file.writeText(json)
                                val uri = FileProvider.getUriForFile(
                                    context,
                                    "${context.packageName}.fileprovider",
                                    file,
                                )
                                val share = Intent(Intent.ACTION_SEND).apply {
                                    type = "application/json"
                                    putExtra(Intent.EXTRA_STREAM, uri)
                                    addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                                }
                                context.startActivity(Intent.createChooser(share, null))
                            } catch (_: Exception) {
                                controlError = L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_error)
                            } finally {
                                controlBusy = false
                            }
                        }
                    },
                    onPause = { confirmingPause = true },
                    onResume = { confirmingResume = true },
                    onReset = {
                        resetPhrase = ""
                        confirmingReset = true
                    },
                )
            }
        }
    }

    if (confirmingPause) {
        AlertDialog(
            onDismissRequest = { confirmingPause = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_pauseConfirmTitle)) },
            text = { Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_pauseConfirmBody)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmingPause = false
                    runControl { LmsApi.pauseLearnerProfile(it) }
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_pause))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingPause = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (confirmingResume) {
        AlertDialog(
            onDismissRequest = { confirmingResume = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resumeConfirmTitle)) },
            text = { Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resumeConfirmBody)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmingResume = false
                    runControl { LmsApi.resumeLearnerProfile(it) }
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resume))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingResume = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (confirmingReset) {
        val required = L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resetConfirmPhrase)
        AlertDialog(
            onDismissRequest = { confirmingReset = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resetConfirmTitle)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resetConfirmBody))
                    OutlinedTextField(
                        value = resetPhrase,
                        onValueChange = { resetPhrase = it },
                        label = { Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resetConfirmPhraseLabel)) },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        confirmingReset = false
                        runControl { LmsApi.resetLearnerProfile(it) }
                    },
                    enabled = resetPhrase.trim().equals(required, ignoreCase = true),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_reset))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingReset = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }
}

@Composable
private fun IntroCard(localePrefs: LocalePreferences) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_howItWorks_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_description),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_howItWorks_body),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun ErrorBanner(message: String) {
    Text(
        message,
        color = textPrimary(),
        fontSize = 12.sp,
        modifier = Modifier
            .fillMaxWidth()
            .padding(12.dp)
            .semantics { contentDescription = message },
    )
}

@Composable
private fun PausedBanner(localePrefs: LocalePreferences) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_paused_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_paused_body),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun EmptyState(localePrefs: LocalePreferences) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_empty_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_empty_body),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun FacetCard(
    facet: LearnerProfileFacetSummary,
    session: AuthSession,
    localePrefs: LocalePreferences,
    isOnline: Boolean,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    var expanded by remember(facet.facetKey) { mutableStateOf(true) }
    var insights by remember(facet.facetKey) { mutableStateOf<List<LearnerProfileInsight>?>(null) }
    var loadingInsights by remember(facet.facetKey) { mutableStateOf(false) }
    var insightError by remember(facet.facetKey) { mutableStateOf<String?>(null) }

    LaunchedEffect(facet.facetKey, facet.state, accessToken) {
        if (facet.state != "ok") return@LaunchedEffect
        val token = accessToken ?: return@LaunchedEffect
        loadingInsights = true
        insightError = null
        try {
            insights = LmsApi.fetchLearnerProfileFacet(facet.facetKey, token)?.insights.orEmpty()
        } catch (_: Exception) {
            insightError = L.text(context, localePrefs, R.string.mobile_learnerProfile_facet_error)
            insights = emptyList()
        } finally {
            loadingInsights = false
        }
    }

    val chartCaption = when (facet.facetKey) {
        "study_rhythm" -> LearnerProfileLogic.rhythmChartCaption(context, localePrefs, facet.summary)
        "content_modality" -> LearnerProfileLogic.modalityChartCaption(context, localePrefs, facet.summary)
        else -> null
    }

    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded }
                    .padding(vertical = 4.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.Top,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        L.text(context, localePrefs, LearnerProfileLogic.facetTitleRes(facet.facetKey)),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    Text(
                        L.text(context, localePrefs, LearnerProfileLogic.facetDescriptionRes(facet.facetKey)),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
                Text(if (expanded) "▲" else "▼", color = textSecondary())
            }

            if (expanded) {
                when {
                    facet.state == "insufficient_data" -> Text(
                        L.text(context, localePrefs, R.string.mobile_learnerProfile_facet_insufficient),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    insightError != null -> Text(insightError!!, fontSize = 12.sp, color = textPrimary())
                    loadingInsights -> CircularProgressIndicator(modifier = Modifier.padding(8.dp))
                    else -> {
                        chartCaption?.let {
                            Text(
                                it,
                                fontSize = 11.sp,
                                fontFamily = FontFamily.Monospace,
                                color = textSecondary(),
                                modifier = Modifier.semantics { contentDescription = it },
                            )
                        }
                        insights.orEmpty().forEach { insight ->
                            InsightRow(
                                facetKey = facet.facetKey,
                                insight = insight,
                                session = session,
                                localePrefs = localePrefs,
                                isOnline = isOnline,
                            )
                        }
                    }
                }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        L.format(
                            context,
                            localePrefs,
                            R.string.mobile_learnerProfile_facet_lastComputed,
                            DateFormatting.formatAbsoluteShort(facet.updatedAt, localePrefs.effectiveLocale),
                        ),
                        fontSize = 11.sp,
                        color = textSecondary(),
                    )
                    Text(
                        L.text(context, localePrefs, LearnerProfileLogic.confidenceLabelRes(facet.confidence)),
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Medium,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}

@Composable
private fun InsightRow(
    facetKey: String,
    insight: LearnerProfileInsight,
    session: AuthSession,
    localePrefs: LocalePreferences,
    isOnline: Boolean,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    var evidenceExpanded by remember(insight.insightKey) { mutableStateOf(false) }
    var evidence by remember(insight.insightKey) { mutableStateOf<List<LearnerProfileEvidenceRow>?>(null) }
    var loadingEvidence by remember(insight.insightKey) { mutableStateOf(false) }
    var evidenceError by remember(insight.insightKey) { mutableStateOf<String?>(null) }

    val rows = evidence ?: insight.evidence.orEmpty()
    val derived = L.format(
        context,
        localePrefs,
        if (LearnerProfileLogic.uniqueCourseCount(rows) <= 0) {
            R.string.mobile_learnerProfile_evidence_derivedFromNoCourses
        } else {
            R.string.mobile_learnerProfile_evidence_derivedFrom
        },
        L.plural(context, localePrefs, R.plurals.mobile_learnerProfile_evidence_observationCount, LearnerProfileLogic.totalObservationCount(rows)),
        if (LearnerProfileLogic.uniqueCourseCount(rows) > 0) {
            L.plural(context, localePrefs, R.plurals.mobile_learnerProfile_evidence_courseCount, LearnerProfileLogic.uniqueCourseCount(rows))
        } else {
            ""
        },
    )

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text(
            L.text(context, localePrefs, LearnerProfileLogic.insightLabelRes(insight.insightKey)),
            fontWeight = FontWeight.SemiBold,
            fontSize = 12.sp,
            color = textPrimary(),
        )
        Text(
            LearnerProfileLogic.formatInsightValue(context, localePrefs, insight, facetKey),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        Text(derived, fontSize = 11.sp, color = textSecondary())
        Text(
            if (evidenceExpanded) {
                L.text(context, localePrefs, R.string.mobile_learnerProfile_evidence_collapse)
            } else {
                L.text(context, localePrefs, R.string.mobile_learnerProfile_evidence_expand)
            },
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
            modifier = Modifier
                .clickable {
                    evidenceExpanded = !evidenceExpanded
                    if (evidenceExpanded && evidence == null) {
                        val token = accessToken
                        if (token == null) return@clickable
                        if (!isOnline && !insight.evidence.isNullOrEmpty()) {
                            evidence = insight.evidence
                            return@clickable
                        }
                        loadingEvidence = true
                        evidenceError = null
                        scope.launch {
                            try {
                                val map = LmsApi.fetchLearnerProfileFacetEvidence(facetKey, token)
                                evidence = map[insight.insightKey].orEmpty()
                            } catch (_: Exception) {
                                evidence = insight.evidence
                                if (evidence.isNullOrEmpty()) {
                                    evidenceError = L.text(context, localePrefs, R.string.mobile_learnerProfile_evidence_error)
                                }
                            } finally {
                                loadingEvidence = false
                            }
                        }
                    }
                }
                .padding(vertical = 8.dp),
        )
        if (evidenceExpanded) {
            when {
                loadingEvidence -> CircularProgressIndicator(modifier = Modifier.padding(4.dp))
                evidenceError != null -> Text(evidenceError!!, fontSize = 11.sp)
                rows.isEmpty() -> Text(
                    L.text(context, localePrefs, R.string.mobile_learnerProfile_evidence_empty),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
                else -> rows.forEach { row ->
                    Column(modifier = Modifier.padding(vertical = 4.dp)) {
                        Text(row.sourceKind, fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
                        Text(
                            L.plural(
                                context,
                                localePrefs,
                                R.plurals.mobile_learnerProfile_evidence_observationCount,
                                row.observationCount,
                            ),
                            fontSize = 11.sp,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ManageCard(
    localePrefs: LocalePreferences,
    profile: LearnerProfile?,
    controlsDisabled: Boolean,
    controlBusy: Boolean,
    isOnline: Boolean,
    onDownload: () -> Unit,
    onPause: () -> Unit,
    onResume: () -> Unit,
    onReset: () -> Unit,
) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_description),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            Button(onClick = onDownload, enabled = !controlsDisabled, modifier = Modifier.fillMaxWidth()) {
                Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_download))
            }
            if (profile != null && LearnerProfileLogic.isPaused(profile)) {
                Button(onClick = onResume, enabled = !controlsDisabled, modifier = Modifier.fillMaxWidth()) {
                    Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_resume))
                }
            } else {
                Button(onClick = onPause, enabled = !controlsDisabled, modifier = Modifier.fillMaxWidth()) {
                    Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_pause))
                }
            }
            Button(onClick = onReset, enabled = !controlsDisabled, modifier = Modifier.fillMaxWidth()) {
                Text(L.text(context, localePrefs, R.string.mobile_learnerProfile_manage_reset))
            }
            if (controlsDisabled && !isOnline) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_learnerProfile_offline_controlsDisabled),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
            } else if (controlBusy) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
            }
        }
    }
}