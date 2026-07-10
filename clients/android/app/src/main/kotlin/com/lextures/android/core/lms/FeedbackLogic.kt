package com.lextures.android.core.lms

import com.lextures.android.BuildConfig
import com.lextures.android.R
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError

/** Product feedback helpers (FB3). */
object FeedbackLogic {
    const val MAX_MESSAGE_LEN = 5000
    const val SOURCE = "android"
    val CATEGORIES = listOf("bug", "idea", "question", "praise", "other")

    fun feedbackEnabled(features: MobilePlatformFeatures): Boolean = features.ffFeedback

    fun messageValid(message: String): Boolean = message.trim().isNotEmpty()

    fun trimmedMessageLength(message: String): Int = message.trim().length

    fun appVersion(): String = BuildConfig.VERSION_NAME

    fun buildSubmitRequest(
        message: String,
        category: String,
        route: String,
        locale: String?,
        viewport: String?,
    ): SubmitFeedbackRequest {
        val trimmedCategory = category.trim()
        return SubmitFeedbackRequest(
            message = message.trim(),
            source = SOURCE,
            appVersion = appVersion(),
            context = FeedbackContextPayload(
                route = route,
                locale = locale?.takeIf { it.isNotBlank() },
                viewport = viewport?.takeIf { it.isNotBlank() },
            ),
            category = trimmedCategory.takeIf { it.isNotEmpty() },
        )
    }

    sealed class SubmitOutcome {
        data object Success : SubmitOutcome()
        data object RateLimited : SubmitOutcome()
        data object Offline : SubmitOutcome()
        data object Error : SubmitOutcome()
    }

    fun mapSubmitError(error: Throwable, isOnline: Boolean): SubmitOutcome {
        if (!isOnline) return SubmitOutcome.Offline
        if (error is ApiError.HttpStatus && error.code == 429) return SubmitOutcome.RateLimited
        if (error is ApiError.Transport) return SubmitOutcome.Offline
        return SubmitOutcome.Error
    }

    fun errorMessageRes(outcome: SubmitOutcome): Int = when (outcome) {
        SubmitOutcome.Success -> R.string.mobile_feedback_success
        SubmitOutcome.RateLimited -> R.string.mobile_feedback_rateLimited
        SubmitOutcome.Offline -> R.string.mobile_feedback_offline
        SubmitOutcome.Error -> R.string.mobile_feedback_error
    }

    fun categoryLabelRes(category: String): Int = when (category.trim()) {
        "bug" -> R.string.mobile_feedback_category_bug
        "idea" -> R.string.mobile_feedback_category_idea
        "question" -> R.string.mobile_feedback_category_question
        "praise" -> R.string.mobile_feedback_category_praise
        "other" -> R.string.mobile_feedback_category_other
        else -> R.string.mobile_feedback_category_none
    }
}
