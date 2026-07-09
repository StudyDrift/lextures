package com.lextures.android.core.lms

import android.content.Context
import java.util.Locale
import kotlin.math.round

/** Course translations / glossary helpers (M13.9). */
object CourseTranslationsLogic {
    const val PAGE_SIZE = 20
    const val DEFAULT_SOURCE_LOCALE = "en"

    data class LocaleOption(val tag: String, val labelResKey: String)

    val targetLocaleOptions: List<LocaleOption> = listOf(
        LocaleOption("es", "mobile_courseSettings_translations_locale_es"),
        LocaleOption("fr", "mobile_courseSettings_translations_locale_fr"),
        LocaleOption("ar", "mobile_courseSettings_translations_locale_ar"),
        LocaleOption("he", "mobile_courseSettings_translations_locale_he"),
        LocaleOption("es-ES", "mobile_courseSettings_translations_locale_esES"),
        LocaleOption("es-MX", "mobile_courseSettings_translations_locale_esMX"),
        LocaleOption("fr-FR", "mobile_courseSettings_translations_locale_frFR"),
        LocaleOption("fr-CA", "mobile_courseSettings_translations_locale_frCA"),
        LocaleOption("ar-SA", "mobile_courseSettings_translations_locale_arSA"),
        LocaleOption("he-IL", "mobile_courseSettings_translations_locale_heIL"),
    )

    private val localeTagPattern = Regex("^[a-z]{2}(-[A-Z]{2})?$")
    private val rtlPrimary = setOf("ar", "he", "fa", "ur", "ps")

    fun isFeatureEnabled(features: com.lextures.android.core.navigation.MobilePlatformFeatures): Boolean =
        features.translationMemoryEnabled

    fun cacheKeyLocales(courseCode: String): String = "course:$courseCode:translations:locales"

    fun cacheKeyLocaleDetail(courseCode: String, locale: String): String =
        "course:$courseCode:translations:locale:$locale"

    fun cacheKeyGlossary(courseCode: String, locale: String): String =
        "course:$courseCode:translations:glossary:$locale"

    fun trackedLocalesPrefsKey(courseCode: String): String = "course_translations_tracked_$courseCode"

    fun loadTrackedLocales(context: Context, courseCode: String): List<String> {
        val prefs = context.getSharedPreferences("lextures_translations", Context.MODE_PRIVATE)
        val raw = prefs.getStringSet(trackedLocalesPrefsKey(courseCode), emptySet()) ?: emptySet()
        return raw.map { normalizeLocaleTag(it) }.filter { isValidLocaleTag(it) }.sorted()
    }

    fun saveTrackedLocales(context: Context, courseCode: String, tags: List<String>) {
        val prefs = context.getSharedPreferences("lextures_translations", Context.MODE_PRIVATE)
        val normalized = tags.map { normalizeLocaleTag(it) }.filter { isValidLocaleTag(it) }.toSet()
        prefs.edit().putStringSet(trackedLocalesPrefsKey(courseCode), normalized).apply()
    }

    fun glossaryIdempotencyKey(courseCode: String, locale: String, sourceTerm: String): String {
        val term = sourceTerm.trim().lowercase(Locale.US)
        return "course-translations:$courseCode:glossary:$locale:$term"
    }

    fun glossaryPath(courseCode: String): String = "/api/v1/courses/$courseCode/glossary"

    fun coveragePath(courseCode: String, targetLocale: String? = null): String =
        if (!targetLocale.isNullOrBlank()) {
            "/api/v1/courses/$courseCode/translation-coverage?target_locale=$targetLocale"
        } else {
            "/api/v1/courses/$courseCode/translation-coverage"
        }

    fun translationsPath(courseCode: String, targetLocale: String): String =
        "/api/v1/courses/$courseCode/translations?target_locale=$targetLocale"

    fun normalizeLocaleTag(raw: String): String = raw.trim()

    fun isValidLocaleTag(raw: String): Boolean {
        val tag = normalizeLocaleTag(raw)
        return tag.isNotEmpty() && localeTagPattern.matches(tag)
    }

    fun isRTLLocale(tag: String): Boolean {
        val primary = normalizeLocaleTag(tag).split("-").firstOrNull()?.lowercase(Locale.US).orEmpty()
        return primary in rtlPrimary
    }

    fun localeLabelResKey(tag: String): String {
        val normalized = normalizeLocaleTag(tag)
        targetLocaleOptions.firstOrNull { it.tag == normalized }?.let { return it.labelResKey }
        val primary = normalized.split("-").firstOrNull()?.lowercase(Locale.US).orEmpty()
        targetLocaleOptions.firstOrNull { it.tag == primary }?.let { return it.labelResKey }
        return "mobile_courseSettings_translations_locale_unknown"
    }

    fun fallbackLocaleDisplayName(tag: String): String =
        tag.uppercase(Locale.US)

    fun coveragePercent(translated: Int, total: Int): Int {
        if (total <= 0) return 100
        return round(translated.toDouble() / total.toDouble() * 100).toInt()
    }

    fun formatCoveragePercentOnly(percent: Double): String = "${round(percent).toInt()}%"

