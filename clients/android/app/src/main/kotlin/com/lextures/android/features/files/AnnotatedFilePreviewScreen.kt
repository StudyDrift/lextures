package com.lextures.android.features.files

import android.graphics.Bitmap
import android.graphics.pdf.PdfRenderer
import android.os.ParcelFileDescriptor
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesType
import androidx.compose.foundation.background
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.CourseFileLogic
import com.lextures.android.core.lms.FileDownloadManager
import com.lextures.android.core.lms.FilePreviewKind
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.SubmissionAnnotation
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/** File preview with optional instructor markup overlay (M6.1). */
@Composable
fun AnnotatedFilePreviewScreen(
    session: AuthSession,
    target: FilePreviewTarget,
    annotations: List<SubmissionAnnotation>,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    var loading by remember { mutableStateOf(true) }
    var bytes by remember { mutableStateOf<ByteArray?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val kind = remember(target) {
        CourseFileLogic.previewKind(target.mimeType, target.displayName)
    }

    LaunchedEffect(accessToken, target) {
        val token = accessToken ?: return@LaunchedEffect
        if (kind != FilePreviewKind.Image && kind != FilePreviewKind.Pdf) {
            loading = false
            return@LaunchedEffect
        }
        loading = true
        errorMessage = null
        try {
            bytes = FileDownloadManager.fetchBytes(target, token)
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    if (kind != FilePreviewKind.Image && kind != FilePreviewKind.Pdf) {
        FilePreviewScreen(session = session, target = target, onBack = onBack, modifier = modifier)
        return
    }

    Column(modifier = modifier.fillMaxSize().background(sceneBackground())) {
        IconButton(onClick = onBack) {
            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
        }
        Text(
            text = target.displayName,
            style = LexturesType.display(18, FontWeight.Bold),
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            errorMessage != null -> Text(errorMessage!!, modifier = Modifier.padding(16.dp))
            kind == FilePreviewKind.Image -> {
                val data = bytes
                if (data != null) {
                    Box(Modifier.fillMaxSize()) {
                        Image(
                            bitmap = android.graphics.BitmapFactory.decodeByteArray(data, 0, data.size).asImageBitmap(),
                            contentDescription = target.displayName,
                            modifier = Modifier.fillMaxSize(),
                            contentScale = ContentScale.Fit,
                        )
                        MarkupOverlay(annotations = annotations)
                    }
                }
            }
            kind == FilePreviewKind.Pdf -> {
                val data = bytes
                if (data != null) {
                    PdfWithOverlay(data = data, annotations = annotations)
                }
            }
        }
    }
}

@Composable
private fun PdfWithOverlay(data: ByteArray, annotations: List<SubmissionAnnotation>) {
    var bitmap by remember { mutableStateOf<Bitmap?>(null) }
    LaunchedEffect(data) {
        bitmap = withContext(Dispatchers.IO) {
            val file = File.createTempFile("preview", ".pdf")
            try {
                file.writeBytes(data)
                ParcelFileDescriptor.open(file, ParcelFileDescriptor.MODE_READ_ONLY).use { fd ->
                    PdfRenderer(fd).use { renderer ->
                        if (renderer.pageCount == 0) return@withContext null
                        renderer.openPage(0).use { page ->
                            val bmp = Bitmap.createBitmap(page.width, page.height, Bitmap.Config.ARGB_8888)
                            page.render(bmp, null, null, PdfRenderer.Page.RENDER_MODE_FOR_DISPLAY)
                            bmp
                        }
                    }
                }
            } finally {
                file.delete()
            }
        }
    }
    Column(Modifier.verticalScroll(rememberScrollState()).padding(16.dp)) {
        bitmap?.let { bmp ->
            Box(Modifier.fillMaxWidth()) {
                Image(
                    bitmap = bmp.asImageBitmap(),
                    contentDescription = "PDF page",
                    modifier = Modifier.fillMaxWidth(),
                    contentScale = ContentScale.FillWidth,
                )
                MarkupOverlay(annotations = annotations, page = 1)
            }
        } ?: CircularProgressIndicator()
    }
}
