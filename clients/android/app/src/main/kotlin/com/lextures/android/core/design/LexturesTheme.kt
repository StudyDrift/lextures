package com.lextures.android.core.design

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

/** Visual tokens aligned with the web auth shell (`lex-auth-scene`, teal primary). */
object LexturesColors {
    val Primary = Color(0xFF0F766E)
    val PrimaryMuted = Color(0xFF135854)
    val SceneBackground = Color(0xFFFAF9F6)
    val CardBackground = Color.White
    val FieldBorder = Color(0xFFE5E2DE)
    val TextPrimary = Color(0xFF1C1917)
    val TextSecondary = Color(0xFF6B645D)
    val Error = Color(0xFFE0316E)
    val StrengthStrong = Color(0xFF0C9857)

    val SceneBackgroundDark = Color(0xFF171717)
    val CardBackgroundDark = Color(0xFF171717)
    val FieldBorderDark = Color(0xFF4F4F4F)
    val TextPrimaryDark = Color(0xFFFAFAFA)
    val TextSecondaryDark = Color(0xFFA3A3A3)
}

@Composable
fun isDarkTheme(): Boolean = isSystemInDarkTheme()

@Composable
fun sceneBackground(): Color =
    if (isDarkTheme()) LexturesColors.SceneBackgroundDark else LexturesColors.SceneBackground

@Composable
fun cardBackground(): Color =
    if (isDarkTheme()) LexturesColors.CardBackgroundDark else LexturesColors.CardBackground

@Composable
fun fieldBorder(): Color =
    if (isDarkTheme()) LexturesColors.FieldBorderDark else LexturesColors.FieldBorder

@Composable
fun textPrimary(): Color =
    if (isDarkTheme()) LexturesColors.TextPrimaryDark else LexturesColors.TextPrimary

@Composable
fun textSecondary(): Color =
    if (isDarkTheme()) LexturesColors.TextSecondaryDark else LexturesColors.TextSecondary

@Composable
fun LexturesTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val scheme = if (dark) {
        darkColorScheme(
            primary = LexturesColors.Primary,
            onPrimary = Color.White,
            background = LexturesColors.SceneBackgroundDark,
            surface = LexturesColors.CardBackgroundDark,
            onBackground = LexturesColors.TextPrimaryDark,
            onSurface = LexturesColors.TextPrimaryDark,
            error = LexturesColors.Error,
        )
    } else {
        lightColorScheme(
            primary = LexturesColors.Primary,
            onPrimary = Color.White,
            background = LexturesColors.SceneBackground,
            surface = LexturesColors.CardBackground,
            onBackground = LexturesColors.TextPrimary,
            onSurface = LexturesColors.TextPrimary,
            error = LexturesColors.Error,
        )
    }

    MaterialTheme(
        colorScheme = scheme,
        content = content,
    )
}
