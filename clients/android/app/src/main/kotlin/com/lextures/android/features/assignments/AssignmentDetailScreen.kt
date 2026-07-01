package com.lextures.android.features.assignments

import android.Manifest
import android.content.pm.PackageManager
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Photo
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.ContextCompat
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.AuthPrimaryButton
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.rememberLocalePreferences
import com.lextures.android.core.lms.AssignmentLogic
import com.lextures.android.core.lms.AssignmentSubmission
import com.lextures.android.core.lms.AssignmentSubmissionStatus
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.GradeFeedbackRoute
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.ModuleItemDetail
import com.lextures.android.core.lms.SubmissionGrade
import com.lextures.android.core.lms.SubmitAssignmentTextRequest
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.courses.detailRows
import com.lextures.android.features.courses.ItemKind
import com.lextures.android.features.courses.MarkdownText
import com.lextures.android.features.courses.RowHeader
import com.lextures.android.features.files.FilePreviewScreen
import com.lextures.android.features.grades.GradeFeedbackScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.io.File

/** Student assignment detail: instructions, compose, submit, status, feedback link (M5.1). */
@Composable
fun AssignmentDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val draftStore = remember { AssignmentDraftStore(context) }
    val scope = rememberCoroutineScope()
    val localePrefs = rememberLocalePreferences()
    val fileTooLargeMessage = L.text(context, localePrefs, R.string.mobile_assignment_fileTooLarge)
    val fileTypeNotAllowedMessage = L.text(context, localePrefs, R.string.mobile_assignment_fileTypeNotAllowed)
    val saveAnswerLabel = L.text(context, localePrefs, R.string.mobile_assignment_saveAnswer)
    val queuedOfflineMessage = L.text(context, localePrefs, R.string.mobile_assignment_queuedOffline)
    val offlineJson = remember { Json { ignoreUnknownKeys = true } }
    val courseCode = course.courseCode
    val resubmissionEnabled = course.resubmissionWorkflowEnabled == true
    val draftKey = AssignmentLogic.draftStorageKey(courseCode, item.id)

    var detail by remember { mutableStateOf<ModuleItemDetail?>(null) }
    var mySubmission by remember { mutableStateOf<AssignmentSubmission?>(null) }
    var myGrade by remember { mutableStateOf<SubmissionGrade?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var draftText by remember { mutableStateOf("") }
    var urlText by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    var submitSuccess by remember { mutableStateOf<String?>(null) }
    var pendingAttachment by remember { mutableStateOf<PendingAttachment?>(null) }
    var uploadProgress by remember { mutableDoubleStateOf(-1.0) }
    var uploadError by remember { mutableStateOf<String?>(null) }
    var openFeedback by remember { mutableStateOf<GradeFeedbackRoute?>(null) }
    var openPreview by remember { mutableStateOf<FilePreviewTarget?>(null) }
    var cameraUri by remember { mutableStateOf<Uri?>(null) }

    BackHandler(onBack = onBack)

    LaunchedEffect(draftKey) {
        draftText = draftStore.load(draftKey)
    }

    LaunchedEffect(draftText) {
        draftStore.save(draftKey, draftText)
    }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            detail = LmsApi.fetchItemDetail(courseCode, item, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
        mySubmission = runCatching { LmsApi.fetchMySubmission(courseCode, item.id, token) }.getOrNull()
        myGrade = mySubmission?.let { submission ->
            runCatching { LmsApi.fetchSubmissionGrade(courseCode, item.id, submission.id, token) }.getOrNull()
        }
    }

    LaunchedEffect(accessToken, item.id) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    openFeedback?.let { route ->
        GradeFeedbackScreen(
            session = session,
            course = course,
            column = route.column,
            onBack = { openFeedback = null },
            modifier = modifier,
        )
        return
    }

    openPreview?.let { target ->
        FilePreviewScreen(
            session = session,
            target = target,
            onBack = { openPreview = null },
            modifier = modifier,
        )
        return
    }

    val status = AssignmentLogic.status(mySubmission, myGrade, detail)
    val composedText = remember(draftText, urlText, detail) {
        val body = draftText.trim()
        val url = urlText.trim()
        if (detail?.submissionAllowUrl == true && url.isNotEmpty()) {
            if (body.isEmpty()) url else "$body\n\n$url"
        } else {
            body
        }
    }

    val pickFileLauncher = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                val bytes = stream.readBytes()
                if (!AssignmentLogic.isAllowedFileSize(bytes.size.toLong())) {
                    errorMessage = fileTooLargeMessage
                    return@runCatching
                }
                val mime = context.contentResolver.getType(uri) ?: "application/octet-stream"
                if (!AssignmentLogic.isAllowedMimeType(mime)) {
                    errorMessage = fileTypeNotAllowedMessage
                    return@runCatching
                }
                val name = uri.lastPathSegment?.substringAfterLast('/') ?: "upload"
                pendingAttachment = PendingAttachment(bytes, name, mime)
            }
        }
    }

    val takePictureLauncher = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { ok ->
        if (!ok) return@rememberLauncherForActivityResult
        val uri = cameraUri ?: return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                val bytes = stream.readBytes()
                if (!AssignmentLogic.isAllowedFileSize(bytes.size.toLong())) {
                    errorMessage = fileTooLargeMessage
                    return@runCatching
                }
                pendingAttachment = PendingAttachment(bytes, "photo.jpg", "image/jpeg")
            }
        }
    }

    val cameraPermissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) {
            val file = File(context.cacheDir, "assignment-camera-${System.currentTimeMillis()}.jpg")
            val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
            cameraUri = uri
            takePictureLauncher.launch(uri)
        }
    }

    fun uploadAttachment(token: String, attachment: PendingAttachment, attempt: Int = 1) {
        scope.launch {
            uploadError = null
            uploadProgress = 0.1
            try {
                val submission = LmsApi.uploadAssignmentFile(
                    courseCode = courseCode,
                    itemId = item.id,
                    fileBytes = attachment.bytes,
                    fileName = attachment.fileName,
                    mimeType = attachment.mimeType,
                    accessToken = token,
                )
                uploadProgress = 1.0
                pendingAttachment = null
                mySubmission = submission
                submitSuccess = localePrefs.localizedContext(context).getString(
                    R.string.mobile_assignment_submitSuccess,
                    LmsDates.relative(submission.submittedAt),
                    submission.versionNumber ?: 1,
                )
                load(token)
            } catch (e: Exception) {
                if (attempt < 3) {
                    delay(attempt * 800L)
                    uploadAttachment(token, attachment, attempt + 1)
                } else {
                    uploadProgress = -1.0
                    uploadError = session.mapError(e)
                }
            }
        }
    }

    fun submitNow() {
        val token = accessToken ?: return
        scope.launch {
            submitting = true
            submitSuccess = null
            errorMessage = null
            try {
                pendingAttachment?.let { attachment ->
                    uploadAttachment(token, attachment)
                    submitting = false
                    return@launch
                }
                if (composedText.isBlank()) {
                    submitting = false
                    return@launch
                }
                if (!isOnline) {
                    offline.enqueueMutation(
                        method = "POST",
                        path = "/api/v1/courses/$courseCode/assignments/${item.id}/submissions/text",
                        bodyJson = offlineJson.encodeToString(
                            SubmitAssignmentTextRequest(composedText),
                        ),
                        label = saveAnswerLabel,
                        accessToken = token,
                        preferQueue = true,
                    )
                    submitSuccess = queuedOfflineMessage
                    submitting = false
                    return@launch
                }
                val submission = LmsApi.submitAssignmentText(courseCode, item.id, composedText, token)
                draftStore.clear(draftKey)
                draftText = ""
                urlText = ""
                mySubmission = submission
                submitSuccess = localePrefs.localizedContext(context).getString(
                    R.string.mobile_assignment_submitSuccess,
                    LmsDates.relative(submission.submittedAt),
                    submission.versionNumber ?: 1,
                )
                load(token)
            } catch (e: Exception) {
                errorMessage = session.mapError(e)
            } finally {
                submitting = false
            }
        }
    }

    Column(modifier = modifier) {
        RowHeader(title = detail?.title ?: item.title, onBack = onBack)

        if (loading && detail == null) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            return
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        AssignmentChip(ItemKind.label(item.kind), accentColor())
                        if (LmsDates.parse(detail?.dueAt ?: item.dueAt) != null) {
                            AssignmentChip(
                                L.format(R.string.mobile_assignment_due, LmsDates.shortDateTime(detail?.dueAt ?: item.dueAt)),
                                LexturesColors.Coral,
                            )
                        }
                    }
                    Text(
                        text = detail?.title ?: item.title,
                        style = LexturesType.display(24),
                        color = textPrimary(),
                    )
                }
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            submitSuccess?.let { message ->
                item {
                    LmsCard {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                            Icon(Icons.Default.CheckCircle, contentDescription = null, tint = LexturesColors.BrandTeal)
                            Text(message, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        }
                    }
                }
            }

            detail?.markdown?.takeIf { it.isNotBlank() }?.let { markdown ->
                item {
                    LmsCard { MarkdownText(markdown) }
                }
            }

            item {
                StatusCard(
                    status = status,
                    submission = mySubmission,
                    grade = myGrade,
                    detail = detail,
                    onPreviewAttachment = { submission ->
                        val path = submission.attachmentContentPath ?: return@StatusCard
                        val name = submission.attachmentFilename ?: return@StatusCard
                        openPreview = FilePreviewTarget.submissionContentPath(
                            courseCode = courseCode,
                            contentPath = path,
                            fileName = name,
                            mimeType = submission.attachmentMimeType,
                        )
                    },
                )
            }

            if (AssignmentLogic.canSubmit(detail, mySubmission, resubmissionEnabled) || pendingAttachment != null) {
                item {
                    ComposerCard(
                        detail = detail,
                        submission = mySubmission,
                        resubmissionEnabled = resubmissionEnabled,
                        draftText = draftText,
                        onDraftChange = { draftText = it },
                        urlText = urlText,
                        onUrlChange = { urlText = it },
                        pendingAttachment = pendingAttachment,
                        uploadProgress = uploadProgress,
                        uploadError = uploadError,
                        isOnline = isOnline,
                        submitting = submitting,
                        canSubmit = (pendingAttachment != null || composedText.isNotBlank()) &&
                            AssignmentLogic.canSubmit(detail, mySubmission, resubmissionEnabled),
                        onPickPhoto = { pickFileLauncher.launch(arrayOf("image/*", "video/*")) },
                        onPickFile = { pickFileLauncher.launch(arrayOf("application/pdf", "image/*", "text/*")) },
                        onCamera = {
                            when {
                                ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) ==
                                    PackageManager.PERMISSION_GRANTED -> {
                                    val file = File(context.cacheDir, "assignment-camera-${System.currentTimeMillis()}.jpg")
                                    val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
                                    cameraUri = uri
                                    takePictureLauncher.launch(uri)
                                }
                                else -> cameraPermissionLauncher.launch(Manifest.permission.CAMERA)
                            }
                        },
                        onRetryUpload = {
                            pendingAttachment?.let { attachment ->
                                accessToken?.let { uploadAttachment(it, attachment) }
                            }
                        },
                        onSubmit = { submitNow() },
                    )
                }
            }

            if (status == AssignmentSubmissionStatus.Graded || myGrade?.posted == true) {
                item {
                    LmsCard(onClick = {
                        openFeedback = GradeFeedbackRoute(AssignmentLogic.gradeColumn(item, detail))
                    }) {
                        Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(L.text(R.string.mobile_assignment_viewFeedback), fontWeight = FontWeight.SemiBold)
                                Text(
                                    L.text(R.string.mobile_assignment_viewFeedbackHint),
                                    fontSize = 12.sp,
                                    color = textSecondary(),
                                )
                            }
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null, tint = textSecondary())
                        }
                    }
                }
            }

            val rows = detailRows(item, detail)
            if (rows.isNotEmpty()) {
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_assignment_details), style = LexturesType.display(18))
                        HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))
                        rows.forEach { (label, value) ->
                            Row(modifier = Modifier.fillMaxWidth().padding(vertical = 3.dp)) {
                                Text(label, fontSize = 14.sp, color = textSecondary(), modifier = Modifier.weight(1f))
                                Text(value, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = textPrimary(), textAlign = TextAlign.End)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AssignmentChip(text: String, tint: androidx.compose.ui.graphics.Color) {
    Text(
        text = text,
        fontSize = 12.sp,
        fontWeight = FontWeight.SemiBold,
        color = tint,
        modifier = Modifier
            .padding(horizontal = 9.dp, vertical = 4.dp)
            .backgroundChip(tint),
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
    )
}

@Composable
private fun Modifier.backgroundChip(tint: androidx.compose.ui.graphics.Color): Modifier =
    this.then(
        Modifier
            .padding(0.dp)
            .clipRounded(tint),
    )

@Composable
private fun Modifier.clipRounded(tint: androidx.compose.ui.graphics.Color): Modifier {
    val shape = androidx.compose.foundation.shape.RoundedCornerShape(999.dp)
    return this.then(
        Modifier
            .background(tint.copy(alpha = 0.13f), shape),
    )
}

private data class PendingAttachment(
    val bytes: ByteArray,
    val fileName: String,
    val mimeType: String,
)

@Composable
private fun StatusCard(
    status: AssignmentSubmissionStatus,
    submission: AssignmentSubmission?,
    grade: SubmissionGrade?,
    detail: ModuleItemDetail?,
    onPreviewAttachment: (AssignmentSubmission) -> Unit,
) {
    LmsCard {
        Text(L.text(R.string.mobile_assignment_yourWork), style = LexturesType.display(18), color = textPrimary())
        when (status) {
            AssignmentSubmissionStatus.NotStarted -> {
                Text(L.text(R.string.mobile_assignment_notSubmitted), fontSize = 14.sp, color = textSecondary())
            }
            AssignmentSubmissionStatus.Submitted, AssignmentSubmissionStatus.Late -> {
                submission?.let {
                    Text(
                        text = if (status == AssignmentSubmissionStatus.Late) {
                            L.text(R.string.mobile_assignment_submittedLate)
                        } else {
                            L.format(R.string.mobile_assignment_submittedAt, LmsDates.shortDateTime(it.submittedAt))
                        },
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Medium,
                    )
                    it.versionNumber?.takeIf { version -> version > 1 }?.let { version ->
                        Text(L.format(R.string.mobile_assignment_version, version), fontSize = 12.sp, color = textSecondary())
                    }
                }
            }
            AssignmentSubmissionStatus.RevisionRequested -> {
                Text(L.text(R.string.mobile_assignment_revisionRequested), color = LexturesColors.Coral, fontWeight = FontWeight.SemiBold)
                submission?.revisionFeedback?.takeIf { it.isNotBlank() }?.let {
                    Text(it, fontSize = 12.sp, color = textSecondary())
                }
            }
            AssignmentSubmissionStatus.Graded -> {
                submission?.let {
                    Text(L.format(R.string.mobile_assignment_submittedAt, LmsDates.shortDateTime(it.submittedAt)), fontSize = 14.sp)
                }
                grade?.pointsEarned?.let { earned ->
                    HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))
                    Row(modifier = Modifier.fillMaxWidth()) {
                        Text(L.text(R.string.mobile_assignment_grade), color = textSecondary())
                        Text(
                            text = grade.maxPoints?.let { max -> "$earned / $max" } ?: earned.toString(),
                            modifier = Modifier.weight(1f),
                            textAlign = TextAlign.End,
                            fontWeight = FontWeight.Bold,
                            color = accentColor(),
                        )
                    }
                }
            }
        }
        if (AssignmentLogic.hasAttachment(submission)) {
            TextButton(onClick = { submission?.let(onPreviewAttachment) }) {
                Text(submission?.attachmentFilename ?: L.text(R.string.mobile_assignment_attachment))
            }
        }
    }
}

