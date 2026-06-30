package com.lextures.android.features.courses

import androidx.annotation.StringRes
import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L

@Composable fun moduleEmptyLabel(): String = L.text(R.string.mobile_modules_empty)

@Composable fun moduleEmptyCourseTitle(): String = L.text(R.string.mobile_modules_emptyCourse)

@Composable fun moduleEmptyCourseHint(): String = L.text(R.string.mobile_modules_emptyCourseHint)

@Composable fun moduleCompleteLabel(): String = L.text(R.string.mobile_modules_complete)

@Composable fun moduleMarkDoneLabel(): String = L.text(R.string.mobile_modules_markDone)

@Composable fun moduleMarkingDoneLabel(): String = L.text(R.string.mobile_modules_markingDone)

@Composable fun moduleLoadErrorLabel(): String = L.text(R.string.mobile_modules_loadError)

@Composable fun moduleLockedDefaultLabel(): String = L.text(R.string.mobile_modules_lockedDefault)

@Composable fun moduleOpenExternalLabel(): String = L.text(R.string.mobile_modules_openExternal)

@Composable fun moduleNoLinkLabel(): String = L.text(R.string.mobile_modules_noLink)

@Composable
fun modulePlaceholderLabel(@StringRes id: Int): String = L.text(id)

@StringRes
fun modulePlaceholderRes(messageKey: String): Int = when (messageKey) {
    "mobile.modules.placeholder.quiz" -> R.string.mobile_modules_placeholder_quiz
    "mobile.modules.placeholder.interactive" -> R.string.mobile_modules_placeholder_interactive
    "mobile.modules.placeholder.file" -> R.string.mobile_modules_placeholder_file
    else -> R.string.mobile_modules_placeholder_unsupported
}
