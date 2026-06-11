package com.lextures.android.core.design

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import kotlin.math.abs

/**
 * Lextures by StudyDrift brand system.
 *
 * Derived from the logo: a rocket lifting off an open book. Warm cream paper,
 * deep-teal ink, coral energy, amber highlights. Serif display type for an
 * editorial, scholarly feel; system sans for body copy.
 */
object LexturesColors {
    // Brand anchors (from logo)
    val BrandTeal = Color(0xFF6EC0B1)
    val BrandCoral = Color(0xFFF6684B)
    val BrandAmber = Color(0xFFF69945)
    val BrandCream = Color(0xFFF4E4C0)

    // Action colors
    val Primary = Color(0xFF12756A)
    val PrimaryDeep = Color(0xFF0C4F47)
    val PrimaryMuted = Color(0xFF135854)
    val Coral = BrandCoral
    val Amber = BrandAmber
    val Error = Color(0xFFDF3250)
    val StrengthStrong = Color(0xFF0C9857)

    // Light surfaces
    val SceneBackground = Color(0xFFFAF5EA)
    val CardBackground = Color.White
    val FieldBorder = Color(0xFFEAE0CC)
    val TextPrimary = Color(0xFF1F2D2A)
    val TextSecondary = Color(0xFF64746F)

    // Dark surfaces (teal-tinted, never pure gray)
    val SceneBackgroundDark = Color(0xFF111B19)
    val CardBackgroundDark = Color(0xFF1B2725)
    val FieldBorderDark = Color(0xFF32423E)
    val TextPrimaryDark = Color(0xFFF2EFE6)
    val TextSecondaryDark = Color(0xFF9CAEA8)
}

/** Serif display type — like a textbook chapter heading. */
object LexturesType {
    val DisplayFamily = FontFamily.Serif

    fun display(size: Int, weight: FontWeight = FontWeight.SemiBold) = TextStyle(
        fontFamily = DisplayFamily,
        fontWeight = weight,
        fontSize = size.sp,
    )
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

/** Brighter primary for dark backgrounds. */
@Composable
fun accentColor(): Color =
    if (isDarkTheme()) LexturesColors.BrandTeal else LexturesColors.Primary

/** Hero gradient (deep teal): dashboard greeting, course banners. */
val HeroBrush: Brush = Brush.linearGradient(
    colors = listOf(LexturesColors.PrimaryDeep, Color(0xFF17897B)),
)

private val CoverPalettes = listOf(
    listOf(Color(0xFF17897B), Color(0xFF6EC0B1)), // teal
    listOf(Color(0xFFE2553A), Color(0xFFF69945)), // coral → amber
    listOf(Color(0xFF0C4F47), Color(0xFF2BA391)), // deep teal
    listOf(Color(0xFFD9822B), Color(0xFFF6B95A)), // amber
    listOf(Color(0xFFC65441), Color(0xFFF6684B)), // coral
)

/** Deterministic course-cover gradient: every course gets a stable brand color. */
fun coverBrush(key: String): Brush {
    val index = abs(key.fold(0) { acc, c -> acc * 31 + c.code }) % CoverPalettes.size
    return Brush.linearGradient(
        colors = CoverPalettes[index],
        start = Offset.Zero,
        end = Offset(420f, 420f),
    )
}

@Composable
fun LexturesTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val scheme = if (dark) {
        darkColorScheme(
            primary = LexturesColors.BrandTeal,
            onPrimary = LexturesColors.PrimaryDeep,
            background = LexturesColors.SceneBackgroundDark,
            surface = LexturesColors.CardBackgroundDark,
            onBackground = LexturesColors.TextPrimaryDark,
            onSurface = LexturesColors.TextPrimaryDark,
            secondary = LexturesColors.BrandCoral,
            tertiary = LexturesColors.BrandAmber,
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
            secondary = LexturesColors.BrandCoral,
            tertiary = LexturesColors.BrandAmber,
            error = LexturesColors.Error,
        )
    }

    MaterialTheme(
        colorScheme = scheme,
        content = content,
    )
}
