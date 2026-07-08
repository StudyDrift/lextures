package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class CourseAccessibilityInfo(
    val altTextCoverage: AltTextCoverage = AltTextCoverage(),
    val hardBlockSave: Boolean = false,
)

@Serializable
data class AltTextCoverage(
    val withAlt: Int = 0,
    val total: Int = 0,
    val percent: Int = 100,
    val uncoveredItems: List<UncoveredAccessibilityItem> = emptyList(),
)

@Serializable
data class UncoveredAccessibilityItem(
    val itemId: String,
    val title: String,
    val kind: String,
    val withAlt: Int = 0,
    val total: Int = 0,
    val missing: Int = 0,
)

@Serializable
data class AltTextSuggestion(
    val suggestion: String = "",
    val confidence: Double = 0.0,
)

@Serializable
data class PatchItemMarkdownBody(
    val markdown: String,
)

@Serializable
data class SuggestAltTextBody(
    val imageUrl: String,
    val language: String = "",
)
