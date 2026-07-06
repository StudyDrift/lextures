package com.lextures.android.features.quiz

import android.app.Activity
import android.view.WindowManager
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.DefaultLifecycleObserver
import androidx.lifecycle.LifecycleOwner

/**
 * Platform lockdown for kiosk-mode quizzes: screen pinning / lock task, FLAG_SECURE,
 * and focus-loss reporting (M4.2).
 */
class LockdownController(
    private val activity: Activity,
) : DefaultLifecycleObserver {
    var focusLossBanner by mutableStateOf<String?>(null)
        private set
    var platformWarning by mutableStateOf<String?>(null)
        private set
    var isActive by mutableStateOf(false)
        private set

    private var onFocusLoss: ((String) -> Unit)? = null
    private var skipNextResumeFocusLoss = true

    fun activate(onFocusLoss: (String) -> Unit) {
        if (isActive) return
        isActive = true
        this.onFocusLoss = onFocusLoss
        focusLossBanner = null
        skipNextResumeFocusLoss = true

        activity.window.addFlags(WindowManager.LayoutParams.FLAG_SECURE)
        activity.window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)

        runCatching { activity.startLockTask() }
        if (!activity.isInLockTaskMode) {
            platformWarning = activity.getString(com.lextures.android.R.string.mobile_quiz_lockdown_pinningWarning)
        }
    }

    fun deactivate() {
        if (!isActive) return
        isActive = false
        onFocusLoss = null
        focusLossBanner = null
        platformWarning = null
        skipNextResumeFocusLoss = true

        activity.window.clearFlags(WindowManager.LayoutParams.FLAG_SECURE)
        activity.window.clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        runCatching { activity.stopLockTask() }
    }

    override fun onStop(owner: LifecycleOwner) {
        if (!isActive) return
        reportIntegrityEvent("app_background", showBanner = false)
    }

    override fun onStart(owner: LifecycleOwner) {
        if (!isActive) return
        if (skipNextResumeFocusLoss) {
            skipNextResumeFocusLoss = false
            return
        }
        focusLossBanner = activity.getString(com.lextures.android.R.string.mobile_quiz_lockdown_focusLossBanner)
    }

    private fun reportIntegrityEvent(eventType: String, showBanner: Boolean = true) {
        onFocusLoss?.invoke(eventType)
        if (showBanner) {
            focusLossBanner = activity.getString(com.lextures.android.R.string.mobile_quiz_lockdown_focusLossBanner)
        }
    }
}
