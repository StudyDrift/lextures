package com.lextures.android.features.files

import android.content.Intent
import android.graphics.Bitmap
import android.graphics.pdf.PdfRenderer
import android.net.Uri
import android.os.ParcelFileDescriptor
import android.widget.Toast
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.DownloadDone
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.FileProvider
import androidx.media3.common.MediaItem
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.ProgressiveMediaSource
import androidx.media3.ui.PlayerView
import coil.compose.AsyncImage
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.CourseFileLogic
import com.lextures.android.core.lms.FileDownloadManager
import com.lextures.android.core.lms.FilePreviewKind
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsEmptyState
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File

/** Reusable inline preview for course files, module file items, and submission attachments (M3.2). */
@Composable
fun FilePreviewScreen(
    session: AuthSession,
    target: FilePreviewTarget,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var previewBytes by remember { mutableStateOf<ByteArray?>(null) }
    var isSaved by remember { mutableStateOf(false) }
    var isDownloading by remember { mutableStateOf(false) }

    val previewKind = remember(target) {
        CourseFileLogic.previewKind(target.mimeType, target.displayName)
    }

    val downloadLabel = fileDownloadLabel()
    val openInLabel = fileOpenInLabel()
    val savedLabel = fileSavedLabel()
    val downloadOnlyHint = fileDownloadOnlyHint()
    val previewUnavailable = filePreviewUnavailableLabel()
    val offlineUnavailable = fileOfflineUnavailableLabel()
    val loadError = fileLoadErrorLabel()
    val downloadError = fileDownloadErrorLabel()
    val openError = fileOpenErrorLabel()

    BackHandler(onBack = onBack)

    LaunchedEffect(target, accessToken, isOnline) {
        loading = true
        errorMessage = null
        isSaved = FileDownloadManager.isDownloaded(target, offline)

        if (previewKind == FilePreviewKind.Audio || previewKind == FilePreviewKind.Video) {
            loading = false
            return@LaunchedEffect
        }

        FileDownloadManager.cachedBytes(target, offline)?.let {
            previewBytes = it
            isSaved = true
            loading = false
            return@LaunchedEffect
        }

        val token = accessToken
        if (token.isNullOrBlank()) {
            errorMessage = loadError
            loading = false
            return@LaunchedEffect
        }

        if (!isOnline) {
            errorMessage = offlineUnavailable
            loading = false
            return@LaunchedEffect
        }

        try {
            previewBytes = FileDownloadManager.fetchBytes(target, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e).ifBlank { loadError }
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.background(sceneBackground())) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = target.displayName,
                fontSize = 17.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            if (isSaved) {
                Icon(
                    Icons.Default.DownloadDone,
                    contentDescription = savedLabel,
                    tint = LexturesColors.StrengthStrong,
                    modifier = Modifier.padding(end = 4.dp),
                )
            }
            if (isDownloading) {
                CircularProgressIndicator(
                    modifier = Modifier.padding(12.dp),
                    color = LexturesColors.Primary,
                    strokeWidth = 2.dp,
                )
            } else {
                IconButton(onClick = {
                    val token = accessToken ?: return@IconButton
                    scope.launch {
                        isDownloading = true
                        try {
                            FileDownloadManager.download(target, token, offline)
                            isSaved = true
                            if (previewBytes == null &&
                                previewKind != FilePreviewKind.Audio &&
                                previewKind != FilePreviewKind.Video
                            ) {
                                previewBytes = FileDownloadManager.cachedBytes(target, offline)
                            }
                        } catch (e: Exception) {
                            errorMessage = session.mapError(e).ifBlank { downloadError }
                        } finally {
                            isDownloading = false
                        }
                    }
                }) {
                    Icon(Icons.Default.Download, contentDescription = downloadLabel)
                }
            }
            IconButton(onClick = {
                scope.launch {
                    try {
                        val bytes = previewBytes
                            ?: FileDownloadManager.cachedBytes(target, offline)
                            ?: accessToken?.let { FileDownloadManager.fetchBytes(target, it) }
                            ?: throw IllegalStateException(openError)
                        shareBytes(context, target, bytes)
                    } catch (e: Exception) {
                        Toast.makeText(context, session.mapError(e).ifBlank { openError }, Toast.LENGTH_SHORT).show()
                    }
                }
            }) {
                Icon(Icons.Default.Share, contentDescription = openInLabel)
            }
        }

        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            when {
                loading -> CircularProgressIndicator(color = LexturesColors.Primary)
                errorMessage != null -> LmsEmptyState(
                    icon = Icons.Default.Share,
                    title = target.displayName,
                    message = errorMessage!!,
                )
                previewKind == FilePreviewKind.Image && previewBytes != null -> {
                    ZoomableImage(bytes = previewBytes!!, label = target.displayName)
                }
                previewKind == FilePreviewKind.Pdf && previewBytes != null -> {
                    PdfPager(bytes = previewBytes!!)
                }
                previewKind == FilePreviewKind.Audio || previewKind == FilePreviewKind.Video -> {
                    val token = accessToken
                    if (!token.isNullOrBlank()) {
                        AuthedMediaPlayer(
                            url = FileDownloadManager.contentUrl(target.courseCode, target),
                            token = token,
                        )
                    } else {
                        LmsEmptyState(
                            icon = Icons.Default.Share,
                            title = target.displayName,
                            message = previewUnavailable,
                        )
                    }
                }
                previewKind == FilePreviewKind.DownloadOnly -> {
                    LmsEmptyState(
                        icon = Icons.Default.Download,
                        title = target.displayName,
                        message = downloadOnlyHint,
                    )
                }
                else -> LmsEmptyState(
                    icon = Icons.Default.Share,
                    title = target.displayName,
                    message = previewUnavailable,
                )
            }
        }
    }
}

