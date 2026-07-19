package com.lextures.android.features.courses.create

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CanvasImportApi
import com.lextures.android.core.lms.CanvasImportLogic
import com.lextures.android.core.lms.CanvasImportObservability
import com.lextures.android.core.lms.CanvasCourseListItem
import com.lextures.android.core.lms.CanvasImportIncludeBody
import com.lextures.android.core.lms.CourseImportExportLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateCourseRequest
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PostCourseImportCanvasRequest
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Full-screen Canvas course import wizard (MOB.2) — credentials → select → importing. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CanvasImportScreen(
    session: AuthSession,
    existingCourses: List<CourseSummary>,
    onFinished: (CourseSummary) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var step by remember { mutableStateOf(CanvasImportLogic.ImportStep.Credentials) }
    var canvasBaseUrl by remember { mutableStateOf("") }
    var canvasToken by remember { mutableStateOf("") }
    var courses by remember { mutableStateOf<List<CanvasCourseListItem>>(emptyList()) }
    var selectedCourseId by remember { mutableStateOf<Long?>(null) }
    var include by remember { mutableStateOf(CanvasImportLogic.Include.ALL) }
    var targetMode by remember { mutableStateOf(CanvasImportLogic.TargetMode.NewCourse) }
    var importMode by remember { mutableStateOf(CourseImportExportLogic.ImportMode.erase) }
    var existingCourseCode by remember { mutableStateOf("") }
    var enableGradeSync by remember { mutableStateOf(false) }
    var nameFilter by remember { mutableStateOf("") }
    var hideUnpublished by remember { mutableStateOf(false) }
    var progressLog by remember { mutableStateOf<List<String>>(emptyList()) }
    var busy by remember { mutableStateOf(false) }
    var importComplete by remember { mutableStateOf(false) }
    var cancelled by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var cancelRequested by remember { mutableStateOf(false) }
    var importedCourse by remember { mutableStateOf<CourseSummary?>(null) }

    val filteredCourses = remember(courses, nameFilter, hideUnpublished) {
        var list = CanvasImportLogic.filterCourses(courses, nameFilter)
        if (hideUnpublished) {
            list = list.filter { !CanvasImportLogic.isUnpublished(it.workflowState) }
        }
        list
    }

    fun appendProgress(message: String) {
        val trimmed = message.trim()
        if (trimmed.isNotEmpty()) progressLog = progressLog + trimmed
    }

    fun connect() {
        errorMessage = null
        val key = CanvasImportLogic.validateCredentials(canvasBaseUrl, canvasToken)
        if (key != null) {
            errorMessage = L.text(context, localePrefs, androidName(key))
            return
        }
        val token = accessToken
        if (token.isNullOrBlank()) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_canvasImport_error_session)
            return
        }
        if (!isOnline) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_canvasImport_error_offline)
            return
        }
        busy = true
        scope.launch {
            try {
                // Token stays in Compose state memory only — never written to DataStore/prefs.
                val listed = CanvasImportApi.fetchCanvasCourses(
                    canvasBaseUrl = canvasBaseUrl,
                    accessToken = canvasToken,
                    sessionAccessToken = token,
                )
                courses = listed
                selectedCourseId = listed.firstOrNull()?.id
                CanvasImportObservability.recordListed(context, listed.size)
                step = CanvasImportLogic.ImportStep.Select
            } catch (e: Exception) {
                errorMessage = e.message
                    ?: L.text(context, localePrefs, R.string.mobile_canvasImport_error_listFailed)
            } finally {
                busy = false
            }
        }
    }

    fun startImport() {
        errorMessage = null
        cancelRequested = false
        cancelled = false
        importComplete = false
        progressLog = emptyList()
        val selectedId = selectedCourseId
        val selected = courses.firstOrNull { it.id == selectedId }
        if (selected == null) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_canvasImport_error_selectCourse)
            return
        }
        val sessionToken = accessToken
        if (sessionToken.isNullOrBlank()) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_canvasImport_error_session)
            return
        }
        if (!isOnline) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_canvasImport_error_offline)
            return
        }
        busy = true
        step = CanvasImportLogic.ImportStep.Importing
        CanvasImportObservability.recordStarted(context, include)
        scope.launch {
            try {
                val targetCode = if (targetMode == CanvasImportLogic.TargetMode.NewCourse) {
                    appendProgress(L.text(context, localePrefs, R.string.mobile_canvasImport_progress_creatingShell))
                    val created = LmsApi.createCourse(
                        CreateCourseRequest(
                            title = selected.name,
                            description = "",
                            courseType = "traditional",
                        ),
                        sessionToken,
                    )
                    created.courseCode
                } else {
                    existingCourseCode
                }

                if (cancelRequested) throw CanvasImportLogic.CanvasImportError.Cancelled

                val request = PostCourseImportCanvasRequest(
                    mode = importMode.name,
                    canvasBaseUrl = CanvasImportLogic.normalizeBaseUrl(canvasBaseUrl),
                    canvasCourseId = selected.id.toString(),
                    accessToken = canvasToken.trim(),
                    include = CanvasImportIncludeBody.from(include),
                    canvasGradeSyncEnabled = if (enableGradeSync) true else null,
                )
                CanvasImportApi.postCourseImportCanvas(
                    courseCode = targetCode,
                    body = request,
                    sessionAccessToken = sessionToken,
                    onProgress = { message ->
                        appendProgress(message)
                        CanvasImportObservability.recordProgress(context)
                    },
                    isCancelled = { cancelRequested },
                )

                val summary = LmsApi.fetchCourse(targetCode, sessionToken)
                importedCourse = summary
                importComplete = true
                busy = false
                CanvasImportObservability.recordSucceeded(context, include)
                appendProgress(L.text(context, localePrefs, R.string.mobile_canvasImport_progress_done))
                delay(600)
                onFinished(summary)
            } catch (e: Exception) {
                busy = false
                if (e is CanvasImportLogic.CanvasImportError.Cancelled || cancelRequested) {
                    cancelled = true
                    appendProgress(CanvasImportLogic.CANCELLED_MESSAGE)
                    CanvasImportObservability.recordCancelled(context)
                } else {
                    CanvasImportObservability.recordFailed(context)
                    errorMessage = e.message
                        ?: L.text(context, localePrefs, R.string.mobile_canvasImport_error_importFailed)
                }
            }
        }
    }

    Surface(modifier = modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize()) {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_canvasImport_title)) },
                navigationIcon = {
                    IconButton(
                        onClick = onDismiss,
                        enabled = !(busy && !importComplete),
                    ) {
                        Icon(
                            Icons.Default.Close,
                            contentDescription = L.text(context, localePrefs, R.string.mobile_common_close),
                        )
                    }
                },
            )

            Column(
                modifier = Modifier
                    .weight(1f)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                Text(
                    text = L.format(
                        context,
                        localePrefs,
                        R.string.mobile_canvasImport_stepOf,
                        step.number + 1,
                        3,
                    ),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textSecondary(),
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                    CanvasImportLogic.ImportStep.entries.forEach { s ->
                        val active = s.number <= step.number
                        Column(modifier = Modifier.weight(1f)) {
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(4.dp)
                                    .clip(RoundedCornerShape(2.dp))
                                    .background(
                                        if (active) LexturesColors.Primary else textSecondary().copy(alpha = 0.25f),
                                    ),
                            )
                            Text(
                                text = L.text(
                                    context,
                                    localePrefs,
                                    when (s) {
                                        CanvasImportLogic.ImportStep.Credentials ->
                                            R.string.mobile_canvasImport_step_credentials
                                        CanvasImportLogic.ImportStep.Select ->
                                            R.string.mobile_canvasImport_step_select
                                        CanvasImportLogic.ImportStep.Importing ->
                                            R.string.mobile_canvasImport_step_importing
                                    },
                                ),
                                fontSize = 11.sp,
                                color = if (active) textPrimary() else textSecondary(),
                                maxLines = 1,
                            )
                        }
                    }
                }

                errorMessage?.let { LmsErrorBanner(it) }

                when (step) {
                    CanvasImportLogic.ImportStep.Credentials -> {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_canvasImport_credentials_intro),
                            color = textSecondary(),
                            fontSize = 14.sp,
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_canvasImport_credentials_tokenNotStored),
                            color = textSecondary(),
                            fontSize = 12.sp,
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_canvasImport_field_baseUrl),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        OutlinedTextField(
                            value = canvasBaseUrl,
                            onValueChange = { canvasBaseUrl = it },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                            placeholder = {
                                Text(L.text(context, localePrefs, R.string.mobile_canvasImport_field_baseUrlPlaceholder))
                            },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_canvasImport_field_token),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        OutlinedTextField(
                            value = canvasToken,
                            onValueChange = { canvasToken = it },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                            visualTransformation = PasswordVisualTransformation(),
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                            placeholder = {
                                Text(L.text(context, localePrefs, R.string.mobile_canvasImport_field_tokenPlaceholder))
                            },
                        )
                    }

                    CanvasImportLogic.ImportStep.Select -> {
                        if (courses.isEmpty()) {
                            LmsEmptyState(
                                icon = Icons.Default.Close,
                                title = L.text(context, localePrefs, R.string.mobile_canvasImport_empty_title),
                                message = L.text(context, localePrefs, R.string.mobile_canvasImport_empty_body),
                            )
                        } else {
                            OutlinedTextField(
                                value = nameFilter,
                                onValueChange = { nameFilter = it },
                                modifier = Modifier.fillMaxWidth(),
                                singleLine = true,
                                placeholder = {
                                    Text(L.text(context, localePrefs, R.string.mobile_canvasImport_filter_placeholder))
                                },
                            )
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_canvasImport_filter_hideUnpublished),
                                    color = textPrimary(),
                                )
                                Switch(checked = hideUnpublished, onCheckedChange = { hideUnpublished = it })
                            }
                            filteredCourses.forEach { course ->
                                CoursePickRow(
                                    course = course,
                                    selected = selectedCourseId == course.id,
                                    onClick = { selectedCourseId = course.id },
                                )
                            }
                            Text(
                                L.text(context, localePrefs, R.string.mobile_canvasImport_include_heading),
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            Text(
                                L.text(context, localePrefs, R.string.mobile_canvasImport_include_piiNotice),
                                color = textSecondary(),
                                fontSize = 12.sp,
                            )
                            CanvasImportLogic.IncludeCategory.entries.forEach { category ->
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Text(
                                        L.text(context, localePrefs, androidName(categoryLabelKey(category))),
                                        color = textPrimary(),
                                    )
                                    Switch(
                                        checked = include.value(category),
                                        onCheckedChange = { include = include.set(category, it) },
                                    )
                                }
                            }
                            Text(
                                L.text(context, localePrefs, R.string.mobile_canvasImport_target_heading),
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
                                CanvasImportLogic.TargetMode.entries.forEachIndexed { index, mode ->
                                    SegmentedButton(
                                        selected = targetMode == mode,
                                        onClick = {
                                            targetMode = mode
                                            importMode = CanvasImportLogic.defaultImportMode(mode)
                                        },
                                        shape = SegmentedButtonDefaults.itemShape(
                                            index = index,
                                            count = CanvasImportLogic.TargetMode.entries.size,
                                        ),
                                    ) {
                                        Text(
                                            L.text(
                                                context,
                                                localePrefs,
                                                when (mode) {
                                                    CanvasImportLogic.TargetMode.NewCourse ->
                                                        R.string.mobile_canvasImport_target_new
                                                    CanvasImportLogic.TargetMode.ExistingCourse ->
                                                        R.string.mobile_canvasImport_target_existing
                                                },
                                            ),
                                        )
                                    }
                                }
                            }
                            if (targetMode == CanvasImportLogic.TargetMode.ExistingCourse) {
                                if (existingCourses.isEmpty()) {
                                    Text(
                                        L.text(context, localePrefs, R.string.mobile_canvasImport_target_noExisting),
                                        color = textSecondary(),
                                        fontSize = 12.sp,
                                    )
                                } else {
                                    existingCourses.forEach { course ->
                                        CoursePickRowExisting(
                                            title = course.title,
                                            selected = existingCourseCode == course.courseCode,
                                            onClick = { existingCourseCode = course.courseCode },
                                        )
                                    }
                                    CourseImportExportLogic.ImportMode.entries.forEach { mode ->
                                        Row(
                                            modifier = Modifier
                                                .fillMaxWidth()
                                                .clickable { importMode = mode }
                                                .padding(vertical = 6.dp),
                                            verticalAlignment = Alignment.CenterVertically,
                                        ) {
                                            Icon(
                                                if (importMode == mode) Icons.Default.CheckCircle else Icons.Default.RadioButtonUnchecked,
                                                contentDescription = null,
                                                tint = if (importMode == mode) LexturesColors.Primary else textSecondary(),
                                            )
                                            Spacer(Modifier = Modifier.padding(6.dp))
                                            Text(
                                                L.text(
                                                    context,
                                                    localePrefs,
                                                    when (mode) {
                                                        CourseImportExportLogic.ImportMode.erase ->
                                                            R.string.mobile_courseSettings_importExport_mode_erase_title
                                                        CourseImportExportLogic.ImportMode.mergeAdd ->
                                                            R.string.mobile_courseSettings_importExport_mode_mergeAdd_title
                                                        CourseImportExportLogic.ImportMode.overwrite ->
                                                            R.string.mobile_courseSettings_importExport_mode_overwrite_title
                                                    },
                                                ),
                                                color = textPrimary(),
                                            )
                                        }
                                    }
                                }
                            }
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(
                                        L.text(context, localePrefs, R.string.mobile_canvasImport_gradeSync_title),
                                        color = textPrimary(),
                                    )
                                    Text(
                                        L.text(context, localePrefs, R.string.mobile_canvasImport_gradeSync_summary),
                                        color = textSecondary(),
                                        fontSize = 12.sp,
                                    )
                                }
                                Switch(checked = enableGradeSync, onCheckedChange = { enableGradeSync = it })
                            }
                        }
                    }

                    CanvasImportLogic.ImportStep.Importing -> {
                        when {
                            importComplete -> {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_canvasImport_success_title),
                                    fontWeight = FontWeight.SemiBold,
                                    color = LexturesColors.Primary,
                                )
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_canvasImport_success_body),
                                    color = textSecondary(),
                                )
                            }
                            cancelled -> {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_canvasImport_cancelled_title),
                                    fontWeight = FontWeight.SemiBold,
                                    color = textPrimary(),
                                )
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_canvasImport_cancelled_body),
                                    color = textSecondary(),
                                )
                            }
                            else -> {
                                CircularProgressIndicator()
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_canvasImport_progress_live),
                                    color = textSecondary(),
                                )
                            }
                        }
                        progressLog.forEach { line ->
                            Text(line, color = textPrimary(), fontSize = 12.sp)
                        }
                    }
                }
            }

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(sceneBackground())
                    .padding(16.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                when (step) {
                    CanvasImportLogic.ImportStep.Credentials -> {
                        Button(onClick = { connect() }, enabled = !busy, modifier = Modifier.fillMaxWidth()) {
                            Text(L.text(context, localePrefs, R.string.mobile_canvasImport_action_connect))
                        }
                    }
                    CanvasImportLogic.ImportStep.Select -> {
                        TextButton(
                            onClick = {
                                step = CanvasImportLogic.ImportStep.Credentials
                                errorMessage = null
                            },
                            enabled = !busy,
                        ) {
                            Text(L.text(context, localePrefs, R.string.mobile_canvasImport_action_back))
                        }
                        Spacer(modifier = Modifier.weight(1f))
                        Button(
                            onClick = { startImport() },
                            enabled = !busy &&
                                selectedCourseId != null &&
                                (
                                    targetMode == CanvasImportLogic.TargetMode.NewCourse ||
                                        existingCourseCode.isNotBlank()
                                    ),
                        ) {
                            Text(L.text(context, localePrefs, R.string.mobile_canvasImport_action_import))
                        }
                    }
                    CanvasImportLogic.ImportStep.Importing -> {
                        when {
                            busy && !importComplete -> {
                                TextButton(
                                    onClick = {
                                        cancelRequested = true
                                        cancelled = true
                                        busy = false
                                        CanvasImportObservability.recordCancelled(context)
                                    },
                                ) {
                                    Text(L.text(context, localePrefs, R.string.mobile_canvasImport_action_cancel))
                                }
                            }
                            importComplete -> {
                                Button(
                                    onClick = { importedCourse?.let(onFinished) },
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    Text(L.text(context, localePrefs, R.string.mobile_canvasImport_action_openCourse))
                                }
                            }
                            else -> {
                                TextButton(
                                    onClick = {
                                        step = CanvasImportLogic.ImportStep.Select
                                        cancelled = false
                                        progressLog = emptyList()
                                        errorMessage = null
                                    },
                                ) {
                                    Text(L.text(context, localePrefs, R.string.mobile_canvasImport_action_back))
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun CoursePickRow(
    course: CanvasCourseListItem,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(cardBackground())
            .clickable(onClick = onClick)
            .padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(
            if (selected) Icons.Default.CheckCircle else Icons.Default.RadioButtonUnchecked,
            contentDescription = null,
            tint = if (selected) LexturesColors.Primary else textSecondary(),
        )
        Column {
            Text(course.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
            val subtitle = listOfNotNull(course.courseCode, course.termName, course.workflowState)
                .map { it.trim() }
                .filter { it.isNotEmpty() }
                .joinToString(" · ")
            if (subtitle.isNotEmpty()) {
                Text(subtitle, fontSize = 12.sp, color = textSecondary())
            }
        }
    }
}

@Composable
private fun CoursePickRowExisting(
    title: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(cardBackground())
            .clickable(onClick = onClick)
            .padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            if (selected) Icons.Default.CheckCircle else Icons.Default.RadioButtonUnchecked,
            contentDescription = null,
            tint = if (selected) LexturesColors.Primary else textSecondary(),
        )
        Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
    }
}

private fun categoryLabelKey(category: CanvasImportLogic.IncludeCategory): String =
    "mobile.canvasImport.include.${category.value}"

private fun androidName(key: String): Int {
    // Resolved via resources after locale sync; fall back through string lookup helpers.
    return when (key) {
        "mobile.canvasImport.error.urlRequired" -> R.string.mobile_canvasImport_error_urlRequired
        "mobile.canvasImport.error.urlInvalid" -> R.string.mobile_canvasImport_error_urlInvalid
        "mobile.canvasImport.error.tokenRequired" -> R.string.mobile_canvasImport_error_tokenRequired
        "mobile.canvasImport.include.modules" -> R.string.mobile_canvasImport_include_modules
        "mobile.canvasImport.include.assignments" -> R.string.mobile_canvasImport_include_assignments
        "mobile.canvasImport.include.quizzes" -> R.string.mobile_canvasImport_include_quizzes
        "mobile.canvasImport.include.enrollments" -> R.string.mobile_canvasImport_include_enrollments
        "mobile.canvasImport.include.grades" -> R.string.mobile_canvasImport_include_grades
        "mobile.canvasImport.include.settings" -> R.string.mobile_canvasImport_include_settings
        "mobile.canvasImport.include.files" -> R.string.mobile_canvasImport_include_files
        "mobile.canvasImport.include.announcements" -> R.string.mobile_canvasImport_include_announcements
        else -> R.string.mobile_canvasImport_title
    }
}
