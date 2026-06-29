package com.lextures.android.core.i18n

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.ZoneId
import java.util.Locale

class I18nTest {
    @Test
    fun isRTLLocale_detectsArabicAndHebrew() {
        assertTrue(LocalePreferences.isRTLLocale("ar"))
        assertTrue(LocalePreferences.isRTLLocale("ar-SA"))
        assertTrue(LocalePreferences.isRTLLocale("he-IL"))
        assertFalse(LocalePreferences.isRTLLocale("en"))
        assertFalse(LocalePreferences.isRTLLocale("fr"))
    }

    @Test
    fun resolveResourceLanguage_mapsSupportedBundles() {
        assertEquals("es", LocalePreferences.resolveResourceLanguage("es"))
        assertEquals("fr", LocalePreferences.resolveResourceLanguage("fr-CA"))
        assertEquals("ar", LocalePreferences.resolveResourceLanguage("ar"))
        assertEquals("en-XA", LocalePreferences.resolveResourceLanguage("en-XA"))
        assertEquals("en", LocalePreferences.resolveResourceLanguage("de"))
    }

    @Test
    fun dateFormatting_parsesIsoAndFormatsDueInTimeZone() {
        val iso = "2026-06-16T07:59:59Z"
        assertTrue(DateFormatting.parse(iso) != null)
        val utc = DateFormatting.formatDue(iso, Locale.US, ZoneId.of("UTC"))
        val pacific = DateFormatting.formatDue(iso, Locale.US, ZoneId.of("America/Los_Angeles"))
        assertFalse(utc.isEmpty())
        assertNotEquals(utc, pacific)
    }

    @Test
    fun dateFormatting_formatsNumbersPerLocale() {
        val english = DateFormatting.formatNumber(1234.5, Locale.US)
        val french = DateFormatting.formatNumber(1234.5, Locale.FRANCE)
        assertTrue(english.contains("1"))
        assertTrue(french.contains("1"))
    }
}
