package com.lextures.android.features.settings.admin

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
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Search
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.ArchivedCourseRow
import com.lextures.android.core.lms.ArchivedCoursesAdminLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ArchivedCoursesAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var courses by remember { mutableStateOf<List<ArchivedCourseRow>>(emptyList()) }
    var searchText by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busyCode by remember { mutableStateOf<String?>(null) }
    var pendingRestore by remember { mutableStateOf<ArchivedCourseRow?>(null) }
    var deleteTarget by remember { mutableStateOf<ArchivedCourseRow?>(null) }
    var deletePhrase by remember { mutableStateOf("") }

    val canView = ArchivedCoursesAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val filteredRows = ArchivedCoursesAdminLogic.filterRows(courses, searchText)

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        runCatching {
            courses = LmsApi.fetchArchivedCourses(token)
        }.onFailure {
            errorMessage = L.text(
                context,
                localePrefs,
                R.string.mobile_admin_archivedCourses_error,
            )
        }
        loading = false
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    if (!canView) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_title)) },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                        }
                    },
                )
            },
            modifier = modifier,
        ) { padding ->
            LmsEmptyState(
                icon = Icons.Filled.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_accessDeniedMessage),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
        }
        return
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
        modifier = modifier,
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_description),
                color = textSecondary(),
            )

            OutlinedTextField(
                value = searchText,
                onValueChange = { searchText = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_search)) },
                singleLine = true,
            )

            errorMessage?.let { LmsErrorBanner(message = it) }
            statusMessage?.let { msg ->
                LmsCard {
                    Text(msg, fontWeight = FontWeight.SemiBold, color = LexturesColors.BrandTeal)
                }
            }

            when {
                loading && courses.isEmpty() -> LmsSkeletonList(count = 3)
                courses.isEmpty() -> LmsEmptyState(
                    icon = Icons.Filled.Archive,
                    title = L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_emptyMessage),
                )
                filteredRows.isEmpty() -> LmsEmptyState(
                    icon = Icons.Filled.Search,
                    title = L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_emptySearch),
                )
                else -> filteredRows.forEach { row ->
                    ArchivedCourseCard(
                        row = row,
                        busy = busyCode == row.courseCode,
                        localePrefs = localePrefs,
                        onRestore = { pendingRestore = row },
                        onDelete = {
                            deletePhrase = ""
                            deleteTarget = row
                        },
                    )
                }
            }
        }
    }

    pendingRestore?.let { row ->
        AlertDialog(
            onDismissRequest = { pendingRestore = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_restoreConfirm)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        pendingRestore = null
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            busyCode = row.courseCode
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.restoreArchivedCourse(row.courseCode, token)
                                courses = ArchivedCoursesAdminLogic.rowsAfterRestore(courses, row.courseCode)
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_archivedCourses_restoreSuccess,
                                )
                            }.onFailure {
                                errorMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_archivedCourses_error,
                                )
                            }
                            busyCode = null
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_restore))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingRestore = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    deleteTarget?.let { row ->
        val required = ArchivedCoursesAdminLogic.deleteConfirmPhrase(row)
        AlertDialog(
            onDismissRequest = { deleteTarget = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_deleteTitle)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_deleteMessage))
                    Text(row.title, fontWeight = FontWeight.SemiBold)
                    Text(row.courseCode, fontFamily = FontFamily.Monospace)
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_deleteWarning),
                        color = LexturesColors.Coral,
                    )
                    OutlinedTextField(
                        value = deletePhrase,
                        onValueChange = { deletePhrase = it },
                        label = {
                            Text(
                                L.format(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_archivedCourses_deleteConfirmPhraseLabel,
                                    required,
                                ),
                            )
                        },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        deleteTarget = null
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            busyCode = row.courseCode
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.deleteArchivedCoursePermanently(row.courseCode, token)
                                courses = ArchivedCoursesAdminLogic.rowsAfterDelete(courses, row.courseCode)
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_archivedCourses_deleteSuccess,
                                )
                            }.onFailure {
                                errorMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_archivedCourses_error,
                                )
                            }
                            busyCode = null
                        }
                    },
                    enabled = ArchivedCoursesAdminLogic.deleteConfirmMatches(deletePhrase, row),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_deletePermanently))
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }
}

@Composable
private fun ArchivedCourseCard(
    row: ArchivedCourseRow,
    busy: Boolean,
    localePrefs: LocalePreferences,
    onRestore: () -> Unit,
    onDelete: () -> Unit,
) {
    val context = LocalContext.current
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(
                row.title.trim().ifEmpty { "—" },
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                "${L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_courseCode)}: ${row.courseCode}",
                fontFamily = FontFamily.Monospace,
                color = textSecondary(),
            )
            Text(
                "${L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_archivedBy)}: " +
                    ArchivedCoursesAdminLogic.archivedByLabel(row),
                color = textSecondary(),
            )
            Text(
                "${L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_archivedAt)}: " +
                    ArchivedCoursesAdminLogic.formatArchivedAt(row.archivedAt),
                color = textSecondary(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onRestore, enabled = !busy) {
                    Text(
                        if (busy) {
                            L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_restoring)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_restore)
                        },
                    )
                }
                TextButton(onClick = onDelete, enabled = !busy) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_archivedCourses_delete),
                        color = LexturesColors.Coral,
                    )
                }
            }
        }
    }
}