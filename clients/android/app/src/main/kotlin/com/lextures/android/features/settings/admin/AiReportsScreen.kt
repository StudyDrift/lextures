package com.lextures.android.features.settings.admin

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
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AiModelsAdminLogic
import com.lextures.android.core.lms.AiReportsPayload
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AiReportsScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val canView = AiModelsAdminLogic.canView(shell.platformFeatures, shell.permissions)

    var preset by remember { mutableStateOf(AiModelsAdminLogic.ReportPreset.HOURS_24) }
    var featureFilter by remember { mutableStateOf("") }
    var userQuery by remember { mutableStateOf("") }
    var courseCode by remember { mutableStateOf("") }
    var report by remember { mutableStateOf<AiReportsPayload?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        val range = AiModelsAdminLogic.utcRange(preset)
        try {
            report = LmsApi.fetchAiReports(
                from = range.first,
                to = range.second,
                feature = featureFilter.ifBlank { null },
                userQuery = userQuery.trim().ifBlank { null },
                courseCode = courseCode.trim().ifBlank { null },
                accessToken = token,
            )
        } catch (e: Exception) {
            report = null
            errorMessage = if (e is ApiError.HttpStatus && !e.message.isNullOrBlank()) {
                e.message!!
            } else {
                L.text(context, localePrefs, R.string.mobile_admin_ai_reports_loadError)
            }
        }
        loading = false
    }

    LaunchedEffect(accessToken, canView, preset) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_reports_title)) },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                        }
                    },
                )
            },
        ) { padding ->
            LmsEmptyState(
                modifier = Modifier.padding(padding).padding(16.dp),
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_ai_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_admin_ai_accessDenied_message),
            )
        }
        return
    }

    val presetLabelRes = mapOf(
        AiModelsAdminLogic.ReportPreset.HOURS_24 to R.string.mobile_admin_ai_reports_preset_24h,
        AiModelsAdminLogic.ReportPreset.DAYS_7 to R.string.mobile_admin_ai_reports_preset_7d,
        AiModelsAdminLogic.ReportPreset.DAYS_30 to R.string.mobile_admin_ai_reports_preset_30d,
        AiModelsAdminLogic.ReportPreset.DAYS_90 to R.string.mobile_admin_ai_reports_preset_90d,
    )

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_reports_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_ai_reports_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )

            Row(
                modifier = Modifier.horizontalScroll(rememberScrollState()),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                AiModelsAdminLogic.ReportPreset.entries.forEach { value ->
                    FilterChip(
                        selected = preset == value,
                        onClick = { preset = value },
                        label = {
                            Text(L.text(context, localePrefs, presetLabelRes.getValue(value)))
                        },
                    )
                }
            }

            OutlinedTextField(
                value = userQuery,
                onValueChange = { userQuery = it },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                label = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_reports_searchUser)) },
            )
            OutlinedTextField(
                value = courseCode,
                onValueChange = { courseCode = it },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                label = { Text(L.text(context, localePrefs, R.string.mobile_admin_ai_reports_searchCourse)) },
            )
            OutlinedButton(
                onClick = {
                    val token = accessToken ?: return@OutlinedButton
                    scope.launch { load(token) }
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_ai_reports_applyFilters))
            }

            errorMessage?.let { LmsErrorBanner(message = it) }

            if (loading && report == null) {
                LmsSkeletonList(count = 4)
            } else {
                report?.let { data ->
                    if (data.range.from.isNotEmpty()) {
                        Text(
                            L.format(
                                context,
                                localePrefs,
                                R.string.mobile_admin_ai_reports_window,
                                data.range.from,
                                data.range.to,
                            ),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }

                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_reports_costTitle),
                        fontWeight = FontWeight.Bold,
                        fontSize = 18.sp,
                    )
                    SummaryCard(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_reports_totalCost),
                        AiModelsAdminLogic.formatUsd(data.cost.summary.totalCostUsd),
                    )
                    SummaryCard(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_reports_totalCalls),
                        AiModelsAdminLogic.formatCount(data.cost.summary.totalCalls),
                    )
                    SummaryCard(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_reports_totalTokens),
                        AiModelsAdminLogic.formatCount(data.cost.summary.totalTokens),
                    )
                    if (data.cost.summary.totalCalls == 0L) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_reports_emptyWindow),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }

                    if (data.cost.byDay.isNotEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_reports_byDay),
                            fontWeight = FontWeight.SemiBold,
                        )
                        data.cost.byDay.forEach { row ->
                            MetricCard(
                                title = row.day,
                                cost = row.costUsd,
                                calls = row.calls,
                                tokens = row.tokens,
                                localePrefs = localePrefs,
                            )
                        }
                    }

                    if (data.cost.byFeature.isNotEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_reports_byFeature),
                            fontWeight = FontWeight.SemiBold,
                        )
                        data.cost.byFeature.forEach { row ->
                            MetricCard(
                                title = AiModelsAdminLogic.featureLabel(row.feature),
                                cost = row.costUsd,
                                calls = row.calls,
                                tokens = row.tokens,
                                localePrefs = localePrefs,
                            )
                        }
                    }

                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_reports_byUser),
                        fontWeight = FontWeight.SemiBold,
                    )
                    if (data.byUser.isEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_reports_noUserUsage),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    } else {
                        data.byUser.forEach { row ->
                            MetricCard(
                                title = row.displayName.ifEmpty { row.email },
                                subtitle = row.email.takeIf { it.isNotEmpty() && it != row.displayName },
                                cost = row.costUsd,
                                calls = row.calls,
                                tokens = row.totalTokens,
                                localePrefs = localePrefs,
                            )
                        }
                    }

                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_ai_reports_byCourse),
                        fontWeight = FontWeight.SemiBold,
                    )
                    if (data.byCourse.isEmpty()) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_ai_reports_noCourseUsage),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    } else {
                        data.byCourse.forEach { row ->
                            MetricCard(
                                title = row.title.ifEmpty { row.courseCode },
                                subtitle = row.courseCode.takeIf { it.isNotEmpty() && it != row.title },
                                cost = row.costUsd,
                                calls = row.calls,
                                tokens = row.totalTokens,
                                localePrefs = localePrefs,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SummaryCard(label: String, value: String) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(label, fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = textSecondary())
            Text(value, fontSize = 18.sp, fontWeight = FontWeight.SemiBold)
        }
    }
}

@Composable
private fun MetricCard(
    title: String,
    cost: Double,
    calls: Long,
    tokens: Long,
    localePrefs: LocalePreferences,
    subtitle: String? = null,
) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, fontWeight = FontWeight.SemiBold)
            subtitle?.let { Text(it, fontSize = 12.sp, color = textSecondary()) }
            Text(
                "${AiModelsAdminLogic.formatUsd(cost)} · ${AiModelsAdminLogic.formatCount(calls)} " +
                    "${L.text(context, localePrefs, R.string.mobile_admin_ai_reports_calls)} · " +
                    "${AiModelsAdminLogic.formatCount(tokens)} " +
                    L.text(context, localePrefs, R.string.mobile_admin_ai_reports_tokens),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}
