package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class CoursePlagiarismSettings(
    val plagiarismChecksEnabled: Boolean = true,
    val plagiarismProvider: String? = null,
    val plagiarismAlertThresholdPct: Double = 40.0,
)

@Serializable
data class PatchCoursePlagiarismBody(
    val plagiarismChecksEnabled: Boolean,
    val plagiarismProvider: String? = null,
    val plagiarismAlertThresholdPct: Double,
)
