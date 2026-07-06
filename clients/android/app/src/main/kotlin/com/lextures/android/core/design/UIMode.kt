package com.lextures.android.core.design

import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.i18n.LocalLocalePreferences
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.accessibility.AccessibilitySupport
import com.lextures.android.core.i18n.L
import com.lextures.android.core.navigation.MobileRoleContext
import com.lextures.android.core.navigation.MoreDestination
import com.lextures.android.core.navigation.RootDestination

/** Age-appropriate UI mode (plan 13.11 / M10.4). */
enum class UIMode {
    Standard,
    Elementary,
    K2,
    ;

    val isYoung: Boolean get() = this != Standard

    val minimumTapTarget: Dp
        get() = when (this) {
            K2 -> 48.dp
            Elementary, Standard -> 44.dp
        }

    val baseBodySp
        get() = when (this) {
            K2 -> 24.sp
            Elementary -> 18.sp
            Standard -> 17.sp
        }

    val drawerIconDp: Dp
        get() = when (this) {
            K2 -> 28.dp
            Elementary -> 22.dp
            Standard -> 20.dp
        }

    val drawerRowVerticalPadding: Dp
        get() = when (this) {
            K2 -> 16.dp
            Elementary -> 13.dp
            Standard -> 11.dp
        }

    val choiceButtonMinHeight: Dp
        get() = when (this) {
            K2 -> 56.dp
            Elementary -> 48.dp
            Standard -> 44.dp
        }
}

/** User-facing override stored on device. [Auto] follows the server-derived mode. */
enum class UIModePreference(val labelRes: Int) {
    Auto(R.string.mobile_uiMode_preference_auto),
    K2(R.string.mobile_uiMode_preference_k2),
    Elementary(R.string.mobile_uiMode_preference_elementary),
    Standard(R.string.mobile_uiMode_preference_standard),
    ;

    val resolvedMode: UIMode?
        get() = when (this) {
            Auto -> null
            K2 -> UIMode.K2
            Elementary -> UIMode.Elementary
            Standard -> UIMode.Standard
        }

    companion object {
        fun fromRaw(raw: String?): UIModePreference =
            entries.firstOrNull { it.name.equals(raw, ignoreCase = true) } ?: Auto
    }
}

/** Derives effective UI mode from grade level and overrides (mirrors server `readingprefs`). */
object UIModeLogic {
    fun gradeToUIMode(gradeLevel: String?): UIMode {
        return when (gradeLevel) {
            "K", "1", "2" -> UIMode.K2
            "3", "4", "5" -> UIMode.Elementary
            else -> UIMode.Standard
        }
    }

    fun parseMode(raw: String?): UIMode? = when (raw) {
        "k2" -> UIMode.K2
        "elementary" -> UIMode.Elementary
        "standard" -> UIMode.Standard
        else -> null
    }

    fun effectiveMode(
        featureEnabled: Boolean,
        roleContext: MobileRoleContext,
        serverOverride: String?,
        serverEffective: String?,
        localPreference: UIModePreference,
    ): UIMode {
        if (!featureEnabled || roleContext != MobileRoleContext.Learning) return UIMode.Standard
        parseMode(serverOverride)?.let { return it }
        localPreference.resolvedMode?.let { return it }
        parseMode(serverEffective)?.let { return it }
        return UIMode.Standard
    }

    @Composable
    fun drawerLabel(destination: RootDestination, mode: UIMode): String {
        if (mode == UIMode.K2) {
            k2DrawerLabelRes(destination)?.let { return L.text(it) }
        }
        if (mode == UIMode.Elementary) {
            elementaryDrawerLabelRes(destination)?.let { return L.text(it) }
        }
        return resolveStringRes(destination.labelRes)
    }

    @Composable
    fun moreLabel(destination: MoreDestination, mode: UIMode): String {
        if (mode == UIMode.K2) {
            k2MoreLabelRes(destination)?.let { return L.text(it) }
        }
        return resolveStringRes(destination.labelRes)
    }

    @Composable
    private fun resolveStringRes(name: String): String {
        val context = LocalContext.current
        val prefs = LocalLocalePreferences.current
        val id = context.resources.getIdentifier(name, "string", context.packageName)
        return if (id != 0) prefs.localizedContext(context).getString(id) else name
    }

