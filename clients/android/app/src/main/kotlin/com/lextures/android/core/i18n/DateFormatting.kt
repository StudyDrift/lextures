package com.lextures.android.core.i18n

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import java.util.Locale

/** Locale- and timezone-aware date/number formatting for LMS timestamps (plan M0.4 / M2.1). */
object DateFormatting {
    fun parse(raw: String?): Instant? {
        if (raw.isNullOrBlank()) return null
        return runCatching { Instant.parse(raw) }.getOrNull()
    }

    fun formatAbsoluteShort(
        raw: String?,
        locale: Locale,
        zoneId: ZoneId = ZoneId.systemDefault(),
    ): String {
        val instant = parse(raw) ?: return EM_DASH
        val zoned = instant.atZone(zoneId)
        return DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM, FormatStyle.SHORT)
            .withLocale(locale)
            .withZone(zoneId)
            .format(zoned)
    }

    fun formatDue(
        raw: String?,
        locale: Locale,
        zoneId: ZoneId = ZoneId.systemDefault(),
        duePrefix: String = "Due %1\$s",
    ): String {
        val formatted = formatAbsoluteShort(raw, locale, zoneId)
        if (formatted == EM_DASH) return formatted
        return String.format(locale, duePrefix, formatted)
    }

    fun formatDate(
        raw: String?,
        locale: Locale,
        zoneId: ZoneId = ZoneId.systemDefault(),
    ): String {
        val instant = parse(raw) ?: return EM_DASH
        return DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM)
            .withLocale(locale)
            .withZone(zoneId)
            .format(instant)
    }

    fun formatRelative(raw: String?, locale: Locale): String {
        val instant = parse(raw) ?: return ""
        val diffSeconds = (Instant.now().epochSecond - instant.epochSecond).coerceAtLeast(0)
        return when {
            diffSeconds < 45 -> "Just now"
            diffSeconds < 3600 -> "${diffSeconds / 60}m"
            diffSeconds < 86400 -> "${diffSeconds / 3600}h"
            diffSeconds < 604800 -> "${diffSeconds / 86400}d"
            else -> formatDate(raw, locale)
        }
    }

    fun formatNumber(value: Double, locale: Locale, maximumFractionDigits: Int = 0): String {
        val format = java.text.NumberFormat.getNumberInstance(locale)
        format.maximumFractionDigits = maximumFractionDigits
        format.minimumFractionDigits = 0
        return format.format(value)
    }

    fun formatPoints(value: Double, locale: Locale, pointsSuffix: String = "%1\$s pts"): String {
        val formatted = formatNumber(value, locale, maximumFractionDigits = 1)
        return String.format(locale, pointsSuffix, formatted)
    }

    const val EM_DASH = "—"
}
