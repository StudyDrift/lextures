package com.lextures.android.features.boards

import android.Manifest
import android.content.pm.PackageManager
import android.media.MediaRecorder
import android.net.Uri
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardComposeValidation
import com.lextures.android.core.lms.BoardContentType
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardPostBody
import com.lextures.android.core.lms.BoardPostsApi
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.launch
import java.io.File

private data class PendingBoardFile(
    val bytes: ByteArray,
    val fileName: String,
    val mimeType: String,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardComposerSheet(
    session: AuthSession,
    courseCode: String,
    boardId: String,
    accessToken: String,
    onDismiss: () -> Unit,
    onCreated: (BoardPost) -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val loadFileError = L.text(R.string.mobile_boards_compose_loadFileError)
    val micDenied = L.text(R.string.mobile_boards_compose_micDenied)
    val bodyRequired = L.text(R.string.mobile_boards_compose_bodyRequired)
    val linkRequired = L.text(R.string.mobile_boards_compose_linkRequired)
    val fileRequired = L.text(R.string.mobile_boards_compose_fileRequired)
    val altRequired = L.text(R.string.mobile_boards_compose_altRequired)
    val audioRequired = L.text(R.string.mobile_boards_compose_audioRequired)
    val quotaExceeded = L.text(R.string.mobile_boards_compose_quotaExceeded)
    val filterBlocked = L.text(R.string.mobile_boards_moderation_filterBlocked)
    val lockedNotice = L.text(R.string.mobile_boards_sync_lockedNotice)

    var contentType by remember { mutableStateOf(BoardContentType.Text) }
    var title by remember { mutableStateOf("") }
    var bodyText by remember { mutableStateOf("") }
    var linkUrl by remember { mutableStateOf("") }
    var altText by remember { mutableStateOf("") }
    var pending by remember { mutableStateOf<PendingBoardFile?>(null) }
    var uploading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var validationHint by remember { mutableStateOf<String?>(null) }
    var showMicExplainer by remember { mutableStateOf(false) }
    var showCameraExplainer by remember { mutableStateOf(false) }
    var isRecording by remember { mutableStateOf(false) }
    var recorder by remember { mutableStateOf<MediaRecorder?>(null) }
    var recordFile by remember { mutableStateOf<File?>(null) }
    var cameraUri by remember { mutableStateOf<Uri?>(null) }

    fun loadUri(uri: Uri, fallbackName: String, fallbackMime: String) {
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                val bytes = stream.readBytes()
                val mime = context.contentResolver.getType(uri) ?: fallbackMime
                val name = uri.lastPathSegment?.substringAfterLast('/') ?: fallbackName
                pending = PendingBoardFile(bytes, name, mime)
            }
        }.onFailure {
            errorMessage = loadFileError
        }
    }

    val pickPhoto = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        loadUri(uri, "photo.jpg", "image/jpeg")
    }
    val pickFile = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        loadUri(uri, "upload.bin", "application/octet-stream")
    }
    val takePicture = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { ok ->
        if (!ok) return@rememberLauncherForActivityResult
        val uri = cameraUri ?: return@rememberLauncherForActivityResult
        loadUri(uri, "photo.jpg", "image/jpeg")
    }
    val cameraPermission = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) {
            val file = File(context.cacheDir, "board-camera-${System.currentTimeMillis()}.jpg")
            val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
            cameraUri = uri
            takePicture.launch(uri)
        }
    }
    val micPermission = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) startRecording(context) { rec, file ->
            recorder = rec
            recordFile = file
            isRecording = true
        } else {
            errorMessage = micDenied
        }
    }

    fun stopRecording() {
        runCatching {
            recorder?.stop()
            recorder?.release()
        }
        recorder = null
        isRecording = false
        val file = recordFile ?: return
        pending = PendingBoardFile(file.readBytes(), file.name, "audio/mp4")
    }

    ModalBottomSheet(onDismissRequest = {
        if (isRecording) stopRecording()
        onDismiss()
    }, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(L.text(R.string.mobile_boards_compose_navTitle))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                listOf(
                    BoardContentType.Text,
                    BoardContentType.Image,
                    BoardContentType.Link,
                    BoardContentType.File,
                    BoardContentType.Audio,
                ).forEach { type ->
                    FilterChip(
                        selected = contentType == type,
                        onClick = { contentType = type },
                        label = {
                            Text(
                                when (type) {
                                    BoardContentType.Text -> L.text(R.string.mobile_boards_post_type_text)
                                    BoardContentType.Image -> L.text(R.string.mobile_boards_post_type_image)
                                    BoardContentType.Link -> L.text(R.string.mobile_boards_post_type_link)
                                    BoardContentType.File -> L.text(R.string.mobile_boards_post_type_file)
                                    BoardContentType.Audio -> L.text(R.string.mobile_boards_post_type_audio)
                                    else -> type.apiValue
                                },
                            )
                        },
                    )
                }
            }
            Text(L.text(R.string.mobile_boards_compose_drawingDisabledHint), color = textSecondary())

            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                label = { Text(L.text(R.string.mobile_boards_compose_titleLabel)) },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )

            when (contentType) {
                BoardContentType.Text -> OutlinedTextField(
                    value = bodyText,
                    onValueChange = { bodyText = it },
                    label = { Text(L.text(R.string.mobile_boards_compose_bodyLabel)) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 3,
                )
                BoardContentType.Link, BoardContentType.Video -> OutlinedTextField(
                    value = linkUrl,
                    onValueChange = { linkUrl = it },
                    label = { Text(L.text(R.string.mobile_boards_compose_linkLabel)) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
                BoardContentType.Image -> {
                    pending?.let { Text(it.fileName, color = textSecondary()) }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        TextButton(onClick = { pickPhoto.launch("image/*") }) {
                            Text(L.text(R.string.mobile_boards_compose_pickPhoto))
                        }
                        TextButton(onClick = { showCameraExplainer = true }) {
                            Text(L.text(R.string.mobile_boards_compose_takePhoto))
                        }
                    }
                    OutlinedTextField(
                        value = altText,
                        onValueChange = { altText = it },
                        label = { Text(L.text(R.string.mobile_boards_compose_altLabel)) },
                        modifier = Modifier.fillMaxWidth(),
                        minLines = 2,
                    )
                    if (altText.isBlank()) {
                        Text(L.text(R.string.mobile_boards_compose_altHint), color = textSecondary())
                    }
                }
                BoardContentType.File -> {
                    pending?.let { Text(it.fileName, color = textSecondary()) }
                    TextButton(onClick = { pickFile.launch(arrayOf("*/*")) }) {
                        Text(L.text(R.string.mobile_boards_compose_pickFile))
                    }
                }
                BoardContentType.Audio -> {
                    pending?.let { Text(it.fileName, color = textSecondary()) }
                    if (isRecording) {
                        TextButton(onClick = { stopRecording() }) {
                            Text(L.text(R.string.mobile_boards_compose_stopRecord))
                        }
                    } else {
                        TextButton(onClick = { showMicExplainer = true }) {
                            Text(L.text(R.string.mobile_boards_compose_recordAudio))
                        }
                    }
                }
                BoardContentType.Drawing -> Unit
            }

            if (uploading) {
                LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
                Text(L.text(R.string.mobile_boards_compose_uploading), color = textSecondary())
            }
            validationHint?.let { Text(it, color = androidx.compose.ui.graphics.Color(0xFFE67E22)) }
            errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color(0xFFC0392B)) }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                TextButton(
                    onClick = {
                        if (isRecording) stopRecording()
                        onDismiss()
                    },
                    enabled = !uploading,
                ) { Text(L.text(R.string.mobile_common_cancel)) }
                TextButton(
                    onClick = {
                        if (isRecording) stopRecording()
                        validationHint = null
                        errorMessage = null
                        val validation = BoardsLogic.validateCompose(
                            contentType = contentType,
                            text = bodyText,
                            linkUrl = linkUrl,
                            hasFile = pending != null,
                            altText = altText,
                            hasAudio = contentType == BoardContentType.Audio && pending != null,
                        )
                        validationHint = when (validation) {
                            BoardComposeValidation.Ok -> null
                            BoardComposeValidation.MissingText -> bodyRequired
                            BoardComposeValidation.MissingLink -> linkRequired
                            BoardComposeValidation.MissingFile -> fileRequired
                            BoardComposeValidation.MissingAltText -> altRequired
                            BoardComposeValidation.MissingAudio -> audioRequired
                        }
                        if (validation != BoardComposeValidation.Ok) return@TextButton
                        scope.launch {
                            uploading = true
                            try {
                                var attachmentId: String? = null
                                var postType = contentType.apiValue
                                var body: BoardPostBody? = null
                                var link: String? = null
                                when (contentType) {
                                    BoardContentType.Text -> body = BoardsLogic.makeTextBody(bodyText)
                                    BoardContentType.Link -> {
                                        link = linkUrl.trim()
                                        if (BoardsLogic.videoEmbedFromUrl(link) != null) {
                                            postType = BoardContentType.Video.apiValue
                                        }
                                    }
                                    BoardContentType.Image, BoardContentType.File, BoardContentType.Audio -> {
                                        val file = pending ?: return@launch
                                        val att = BoardPostsApi.uploadAttachment(
                                            courseCode = courseCode,
                                            boardId = boardId,
                                            fileName = file.fileName,
                                            mimeType = file.mimeType,
                                            fileBytes = file.bytes,
                                            altText = if (contentType == BoardContentType.Image) altText.trim() else null,
                                            contentType = contentType.apiValue,
                                            accessToken = accessToken,
                                        )
                                        attachmentId = att.id
                                    }
                                    else -> Unit
                                }
                                val created = BoardPostsApi.createPost(
                                    courseCode = courseCode,
                                    boardId = boardId,
                                    contentType = postType,
                                    title = title.trim().ifEmpty { null },
                                    body = body,
                                    linkUrl = link,
                                    attachmentId = attachmentId,
                                    accessToken = accessToken,
                                )
                                onCreated(created)
                                onDismiss()
                            } catch (e: ApiError.HttpStatus) {
                                errorMessage = when {
                                    e.message?.contains("Storage limit") == true -> quotaExceeded
                                    BoardsLogic.isFilterBlockMessage(e.message) -> filterBlocked
                                    BoardsLogic.isLockOrFreezeMessage(e.message) -> lockedNotice
                                    else -> session.mapError(e)
                                }
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            } finally {
                                uploading = false
                            }
                        }
                    },
                    enabled = !uploading,
                ) { Text(L.text(R.string.mobile_boards_compose_submit)) }
            }
        }
    }

    if (showMicExplainer) {
        AlertDialog(
            onDismissRequest = { showMicExplainer = false },
            title = { Text(L.text(R.string.mobile_boards_compose_micPermissionTitle)) },
            text = { Text(L.text(R.string.mobile_boards_compose_micPermissionMessage)) },
            confirmButton = {
                TextButton(onClick = {
                    showMicExplainer = false
                    when {
                        ContextCompat.checkSelfPermission(context, Manifest.permission.RECORD_AUDIO) ==
                            PackageManager.PERMISSION_GRANTED -> {
                            startRecording(context) { rec, file ->
                                recorder = rec
                                recordFile = file
                                isRecording = true
                            }
                        }
                        else -> micPermission.launch(Manifest.permission.RECORD_AUDIO)
                    }
                }) { Text(L.text(R.string.mobile_boards_compose_continue)) }
            },
            dismissButton = {
                TextButton(onClick = { showMicExplainer = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (showCameraExplainer) {
        AlertDialog(
            onDismissRequest = { showCameraExplainer = false },
            title = { Text(L.text(R.string.mobile_boards_compose_cameraPermissionTitle)) },
            text = { Text(L.text(R.string.mobile_boards_compose_cameraPermissionMessage)) },
            confirmButton = {
                TextButton(onClick = {
                    showCameraExplainer = false
                    when {
                        ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) ==
                            PackageManager.PERMISSION_GRANTED -> {
                            val file = File(context.cacheDir, "board-camera-${System.currentTimeMillis()}.jpg")
                            val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
                            cameraUri = uri
                            takePicture.launch(uri)
                        }
                        else -> cameraPermission.launch(Manifest.permission.CAMERA)
                    }
                }) { Text(L.text(R.string.mobile_boards_compose_continue)) }
            },
            dismissButton = {
                TextButton(onClick = { showCameraExplainer = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }
}

private fun startRecording(
    context: android.content.Context,
    onReady: (MediaRecorder, File) -> Unit,
) {
    val file = File(context.cacheDir, "board-audio-${System.currentTimeMillis()}.m4a")
    val recorder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
        MediaRecorder(context)
    } else {
        @Suppress("DEPRECATION")
        MediaRecorder()
    }
    recorder.setAudioSource(MediaRecorder.AudioSource.MIC)
    recorder.setOutputFormat(MediaRecorder.OutputFormat.MPEG_4)
    recorder.setAudioEncoder(MediaRecorder.AudioEncoder.AAC)
    recorder.setOutputFile(file.absolutePath)
    recorder.prepare()
    recorder.start()
    onReady(recorder, file)
}
