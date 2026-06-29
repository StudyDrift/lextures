package com.lextures.android.core.i18n

import android.content.Context
import android.content.SharedPreferences
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.platform.LocalContext
import java.util.Locale

/** Device locale with optional in-app override (matches web `lextures.locale` storage key). */
@Stable
class LocalePreferences(context: Context) {
    private val prefs: SharedPreferences =
        context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    var localeTag by mutableStateOf(prefs.getString(KEY_LOCALE, SYSTEM_TAG) ?: SYSTEM_TAG)
        private set

    val usesSystemLocale: Boolean
        get() = localeTag == SYSTEM_TAG || localeTag.isBlank()

    val effectiveTag: String
        get() = if (usesSystemLocale) Locale.getDefault().toLanguageTag() else localeTag

    val effectiveLocale: Locale
        get() = Locale.forLanguageTag(effectiveTag)

    val isRTL: Boolean
        get() = isRTLLocale(effectiveTag)

    val acceptLanguageHeader: String
        get() = effectiveTag

    fun updateLocaleTag(tag: String) {
        localeTag = tag
        prefs.edit().putString(KEY_LOCALE, tag).apply()
        MobileLocale.acceptLanguage = acceptLanguageHeader
    }

    init {
        MobileLocale.acceptLanguage = acceptLanguageHeader
    }

    fun localizedContext(base: Context): Context {
        val locale = effectiveLocale
        val config = base.resources.configuration
        config.setLocale(locale)
        return base.createConfigurationContext(config)
    }

    companion object {
        const val PREFS_NAME = "lextures.locale.prefs"
        const val KEY_LOCALE = "lextures.locale"
        const val SYSTEM_TAG = "system"

        val localeOptions: List<LocaleOption> = listOf(
            LocaleOption(SYSTEM_TAG, "System default"),
            LocaleOption("en", "English"),
            LocaleOption("es", "Español"),
            LocaleOption("fr", "Français"),
            LocaleOption("ar", "العربية"),
            LocaleOption("en-XA", "Pseudo (en-XA)"),
        )

        private val rtlLocales = setOf("ar", "he", "fa", "ur", "ps")

        fun isRTLLocale(tag: String): Boolean {
            val primary = tag.substringBefore('-').substringBefore('_').lowercase(Locale.ROOT)
            return primary in rtlLocales
        }

        fun resolveResourceLanguage(tag: String): String {
            val primary = tag.substringBefore('-').substringBefore('_').lowercase(Locale.ROOT)
            return when (primary) {
                "es", "fr", "ar" -> primary
                else -> if (tag == "en-XA") "en-XA" else "en"
            }
        }
    }
}

data class LocaleOption(
    val tag: String,
    val label: String,
)

/** Global Accept-Language value for API requests (updated by [LocalePreferences]). */
object MobileLocale {
    @Volatile
    var acceptLanguage: String = Locale.getDefault().toLanguageTag()
}

val LocalLocalePreferences = staticCompositionLocalOf<LocalePreferences> {
    error("LocalePreferences not provided")
}

/** Resolves a string resource in the active app locale. */
object L {
    @Composable
    fun text(@androidx.annotation.StringRes id: Int): String {
        val context = LocalContext.current
        val prefs = LocalLocalePreferences.current
        return prefs.localizedContext(context).getString(id)
    }

    @Composable
    fun plural(@androidx.annotation.PluralsRes id: Int, count: Int): String {
        val context = LocalContext.current
        val prefs = LocalLocalePreferences.current
        return prefs.localizedContext(context).resources.getQuantityString(id, count, count)
    }

    fun text(context: Context, prefs: LocalePreferences, @androidx.annotation.StringRes id: Int): String =
        prefs.localizedContext(context).getString(id)

    fun plural(context: Context, prefs: LocalePreferences, @androidx.annotation.PluralsRes id: Int, count: Int): String =
        prefs.localizedContext(context).resources.getQuantityString(id, count, count)
}

@Composable
fun rememberLocalePreferences(): LocalePreferences {
    val context = LocalContext.current
    return remember { LocalePreferences(context.applicationContext) }
}
