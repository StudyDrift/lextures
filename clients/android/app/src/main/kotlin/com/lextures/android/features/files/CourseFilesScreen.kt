package com.lextures.android.features.files

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.DownloadDone
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.realtime.CourseFilesSocket
import androidx.compose.foundation.shape.RoundedCornerShape
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseFileBreadcrumb
import com.lextures.android.core.lms.CourseFileFolder
import com.lextures.android.core.lms.CourseFileFolderContents
import com.lextures.android.core.lms.CourseFileItem
import com.lextures.android.core.lms.CourseFileLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.FileDownloadManager
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import android.text.format.Formatter

@Composable
fun CourseFilesScreen(
    session: AuthSession,
    course: CourseSummary,
    onOpenPreview: (FilePreviewTarget) -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val storageBytes by offline.storageBytes.collectAsState()

    var folderId by remember { mutableStateOf<String?>(null) }
    var breadcrumbs by remember { mutableStateOf<List<CourseFileBreadcrumb>>(emptyList()) }
    var folders by remember { mutableStateOf<List<CourseFileFolder>>(emptyList()) }
    var files by remember { mutableStateOf<List<CourseFileItem>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var savedKeys by remember { mutableStateOf<Set<String>>(emptySet()) }
    var showClearConfirm by remember { mutableStateOf(false) }

    val filesSocket = remember(course.courseCode) { CourseFilesSocket() }
    val filesRevision by filesSocket.revision.collectAsState()
    DisposableEffect(course.courseCode) {
        filesSocket.connect(course.courseCode) { accessToken }
        onDispose { filesSocket.disconnect() }
    }

    val rootLabel = fileRootLabel()
    val emptyTitle = fileEmptyFolderTitle()
    val emptyHint = fileEmptyFolderHint()
    val loadError = fileLoadErrorLabel()
    val cacheSizeLabel = fileCacheSizeLabel()
    val clearCacheLabel = fileClearCacheLabel()
    val clearCacheTitle = fileClearCacheTitle()
    val clearCacheMessage = fileClearCacheMessage()
    val clearCacheConfirm = fileClearCacheConfirmLabel()
    val folderLabel = fileFolderLabel()
    val savedLabel = fileSavedLabel()

    LaunchedEffect(accessToken, folderId, filesRevision) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val cacheKey = OfflineCacheKey.courseFiles(course.courseCode, folderId)
            val result = offline.cachedFetch(
                key = cacheKey,
                accessToken = token,
                serializer = CourseFileFolderContents.serializer(),
            ) {
                if (folderId != null) {
                    LmsApi.fetchCourseFilesFolder(course.courseCode, folderId!!, token)
                } else {
                    LmsApi.fetchCourseFilesRoot(course.courseCode, token)
                }
            }
            folders = result.first.folders.sortedBy { it.name.lowercase() }
            files = result.first.files.sortedBy { it.title.lowercase() }
            breadcrumbs = result.first.breadcrumbs.orEmpty()
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e).ifBlank { loadError }
        } finally {
            loading = false
        }
    }

    LaunchedEffect(files, accessToken) {
        val keys = mutableSetOf<String>()
        for (file in files) {
            val target = FilePreviewTarget.from(file, course.courseCode)
            if (FileDownloadManager.isDownloaded(target, offline)) {
                keys += file.id
            }
        }
        savedKeys = keys
    }

    if (showClearConfirm) {
        AlertDialog(
            onDismissRequest = { showClearConfirm = false },
            title = { Text(clearCacheTitle) },
            text = { Text(clearCacheMessage) },
            confirmButton = {
                TextButton(onClick = {
                    showClearConfirm = false
                    offline.clearStorage()
                }) { Text(clearCacheConfirm) }
            },
            dismissButton = {
                TextButton(onClick = { showClearConfirm = false }) { Text("Cancel") }
            },
        )
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            BreadcrumbChip(title = rootLabel, active = folderId == null) { folderId = null }
            breadcrumbs.forEach { crumb ->
                Icon(
                    Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = null,
                    tint = textSecondary(),
                    modifier = Modifier.padding(horizontal = 2.dp),
                )
                BreadcrumbChip(title = crumb.name, active = folderId == crumb.id) { folderId = crumb.id }
            }
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(cacheSizeLabel, fontSize = 12.sp, color = textSecondary())
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    Formatter.formatFileSize(context, storageBytes),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                )
                if (storageBytes > 0) {
                    TextButton(onClick = { showClearConfirm = true }) {
                        Text(clearCacheLabel, fontSize = 12.sp)
                    }
                }
            }
        }

        errorMessage?.let { LmsErrorBanner(it) }
        cacheLabel?.let { StalenessChip(label = it) }

        when {
            loading && folders.isEmpty() && files.isEmpty() -> LmsSkeletonList(count = 4)
            folders.isEmpty() && files.isEmpty() && errorMessage == null -> {
                LmsEmptyState(icon = Icons.Default.Folder, title = emptyTitle, message = emptyHint)
            }
            else -> {
                LazyColumn(
                    modifier = Modifier
                        .clip(RoundedCornerShape(16.dp))
                        .background(cardBackground()),
                ) {
                    items(folders, key = { "folder-${it.id}" }) { folder ->
                        FileRow(
                            name = folder.name,
                            subtitle = folderLabel,
                            saved = false,
                            savedLabel = savedLabel,
                            onClick = { folderId = folder.id },
                        )
                        HorizontalDivider(modifier = Modifier.padding(start = 44.dp))
                    }
                    items(files, key = { "file-${it.id}" }) { file ->
                        FileRow(
                            name = file.title,
                            subtitle = fileSubtitle(file),
                            saved = savedKeys.contains(file.id),
                            savedLabel = savedLabel,
                            onClick = { onOpenPreview(FilePreviewTarget.from(file, course.courseCode)) },
                        )
                        HorizontalDivider(modifier = Modifier.padding(start = 44.dp))
                    }
                }
            }
        }
    }
}

@Composable
private fun BreadcrumbChip(title: String, active: Boolean, onClick: () -> Unit) {
    Text(
        text = title,
        fontSize = 12.sp,
        fontWeight = if (active) FontWeight.SemiBold else FontWeight.Normal,
        color = if (active) accentColor() else textSecondary(),
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
        modifier = Modifier
            .clickable(enabled = !active, onClick = onClick)
            .semantics { contentDescription = title },
    )
}

@Composable
private fun FileRow(
    name: String,
    subtitle: String,
    saved: Boolean,
    savedLabel: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 12.dp)
            .semantics { contentDescription = "$name, $subtitle" },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(Icons.Default.Folder, contentDescription = null, tint = accentColor())
        Column(modifier = Modifier.weight(1f)) {
            Text(name, fontSize = 14.sp, fontWeight = FontWeight.Medium, color = textPrimary(), maxLines = 2)
            Text(subtitle, fontSize = 12.sp, color = textSecondary())
        }
        if (saved) {
            Icon(
                Icons.Default.DownloadDone,
                contentDescription = savedLabel,
                tint = LexturesColors.StrengthStrong,
            )
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = textSecondary())
    }
}

private fun fileSubtitle(file: CourseFileItem): String {
    val size = CourseFileLogic.formatBytes(file.byteSize)
    val date = LmsDates.parse(file.updatedAt)?.let { LmsDates.shortDate(file.updatedAt) }
    return if (date != null) "$size · $date" else size
}
