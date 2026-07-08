package com.lextures.android.features.courses.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.UnsavedChangesBanner
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CoursePlagiarismLogic
import com.lextures.android.core.lms.CoursePlagiarismSettings
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val plagiarismJson = Json { ignoreUnknownKeys = true }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CoursePlagiarismSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var baseline by remember(course.courseCode) { mutableStateOf(CoursePlagiarismLogic.draft(null)) }
    var form by remember(course.courseCode) { mutableStateOf(baseline) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }
    var providerExpanded by remember { mutableStateOf(false) }

    val isDirty = CoursePlagiarismLogic.isDirty(form, baseline)
    val selectedProvider = CoursePlagiarismLogic.providerOptions.firstOrNull { it.value == form.provider }
        ?: CoursePlagiarismLogic.providerOptions.first()

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val cached = offline.cachedFetch(
                key = CoursePlagiarismLogic.cacheKey(course.courseCode),
                accessToken = token,
                serializer = CoursePlagiarismSettings.serializer(),
            ) {
                LmsApi.fetchCoursePlagiarismSettings(course.courseCode, token)
            }
            val loaded = CoursePlagiarismLogic.draft(cached.first)
            baseline = loaded
            if (!isDirty) form = loaded
            cacheLabel = cached.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure { loadError = it.message }
        loading = false
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            actionError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            actionSuccess?.let { msg ->
                item {
                    LmsCard {
                        Text(msg, fontWeight = FontWeight.SemiBold)
                    }
                }
            }

            if (loading) {
                item { LmsSkeletonList(count = 3) }
            } else {
                item {
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                            Text(
                                L.text(R.string.mobile_courseSettings_plagiarism_introTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(L.text(R.string.mobile_courseSettings_plagiarism_introDescription))

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text(L.text(R.string.mobile_courseSettings_plagiarism_enableLabel))
                                Switch(
                                    checked = form.checksEnabled,
                                    onCheckedChange = { form = form.copy(checksEnabled = it) },
                                )
                            }

                            Text(
                                L.text(R.string.mobile_courseSettings_plagiarism_providerLabel),
                                fontWeight = FontWeight.Medium,
                            )
                            ExposedDropdownMenuBox(
                                expanded = providerExpanded,
                                onExpandedChange = { providerExpanded = it },
                            ) {
                                OutlinedTextField(
                                    value = L.text(context, localePrefs, selectedProvider.labelRes),
                                    onValueChange = {},
                                    readOnly = true,
                                    label = { Text(L.text(R.string.mobile_courseSettings_plagiarism_providerLabel)) },
                                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(providerExpanded) },
                                    modifier = Modifier
                                        .menuAnchor()
                                        .fillMaxWidth(),
                                )
                                ExposedDropdownMenu(
                                    expanded = providerExpanded,
                                    onDismissRequest = { providerExpanded = false },
                                ) {
                                    CoursePlagiarismLogic.providerOptions.forEach { option ->
                                        DropdownMenuItem(
                                            text = { Text(L.text(context, localePrefs, option.labelRes)) },
                                            onClick = {
                                                form = form.copy(provider = option.value)
                                                providerExpanded = false
                                            },
                                        )
                                    }
                                }
                            }

                            Text(
                                L.text(R.string.mobile_courseSettings_plagiarism_thresholdLabel),
                                fontWeight = FontWeight.Medium,
                            )
                            OutlinedTextField(
                                value = form.thresholdPct,
                                onValueChange = { form = form.copy(thresholdPct = it) },
                                label = { Text(L.text(R.string.mobile_courseSettings_plagiarism_thresholdLabel)) },
                                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Text(L.text(R.string.mobile_courseSettings_plagiarism_thresholdHint))

                            Text(L.text(R.string.mobile_courseSettings_plagiarism_privacyNote))
                            Text(L.text(R.string.mobile_courseSettings_plagiarism_signalNote))
                        }
                    }
                }
            }
        }

        if (isDirty) {
            UnsavedChangesBanner(
                isSaving = saving,
                onDiscard = {
                    form = baseline
                    actionError = null
                    actionSuccess = null
                },
                onSave = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        if (CoursePlagiarismLogic.validateDraft(form) != null) {
                            actionError = L.text(
                                context,
                                localePrefs,
                                R.string.mobile_courseSettings_plagiarism_validation_thresholdInvalid,
                            )
                            return@launch
                        }
                        saving = true
                        actionError = null
                        actionSuccess = null
                        runCatching {
                            val body = CoursePlagiarismLogic.buildPatchBody(form)
                            offline.enqueueMutation(
                                method = "PATCH",
                                path = CoursePlagiarismLogic.patchPath(course.courseCode),
                                bodyJson = plagiarismJson.encodeToString(body),
                                label = L.text(context, localePrefs, R.string.mobile_courseSettings_plagiarism_saveLabel),
                                accessToken = token,
                                idempotencyKey = CoursePlagiarismLogic.saveIdempotencyKey(course.courseCode),
                            )
                            val refreshed = LmsApi.fetchCoursePlagiarismSettings(course.courseCode, token)
                            val loaded = CoursePlagiarismLogic.draft(refreshed)
                            baseline = loaded
                            form = loaded
                            actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_plagiarism_saved)
                        }.onFailure { actionError = it.message }
                        saving = false
                    }
                },
            )
        }
    }
}
