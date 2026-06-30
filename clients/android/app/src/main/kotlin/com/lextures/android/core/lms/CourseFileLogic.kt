package com.lextures.android.core.lms

import java.net.URLEncoder
import java.text.DecimalFormat

object CourseFileLogic {
    fun encodePath(value: String): String = URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    fun contentPath(courseCode: String, source: CourseFileContentSource, sourceId: String): String =
        when (source) {
            CourseFileContentSource.FileManager ->
                "/api/v1/courses/${encodePath(courseCode)}/files/items/${encodePath(sourceId)}/content"
            CourseFileContentSource.CourseFile ->
                "/api/v1/courses/${encodePath(courseCode)}/course-files/${encodePath(sourceId)}/content"
        }

    fun previewPath(courseCode: String, itemId: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/files/items/${encodePath(itemId)}/preview"

    fun downloadKey(courseCode: String, target: FilePreviewTarget): String =
        "download:$courseCode:${target.id}"

    fun courseFilesCacheKey(courseCode: String, folderId: String?): String =
        if (!folderId.isNullOrEmpty()) "course:$courseCode:files:folder:$folderId"
        else "course:$courseCode:files:root"

    fun previewKind(mimeType: String?, fileName: String): FilePreviewKind {
        val mime = mimeType.orEmpty().lowercase()
        val ext = fileName.substringAfterLast('.', "").lowercase()
        return when {
            mime.startsWith("image/") || ext in setOf("png", "jpg", "jpeg", "gif", "webp", "heic", "svg") ->
                FilePreviewKind.Image
            mime == "application/pdf" || ext == "pdf" -> FilePreviewKind.Pdf
            mime.startsWith("audio/") || ext in setOf("mp3", "wav", "m4a", "aac", "ogg") -> FilePreviewKind.Audio
            mime.startsWith("video/") || ext in setOf("mp4", "mov", "webm", "m4v") -> FilePreviewKind.Video
            else -> FilePreviewKind.DownloadOnly
        }
    }

    fun guessMimeType(fileName: String): String? {
        return when (fileName.substringAfterLast('.', "").lowercase()) {
            "pdf" -> "application/pdf"
            "png" -> "image/png"
            "jpg", "jpeg" -> "image/jpeg"
            "gif" -> "image/gif"
            "webp" -> "image/webp"
            "mp4" -> "video/mp4"
            "mov" -> "video/quicktime"
            "mp3" -> "audio/mpeg"
            "wav" -> "audio/wav"
            "m4a" -> "audio/mp4"
            else -> null
        }
    }

    fun formatBytes(bytes: Long): String {
        if (bytes <= 0) return "0 B"
        val units = arrayOf("B", "KB", "MB", "GB")
        var value = bytes.toDouble()
        var unit = 0
        while (value >= 1024 && unit < units.lastIndex) {
            value /= 1024
            unit++
        }
        return DecimalFormat("#.#").format(value) + " " + units[unit]
    }
}
