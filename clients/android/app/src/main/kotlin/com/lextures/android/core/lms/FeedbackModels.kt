package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class FeedbackContextPayload(
    val route: String,
    val locale: String? = null,
    val viewport: String? = null,
)

@Serializable
data class SubmitFeedbackRequest(
    val message: String,
    val source: String,
    @SerialName("app_version") val appVersion: String,
    val context: FeedbackContextPayload,
    val category: String? = null,
)

@Serializable
data class SubmitFeedbackResponse(
    val id: String,
    @SerialName("created_at") val createdAt: String? = null,
)
