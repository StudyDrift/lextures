package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseTranslationsLogicTest {
    @Test
    fun featureGate_requiresTranslationMemory() {
        assertFalse(CourseTranslationsLogic.isFeatureEnabled(MobilePlatformFeatures(translationMemoryEnabled = false)))
        assertTrue(CourseTranslationsLogic.isFeatureEnabled(MobilePlatformFeatures(translationMemoryEnabled = true)))
    }

    @Test
    fun coveragePercent_emptyTotalIs100() {
        assertEquals(100, CourseTranslationsLogic.coveragePercent(0, 0))
    }

    @Test
    fun coveragePercent_roundsCorrectly() {
        assertEquals(33, CourseTranslationsLogic.coveragePercent(1, 3))
        assertEquals(100, CourseTranslationsLogic.coveragePercent(2, 2))
    }

    @Test
    fun mergeLocales_includesTracked() {
        val server = listOf(
            TranslationCoverage(targetLocale = "es", totalItems = 10, translatedItems = 4, percent = 40.0),
        )
        val merged = CourseTranslationsLogic.mergeLocales(server, listOf("fr", "es"))
        assertEquals(listOf("es", "fr"), merged.map { it.targetLocale }.sorted())
        val fr = merged.first { it.targetLocale == "fr" }
        assertEquals(0, fr.translatedItems)
        assertEquals(10, fr.totalItems)
    }

    @Test
    fun availableLocales_excludesExisting() {
        val existing = listOf(
            TranslationCoverage(targetLocale = "es", totalItems = 1, translatedItems = 0, percent = 0.0),
        )
        val available = CourseTranslationsLogic.availableLocalesToAdd(existing)
        assertFalse(available.any { it.tag == "es" })
        assertTrue(available.any { it.tag == "fr" })
    }

    @Test
    fun glossaryValidation() {
        assertEquals(
            CourseTranslationsLogic.GlossaryValidation.SourceRequired,
            CourseTranslationsLogic.validateGlossaryDraft(
                CourseTranslationsLogic.GlossaryDraft(sourceTerm = "", targetTerm = "x"),
            ),
        )
        assertEquals(
            CourseTranslationsLogic.GlossaryValidation.TargetRequired,
            CourseTranslationsLogic.validateGlossaryDraft(
                CourseTranslationsLogic.GlossaryDraft(sourceTerm = "x", targetTerm = "  "),
            ),
        )
        assertEquals(
            CourseTranslationsLogic.GlossaryValidation.Ok,
            CourseTranslationsLogic.validateGlossaryDraft(
                CourseTranslationsLogic.GlossaryDraft(sourceTerm = "a", targetTerm = "b"),
            ),
        )
    }

    @Test
    fun glossaryDiff_detectsChanges() {
        val existing = CourseGlossaryEntry(id = "1", sourceTerm = "term", targetTerm = "término")
        assertFalse(CourseTranslationsLogic.glossaryDiff("term", "término", existing))
        assertTrue(CourseTranslationsLogic.glossaryDiff("term", "palabra", existing))
        assertTrue(CourseTranslationsLogic.glossaryDiff("new", "nuevo", null))
    }

    @Test
    fun filterAndPaginateGlossary() {
        val entries = (0 until 25).map {
            CourseGlossaryEntry(id = "$it", sourceTerm = "term-$it", targetTerm = "t-$it")
        }
        assertEquals(20, CourseTranslationsLogic.paginatedGlossary(entries, 0).size)
        assertTrue(CourseTranslationsLogic.hasMoreGlossaryPages(entries, 0))
        assertFalse(CourseTranslationsLogic.hasMoreGlossaryPages(entries, 1))
        val filtered = CourseTranslationsLogic.filterGlossary(entries, "term-1")
        assertTrue(filtered.all { it.sourceTerm.contains("term-1") })
    }

    @Test
    fun upsertGlossaryEntry_replacesBySourceTerm() {
        val existing = listOf(
            CourseGlossaryEntry(id = "1", sourceTerm = "Alpha", targetTerm = "A1"),
            CourseGlossaryEntry(id = "2", sourceTerm = "Beta", targetTerm = "B1"),
        )
        val updated = CourseTranslationsLogic.upsertGlossaryEntry(
            CourseGlossaryEntry(id = "3", sourceTerm = "alpha", targetTerm = "A2"),
            existing,
        )
        assertEquals(2, updated.size)
        assertEquals("A2", updated.first { it.sourceTerm.equals("alpha", ignoreCase = true) }.targetTerm)
    }

    @Test
    fun localeTagValidation() {
        assertTrue(CourseTranslationsLogic.isValidLocaleTag("es"))
        assertTrue(CourseTranslationsLogic.isValidLocaleTag("es-MX"))
        assertTrue(CourseTranslationsLogic.isValidLocaleTag("xx")) // shape only; server enforces allow-list
        assertFalse(CourseTranslationsLogic.isValidLocaleTag("ES"))
        assertFalse(CourseTranslationsLogic.isValidLocaleTag("e"))
        assertFalse(CourseTranslationsLogic.isValidLocaleTag(""))
    }

    @Test
    fun isRTLLocale() {
        assertTrue(CourseTranslationsLogic.isRTLLocale("ar"))
        assertTrue(CourseTranslationsLogic.isRTLLocale("he-IL"))
        assertFalse(CourseTranslationsLogic.isRTLLocale("es"))
    }

    @Test
    fun statusLabelResKey() {
        val published = CourseTranslationListItem(
            itemId = "1",
            itemType = "content_page",
            title = "A",
            body = "",
            hasPublished = true,
        )
        val draft = CourseTranslationListItem(
            itemId = "2",
            itemType = "content_page",
            title = "B",
            body = "",
            hasDraft = true,
        )
        val missing = CourseTranslationListItem(
            itemId = "3",
            itemType = "content_page",
            title = "C",
            body = "",
        )
        assertEquals(
            "mobile_courseSettings_translations_status_published",
            CourseTranslationsLogic.statusLabelResKey(published),
        )
        assertEquals(
            "mobile_courseSettings_translations_status_draft",
            CourseTranslationsLogic.statusLabelResKey(draft),
        )
        assertEquals(
            "mobile_courseSettings_translations_status_missing",
            CourseTranslationsLogic.statusLabelResKey(missing),
        )
    }

    @Test
    fun trackLocale_dedupes() {
        val once = CourseTranslationsLogic.trackLocale("es", emptyList())
        val twice = CourseTranslationsLogic.trackLocale("es", once)
        assertEquals(listOf("es"), once)
        assertEquals(listOf("es"), twice)
    }
}