@Composable
private fun ComposerCard(
    detail: ModuleItemDetail?,
    submission: AssignmentSubmission?,
    resubmissionEnabled: Boolean,
    draftText: String,
    onDraftChange: (String) -> Unit,
    urlText: String,
    onUrlChange: (String) -> Unit,
    pendingAttachment: PendingAttachment?,
    uploadProgress: Double,
    uploadError: String?,
    isOnline: Boolean,
    submitting: Boolean,
    canSubmit: Boolean,
    onPickPhoto: () -> Unit,
    onPickFile: () -> Unit,
    onCamera: () -> Unit,
    onRetryUpload: () -> Unit,
    onSubmit: () -> Unit,
) {
    LmsCard {
        Text(L.text(R.string.mobile_assignment_submitWork), style = LexturesType.display(18))
        AssignmentLogic.submitDisabledReasonKey(detail, submission, resubmissionEnabled)?.let { key ->
            Text(
                text = assignmentDisabledReason(key),
                fontSize = 12.sp,
                color = LexturesColors.Coral,
            )
        }
        if (AssignmentLogic.canSubmitText(detail, submission, resubmissionEnabled)) {
            OutlinedTextField(
                value = draftText,
                onValueChange = onDraftChange,
                modifier = Modifier.fillMaxWidth().heightIn(min = 120.dp),
                placeholder = { Text(L.text(R.string.mobile_assignment_textPlaceholder)) },
            )
        }
        if (detail?.submissionAllowUrl == true) {
            OutlinedTextField(
                value = urlText,
                onValueChange = onUrlChange,
                modifier = Modifier.fillMaxWidth(),
                placeholder = { Text(L.text(R.string.mobile_assignment_urlPlaceholder)) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
            )
        }
        if (AssignmentLogic.canSubmitFile(detail, submission, resubmissionEnabled)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                TextButton(onClick = onPickPhoto) {
                    Icon(Icons.Default.Photo, contentDescription = null)
                    Text(L.text(R.string.mobile_assignment_pickPhoto))
                }
                TextButton(onClick = onCamera) {
                    Icon(Icons.Default.CameraAlt, contentDescription = null)
                    Text(L.text(R.string.mobile_assignment_camera))
                }
                TextButton(onClick = onPickFile) {
                    Icon(Icons.Default.Folder, contentDescription = null)
                    Text(L.text(R.string.mobile_assignment_pickFile))
                }
            }
            if (uploadProgress in 0.0..1.0) {
                LinearProgressIndicator(progress = { uploadProgress.toFloat() }, modifier = Modifier.fillMaxWidth())
            }
            uploadError?.let {
                Column {
                    Text(it, fontSize = 12.sp, color = LexturesColors.Coral)
                    TextButton(onClick = onRetryUpload) { Text(L.text(R.string.mobile_common_retry)) }
                }
            }
            pendingAttachment?.let {
                Text(it.fileName, fontSize = 12.sp, color = textSecondary())
            }
        }
        if (!isOnline) {
            Text(L.text(R.string.mobile_assignment_offlineHint), fontSize = 12.sp, color = LexturesColors.Amber)
        }
        AuthPrimaryButton(
            text = when {
                submitting -> L.text(R.string.mobile_assignment_submitting)
                submission != null -> L.text(R.string.mobile_assignment_resubmit)
                else -> L.text(R.string.mobile_assignment_submit)
            },
            onClick = onSubmit,
            enabled = canSubmit && !submitting && uploadProgress < 0,
        )
    }
}

@Composable
private fun assignmentDisabledReason(key: String): String = when (key) {
    "mobile.assignment.noSubmissionTypes" -> L.text(R.string.mobile_assignment_noSubmissionTypes)
    "mobile.assignment.closed" -> L.text(R.string.mobile_assignment_closed)
    "mobile.assignment.revisionPastDue" -> L.text(R.string.mobile_assignment_revisionPastDue)
    "mobile.assignment.pastDueBlocked" -> L.text(R.string.mobile_assignment_pastDueBlocked)
    "mobile.assignment.fileLocked" -> L.text(R.string.mobile_assignment_fileLocked)
    else -> key
}
