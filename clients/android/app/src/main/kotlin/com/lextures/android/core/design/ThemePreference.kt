package com.lextures.android.core.design

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/** App-wide appearance override. */
enum class ThemeAppearance(val storageValue: String) {
    SYSTEM("system"),
    LIGHT("light"),
    DARK("dark"),
    ;

    companion object {
        fun fromStorage(value: String?): ThemeAppearance =
            entries.firstOrNull { it.storageValue == value } ?: SYSTEM
    }
}

/**
 * Device-only theme override (system / light / dark), persisted in SharedPreferences.
 *
 * A process singleton so the settings picker and [LexturesTheme] / [isDarkTheme]
 * observe the same Compose state and update live (FR-2 / FR-5: device-only prefs).
 */
class ThemePreference private constructor(context: Context) {
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    var appearance by mutableStateOf(ThemeAppearance.fromStorage(prefs.getString(KEY_APPEARANCE, null)))
        private set

    fun update(value: ThemeAppearance) {
        appearance = value
        prefs.edit().putString(KEY_APPEARANCE, value.storageValue).apply()
    }

    companion object {
        private const val PREFS_NAME = "lextures_theme"
        private const val KEY_APPEARANCE = "appearance"

        @Volatile
        private var instance: ThemePreference? = null

        fun get(context: Context): ThemePreference =
            instance ?: synchronized(this) {
                instance ?: ThemePreference(context.applicationContext).also { instance = it }
            }
    }
}
