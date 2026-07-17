package com.lextures.android.core.accessibility

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.platform.LocalContext

class AccessibilityPreferences(context: Context) {
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    var dyslexiaDisplayEnabled: Boolean
        get() = prefs.getBoolean(KEY_DYSLEXIA, false)
        set(value) {
            prefs.edit().putBoolean(KEY_DYSLEXIA, value).apply()
        }

    /** In-app reduce-motion override (AN.1). Combined with OS animator scale via LocalReduceMotion. */
    var reducedMotionEnabled: Boolean
        get() = prefs.getBoolean(KEY_REDUCED_MOTION, false)
        set(value) {
            prefs.edit().putBoolean(KEY_REDUCED_MOTION, value).apply()
        }

    var ttsSpeed: Float
        get() = prefs.getFloat(KEY_TTS_SPEED, 1f)
        set(value) {
            prefs.edit().putFloat(KEY_TTS_SPEED, value.coerceIn(0.5f, 2f)).apply()
        }

    fun reset() {
        prefs.edit()
            .putBoolean(KEY_DYSLEXIA, false)
            .putBoolean(KEY_REDUCED_MOTION, false)
            .putFloat(KEY_TTS_SPEED, 1f)
            .apply()
    }

    companion object {
        private const val PREFS_NAME = "lextures_a11y"
        private const val KEY_DYSLEXIA = "dyslexiaDisplay"
        private const val KEY_REDUCED_MOTION = "reducedMotion"
        private const val KEY_TTS_SPEED = "ttsSpeed"
    }
}

val LocalAccessibilityPreferences = staticCompositionLocalOf<AccessibilityPreferences> {
    error("AccessibilityPreferences not provided")
}

@Composable
fun rememberAccessibilityPreferences(): AccessibilityPreferences {
    val context = LocalContext.current
    return remember(context) { AccessibilityPreferences(context.applicationContext) }
}

/** Stateful dyslexia toggle for profile/settings screens. */
class AccessibilityPreferencesState(context: Context) {
    private val backing = AccessibilityPreferences(context)

    var dyslexiaDisplayEnabled by mutableStateOf(backing.dyslexiaDisplayEnabled)
        private set

    var reducedMotionEnabled by mutableStateOf(backing.reducedMotionEnabled)
        private set

    var ttsSpeed by mutableFloatStateOf(backing.ttsSpeed)
        private set

    fun updateDyslexiaDisplayEnabled(enabled: Boolean) {
        dyslexiaDisplayEnabled = enabled
        backing.dyslexiaDisplayEnabled = enabled
    }

    fun updateReducedMotionEnabled(enabled: Boolean) {
        reducedMotionEnabled = enabled
        backing.reducedMotionEnabled = enabled
    }

    fun updateTtsSpeed(speed: Float) {
        ttsSpeed = speed
        backing.ttsSpeed = speed
    }
}

@Composable
fun rememberAccessibilityPreferencesState(): AccessibilityPreferencesState {
    val context = LocalContext.current
    return remember(context) { AccessibilityPreferencesState(context.applicationContext) }
}