    private fun k2DrawerLabelRes(destination: RootDestination): Int? = when (destination) {
        RootDestination.Dashboard -> R.string.mobile_uiMode_young_dashboard
        RootDestination.Courses -> R.string.mobile_uiMode_young_courses
        RootDestination.Todos -> R.string.mobile_uiMode_young_todos
        RootDestination.Calendar -> R.string.mobile_uiMode_young_calendar
        RootDestination.Inbox -> R.string.mobile_uiMode_young_inbox
        RootDestination.Settings -> R.string.mobile_uiMode_young_settings
        else -> null
    }

    private fun elementaryDrawerLabelRes(destination: RootDestination): Int? = when (destination) {
        RootDestination.Dashboard -> R.string.mobile_uiMode_elementary_dashboard
        RootDestination.Courses -> R.string.mobile_uiMode_elementary_courses
        RootDestination.Todos -> R.string.mobile_uiMode_elementary_todos
        else -> null
    }

    private fun k2MoreLabelRes(destination: MoreDestination): Int? = when (destination) {
        MoreDestination.Reading -> R.string.mobile_uiMode_young_reading
        MoreDestination.Settings -> R.string.mobile_uiMode_young_settings
        else -> null
    }
}

class UIModeStore(context: android.content.Context) {
    private val prefs = context.getSharedPreferences(PREFS_NAME, android.content.Context.MODE_PRIVATE)

    var featureEnabled by mutableStateOf(false)
        private set
    var serverEffectiveMode by mutableStateOf(UIMode.Standard)
        private set
    var serverOverrideMode by mutableStateOf<UIMode?>(null)
        private set
    var localPreference by mutableStateOf(loadPreference())
        private set

    val hasAdminOverride: Boolean get() = serverOverrideMode != null

    var lastRoleContext by mutableStateOf(MobileRoleContext.Learning)
        private set

    val resolvedMode: UIMode
        get() = effectiveMode(lastRoleContext)

    fun effectiveMode(roleContext: MobileRoleContext): UIMode {
        lastRoleContext = roleContext
        return UIModeLogic.effectiveMode(
            featureEnabled = featureEnabled,
            roleContext = roleContext,
            serverOverride = serverOverrideMode?.name?.lowercase(),
            serverEffective = serverEffectiveMode.name.lowercase(),
            localPreference = localPreference,
        )
    }

    fun updatePlatform(featureEnabled: Boolean) {
        this.featureEnabled = featureEnabled
    }

    fun updateLocalPreference(preference: UIModePreference) {
        localPreference = preference
        prefs.edit().putString(KEY_PREFERENCE, preference.name).apply()
    }

    fun applyReadingPreferences(
        effectiveUiMode: String?,
        uiModeOverride: String?,
        featureEnabled: Boolean,
    ) {
        this.featureEnabled = featureEnabled
        UIModeLogic.parseMode(effectiveUiMode)?.let {
            serverEffectiveMode = it
            prefs.edit().putString(KEY_SERVER_EFFECTIVE, it.name.lowercase()).apply()
        }
        if (uiModeOverride != null) {
            UIModeLogic.parseMode(uiModeOverride)?.let {
                serverOverrideMode = it
                prefs.edit().putString(KEY_SERVER_OVERRIDE, it.name.lowercase()).apply()
            }
        } else {
            serverOverrideMode = null
            prefs.edit().remove(KEY_SERVER_OVERRIDE).apply()
        }
    }

    private fun loadPreference(): UIModePreference {
        val stored = prefs.getString(KEY_PREFERENCE, UIModePreference.Auto.name)
        return UIModePreference.fromRaw(stored)
    }

    init {
        serverEffectiveMode = UIModeLogic.parseMode(prefs.getString(KEY_SERVER_EFFECTIVE, null)) ?: UIMode.Standard
        serverOverrideMode = UIModeLogic.parseMode(prefs.getString(KEY_SERVER_OVERRIDE, null))
    }

    companion object {
        private const val PREFS_NAME = "lextures_ui_mode"
        private const val KEY_PREFERENCE = "preference"
        private const val KEY_SERVER_EFFECTIVE = "serverEffective"
        private const val KEY_SERVER_OVERRIDE = "serverOverride"
    }
}

val LocalUIModeStore = staticCompositionLocalOf<UIModeStore> {
    error("UIModeStore not provided")
}

@Composable
fun rememberUIModeStore(): UIModeStore {
    val context = LocalContext.current
    return remember(context) { UIModeStore(context.applicationContext) }
}
