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
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseArchivedContentLogic
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val archivedJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseArchivedContentScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    permissions: List<String>,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var structureItems by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var restoringId by remember { mutableStateOf<String?>(null) }
    var pendingRestore by remember { mutableStateOf<CourseArchivedContentLogic.ArchivedContentRow?>(null) }

    val canView = CourseArchivedContentLogic.canViewArchivedContent(course.courseCode, permissions)
    val rows = CourseArchivedContentLogic.archivedRows(structureItems)

    suspend fun reload() {
        val token = session.accessToken.value ?: return
        loading = true
        loadError = null
        runCatching {
            val result = offline.cachedFetch(
                key = CourseArchivedContentLogic.cacheKeyArchivedStructure(course.courseCode),
                accessToken = token,
                serializer = ListSerializer(CourseStructureItem.serializer()),
            ) { LmsApi.fetchCourseArchivedStructure(course.courseCode, token) }
            structureItems = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure { loadError = CourseArchivedContentLogic.userFacingError(it) }
        loading = false
    }

    LaunchedEffect(course.courseCode) {
        reload()
    }

    if (!canView) {
        LmsEmptyState(
            icon = Icons.Filled.Lock,
            title = L.text(R.string.mobile_courseSettings_accessDeniedTitle),
            message = L.text(R.string.mobile_courseSettings_archivedContent_accessDeniedMessage),
        )
        return
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (!isOnline) {
            OfflineBanner()
        }
        cacheLabel?.let { StalenessChip(label = it) }
        loadError?.let { LmsErrorBanner(message = it) }
        actionError?.let { LmsErrorBanner(message = it) }
        actionSuccess?.let { msg ->
            LmsCard {
                Text(msg, fontWeight = FontWeight.SemiBold)
            }
        }

        if (loading) {
            LmsSkeletonList(count = 3)
        } else {
            Text(L.text(R.string.mobile_courseSettings_archivedContent_description))

            if (rows.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Filled.Archive,
                    title = L.text(R.string.mobile_courseSettings_archivedContent_emptyTitle),
                    message = L.text(R.string.mobile_courseSettings_archivedContent_emptyMessage),
                )
            } else {
                rows.forEach { row ->
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(row.title, fontWeight = FontWeight.SemiBold)
                            Text(
                                "${L.text(R.string.mobile_courseSettings_archivedContent_type)}: " +
                                    L.text(context, localePrefs, archivedKindRes(row.kindLabelKey)),
                            )
                            Text(
                                "${L.text(R.string.mobile_courseSettings_archivedContent_module)}: ${row.moduleTitle}",
                            )
                            Text(
                                "${L.text(R.string.mobile_courseSettings_archivedContent_archivedAt)}: " +
                                    CourseArchivedContentLogic.formatArchivedAt(row.archivedAt),
                            )
                            Button(
                                onClick = { pendingRestore = row },
                                enabled = restoringId == null,
                            ) {
                                if (restoringId == row.id) {
                                    CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp))
                                }
                                Text(
                                    if (restoringId == row.id) {
                                        L.text(R.string.mobile_courseSettings_archivedContent_restoring)
                                    } else {
                                        L.text(R.string.mobile_courseSettings_archivedContent_restore)
                                    },
                                )
                            }
                        }
                    }
                }
            }
        }
    }

    pendingRestore?.let { row ->
        AlertDialog(
            onDismissRequest = { pendingRestore = null },
            title = { Text(L.text(R.string.mobile_courseSettings_archivedContent_restoreConfirmTitle)) },
            text = {
                Text(
                    L.format(
                        R.string.mobile_courseSettings_archivedContent_restoreConfirmMessage,
                        row.title,
                    ),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        pendingRestore = null
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            restoringId = row.id
                            actionError = null
                            actionSuccess = null
                            runCatching {
                                offline.enqueueMutation(
                                    method = "PATCH",
                                    path = "/api/v1/courses/${course.courseCode}/structure/items/${row.id}",
                                    bodyJson = archivedJson.encodeToString(
                                        mapOf("archived" to false),
                                    ),
                                    label = L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_courseSettings_archivedContent_restoreLabel,
                                    ),
                                    accessToken = token,
                                    idempotencyKey = "course-archived-restore:${course.courseCode}:${row.id}",
                                )
                                structureItems = CourseArchivedContentLogic.itemsAfterRestore(
                                    structureItems,
                                    row.id,
                                )
                                actionSuccess = L.text(
                                    context,
                                    localePrefs,
                                    if (isOnline) {
                                        R.string.mobile_courseSettings_archivedContent_restoreSuccess
                                    } else {
                                        R.string.mobile_courseSettings_archivedContent_restoreQueued
                                    },
                                )
                            }.onFailure {
                                actionError = CourseArchivedContentLogic.userFacingError(it)
                            }
                            restoringId = null
                        }
                    },
                ) {
                    Text(L.text(R.string.mobile_courseSettings_archivedContent_restore))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingRestore = null }) {
                    Text(L.text(R.string.mobile_courseSettings_archivedContent_cancel))
                }
            },
        )
    }
}

private fun archivedKindRes(kindLabelKey: String): Int = when (kindLabelKey) {
    "mobile.courseSettings.archivedContent.kind.heading" ->
        R.string.mobile_courseSettings_archivedContent_kind_heading
    "mobile.courseSettings.archivedContent.kind.contentPage" ->
        R.string.mobile_courseSettings_archivedContent_kind_contentPage
    "mobile.courseSettings.archivedContent.kind.assignment" ->
        R.string.mobile_courseSettings_archivedContent_kind_assignment
    "mobile.courseSettings.archivedContent.kind.quiz" ->
        R.string.mobile_courseSettings_archivedContent_kind_quiz
    "mobile.courseSettings.archivedContent.kind.externalLink" ->
        R.string.mobile_courseSettings_archivedContent_kind_externalLink
    else -> R.string.mobile_courseSettings_archivedContent_kind_other
}
