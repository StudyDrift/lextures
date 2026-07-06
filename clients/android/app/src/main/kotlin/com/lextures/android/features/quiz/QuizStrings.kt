package com.lextures.android.features.quiz

import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.lextures.android.R
import com.lextures.android.core.i18n.L

@Composable fun quizLabel(): String = L.text(R.string.mobile_quiz_label)
@Composable fun quizStartLabel(): String = L.text(R.string.mobile_quiz_startAttempt)
@Composable fun quizPreviewLabel(): String = L.text(R.string.mobile_quiz_previewQuiz)
@Composable fun quizPreviewNoteLabel(): String = L.text(R.string.mobile_quiz_previewNote)
@Composable fun quizPreviewEmptyLabel(): String = L.text(R.string.mobile_quiz_previewEmpty)
@Composable fun quizPreviousLabel(): String = L.text(R.string.mobile_quiz_previous)
@Composable fun quizNextLabel(): String = L.text(R.string.mobile_quiz_next)
@Composable fun quizSubmitLabel(): String = L.text(R.string.mobile_quiz_submit)
@Composable fun quizDoneLabel(): String = L.text(R.string.mobile_quiz_done)
@Composable fun quizSavedLabel(): String = L.text(R.string.mobile_quiz_saved)
@Composable fun quizSaveFailedLabel(): String = L.text(R.string.mobile_quiz_saveFailed)
@Composable fun quizNotYetSavedLabel(): String = L.text(R.string.mobile_quiz_notYetSaved)
@Composable fun quizOpenOnWebLabel(): String = L.text(R.string.mobile_quiz_openOnWeb)
@Composable fun quizOpenOnWebHintLabel(): String = L.text(R.string.mobile_quiz_openOnWebHint)
@Composable fun quizShortAnswerPlaceholder(): String = L.text(R.string.mobile_quiz_shortAnswerPlaceholder)
@Composable fun quizPreviousAttemptsLabel(): String = L.text(R.string.mobile_quiz_previousAttempts)
@Composable fun quizAdaptiveWebOnlyLabel(): String = L.text(R.string.mobile_quiz_adaptiveWebOnly)
@Composable fun quizNoAttemptsLabel(): String = L.text(R.string.mobile_quiz_noAttemptsRemaining)
@Composable fun quizSubmittedLabel(): String = L.text(R.string.mobile_quiz_submitted)
@Composable fun quizYourScoreLabel(): String = L.text(R.string.mobile_quiz_yourScore)
@Composable fun quizPendingReviewLabel(): String = L.text(R.string.mobile_quiz_pendingReview)

@Composable fun quizLockdownConfirmLabel(): String = L.text(R.string.mobile_quiz_lockdown_confirm)
@Composable fun quizLockdownCancelLabel(): String = L.text(R.string.mobile_quiz_lockdown_cancel)
@Composable fun quizLockdownKioskTitle(): String = L.text(R.string.mobile_quiz_lockdown_kioskTitle)
@Composable fun quizLockdownOneAtATimeTitle(): String = L.text(R.string.mobile_quiz_lockdown_oneAtATimeTitle)
@Composable fun quizLockdownKioskBulletBack(): String = L.text(R.string.mobile_quiz_lockdown_kioskBulletBack)
@Composable fun quizLockdownKioskBulletHints(): String = L.text(R.string.mobile_quiz_lockdown_kioskBulletHints)
@Composable fun quizLockdownKioskBulletFocus(): String = L.text(R.string.mobile_quiz_lockdown_kioskBulletFocus)
@Composable fun quizLockdownOneAtATimeBulletBack(): String = L.text(R.string.mobile_quiz_lockdown_oneAtATimeBulletBack)
@Composable fun quizLockdownOneAtATimeBulletHints(): String = L.text(R.string.mobile_quiz_lockdown_oneAtATimeBulletHints)
@Composable fun quizLockdownKioskBanner(): String = L.text(R.string.mobile_quiz_lockdown_kioskBanner)
@Composable fun quizLockdownFocusLossBanner(): String = L.text(R.string.mobile_quiz_lockdown_focusLossBanner)

@Composable fun quizCodeRunLabel(): String = L.text(R.string.mobile_quiz_code_run)
@Composable fun quizCodeRunningLabel(): String = L.text(R.string.mobile_quiz_code_running)
@Composable fun quizCodeRunFailedLabel(): String = L.text(R.string.mobile_quiz_code_runFailed)
@Composable fun quizCodeOversizedTitle(): String = L.text(R.string.mobile_quiz_code_oversizedTitle)
@Composable fun quizCodeOversizedHint(): String = L.text(R.string.mobile_quiz_code_oversizedHint)
@Composable fun quizCodeStatusPassLabel(): String = L.text(R.string.mobile_quiz_code_statusPass)
@Composable fun quizCodeStatusFailLabel(): String = L.text(R.string.mobile_quiz_code_statusFail)
@Composable fun quizCodeStatusTleLabel(): String = L.text(R.string.mobile_quiz_code_statusTle)
@Composable fun quizCodeStatusMleLabel(): String = L.text(R.string.mobile_quiz_code_statusMle)
@Composable fun quizCodeStatusReLabel(): String = L.text(R.string.mobile_quiz_code_statusRe)
@Composable fun quizCodeStatusCeLabel(): String = L.text(R.string.mobile_quiz_code_statusCe)

@Composable fun quizAttemptNumberLabel(number: Int): String =
    L.format(R.string.mobile_quiz_attemptNumber, number)

@Composable fun quizProgressLabel(current: Int, total: Int): String =
    L.format(R.string.mobile_quiz_progress, current, total)

@Composable fun quizQuestionNumberLabel(number: Int): String =
    L.format(R.string.mobile_quiz_questionNumber, number)

@Composable fun quizTimerLabel(value: String): String =
    L.format(R.string.mobile_quiz_timeRemainingA11y, value)
