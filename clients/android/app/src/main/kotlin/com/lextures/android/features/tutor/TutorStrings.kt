package com.lextures.android.features.tutor

import android.content.Context
import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.network.ApiError

@Composable fun tutorTitle(): String = L.text(R.string.mobile_tutor_title)
@Composable fun tutorAskAi(): String = L.text(R.string.mobile_tutor_askAi)
@Composable fun tutorClose(): String = L.text(R.string.mobile_tutor_close)
@Composable fun tutorSend(): String = L.text(R.string.mobile_tutor_send)
@Composable fun tutorStop(): String = L.text(R.string.mobile_tutor_stop)
@Composable fun tutorPlaceholder(): String = L.text(R.string.mobile_tutor_placeholder)
@Composable fun tutorNewConversation(): String = L.text(R.string.mobile_tutor_newConversation)
@Composable fun tutorReset(): String = L.text(R.string.mobile_tutor_reset)
@Composable fun tutorDisclosureTitle(): String = L.text(R.string.mobile_tutor_disclosureTitle)
@Composable fun tutorDisclosureBody(): String = L.text(R.string.mobile_tutor_disclosureBody)
@Composable fun tutorDisclosureAccept(): String = L.text(R.string.mobile_tutor_disclosureAccept)
@Composable fun tutorLoadError(): String = L.text(R.string.mobile_tutor_loadError)
@Composable fun tutorSendError(): String = L.text(R.string.mobile_tutor_sendError)
@Composable fun tutorAskAiCourseHint(): String = L.text(R.string.mobile_tutor_askAiCourseHint)
@Composable fun tutorAskAiNotebookHint(): String = L.text(R.string.mobile_tutor_askAiNotebookHint)
@Composable fun tutorSource(): String = L.text(R.string.mobile_tutor_source)
@Composable fun tutorYou(): String = L.text(R.string.mobile_tutor_you)
@Composable fun tutorAssistant(): String = L.text(R.string.mobile_tutor_assistant)
@Composable fun tutorTokenBudget(used: Int, limit: Int): String = L.format(R.string.mobile_tutor_tokenBudget, used, limit)

fun tutorOffline(context: Context, prefs: LocalePreferences): String =
    L.text(context, prefs, R.string.mobile_tutor_offline)

fun tutorNoNotebooks(context: Context, prefs: LocalePreferences): String =
    L.text(context, prefs, R.string.mobile_tutor_noNotebooks)

fun tutorMapError(context: Context, prefs: LocalePreferences, error: Throwable): String = when (error) {
    is ApiError.HttpStatus -> when (error.message) {
        "BUDGET_EXCEEDED" -> L.text(context, prefs, R.string.mobile_tutor_budgetExceeded)
        "FORBIDDEN" -> L.text(context, prefs, R.string.mobile_tutor_disabled)
        "UNAVAILABLE" -> L.text(context, prefs, R.string.mobile_tutor_unavailable)
        else -> error.message?.takeIf { it.isNotBlank() }
            ?: L.text(context, prefs, R.string.mobile_tutor_sendError)
    }
    else -> (error as? java.io.IOException)?.message?.takeIf { it.isNotBlank() }
        ?: L.text(context, prefs, R.string.mobile_tutor_loadError)
}