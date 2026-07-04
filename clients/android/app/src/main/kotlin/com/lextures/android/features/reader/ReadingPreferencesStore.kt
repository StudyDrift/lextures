package com.lextures.android.features.reader

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.accessibility.AccessibilityPreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ReadingPreferencesPatch
import com.lextures.android.core.lms.ReadingPreferencesRow
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

class ReadingPreferencesStore(context: Context) {
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val accessibility = AccessibilityPreferences(context)

    var row by mutableStateOf(loadLocal())
        private set
    var loading by mutableStateOf(false)
        private set
    var serverSyncEnabled by mutableStateOf(false)
        private set

    val usesDyslexiaFont: Boolean
        get() = row.dyslexiaDisplayEnabled || row.fontFace == "open-dyslexic"

    suspend fun loadFromServer(accessToken: String, apiEnabled: Boolean) {
        serverSyncEnabled = apiEnabled
        if (!apiEnabled) return
        loading = true
        try {
            row = LmsApi.fetchReadingPreferences(accessToken)
            persistLocal()
            syncAccessibility()
        } catch (_: Exception) {
            // Keep local prefs.
        } finally {
            loading = false
        }
    }

    suspend fun update(patch: ReadingPreferencesPatch, accessToken: String?) {
        row = row.copy(
            fontFace = patch.fontFace ?: row.fontFace,
            letterSpacing = patch.letterSpacing ?: row.letterSpacing,
            wordSpacing = patch.wordSpacing ?: row.wordSpacing,
            lineHeight = patch.lineHeight ?: row.lineHeight,
            ttsSpeed = patch.ttsSpeed ?: row.ttsSpeed,
            dyslexiaDisplayEnabled = patch.dyslexiaDisplayEnabled ?: row.dyslexiaDisplayEnabled,
        )
        if (patch.fontFace != null) {
            row = row.copy(dyslexiaDisplayEnabled = ReaderLogic.dyslexiaFromFontFace(patch.fontFace))
        }
        if (patch.dyslexiaDisplayEnabled == true) {
            row = row.copy(fontFace = ReaderLogic.fontFaceFromDyslexia(true, row.fontFace))
        }
        persistLocal()
        syncAccessibility()
        if (serverSyncEnabled && accessToken != null) {
            runCatching { LmsApi.patchReadingPreferences(patch, accessToken) }
        }
    }

    private fun loadLocal(): ReadingPreferencesRow = ReadingPreferencesRow(
        fontFace = prefs.getString(KEY_FONT_FACE, "default") ?: "default",
        letterSpacing = prefs.getString(KEY_LETTER_SPACING, "normal") ?: "normal",
        wordSpacing = prefs.getString(KEY_WORD_SPACING, "normal") ?: "normal",
        lineHeight = prefs.getString(KEY_LINE_HEIGHT, "normal") ?: "normal",
        ttsSpeed = prefs.getFloat(KEY_TTS_SPEED, 1f).toDouble(),
        dyslexiaDisplayEnabled = prefs.getBoolean(KEY_DYSLEXIA, false),
    )

    private fun persistLocal() {
        prefs.edit()
            .putString(KEY_FONT_FACE, row.fontFace)
            .putString(KEY_LETTER_SPACING, row.letterSpacing)
            .putString(KEY_WORD_SPACING, row.wordSpacing)
            .putString(KEY_LINE_HEIGHT, row.lineHeight)
            .putFloat(KEY_TTS_SPEED, row.ttsSpeed.toFloat())
            .putBoolean(KEY_DYSLEXIA, row.dyslexiaDisplayEnabled)
            .apply()
    }

    private fun syncAccessibility() {
        accessibility.dyslexiaDisplayEnabled = usesDyslexiaFont
        accessibility.ttsSpeed = row.ttsSpeed.toFloat()
    }

    companion object {
        private const val PREFS_NAME = "lextures_reader_prefs"
        private const val KEY_FONT_FACE = "fontFace"
        private const val KEY_LETTER_SPACING = "letterSpacing"
        private const val KEY_WORD_SPACING = "wordSpacing"
        private const val KEY_LINE_HEIGHT = "lineHeight"
        private const val KEY_TTS_SPEED = "ttsSpeed"
        private const val KEY_DYSLEXIA = "dyslexia"
    }
}

val LocalReadingPreferencesStore = staticCompositionLocalOf<ReadingPreferencesStore> {
    error("ReadingPreferencesStore not provided")
}

@Composable
fun rememberReadingPreferencesStore(): ReadingPreferencesStore {
    val context = LocalContext.current
    return remember(context) { ReadingPreferencesStore(context.applicationContext) }
}

fun ReadingPreferencesStore.updateAsync(
    scope: CoroutineScope,
    patch: ReadingPreferencesPatch,
    accessToken: String?,
) {
    scope.launch { update(patch, accessToken) }
}