    fun mergeLocales(
        server: List<TranslationCoverage>,
        tracked: List<String>,
        totalItemsFallback: Int = 0,
    ): List<TranslationCoverage> {
        val byLocale = linkedMapOf<String, TranslationCoverage>()
        for (row in server) {
            val tag = normalizeLocaleTag(row.targetLocale)
            if (tag.isNotEmpty()) byLocale[tag] = row
        }
        val total = server.firstOrNull()?.totalItems ?: totalItemsFallback
        for (raw in tracked) {
            val tag = normalizeLocaleTag(raw)
            if (tag.isEmpty() || byLocale.containsKey(tag)) continue
            byLocale[tag] = TranslationCoverage(
                targetLocale = tag,
                totalItems = total,
                translatedItems = 0,
                percent = if (total == 0) 100.0 else 0.0,
            )
        }
        return byLocale.values.sortedBy { it.targetLocale.lowercase(Locale.US) }
    }

    fun availableLocalesToAdd(existing: List<TranslationCoverage>): List<LocaleOption> {
        val present = existing.map { normalizeLocaleTag(it.targetLocale) }.toSet()
        return targetLocaleOptions.filter { it.tag !in present }
    }

    fun trackLocale(tag: String, into: List<String>): List<String> {
        val normalized = normalizeLocaleTag(tag)
        if (!isValidLocaleTag(normalized)) return into
        if (into.any { normalizeLocaleTag(it) == normalized }) return into
        return into + normalized
    }

    fun mergeTracked(cached: List<String>, current: List<String>): List<String> {
        val seen = linkedSetOf<String>()
        for (tag in cached + current) {
            val n = normalizeLocaleTag(tag)
            if (isValidLocaleTag(n)) seen.add(n)
        }
        return seen.toList()
    }

    data class GlossaryDraft(
        val id: String? = null,
        val sourceTerm: String = "",
        val targetTerm: String = "",
    ) {
        val isEditing: Boolean get() = id != null
    }

    enum class GlossaryValidation { Ok, SourceRequired, TargetRequired }

    fun validateGlossaryDraft(draft: GlossaryDraft): GlossaryValidation {
        if (draft.sourceTerm.trim().isEmpty()) return GlossaryValidation.SourceRequired
        if (draft.targetTerm.trim().isEmpty()) return GlossaryValidation.TargetRequired
        return GlossaryValidation.Ok
    }

    fun glossaryDiff(
        sourceTerm: String,
        targetTerm: String,
        existing: CourseGlossaryEntry?,
    ): Boolean {
        val src = sourceTerm.trim()
        val tgt = targetTerm.trim()
        if (existing == null) return src.isNotEmpty() && tgt.isNotEmpty()
        return src != existing.sourceTerm.trim() || tgt != existing.targetTerm.trim()
    }

    fun filterGlossary(entries: List<CourseGlossaryEntry>, query: String): List<CourseGlossaryEntry> {
        val q = query.trim().lowercase(Locale.US)
        if (q.isEmpty()) return entries
        return entries.filter {
            it.sourceTerm.lowercase(Locale.US).contains(q) ||
                it.targetTerm.lowercase(Locale.US).contains(q)
        }
    }

    fun paginatedGlossary(entries: List<CourseGlossaryEntry>, page: Int): List<CourseGlossaryEntry> {
        val end = minOf(entries.size, maxOf(0, page + 1) * PAGE_SIZE)
        return entries.take(end)
    }

    fun hasMoreGlossaryPages(entries: List<CourseGlossaryEntry>, page: Int): Boolean =
        entries.size > (page + 1) * PAGE_SIZE

    fun draft(from: CourseGlossaryEntry?): GlossaryDraft =
        if (from == null) GlossaryDraft()
        else GlossaryDraft(id = from.id, sourceTerm = from.sourceTerm, targetTerm = from.targetTerm)

    fun buildGlossaryBody(
        draft: GlossaryDraft,
        targetLocale: String,
        sourceLocale: String = DEFAULT_SOURCE_LOCALE,
    ): AddGlossaryEntryBody = AddGlossaryEntryBody(
        sourceTerm = draft.sourceTerm.trim(),
        targetTerm = draft.targetTerm.trim(),
        targetLocale = targetLocale,
        sourceLocale = sourceLocale,
    )

    fun upsertGlossaryEntry(
        entry: CourseGlossaryEntry,
        into: List<CourseGlossaryEntry>,
    ): List<CourseGlossaryEntry> {
        val key = entry.sourceTerm.trim().lowercase(Locale.US)
        return (into.filter {
            it.sourceTerm.trim().lowercase(Locale.US) != key && it.id != entry.id
        } + entry).sortedBy { it.sourceTerm.lowercase(Locale.US) }
    }

    fun paginatedItems(items: List<CourseTranslationListItem>, page: Int): List<CourseTranslationListItem> {
        val end = minOf(items.size, maxOf(0, page + 1) * PAGE_SIZE)
        return items.take(end)
    }

    fun hasMoreItemPages(items: List<CourseTranslationListItem>, page: Int): Boolean =
        items.size > (page + 1) * PAGE_SIZE

    fun filterItems(items: List<CourseTranslationListItem>, query: String): List<CourseTranslationListItem> {
        val q = query.trim().lowercase(Locale.US)
        if (q.isEmpty()) return items
        return items.filter { it.title.lowercase(Locale.US).contains(q) }
    }

    fun unpublishedCount(items: List<CourseTranslationListItem>): Int =
        items.count { it.hasPublished != true }

    fun statusLabelResKey(item: CourseTranslationListItem): String = when {
        item.hasPublished == true -> "mobile_courseSettings_translations_status_published"
        item.hasDraft == true || item.isDraft == true -> "mobile_courseSettings_translations_status_draft"
        else -> "mobile_courseSettings_translations_status_missing"
    }
}
