package com.lextures.android.features.courses.settings

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
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
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseImportExportLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.library.libraryLargerScreenHint
import com.lextures.android.features.library.libraryOpenOnWeb
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject

@Composable
fun CourseImportExportScreen(
    session: AuthSession,
    course: CourseSummary,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var importMode by remember { mutableStateOf(CourseImportExportLogic.ImportMode.erase) }
    var busy by remember { mutableStateOf(false) }
    var exporting by remember { mutableStateOf(false) }
    var importing by remember { mutableStateOf(false) }
    var feedbackSuccess by remember { mutableStateOf<String?>(null) }
    var feedbackError by remember { mutableStateOf<String?>(null) }
    var pendingImport by remember { mutableStateOf<JsonObject?>(null) }
    var showConfirm by remember { mutableStateOf(false) }

    val pickFileLauncher = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        scope.launch {
            feedbackError = null
            feedbackSuccess = null
            runCatching {
                val text = withContext(Dispatchers.IO) {
                    context.contentResolver.openInputStream(uri)?.use { stream ->
                        stream.readBytes().toString(Charsets.UTF_8)
                    } ?: error("read-failed")
                }
                pendingImport = CourseImportExportLogic.parseImportFileText(text)
                showConfirm = true
            }.onFailure {
                feedbackError = importExportErrorText(context, localePrefs, it)
            }
        }
    }

    if (showConfirm) {
        AlertDialog(
            onDismissRequest = {
                showConfirm = false
                pendingImport = null
            },
            title = { Text(L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_confirmTitle)) },
            text = {
                Text(L.text(context, localePrefs, importConfirmMessageRes(importMode)))
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        showConfirm = false
                        val bundle = pendingImport ?: return@TextButton
                        pendingImport = null
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            importing = true
                            busy = true
                            feedbackError = null
                            feedbackSuccess = null
                            runCatching {
                                LmsApi.postCourseImport(course.courseCode, importMode, bundle, token)
                            }.onSuccess {
                                feedbackSuccess = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_courseSettings_importExport_importSuccess,
                                )
                            }.onFailure {
                                feedbackError = importExportErrorText(context, localePrefs, it)
                            }
                            importing = false
                            busy = false
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_confirmImport))
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    showConfirm = false
                    pendingImport = null
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_cancel))
                }
            },
        )
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        feedbackError?.let { LmsErrorBanner(message = it) }
        feedbackSuccess?.let { message ->
            LmsCard {
                Text(message, fontWeight = FontWeight.SemiBold)
            }
        }

        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_exportTitle),
                    fontWeight = FontWeight.SemiBold,
                )
                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_exportDescription))
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_exportPrivacyWarning),
                    fontSize = 12.sp,
                )
                Button(
                    onClick = {
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            exporting = true
                            busy = true
                            feedbackError = null
                            feedbackSuccess = null
                            runCatching {
                                val bundle = LmsApi.fetchCourseExport(course.courseCode, token)
                                shareExportFile(
                                    context = context,
                                    courseCode = course.courseCode,
                                    bundle = bundle,
                                )
                            }.onSuccess {
                                feedbackSuccess = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_courseSettings_importExport_exportSuccess,
                                )
                            }.onFailure {
                                feedbackError = importExportErrorText(context, localePrefs, it)
                            }
                            exporting = false
                            busy = false
                        }
                    },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        if (exporting) {
                            L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_exportPreparing)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_exportButton)
                        },
                    )
                }
            }
        }

        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_importTitle),
                    fontWeight = FontWeight.SemiBold,
                )
                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_importDescription))
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_importModeTitle),
                    fontWeight = FontWeight.SemiBold,
                )
                CourseImportExportLogic.ImportMode.entries.forEach { mode ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { importMode = mode }
                            .padding(vertical = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        RadioButton(selected = importMode == mode, onClick = { importMode = mode })
                        Column {
                            Text(
                                L.text(context, localePrefs, importModeTitleRes(mode)),
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(L.text(context, localePrefs, importModeDetailRes(mode)))
                        }
                    }
                }
                OutlinedButton(
                    onClick = { pickFileLauncher.launch(arrayOf("application/json")) },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        if (importing) {
                            L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_importing)
                        } else {
                            L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_chooseFile)
                        },
                    )
                }
            }
        }

        LmsCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_webImportTitle),
                    fontWeight = FontWeight.SemiBold,
                )
                Text(L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_webImportDescription))
                Text(libraryLargerScreenHint())
                OutlinedButton(
                    onClick = {
                        val url = AppConfiguration.webUrl(
                            CourseImportExportLogic.webImportExportPath(course.courseCode),
                        )
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(libraryOpenOnWeb())
                }
            }
        }
    }
}

private val exportShareJson = Json { prettyPrint = true }

private fun shareExportFile(context: android.content.Context, courseCode: String, bundle: JsonObject) {
    val safeName = CourseImportExportLogic.exportFileName(courseCode).replace(Regex("[^a-zA-Z0-9._-]"), "_")
    val file = File(context.cacheDir, safeName)
    file.writeText(exportShareJson.encodeToString(JsonObject.serializer(), bundle))
    val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
    val intent = Intent(Intent.ACTION_SEND).apply {
        type = "application/json"
        putExtra(Intent.EXTRA_STREAM, uri)
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }
    context.startActivity(Intent.createChooser(intent, null))
}

private fun importModeTitleRes(mode: CourseImportExportLogic.ImportMode): Int = when (mode) {
    CourseImportExportLogic.ImportMode.erase ->
        R.string.mobile_courseSettings_importExport_mode_erase_title
    CourseImportExportLogic.ImportMode.mergeAdd ->
        R.string.mobile_courseSettings_importExport_mode_mergeAdd_title
    CourseImportExportLogic.ImportMode.overwrite ->
        R.string.mobile_courseSettings_importExport_mode_overwrite_title
}

private fun importModeDetailRes(mode: CourseImportExportLogic.ImportMode): Int = when (mode) {
    CourseImportExportLogic.ImportMode.erase ->
        R.string.mobile_courseSettings_importExport_mode_erase_detail
    CourseImportExportLogic.ImportMode.mergeAdd ->
        R.string.mobile_courseSettings_importExport_mode_mergeAdd_detail
    CourseImportExportLogic.ImportMode.overwrite ->
        R.string.mobile_courseSettings_importExport_mode_overwrite_detail
}

private fun importConfirmMessageRes(mode: CourseImportExportLogic.ImportMode): Int = when (mode) {
    CourseImportExportLogic.ImportMode.erase ->
        R.string.mobile_courseSettings_importExport_confirmMessage_erase
    CourseImportExportLogic.ImportMode.mergeAdd ->
        R.string.mobile_courseSettings_importExport_confirmMessage_mergeAdd
    CourseImportExportLogic.ImportMode.overwrite ->
        R.string.mobile_courseSettings_importExport_confirmMessage_overwrite
}

private fun importExportErrorText(
    context: android.content.Context,
    localePrefs: com.lextures.android.core.i18n.LocalePreferences,
    error: Throwable,
): String = when (error) {
    CourseImportExportLogic.ImportExportError.InvalidJson ->
        L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_invalidJson)
    CourseImportExportLogic.ImportExportError.InvalidObject ->
        L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_invalidObject)
    CourseImportExportLogic.ImportExportError.FileTooLarge ->
        L.text(context, localePrefs, R.string.mobile_courseSettings_importExport_fileTooLarge)
    else -> CourseImportExportLogic.userFacingError(error)
}
