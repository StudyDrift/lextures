package com.lextures.android.features.portfolio

import android.Manifest
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenu
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.LinearProgressIndicator
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateArtifactRequest
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PatchArtifactRequest
import com.lextures.android.core.lms.PortfolioArtifact
import com.lextures.android.core.lms.PortfolioLogic
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSectionHeader
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.File

private enum class ArtifactEditorKind(val apiType: String, val labelRes: Int) {
    Upload("upload", R.string.mobile_portfolio_kind_upload),
    Url("url", R.string.mobile_portfolio_kind_url),
    TextPage("text_page", R.string.mobile_portfolio_kind_textPage),
    Heading("heading", R.string.mobile_portfolio_kind_heading),
    Submission("submission", R.string.mobile_portfolio_kind_submission),
}

private data class PendingPortfolioAttachment(
    val bytes: ByteArray,
    val fileName: String,
    val mimeType: String,
)

private data class SubmissionPick(
    val submissionId: String,
    val label: String,
)

/** Add or edit a portfolio artifact (M12.1). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ArtifactEditorScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    portfolioId: String,
    existing: PortfolioArtifact?,
    onSaved: (PortfolioArtifact) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()

    val isEditing = existing != null
    var kind by remember(existing) {
        mutableStateOf(
            when (existing?.artifactType) {
                "url" -> ArtifactEditorKind.Url
                "text_page" -> ArtifactEditorKind.TextPage
                "heading" -> ArtifactEditorKind.Heading
                "submission" -> ArtifactEditorKind.Submission
                else -> ArtifactEditorKind.Upload
            },
        )
    }
    var title by remember(existing) { mutableStateOf(existing?.title.orEmpty()) }
    var reflection by remember(existing) { mutableStateOf(existing?.description.orEmpty()) }
    var externalUrl by remember(existing) { mutableStateOf(existing?.externalUrl.orEmpty()) }
    var textContent by remember(existing) { mutableStateOf(existing?.textContent.orEmpty()) }
    var outcomeTags by remember(existing) {
        mutableStateOf(existing?.outcomeIds?.joinToString(", ").orEmpty())
    }
    var isPublic by remember(existing) { mutableStateOf(existing?.isPublic == true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var pendingAttachment by remember { mutableStateOf<PendingPortfolioAttachment?>(null) }
    var uploadProgress by remember { mutableStateOf(-1.0) }
    var uploadDone by remember { mutableStateOf(false) }

    var enrolledCourses by remember { mutableStateOf<List<CourseSummary>>(emptyList()) }
    var submissionPickerCourse by remember { mutableStateOf<CourseSummary?>(null) }
    var submissionOptions by remember { mutableStateOf<List<SubmissionPick>>(emptyList()) }
    var selectedSubmissionId by remember { mutableStateOf<String?>(null) }
    var loadingSubmissions by remember { mutableStateOf(false) }

    var kindMenuExpanded by remember { mutableStateOf(false) }
    var courseMenuExpanded by remember { mutableStateOf(false) }
    var submissionMenuExpanded by remember { mutableStateOf(false) }
    var cameraUri by remember { mutableStateOf<Uri?>(null) }

    val pickFileLauncher = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                val bytes = stream.readBytes()
                val mime = context.contentResolver.getType(uri) ?: "application/octet-stream"
                val name = uri.lastPathSegment?.substringAfterLast('/') ?: "upload"
                pendingAttachment = PendingPortfolioAttachment(bytes, name, mime)
                uploadDone = false
            }
        }
    }

    val pickPhotoLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                val bytes = stream.readBytes()
                val mime = context.contentResolver.getType(uri) ?: "image/jpeg"
                pendingAttachment = PendingPortfolioAttachment(bytes, "photo.jpg", mime)
                uploadDone = false
            }
        }
    }

    val takePictureLauncher = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { ok ->
        if (!ok) return@rememberLauncherForActivityResult
        val uri = cameraUri ?: return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                val bytes = stream.readBytes()
                pendingAttachment = PendingPortfolioAttachment(bytes, "photo.jpg", "image/jpeg")
                uploadDone = false
            }
        }
    }

    val cameraPermissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) {
            val file = File(context.cacheDir, "portfolio-camera-${System.currentTimeMillis()}.jpg")
            val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
            cameraUri = uri
            takePictureLauncher.launch(uri)
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        enrolledCourses = runCatching { LmsApi.fetchCourses(token) }
            .getOrDefault(emptyList())
            .filter { it.viewerIsStudent }
    }

    LaunchedEffect(accessToken, submissionPickerCourse) {
        val token = accessToken ?: return@LaunchedEffect
        val course = submissionPickerCourse ?: return@LaunchedEffect
        loadingSubmissions = true
        submissionOptions = emptyList()
        selectedSubmissionId = null
        try {
            val structure = LmsApi.fetchCourseStructure(course.courseCode, token)
            val picks = mutableListOf<SubmissionPick>()
            for (item in structure.filter { it.kind == "assignment" }) {
                val submission = runCatching {
                    LmsApi.fetchMySubmission(course.courseCode, item.id, token)
                }.getOrNull()
                if (submission != null) {
                    picks.add(
                        SubmissionPick(
                            submissionId = submission.id,
                            label = "${item.title} · v${submission.versionNumber ?: 1}",
                        ),
                    )
                }
            }
            submissionOptions = picks
        } catch (_: Exception) {
            submissionOptions = emptyList()
        } finally {
            loadingSubmissions = false
        }
    }

    fun uploadWithRetry(
        token: String,
        attachment: PendingPortfolioAttachment,
        trimmedTitle: String,
        outcomeIds: List<String>,
        attempt: Int = 1,
    ) {
        scope.launch {
            uploadProgress = 0.1
            errorMessage = null
            try {
                val artifact = LmsApi.uploadPortfolioArtifactFile(
                    portfolioId = portfolioId,
                    fileBytes = attachment.bytes,
                    fileName = attachment.fileName,
                    mimeType = attachment.mimeType,
                    title = trimmedTitle,
                    description = reflection.trim(),
                    outcomeIds = outcomeIds,
                    isPublic = isPublic,
                    accessToken = token,
                )
                uploadProgress = 1.0
                uploadDone = true
                onSaved(artifact)
            } catch (e: Exception) {
                if (attempt < 3) {
                    delay(attempt * 800L)
                    uploadWithRetry(token, attachment, trimmedTitle, outcomeIds, attempt + 1)
                } else {
                    uploadProgress = -1.0
                    errorMessage = session.mapError(e).ifBlank {
                        L.text(context, localePrefs, R.string.mobile_portfolio_uploadFailed)
                    }
                }
            }
        }
    }

    fun save() {
        val token = accessToken ?: return
        val trimmedTitle = title.trim()
        if (trimmedTitle.isEmpty()) return
        val outcomeIds = PortfolioLogic.parseOutcomeIds(outcomeTags)

        if (isEditing && existing != null) {
            scope.launch {
                saving = true
                errorMessage = null
                try {
                    val updated = LmsApi.patchArtifact(
                        portfolioId = portfolioId,
                        artifactId = existing.id,
                        payload = PatchArtifactRequest(
                            title = trimmedTitle,
                            description = reflection.trim(),
                            textContent = if (kind == ArtifactEditorKind.TextPage) textContent else null,
                            externalUrl = if (kind == ArtifactEditorKind.Url) externalUrl.trim() else null,
                            outcomeIds = outcomeIds.ifEmpty { null },
                            isPublic = isPublic,
                        ),
                        accessToken = token,
                    )
                    onSaved(updated)
                } catch (_: Exception) {
                    errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_saveError)
                } finally {
                    saving = false
                }
            }
            return
        }

        when (kind) {
            ArtifactEditorKind.Upload -> {
                val attachment = pendingAttachment
                if (attachment == null) {
                    errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_fileRequired)
                    return
                }
                uploadWithRetry(token, attachment, trimmedTitle, outcomeIds)
            }
            else -> {
                scope.launch {
                    saving = true
                    errorMessage = null
                    try {
                        val submissionId = selectedSubmissionId
                        if (kind == ArtifactEditorKind.Submission && submissionId.isNullOrBlank()) {
                            errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_submissionRequired)
                            saving = false
                            return@launch
                        }
                        val created = LmsApi.createArtifact(
                            portfolioId = portfolioId,
                            payload = CreateArtifactRequest(
                                artifactType = kind.apiType,
                                title = trimmedTitle,
                                description = reflection.trim(),
                                sourceSubmissionId = submissionId,
                                textContent = if (kind == ArtifactEditorKind.TextPage) textContent else null,
                                externalUrl = if (kind == ArtifactEditorKind.Url) externalUrl.trim() else null,
                                outcomeIds = outcomeIds.ifEmpty { null },
                                isPublic = isPublic,
                            ),
                            accessToken = token,
                        )
                        onSaved(created)
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_portfolio_saveError)
                    } finally {
                        saving = false
                    }
                }
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            TextButton(onClick = onBack) {
                Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
            }
            TextButton(
                onClick = { save() },
                enabled = !saving && title.trim().isNotEmpty(),
            ) {
                Text(
                    L.text(
                        context,
                        localePrefs,
                        if (isEditing) R.string.mobile_common_save else R.string.mobile_portfolio_add,
                    ),
                )
            }
        }

        Text(
            L.text(
                context,
                localePrefs,
                if (isEditing) R.string.mobile_portfolio_editArtifact else R.string.mobile_portfolio_addArtifact,
            ),
            fontSize = 18.sp,
        )

        errorMessage?.let { LmsErrorBanner(message = it) }

        if (!isEditing) {
            ExposedDropdownMenuBox(
                expanded = kindMenuExpanded,
                onExpandedChange = { kindMenuExpanded = it },
            ) {
                OutlinedTextField(
                    value = L.text(context, localePrefs, kind.labelRes),
                    onValueChange = {},
                    readOnly = true,
                    label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_artifactKind)) },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = kindMenuExpanded) },
                    modifier = Modifier.menuAnchor().fillMaxWidth(),
                )
                ExposedDropdownMenu(
                    expanded = kindMenuExpanded,
                    onDismissRequest = { kindMenuExpanded = false },
                ) {
                    ArtifactEditorKind.entries.forEach { option ->
                        DropdownMenuItem(
                            text = { Text(L.text(context, localePrefs, option.labelRes)) },
                            onClick = {
                                kind = option
                                kindMenuExpanded = false
                            },
                        )
                    }
                }
            }
        }

        OutlinedTextField(
            value = title,
            onValueChange = { title = it },
            label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldTitle)) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedTextField(
            value = reflection,
            onValueChange = { reflection = it },
            label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldReflection)) },
            modifier = Modifier.fillMaxWidth(),
            minLines = 3,
            maxLines = 6,
        )
        OutlinedTextField(
            value = outcomeTags,
            onValueChange = { outcomeTags = it },
            label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldOutcomeTags)) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(L.text(context, localePrefs, R.string.mobile_portfolio_artifactPublic))
            Switch(checked = isPublic, onCheckedChange = { isPublic = it })
        }

        when (kind) {
            ArtifactEditorKind.Url -> {
                OutlinedTextField(
                    value = externalUrl,
                    onValueChange = { externalUrl = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldUrl)) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            }
            ArtifactEditorKind.TextPage -> {
                OutlinedTextField(
                    value = textContent,
                    onValueChange = { textContent = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_fieldContent)) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 4,
                    maxLines = 10,
                )
            }
            else -> Unit
        }

        if (kind == ArtifactEditorKind.Upload && !isEditing) {
            LmsSectionHeader(title = L.text(context, localePrefs, R.string.mobile_portfolio_attachFile))
            LmsCard {
                pendingAttachment?.let {
                    Text(it.fileName, fontSize = 12.sp, color = textSecondary(), modifier = Modifier.padding(bottom = 8.dp))
                }
                Button(onClick = { pickPhotoLauncher.launch("image/*") }, modifier = Modifier.fillMaxWidth()) {
                    Text(L.text(context, localePrefs, R.string.mobile_assignment_pickPhoto))
                }
                Button(
                    onClick = {
                        when {
                            context.checkSelfPermission(Manifest.permission.CAMERA) ==
                                android.content.pm.PackageManager.PERMISSION_GRANTED -> {
                                val file = File(context.cacheDir, "portfolio-camera-${System.currentTimeMillis()}.jpg")
                                val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
                                cameraUri = uri
                                takePictureLauncher.launch(uri)
                            }
                            else -> cameraPermissionLauncher.launch(Manifest.permission.CAMERA)
                        }
                    },
                    modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_assignment_camera))
                }
                Button(
                    onClick = { pickFileLauncher.launch(arrayOf("*/*")) },
                    modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_assignment_pickFile))
                }
                when {
                    uploadProgress in 0.0..1.0 && uploadProgress < 1.0 -> {
                        LinearProgressIndicator(
                            progress = { uploadProgress.toFloat() },
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_portfolio_uploading),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                    uploadDone -> Text(
                        L.text(context, localePrefs, R.string.mobile_portfolio_uploadDone),
                        fontSize = 12.sp,
                        color = textSecondary(),
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
        }

        if (kind == ArtifactEditorKind.Submission && !isEditing) {
            LmsSectionHeader(title = L.text(context, localePrefs, R.string.mobile_portfolio_fromSubmission))
            LmsCard {
                ExposedDropdownMenuBox(
                    expanded = courseMenuExpanded,
                    onExpandedChange = { courseMenuExpanded = it },
                ) {
                    OutlinedTextField(
                        value = submissionPickerCourse?.name
                            ?: L.text(context, localePrefs, R.string.mobile_portfolio_selectCourse),
                        onValueChange = {},
                        readOnly = true,
                        label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_pickCourse)) },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = courseMenuExpanded) },
                        modifier = Modifier.menuAnchor().fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = courseMenuExpanded,
                        onDismissRequest = { courseMenuExpanded = false },
                    ) {
                        enrolledCourses.forEach { course ->
                            DropdownMenuItem(
                                text = { Text(course.name) },
                                onClick = {
                                    submissionPickerCourse = course
                                    courseMenuExpanded = false
                                },
                            )
                        }
                    }
                }

                if (loadingSubmissions) {
                    CircularProgressIndicator(modifier = Modifier.padding(top = 12.dp))
                } else if (submissionPickerCourse != null && submissionOptions.isEmpty()) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_portfolio_noSubmissions),
                        fontSize = 12.sp,
                        color = textSecondary(),
                        modifier = Modifier.padding(top = 8.dp),
                    )
                } else if (submissionOptions.isNotEmpty()) {
                    ExposedDropdownMenuBox(
                        expanded = submissionMenuExpanded,
                        onExpandedChange = { submissionMenuExpanded = it },
                        modifier = Modifier.padding(top = 8.dp),
                    ) {
                        val selectedLabel = submissionOptions.firstOrNull { it.submissionId == selectedSubmissionId }?.label
                            ?: L.text(context, localePrefs, R.string.mobile_portfolio_selectSubmission)
                        OutlinedTextField(
                            value = selectedLabel,
                            onValueChange = {},
                            readOnly = true,
                            label = { Text(L.text(context, localePrefs, R.string.mobile_portfolio_pickSubmission)) },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = submissionMenuExpanded) },
                            modifier = Modifier.menuAnchor().fillMaxWidth(),
                        )
                        ExposedDropdownMenu(
                            expanded = submissionMenuExpanded,
                            onDismissRequest = { submissionMenuExpanded = false },
                        ) {
                            submissionOptions.forEach { option ->
                                DropdownMenuItem(
                                    text = { Text(option.label) },
                                    onClick = {
                                        selectedSubmissionId = option.submissionId
                                        submissionMenuExpanded = false
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