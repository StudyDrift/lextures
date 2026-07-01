package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.offline.OfflineService
import okhttp3.OkHttpClient
import okhttp3.Request

object FileDownloadManager {
    private val http = OkHttpClient()

    fun contentUrl(courseCode: String, target: FilePreviewTarget): String =
        AppConfiguration.apiUrl(
            CourseFileLogic.contentPath(courseCode, target.source, target.sourceId),
        ).toString()

    fun previewUrl(courseCode: String, itemId: String): String =
        AppConfiguration.apiUrl(CourseFileLogic.previewPath(courseCode, itemId)).toString()

    fun authorizedRequest(url: String, accessToken: String, range: String? = null): Request {
        val builder = Request.Builder()
            .url(url)
            .header("Authorization", "Bearer $accessToken")
            .header("X-Platform", "android")
        if (range != null) {
            builder.header("Range", "bytes=$range")
        }
        return builder.build()
    }

    suspend fun fetchBytes(target: FilePreviewTarget, accessToken: String): ByteArray {
        val request = authorizedRequest(contentUrl(target.courseCode, target), accessToken)
        http.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                throw ApiError.HttpStatus(response.code, null)
            }
            return response.body?.bytes() ?: ByteArray(0)
        }
    }

    suspend fun fetchReportCardPdf(cardId: String, accessToken: String): ByteArray {
        val url = AppConfiguration.apiUrl("/api/v1/report-cards/$cardId/pdf").toString()
        val request = authorizedRequest(url, accessToken)
        http.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                throw ApiError.HttpStatus(response.code, null)
            }
            return response.body?.bytes() ?: ByteArray(0)
        }
    }

    suspend fun download(
        target: FilePreviewTarget,
        accessToken: String,
        offline: OfflineService,
    ) {
        val key = CourseFileLogic.downloadKey(target.courseCode, target)
        if (offline.isDownloaded(key)) return
        val data = fetchBytes(target, accessToken)
        offline.downloadContent(key, data, target.displayName, target.mimeType)
    }

    fun cachedBytes(target: FilePreviewTarget, offline: OfflineService): ByteArray? {
        val key = CourseFileLogic.downloadKey(target.courseCode, target)
        return offline.downloadedData(key)
    }

    fun isDownloaded(target: FilePreviewTarget, offline: OfflineService): Boolean {
        val key = CourseFileLogic.downloadKey(target.courseCode, target)
        return offline.isDownloaded(key)
    }
}
