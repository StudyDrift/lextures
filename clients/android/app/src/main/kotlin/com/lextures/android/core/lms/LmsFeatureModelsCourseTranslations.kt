package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class TranslationCoverage(
    val targetLocale: String = "",
    val totalItems: Int = 0,
    val translatedItems: Int = 0,
    val percent: Double = 0.0,
    val untranslated: List<TranslatableCourseItem> = emptyList(),
) {
    val percentInt: Int get() = percent.toInt().let {
        // Prefer rounded whole percent for display.
        kotlin.math.round(percent).toInt()
    }
}

@Serializable
data class TranslatableCourseItem(
    val itemId: String = "",
    val itemType: String = "",
    val title: String = "",
    val body: String = "",
    val hasPublished: Boolean? = null,
    val hasDraft: Boolean? = null,
)

@Serializable
data class CourseTranslationListResponse(
    val items: List<CourseTranslationListItem> = emptyList(),
    val coverage: TranslationCoverage = TranslationCoverage(),
)

@Serializable
data class CourseTranslationListItem(
    val itemId: String = "",
    val itemType: String = "",
    val title: String = "",
    val body: String = "",
    val hasPublished: Boolean? = null,
    val hasDraft: Boolean? = null,
    val targetLocale: String? = null,
    val translatedTitle: String? = null,
    val translatedBody: String? = null,
    val isDraft: Boolean? = null,
    val machineTranslationDraft: Boolean? = null,
    val publishedAt: String? = null,
    val version: Long? = null,
    val glossaryMatches: List<GlossaryMatchSpan> = emptyList(),
)

@Serializable
data class GlossaryMatchSpan(
    val sourceTerm: String = "",
    val targetTerm: String = "",
    val start: Int? = null,
    val end: Int? = null,
)

@Serializable
data class CourseGlossaryListResponse(
    val entries: List<CourseGlossaryEntry> = emptyList(),
)

@Serializable
data class CourseGlossaryEntry(
    val id: String = "",
    val sourceTerm: String = "",
    val targetTerm: String = "",
    val sourceLocale: String? = null,
    val targetLocale: String? = null,
)

@Serializable
data class TranslationLocalesResponse(
    val locales: List<TranslationCoverage> = emptyList(),
)

@Serializable
data class SaveCourseTranslationBody(
    val targetLocale: String,
    val sourceLocale: String? = null,
    val translatedTitle: String? = null,
    val translatedBody: String? = null,
    val isDraft: Boolean? = null,
    val machineTranslationDraft: Boolean? = null,
    val version: Long? = null,
)

@Serializable
data class PublishCourseTranslationBody(
    val targetLocale: String,
)

@Serializable
data class AddGlossaryEntryBody(
    val sourceTerm: String,
    val targetTerm: String,
    val targetLocale: String,
    val sourceLocale: String? = "en",
)
