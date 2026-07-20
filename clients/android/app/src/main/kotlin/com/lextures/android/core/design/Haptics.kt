package com.lextures.android.core.design

import android.content.Context
import android.os.Build
import android.view.HapticFeedbackConstants
import android.view.View

/**
 * AN.6 — Standardized haptics mapping for control interactions (FR-5).
 *
 * Views perform feedback via [View.performHapticFeedback]; the OS haptics
 * setting is honored automatically. Never gate the control action on this (FR-9).
 */
object Haptics {
    enum class Kind {
        Tap,
        Selection,
        Success,
        Error,
    }

    /** System constant name for tests / logging. */
    fun systemName(kind: Kind): String = when (kind) {
        Kind.Tap -> "CONTEXT_CLICK"
        Kind.Selection -> "CLOCK_TICK"
        Kind.Success -> "CONFIRM"
        Kind.Error -> "REJECT"
    }

    fun feedbackConstant(kind: Kind): Int = when (kind) {
        Kind.Tap -> HapticFeedbackConstants.CONTEXT_CLICK
        Kind.Selection -> HapticFeedbackConstants.CLOCK_TICK
        Kind.Success -> if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            HapticFeedbackConstants.CONFIRM
        } else {
            HapticFeedbackConstants.CONTEXT_CLICK
        }
        Kind.Error -> if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            HapticFeedbackConstants.REJECT
        } else {
            HapticFeedbackConstants.LONG_PRESS
        }
    }

    fun trigger(view: View?, kind: Kind, enabled: Boolean = true) {
        if (!enabled || view == null) return
        view.performHapticFeedback(feedbackConstant(kind))
    }

    /** Pure helper for unit tests without a View. */
    fun shouldFire(enabled: Boolean, systemHapticsOff: Boolean = false): Boolean {
        if (!enabled) return false
        if (systemHapticsOff) return false
        return true
    }

    @Suppress("UNUSED_PARAMETER")
    fun isSystemHapticsOff(context: Context): Boolean {
        // View.performHapticFeedback already respects the user setting; this
        // helper exists for testability / explicit checks.
        return false
    }
}
