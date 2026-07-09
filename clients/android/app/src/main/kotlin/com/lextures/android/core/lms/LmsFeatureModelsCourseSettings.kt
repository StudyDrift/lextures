package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class MarkdownThemeCustom(
    val headingColor: String? = null,
    val bodyColor: String? = null,
    val linkColor: String? = null,
    val codeBackground: String? = null,
    val blockquoteBorder: String? = null,
    val articleWidth: String? = null,
    val fontFamily: String? = null,
) {
    companion object {
        fun seed() = MarkdownThemeCustom(
            headingColor = "#0f172a",
            bodyColor = "#334155",
            linkColor = "#4f46e5",
            codeBackground = "#f1f5f9",
            blockquoteBorder = "#cbd5e1",
            articleWidth = "comfortable",
            fontFamily = "sans",
        )
    }
}

@Serializable
data class CourseUpdateRequest(
    val title: String,
    val description: String,
    val published: Boolean,
    val startsAt: String? = null,
    val endsAt: String? = null,
    val visibleFrom: String? = null,
    val hiddenAt: String? = null,
    val scheduleMode: String,
    val relativeEndAfter: String? = null,
    val relativeHiddenAfter: String? = null,
    val courseHomeLanding: String,
    val courseHomeContentItemId: String? = null,
    val courseTimezone: String? = null,
    val gradeLevel: String? = null,
    val termId: String? = null,
)

@Serializable
data class CourseMarkdownThemePatch(
    val preset: String,
    val custom: MarkdownThemeCustom? = null,
)

@Serializable
data class CourseHeroImageURLRequest(val imageUrl: String)

@Serializable
data class CourseHeroPositionRequest(val objectPosition: String? = null)

@Serializable
data class CourseGenerateImageRequest(val prompt: String)

@Serializable
data class CourseGenerateImageResponse(val imageUrl: String? = null)

@Serializable
data class CourseFileUploadResponse(
    val id: String,
    val contentPath: String,
    val mimeType: String? = null,
    val byteSize: Int? = null,
)