@Composable
private fun ZoomableImage(bytes: ByteArray, label: String) {
    var scale by remember { mutableFloatStateOf(1f) }
    val state = rememberTransformableState { zoomChange, _, _ ->
        scale = (scale * zoomChange).coerceIn(1f, 4f)
    }
    AsyncImage(
        model = bytes,
        contentDescription = label,
        contentScale = ContentScale.Fit,
        modifier = Modifier
            .fillMaxSize()
            .graphicsLayer(scaleX = scale, scaleY = scale)
            .transformable(state = state),
    )
}

@Composable
private fun PdfPager(bytes: ByteArray) {
    var pageIndex by remember { mutableIntStateOf(0) }
    var pageCount by remember { mutableIntStateOf(0) }
    var bitmap by remember { mutableStateOf<Bitmap?>(null) }

    LaunchedEffect(bytes, pageIndex) {
        bitmap = withContext(Dispatchers.IO) { renderPdfPage(bytes, pageIndex) { pageCount = it } }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        bitmap?.let {
            Image(
                bitmap = it.asImageBitmap(),
                contentDescription = "PDF page ${pageIndex + 1}",
                modifier = Modifier.fillMaxWidth(),
            )
        }
        if (pageCount > 1) {
            Text("Page ${pageIndex + 1} of $pageCount", color = textPrimary())
            Slider(
                value = pageIndex.toFloat(),
                onValueChange = { pageIndex = it.toInt() },
                valueRange = 0f..(pageCount - 1).coerceAtLeast(0).toFloat(),
                steps = (pageCount - 2).coerceAtLeast(0),
            )
        }
    }
}

@androidx.annotation.OptIn(UnstableApi::class)
@Composable
private fun AuthedMediaPlayer(url: String, token: String) {
    val context = LocalContext.current
    val player = remember(url, token) {
        val dataSourceFactory = DefaultHttpDataSource.Factory()
            .setDefaultRequestProperties(mapOf("Authorization" to "Bearer $token"))
        val mediaSource = ProgressiveMediaSource.Factory(dataSourceFactory)
            .createMediaSource(MediaItem.fromUri(url))
        ExoPlayer.Builder(context).build().apply {
            setMediaSource(mediaSource)
            prepare()
        }
    }
    DisposableEffect(player) {
        onDispose { player.release() }
    }
    AndroidView(
        factory = { ctx -> PlayerView(ctx).apply { this.player = player } },
        modifier = Modifier.fillMaxSize(),
    )
}

private fun renderPdfPage(bytes: ByteArray, pageIndex: Int, onPageCount: (Int) -> Unit): Bitmap? {
    val temp = File.createTempFile("preview", ".pdf")
    return try {
        temp.writeBytes(bytes)
        ParcelFileDescriptor.open(temp, ParcelFileDescriptor.MODE_READ_ONLY).use { fd ->
            PdfRenderer(fd).use { renderer ->
                onPageCount(renderer.pageCount)
                if (pageIndex >= renderer.pageCount) return null
                renderer.openPage(pageIndex).use { page ->
                    val bmp = Bitmap.createBitmap(page.width, page.height, Bitmap.Config.ARGB_8888)
                    page.render(bmp, null, null, PdfRenderer.Page.RENDER_MODE_FOR_DISPLAY)
                    bmp
                }
            }
        }
    } finally {
        temp.delete()
    }
}

private fun shareBytes(context: android.content.Context, target: FilePreviewTarget, bytes: ByteArray) {
    val safeName = target.displayName.replace(Regex("[^a-zA-Z0-9._-]"), "_")
    val file = File(context.cacheDir, safeName)
    file.writeBytes(bytes)
    val uri: Uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
    val intent = Intent(Intent.ACTION_VIEW).apply {
        setDataAndType(uri, target.mimeType?.takeIf { it.isNotBlank() } ?: "*/*")
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }
    context.startActivity(Intent.createChooser(intent, null))
}
