package com.lextures.android.features.courses.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseCaptionPolicyPatch
import com.lextures.android.core.lms.CourseConsortiumSettings
import com.lextures.android.core.lms.CourseConsortiumSettingsPatch
import com.lextures.android.core.lms.CourseFeaturesLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.offline.OutboxStatus
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val featuresJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseFeaturesSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    platformFeatures: MobilePlatformFeatures,
    onCourseUpdated: (CourseSummary) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    var serverCourse by remember(course.courseCode) { mutableStateOf(course) }
    var consortiumShareable by remember { mutableStateOf(false) }
    var query by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var consortiumLoading by remember { mutableStateOf(false) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var savingTool by remember { mutableStateOf<CourseFeaturesLogic.Tool?>(null) }
    var savingCaption by remember { mutableStateOf(false) }
    var savingConsortium by remember { mutableStateOf(false) }
    var pendingTools by remember { mutableStateOf(setOf<CourseFeaturesLogic.Tool>()) }
    var pendingDisableTool by remember { mutableStateOf<CourseFeaturesLogic.Tool?>(null) }

    val showCaptions = CourseFeaturesLogic.videoCaptionsSectionEnabled(platformFeatures)
    val showConsortium = CourseFeaturesLogic.consortiumSectionEnabled(platformFeatures)
    val visibleTools = CourseFeaturesLogic.filterTools(CourseFeaturesLogic.allToolRows, query)

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val result = offline.cachedFetch(
                key = CourseFeaturesLogic.cacheKeyFeatures(course.courseCode),
                accessToken = token,
                serializer = CourseSummary.serializer(),
            ) {
                LmsApi.fetchCourse(course.courseCode, token)
            }
            serverCourse = result.first
            onCourseUpdated(result.first)
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure { loadError = it.message }
        loading = false

        if (showConsortium) {
            consortiumLoading = true
            runCatching {
                val consortiumResult = offline.cachedFetch(
                    key = CourseFeaturesLogic.cacheKeyConsortium(course.courseCode),
                    accessToken = token,
                    serializer = CourseConsortiumSettings.serializer(),
                ) {
                    LmsApi.fetchCourseConsortiumSettings(course.courseCode, token)
                        ?: CourseConsortiumSettings(consortiumShareable = false)
                }
                consortiumShareable = consortiumResult.first.consortiumShareable
            }.onFailure { loadError = it.message }
            consortiumLoading = false
        }
    }

    fun persistToggle(tool: CourseFeaturesLogic.Tool, enabled: Boolean) {
        scope.launch {
            val token = session.accessToken.value ?: return@launch
            val previous = serverCourse
            val optimistic = CourseFeaturesLogic.applyToggle(serverCourse, tool, enabled)
            serverCourse = optimistic
            onCourseUpdated(optimistic)
            savingTool = tool
            actionError = null
            actionSuccess = null
            runCatching {
                val item = offline.enqueueMutation(
                    method = "PATCH",
                    path = "/api/v1/courses/${course.courseCode}/features",
                    bodyJson = featuresJson.encodeToString(CourseFeaturesLogic.buildFeaturesPatch(optimistic)),
                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_features_saveLabel),
                    accessToken = token,
                    idempotencyKey = CourseFeaturesLogic.toggleIdempotencyKey(course.courseCode, tool),
                )
                if (item.outboxStatus() != OutboxStatus.Synced) {
                    pendingTools = pendingTools + tool
                } else {
                    pendingTools = pendingTools - tool
                    val refreshed = LmsApi.fetchCourse(course.courseCode, token)
                    serverCourse = refreshed
                    onCourseUpdated(refreshed)
                    actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_features_saved)
                }
            }.onFailure {
                serverCourse = previous
                onCourseUpdated(previous)
                actionError = it.message
            }
            savingTool = null
        }
    }

    pendingDisableTool?.let { tool ->
        AlertDialog(
            onDismissRequest = { pendingDisableTool = null },
            title = { Text(L.text(R.string.mobile_courseSettings_features_disableConfirmTitle)) },
            text = {
                Text(
                    L.format(
                        R.string.mobile_courseSettings_features_disableConfirmMessage,
                        L.text(CourseFeaturesLogic.toolLabelRes(tool)),
                    ),
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    persistToggle(tool, false)
                    pendingDisableTool = null
                }) {
                    Text(L.text(R.string.mobile_courseSettings_features_disableConfirmAction))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingDisableTool = null }) {
                    Text(L.text(R.string.mobile_courseSettings_features_cancel))
                }
            },
        )
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            actionError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }
            actionSuccess?.let { msg ->
                item {
                    LmsCard {
                        Text(msg, fontWeight = FontWeight.SemiBold)
                    }
                }
            }

            if (loading) {
                item { LmsSkeletonList(count = 4) }
            } else {
                item {
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(L.text(R.string.mobile_courseSettings_features_toolsTitle), fontWeight = FontWeight.SemiBold)
                            Text(L.text(R.string.mobile_courseSettings_features_toolsDescription))
                            OutlinedTextField(
                                value = query,
                                onValueChange = { query = it },
                                label = { Text(L.text(R.string.mobile_courseSettings_features_searchPlaceholder)) },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            if (visibleTools.isEmpty()) {
                                Text(L.format(R.string.mobile_courseSettings_features_noToolsMatch, query))
                            } else {
                                visibleTools.forEachIndexed { index, row ->
                                    val tool = row.tool
                                    val enabled = CourseFeaturesLogic.isEnabled(tool, serverCourse)
                                    val isSaving = savingTool == tool
                                    val isPending = pendingTools.contains(tool)
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.Top,
                                    ) {
                                        Column(modifier = Modifier.weight(1f)) {
                                            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                                Text(
                                                    L.text(context, localePrefs, CourseFeaturesLogic.toolLabelRes(tool)),
                                                    fontWeight = FontWeight.SemiBold,
                                                )
                                                if (isPending) {
                                                    Text(
                                                        L.text(R.string.mobile_courseSettings_features_pending),
                                                        fontWeight = FontWeight.SemiBold,
                                                    )
                                                }
                                            }
                                            Text(L.text(context, localePrefs, CourseFeaturesLogic.toolDescriptionRes(tool)))
                                        }
                                        if (!isSaving) {
                                            Switch(
                                                checked = enabled,
                                                onCheckedChange = { newValue ->
                                                    if (!newValue && CourseFeaturesLogic.shouldConfirmDisable(enabled)) {
                                                        pendingDisableTool = tool
                                                    } else {
                                                        persistToggle(tool, newValue)
                                                    }
                                                },
                                            )
                                        }
                                    }
                                    if (index < visibleTools.lastIndex) HorizontalDivider()
                                }
                            }
                        }
                    }
                }

                if (showCaptions) {
                    item {
                        LmsCard {
                            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text(L.text(R.string.mobile_courseSettings_features_captionsTitle), fontWeight = FontWeight.SemiBold)
                                Text(L.text(R.string.mobile_courseSettings_features_captionsDescription))
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Text(L.text(R.string.mobile_courseSettings_features_captionsMandatory))
                                    if (!savingCaption) {
                                        Switch(
                                            checked = serverCourse.requireCaptions == true,
                                            onCheckedChange = { enabled ->
                                                scope.launch {
                                                    val token = session.accessToken.value ?: return@launch
                                                    val previous = serverCourse
                                                    serverCourse = previous.copy(requireCaptions = enabled)
                                                    onCourseUpdated(serverCourse)
                                                    savingCaption = true
                                                    actionError = null
                                                    runCatching {
                                                        offline.enqueueMutation(
                                                            method = "PATCH",
                                                            path = "/api/v1/courses/${course.courseCode}/caption-policy",
                                                            bodyJson = featuresJson.encodeToString(
                                                                CourseCaptionPolicyPatch(requireCaptions = enabled),
                                                            ),
                                                            label = L.text(
                                                                context,
                                                                localePrefs,
                                                                R.string.mobile_courseSettings_features_captionSaveLabel,
                                                            ),
                                                            accessToken = token,
                                                            idempotencyKey = CourseFeaturesLogic.captionPolicyIdempotencyKey(
                                                                course.courseCode,
                                                            ),
                                                        )
                                                        actionSuccess = L.text(
                                                            context,
                                                            localePrefs,
                                                            R.string.mobile_courseSettings_features_captionSaved,
                                                        )
                                                    }.onFailure {
                                                        serverCourse = previous
                                                        onCourseUpdated(previous)
                                                        actionError = it.message
                                                    }
                                                    savingCaption = false
                                                }
                                            },
                                        )
                                    }
                                }
                            }
                        }
                    }
                }

                if (showConsortium) {
                    item {
                        LmsCard {
                            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text(L.text(R.string.mobile_courseSettings_features_consortiumTitle), fontWeight = FontWeight.SemiBold)
                                Text(L.text(R.string.mobile_courseSettings_features_consortiumDescription))
                                if (consortiumLoading) {
                                    Text(L.text(R.string.mobile_courseSettings_features_consortiumLoading))
                                } else {
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Text(L.text(R.string.mobile_courseSettings_features_consortiumAllow))
                                        if (!savingConsortium) {
                                            Switch(
                                                checked = consortiumShareable,
                                                onCheckedChange = { enabled ->
                                                    scope.launch {
                                                        val token = session.accessToken.value ?: return@launch
                                                        val previous = consortiumShareable
                                                        consortiumShareable = enabled
                                                        savingConsortium = true
                                                        actionError = null
                                                        runCatching {
                                                            offline.enqueueMutation(
                                                                method = "PATCH",
                                                                path = "/api/v1/courses/${course.courseCode}/consortium-settings",
                                                                bodyJson = featuresJson.encodeToString(
                                                                    CourseConsortiumSettingsPatch(consortiumShareable = enabled),
                                                                ),
                                                                label = L.text(
                                                                    context,
                                                                    localePrefs,
                                                                    R.string.mobile_courseSettings_features_consortiumSaveLabel,
                                                                ),
                                                                accessToken = token,
                                                                idempotencyKey = CourseFeaturesLogic.consortiumIdempotencyKey(
                                                                    course.courseCode,
                                                                ),
                                                            )
                                                            actionSuccess = L.text(
                                                                context,
                                                                localePrefs,
                                                                R.string.mobile_courseSettings_features_consortiumSaved,
                                                            )
                                                        }.onFailure {
                                                            consortiumShareable = previous
                                                            actionError = it.message
                                                        }
                                                        savingConsortium = false
                                                    }
                                                },
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
