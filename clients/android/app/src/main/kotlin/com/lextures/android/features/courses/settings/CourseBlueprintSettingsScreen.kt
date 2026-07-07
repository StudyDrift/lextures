package com.lextures.android.features.courses.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
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
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.UnsavedChangesBanner
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.BlueprintChildRow
import com.lextures.android.core.lms.BlueprintPushResult
import com.lextures.android.core.lms.BlueprintSyncLogRow
import com.lextures.android.core.lms.CourseBlueprintLogic
import com.lextures.android.core.lms.CourseSettingsLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@Composable
fun CourseBlueprintSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    permissions: List<String>,
    onCourseUpdated: (CourseSummary) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var serverCourse by remember(course.courseCode) { mutableStateOf(course) }
    var isBlueprintDraft by remember(course.courseCode) { mutableStateOf(course.isBlueprint == true) }
    var children by remember { mutableStateOf<List<BlueprintChildRow>>(emptyList()) }
    var syncLogs by remember { mutableStateOf<List<BlueprintSyncLogRow>>(emptyList()) }
    var childCodeDraft by remember { mutableStateOf("") }
    var pushResult by remember { mutableStateOf<BlueprintPushResult?>(null) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var savingDesignation by remember { mutableStateOf(false) }
    var showPushConfirm by remember { mutableStateOf(false) }
    var pendingUnlinkCode by remember { mutableStateOf<String?>(null) }

    val canOrgBlueprint = CourseBlueprintLogic.canManageBlueprint(serverCourse, permissions)
    val isDesignationDirty = isBlueprintDraft != (serverCourse.isBlueprint == true)
    val pushDisabledReason = CourseBlueprintLogic.pushDisabledReason(isOnline, children.size)
    val mutationsDisabledReason = CourseBlueprintLogic.mutationsDisabledReason(isOnline)

    suspend fun reload() {
        val token = session.accessToken.value ?: return
        loading = true
        loadError = null
        runCatching {
            val courseResult = offline.cachedFetch(
                key = CourseSettingsLogic.cacheKeySettings(course.courseCode),
                accessToken = token,
                serializer = CourseSummary.serializer(),
            ) { LmsApi.fetchCourse(course.courseCode, token) }
            serverCourse = courseResult.first
            isBlueprintDraft = serverCourse.isBlueprint == true
            onCourseUpdated(serverCourse)
            cacheLabel = courseResult.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()

            if (!CourseBlueprintLogic.shouldLoadBlueprintDetails(serverCourse, canOrgBlueprint)) {
                children = emptyList()
                syncLogs = emptyList()
                return@runCatching
            }

            val payloadResult = offline.cachedFetch(
                key = CourseBlueprintLogic.cacheKeyBlueprintData(course.courseCode),
                accessToken = token,
                serializer = com.lextures.android.core.lms.BlueprintCachedPayload.serializer(),
            ) { LmsApi.fetchBlueprintPayload(course.courseCode, token) }
            children = payloadResult.first.children
            syncLogs = payloadResult.first.syncLogs
            if (payloadResult.second?.isStale(isOnline) == true && cacheLabel == null) {
                cacheLabel = payloadResult.second?.lastUpdatedLabel()
            }
        }.onFailure { loadError = blueprintErrorText(context, localePrefs, it) }
        loading = false
    }

    LaunchedEffect(course.courseCode, permissions) {
        reload()
    }

    if (!canOrgBlueprint && !loading) {
        LmsEmptyState(
            icon = Icons.Filled.Lock,
            title = L.text(context, localePrefs, R.string.mobile_courseSettings_section_blueprint),
            message = L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_accessDeniedMessage),
        )
        return
    }

    Column(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (loading) {
                CircularProgressIndicator()
            } else {
                if (!isOnline) OfflineBanner()
                cacheLabel?.let { StalenessChip(label = it) }
                loadError?.let { LmsErrorBanner(message = it) }
                actionError?.let { LmsErrorBanner(message = it) }
                actionSuccess?.let { msg ->
                    LmsCard {
                        Text(msg, fontWeight = FontWeight.SemiBold)
                    }
                }

                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_courseSettings_section_blueprint),
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_description))

                        when (CourseBlueprintLogic.blueprintRole(serverCourse).role) {
                            CourseBlueprintLogic.BlueprintRole.Child -> {
                                val parent = CourseBlueprintLogic.blueprintRole(serverCourse).parentCode.orEmpty()
                                Text(
                                    L.format(
                                        R.string.mobile_courseSettings_blueprint_childLinkedBanner,
                                        parent,
                                    ),
                                    fontWeight = FontWeight.SemiBold,
                                )
                                Text(
                                    L.format(
                                        R.string.mobile_courseSettings_blueprint_lastSync,
                                        CourseBlueprintLogic.formatSyncAt(serverCourse.blueprintLastSyncAt),
                                    ),
                                )
                            }
                            CourseBlueprintLogic.BlueprintRole.Master -> {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_statusBlueprintMaster),
                                    fontWeight = FontWeight.SemiBold,
                                )
                            }
                            CourseBlueprintLogic.BlueprintRole.None -> {
                                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_notBlueprintInfo))
                            }
                        }

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_enableDesignation),
                                    fontWeight = FontWeight.SemiBold,
                                )
                                Text(
                                    if (!serverCourse.blueprintParentCourseCode.isNullOrBlank()) {
                                        L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_enableDesignationDisabledHint)
                                    } else {
                                        L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_enableDesignationHint)
                                    },
                                )
                            }
                            Switch(
                                checked = isBlueprintDraft,
                                onCheckedChange = { isBlueprintDraft = it },
                                enabled = !busy && !savingDesignation && serverCourse.blueprintParentCourseCode.isNullOrBlank(),
                            )
                        }
                    }
                }

                if (serverCourse.isBlueprint == true) {
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_linkedChildrenTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_linkedChildrenDescription))
                            mutationsDisabledReason?.let {
                                Text(blueprintDisabledText(context, localePrefs, it))
                            }
                            OutlinedTextField(
                                value = childCodeDraft,
                                onValueChange = { childCodeDraft = it },
                                label = { Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_childCourseCode)) },
                                placeholder = { Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_childCourseCodePlaceholder)) },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Button(
                                onClick = {
                                    scope.launch {
                                        val code = childCodeDraft.trim()
                                        val token = session.accessToken.value ?: return@launch
                                        busy = true
                                        actionError = null
                                        actionSuccess = null
                                        runCatching {
                                            LmsApi.postBlueprintChildLink(course.courseCode, code, token)
                                            childCodeDraft = ""
                                            actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_linkSuccess)
                                            reload()
                                        }.onFailure { actionError = blueprintErrorText(context, localePrefs, it) }
                                        busy = false
                                    }
                                },
                                enabled = !busy && mutationsDisabledReason == null && childCodeDraft.isNotBlank(),
                            ) {
                                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_linkAndSync))
                            }

                            if (children.isEmpty()) {
                                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_noChildren))
                            } else {
                                children.forEach { child ->
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                    ) {
                                        Column {
                                            Text(child.courseCode, fontFamily = FontFamily.Monospace, fontWeight = FontWeight.SemiBold)
                                            Text(child.title)
                                            Text(
                                                L.format(
                                                    R.string.mobile_courseSettings_blueprint_lastSync,
                                                    CourseBlueprintLogic.formatSyncAt(child.lastSyncAt),
                                                ),
                                            )
                                        }
                                        TextButton(
                                            onClick = { pendingUnlinkCode = child.courseCode },
                                            enabled = !busy && mutationsDisabledReason == null,
                                        ) {
                                            Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_unlink))
                                        }
                                    }
                                    HorizontalDivider()
                                }
                            }
                        }
                    }

                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushDescription))
                            pushDisabledReason?.let {
                                Text(blueprintDisabledText(context, localePrefs, it))
                            }
                            Button(
                                onClick = { showPushConfirm = true },
                                enabled = !busy && pushDisabledReason == null,
                                colors = ButtonDefaults.buttonColors(containerColor = androidx.compose.ui.graphics.Color(0xFF059669)),
                            ) {
                                Text(
                                    if (busy) {
                                        L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushWorking)
                                    } else {
                                        L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushButton)
                                    },
                                )
                            }
                            pushResult?.let { result ->
                                Text(
                                    L.format(
                                        R.string.mobile_courseSettings_blueprint_pushResult,
                                        result.childrenSuccess.toString(),
                                        result.childrenTotal.toString(),
                                        result.childrenError.toString(),
                                    ),
                                    fontWeight = FontWeight.SemiBold,
                                )
                                result.detail.forEach { row ->
                                    Text(
                                        "${row.courseCode ?: "—"}: ${if (row.ok == true) "ok" else row.error ?: "error"}",
                                        fontFamily = FontFamily.Monospace,
                                    )
                                }
                            }
                        }
                    }

                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_syncHistoryTitle),
                                fontWeight = FontWeight.SemiBold,
                            )
                            if (syncLogs.isEmpty()) {
                                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_noSyncHistory))
                            } else {
                                syncLogs.forEach { log ->
                                    Text(CourseBlueprintLogic.formatSyncAt(log.triggeredAt), fontWeight = FontWeight.SemiBold)
                                    Text(
                                        L.format(
                                            R.string.mobile_courseSettings_blueprint_syncHistoryRow,
                                            log.childrenSuccess.toString(),
                                            log.childrenTotal.toString(),
                                            log.childrenError.toString(),
                                        ),
                                    )
                                    HorizontalDivider()
                                }
                            }
                        }
                    }
                }
            }
        }

        if (isDesignationDirty) {
            UnsavedChangesBanner(
                isSaving = savingDesignation,
                onDiscard = {
                    isBlueprintDraft = serverCourse.isBlueprint == true
                    actionError = null
                },
                onSave = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        savingDesignation = true
                        actionError = null
                        runCatching {
                            val updated = LmsApi.patchCourseBlueprint(course.courseCode, isBlueprintDraft, token)
                            serverCourse = updated
                            onCourseUpdated(updated)
                            actionSuccess = L.text(
                                context,
                                localePrefs,
                                if (isBlueprintDraft) {
                                    R.string.mobile_courseSettings_blueprint_designationEnabled
                                } else {
                                    R.string.mobile_courseSettings_blueprint_designationDisabled
                                },
                            )
                            reload()
                        }.onFailure { actionError = blueprintErrorText(context, localePrefs, it) }
                        savingDesignation = false
                    }
                },
            )
        }
    }

    if (showPushConfirm) {
        AlertDialog(
            onDismissRequest = { showPushConfirm = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushConfirmTitle)) },
            text = { Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushConfirmMessage)) },
            confirmButton = {
                TextButton(onClick = {
                    showPushConfirm = false
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        busy = true
                        actionError = null
                        actionSuccess = null
                        pushResult = null
                        runCatching {
                            pushResult = LmsApi.postBlueprintPush(course.courseCode, token)
                            actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushSuccess)
                            reload()
                        }.onFailure { actionError = blueprintErrorText(context, localePrefs, it) }
                        busy = false
                    }
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_pushButton))
                }
            },
            dismissButton = {
                TextButton(onClick = { showPushConfirm = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_cancel))
                }
            },
        )
    }

    pendingUnlinkCode?.let { childCode ->
        AlertDialog(
            onDismissRequest = { pendingUnlinkCode = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_unlinkConfirmTitle)) },
            text = {
                Text(
                    L.format(
                        R.string.mobile_courseSettings_blueprint_unlinkConfirmMessage,
                        childCode,
                    ),
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    pendingUnlinkCode = null
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        busy = true
                        actionError = null
                        actionSuccess = null
                        runCatching {
                            LmsApi.deleteBlueprintChildLink(course.courseCode, childCode, token)
                            actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_unlinkSuccess)
                            reload()
                        }.onFailure { actionError = blueprintErrorText(context, localePrefs, it) }
                        busy = false
                    }
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_unlink))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingUnlinkCode = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_cancel))
                }
            },
        )
    }
}

private fun blueprintDisabledText(context: android.content.Context, localePrefs: com.lextures.android.core.i18n.LocalePreferences, reason: String): String =
    when (reason) {
        "offline-push" -> L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_offlinePushDisabled)
        "no-children" -> L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_noChildrenPushDisabled)
        "offline-mutations" -> L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_offlineMutationsDisabled)
        else -> reason
    }

private fun blueprintErrorText(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    error: Throwable,
): String = error.message ?: L.text(context, localePrefs, R.string.mobile_courseSettings_blueprint_genericError)
