package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

object CredentialsLogic {
    fun credentialsEnabled(features: MobilePlatformFeatures): Boolean = features.ffCompletionCredentials

    fun cacheKey(): String = "credentials:list"

    fun credentialDetailCacheKey(id: String): String = "credentials:$id"

    fun sourceTypeLabel(sourceType: String): String = when (sourceType) {
        "course" -> "Course"
        "path" -> "Learning path"
        "ceu" -> "CEU"
        else -> sourceType
    }

    fun issuedDateLabel(iso: String): String = runCatching {
        Instant.parse(iso).atZone(ZoneId.systemDefault())
            .format(DateTimeFormatter.ofPattern("MMM d, yyyy"))
    }.getOrDefault(iso)

    fun shareText(title: String, verificationUrl: String): String =
        "I earned \"$title\" on Lextures. Verify: $verificationUrl"
}