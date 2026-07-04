package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ReadingPreferencesRow(
    val fontFace: String = "default",
    val letterSpacing: String = "normal",
    val wordSpacing: String = "normal",
    val lineHeight: String = "normal",
    val rulerEnabled: Boolean = false,
    val rulerColor: String = "yellow",
    val ttsEnabled: Boolean = false,
    val ttsSpeed: Double = 1.0,
    val ttsVoiceName: String? = null,
    val sttEnabled: Boolean = false,
    val sttLanguage: String = "en-US",
    val dyslexiaDisplayEnabled: Boolean = false,
    val highContrastEnabled: Boolean = false,
    val reducedMotionEnabled: Boolean = false,
    val updatedAt: String? = null,
)

@Serializable
data class ReadingPreferencesPatch(
    val fontFace: String? = null,
    val letterSpacing: String? = null,
    val wordSpacing: String? = null,
    val lineHeight: String? = null,
    val rulerEnabled: Boolean? = null,
    val rulerColor: String? = null,
    val ttsEnabled: Boolean? = null,
    val ttsSpeed: Double? = null,
    val ttsVoiceName: String? = null,
    val sttEnabled: Boolean? = null,
    val sttLanguage: String? = null,
    val dyslexiaDisplayEnabled: Boolean? = null,
    val highContrastEnabled: Boolean? = null,
    val reducedMotionEnabled: Boolean? = null,
)

@Serializable
data class CaptionRecord(
    val id: String,
    @SerialName("storage_object_id") val storageObjectId: String? = null,
    val lang: String,
    val status: String,
    @SerialName("has_low_confidence") val hasLowConfidence: Boolean? = null,
    @SerialName("confidence_avg") val confidenceAvg: Double? = null,
    val backend: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("reviewed_at") val reviewedAt: String? = null,
)

@Serializable
data class TranslateContentRequest(
    @SerialName("content_type") val contentType: String,
    @SerialName("content_id") val contentId: String,
    @SerialName("target_lang") val targetLang: String,
    val text: String,
)

@Serializable
data class TranslateContentResponse(
    val translated: String,
    @SerialName("source_lang") val sourceLang: String,
    val cached: Boolean,
)

@Serializable
data class TranslationCoverageLocale(
    @SerialName("target_locale") val targetLocale: String,
    val percent: Double,
)

@Serializable
data class TranslationCoverageResponse(
    val locales: List<TranslationCoverageLocale>? = null,
    @SerialName("target_locale") val targetLocale: String? = null,
    val percent: Double? = null,
)

@Serializable
data class PatchContentLocaleBody(
    @SerialName("contentLocale") val contentLocale: String? = null,
)

data class ImmersiveReaderCapabilities(
    val toolbarEnabled: Boolean = true,
    val readAloudEnabled: Boolean = true,
    val translationEnabled: Boolean = true,
    val captionsEnabled: Boolean = true,
    val preferencesEnabled: Boolean = true,
)