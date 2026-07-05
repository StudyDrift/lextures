package com.lextures.android.features.evaluations

import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L

@Composable fun evaluationsAnonymityBanner(): String = L.text(R.string.mobile_evaluations_anonymityBanner)
@Composable fun evaluationsSubmit(): String = L.text(R.string.mobile_evaluations_submit)
@Composable fun evaluationsSubmitting(): String = L.text(R.string.mobile_evaluations_submitting)
@Composable fun evaluationsSubmittedTitle(): String = L.text(R.string.mobile_evaluations_submittedTitle)
@Composable fun evaluationsSubmittedMessage(): String = L.text(R.string.mobile_evaluations_submittedMessage)
@Composable fun evaluationsNotOpenTitle(): String = L.text(R.string.mobile_evaluations_notOpenTitle)
@Composable fun evaluationsNotOpenMessage(): String = L.text(R.string.mobile_evaluations_notOpenMessage)
@Composable fun evaluationsValidationRequired(): String = L.text(R.string.mobile_evaluations_validationRequired)
@Composable fun evaluationsLoadError(): String = L.text(R.string.mobile_evaluations_loadError)
@Composable fun evaluationsSubmitError(): String = L.text(R.string.mobile_evaluations_submitError)
@Composable fun evaluationsDeadline(closesAt: String): String = L.format(R.string.mobile_evaluations_deadline, closesAt)
@Composable fun evaluationsResponses(): String = L.text(R.string.mobile_evaluations_responses)
@Composable fun evaluationsEnrolled(): String = L.text(R.string.mobile_evaluations_enrolled)
@Composable fun evaluationsCompletion(): String = L.text(R.string.mobile_evaluations_completion)
@Composable fun evaluationsThresholdTitle(): String = L.text(R.string.mobile_evaluations_thresholdTitle)
@Composable fun evaluationsThresholdMessage(): String = L.text(R.string.mobile_evaluations_thresholdMessage)
@Composable fun evaluationsNoResultsTitle(): String = L.text(R.string.mobile_evaluations_noResultsTitle)
@Composable fun evaluationsNoResultsMessage(): String = L.text(R.string.mobile_evaluations_noResultsMessage)
@Composable fun evaluationsResultsErrorTitle(): String = L.text(R.string.mobile_evaluations_resultsErrorTitle)
@Composable fun evaluationsResultsLoadError(): String = L.text(R.string.mobile_evaluations_resultsLoadError)
@Composable fun evaluationsNoOpenResponses(): String = L.text(R.string.mobile_evaluations_noOpenResponses)
@Composable fun evaluationsResponseCount(count: Int): String = L.format(R.string.mobile_evaluations_responseCount, count)
@Composable fun evaluationsAverageRating(value: String): String = L.format(R.string.mobile_evaluations_averageRating, value)
@Composable fun evaluationsWindowRange(opens: String, closes: String): String =
    L.format(R.string.mobile_evaluations_windowRange, opens, closes)
