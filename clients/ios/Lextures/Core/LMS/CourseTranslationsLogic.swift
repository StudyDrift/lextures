import Foundation

/// Course translations / glossary helpers (M13.9).
enum CourseTranslationsLogic {
    static let pageSize = 20
    static let defaultSourceLocale = "en"

    /// Target locales instructors can add (matches server supported content locales).
    static let targetLocaleOptions: [(tag: String, labelKey: String)] = [
        ("es", "mobile.courseSettings.translations.locale.es"),
        ("fr", "mobile.courseSettings.translations.locale.fr"),
        ("ar", "mobile.courseSettings.translations.locale.ar"),
        ("he", "mobile.courseSettings.translations.locale.he"),
        ("es-ES", "mobile.courseSettings.translations.locale.esES"),
        ("es-MX", "mobile.courseSettings.translations.locale.esMX"),
        ("fr-FR", "mobile.courseSettings.translations.locale.frFR"),
        ("fr-CA", "mobile.courseSettings.translations.locale.frCA"),
        ("ar-SA", "mobile.courseSettings.translations.locale.arSA"),
        ("he-IL", "mobile.courseSettings.translations.locale.heIL"),
    ]

    static func isFeatureEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.translationMemoryEnabled
    }

    static func cacheKeyLocales(courseCode: String) -> String {
        "course:\(courseCode):translations:locales"
    }

    static func cacheKeyLocaleDetail(courseCode: String, locale: String) -> String {
        "course:\(courseCode):translations:locale:\(locale)"
    }

    static func cacheKeyGlossary(courseCode: String, locale: String) -> String {
        "course:\(courseCode):translations:glossary:\(locale)"
    }

    static func trackedLocalesDefaultsKey(courseCode: String) -> String {
        "course-translations-tracked:\(courseCode)"
    }

    static func loadTrackedLocales(courseCode: String) -> [String] {
        UserDefaults.standard.stringArray(forKey: trackedLocalesDefaultsKey(courseCode: courseCode)) ?? []
    }

    static func saveTrackedLocales(_ tags: [String], courseCode: String) {
        let normalized = tags
            .map { normalizeLocaleTag($0) }
            .filter { isValidLocaleTag($0) }
        UserDefaults.standard.set(Array(Set(normalized)).sorted(), forKey: trackedLocalesDefaultsKey(courseCode: courseCode))
    }

    static func mergeTracked(cached: [String], current: [String]) -> [String] {
        var seen = Set<String>()
        var out: [String] = []
        for tag in cached + current {
            let n = normalizeLocaleTag(tag)
            guard isValidLocaleTag(n), !seen.contains(n) else { continue }
            seen.insert(n)
            out.append(n)
        }
        return out
    }

    static func glossaryIdempotencyKey(
        courseCode: String,
        locale: String,
        sourceTerm: String
    ) -> String {
        let term = sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return "course-translations:\(courseCode):glossary:\(locale):\(term)"
    }

    static func translationSaveIdempotencyKey(
        courseCode: String,
        itemId: String,
        locale: String
    ) -> String {
        "course-translations:\(courseCode):item:\(itemId):\(locale)"
    }

    static func publishIdempotencyKey(
        courseCode: String,
        itemId: String,
        locale: String
    ) -> String {
        "course-translations:\(courseCode):publish:\(itemId):\(locale)"
    }

    static func translationsPath(courseCode: String, targetLocale: String) -> String {
        "/api/v1/courses/\(courseCode)/translations?target_locale=\(targetLocale)"
    }

    static func translationItemPath(courseCode: String, itemId: String) -> String {
        "/api/v1/courses/\(courseCode)/translations/\(itemId)"
    }

    static func publishPath(courseCode: String, itemId: String) -> String {
        "/api/v1/courses/\(courseCode)/translations/\(itemId)/publish"
    }

    static func glossaryPath(courseCode: String) -> String {
        "/api/v1/courses/\(courseCode)/glossary"
    }

    static func coveragePath(courseCode: String, targetLocale: String? = nil) -> String {
        if let targetLocale, !targetLocale.isEmpty {
            return "/api/v1/courses/\(courseCode)/translation-coverage?target_locale=\(targetLocale)"
        }
        return "/api/v1/courses/\(courseCode)/translation-coverage"
    }

    static func normalizeLocaleTag(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func isValidLocaleTag(_ raw: String) -> Bool {
        let tag = normalizeLocaleTag(raw)
        guard !tag.isEmpty else { return false }
        let pattern = /^[a-z]{2}(-[A-Z]{2})?$/
        return tag.wholeMatch(of: pattern) != nil
    }

    static func localeLabelKey(for tag: String) -> String {
        let normalized = normalizeLocaleTag(tag)
        if let match = targetLocaleOptions.first(where: { $0.tag == normalized }) {
            return match.labelKey
        }
        let primary = normalized.split(separator: "-").first.map(String.init)?.lowercased() ?? normalized
        if let match = targetLocaleOptions.first(where: { $0.tag == primary }) {
            return match.labelKey
        }
        return "mobile.courseSettings.translations.locale.unknown"
    }

    static func localeDisplayName(for tag: String) -> String {
        let key = localeLabelKey(for: tag)
        if key == "mobile.courseSettings.translations.locale.unknown" {
            return ReaderLogic.localeLabel(tag)
        }
        return L.text(String.LocalizationValue(key))
    }

    static func isRTLLocale(_ tag: String) -> Bool {
        LocalePreferences.isRTLLocale(tag)
    }

    static func coveragePercent(translated: Int, total: Int) -> Int {
        if total <= 0 { return 100 }
        return Int((Double(translated) / Double(total) * 100).rounded())
    }

    static func formatCoverageLabel(translated: Int, total: Int, locale: String) -> String {
        let pct = coveragePercent(translated: translated, total: total)
        return L.format(
            "mobile.courseSettings.translations.coverageValue",
            pct,
            translated,
            total,
            localeDisplayName(for: locale)
        )
    }

    static func formatCoveragePercentOnly(percent: Double) -> String {
        "\(Int(percent.rounded()))%"
    }

    /// Merge server locales with locally tracked locales (e.g. newly added with 0 published).
    static func mergeLocales(
        server: [TranslationCoverage],
        tracked: [String],
        totalItemsFallback: Int = 0
    ) -> [TranslationCoverage] {
        var byLocale: [String: TranslationCoverage] = [:]
        for row in server {
            let tag = normalizeLocaleTag(row.targetLocale)
            guard !tag.isEmpty else { continue }
            byLocale[tag] = row
        }
        let total = server.first?.totalItems ?? totalItemsFallback
        for raw in tracked {
            let tag = normalizeLocaleTag(raw)
            guard !tag.isEmpty, byLocale[tag] == nil else { continue }
            byLocale[tag] = TranslationCoverage(
                targetLocale: tag,
                totalItems: total,
                translatedItems: 0,
                percent: total == 0 ? 100 : 0,
                untranslated: nil
            )
        }
        return byLocale.values.sorted {
            localeDisplayName(for: $0.targetLocale).localizedCaseInsensitiveCompare(
                localeDisplayName(for: $1.targetLocale)
            ) == .orderedAscending
        }
    }

    static func availableLocalesToAdd(existing: [TranslationCoverage]) -> [(tag: String, labelKey: String)] {
        let present = Set(existing.map { normalizeLocaleTag($0.targetLocale) })
        return targetLocaleOptions.filter { !present.contains($0.tag) }
    }

    static func trackLocale(_ tag: String, into tracked: [String]) -> [String] {
        let normalized = normalizeLocaleTag(tag)
        guard isValidLocaleTag(normalized) else { return tracked }
        if tracked.contains(where: { normalizeLocaleTag($0) == normalized }) {
            return tracked
        }
        return tracked + [normalized]
    }

    // MARK: - Glossary

    struct GlossaryDraft: Equatable, Hashable {
        var id: String?
        var sourceTerm: String = ""
        var targetTerm: String = ""

        var isEditing: Bool { id != nil }
    }

    enum GlossaryValidation: Equatable {
        case ok
        case sourceRequired
        case targetRequired
    }

    static func validateGlossaryDraft(_ draft: GlossaryDraft) -> GlossaryValidation {
        if draft.sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return .sourceRequired
        }
        if draft.targetTerm.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return .targetRequired
        }
        return .ok
    }

    static func glossaryDiff(
        sourceTerm: String,
        targetTerm: String,
        existing: CourseGlossaryEntry?
    ) -> Bool {
        let src = sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines)
        let tgt = targetTerm.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let existing else {
            return !src.isEmpty && !tgt.isEmpty
        }
        return src != existing.sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines)
            || tgt != existing.targetTerm.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func filterGlossary(
        _ entries: [CourseGlossaryEntry],
        query: String
    ) -> [CourseGlossaryEntry] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return entries }
        return entries.filter {
            $0.sourceTerm.lowercased().contains(q) || $0.targetTerm.lowercased().contains(q)
        }
    }

    static func paginatedGlossary(
        _ entries: [CourseGlossaryEntry],
        page: Int
    ) -> [CourseGlossaryEntry] {
        let end = min(entries.count, max(0, page + 1) * pageSize)
        return Array(entries.prefix(end))
    }

    static func hasMoreGlossaryPages(entries: [CourseGlossaryEntry], page: Int) -> Bool {
        entries.count > (page + 1) * pageSize
    }

    static func draft(from entry: CourseGlossaryEntry?) -> GlossaryDraft {
        guard let entry else { return GlossaryDraft() }
        return GlossaryDraft(id: entry.id, sourceTerm: entry.sourceTerm, targetTerm: entry.targetTerm)
    }

    static func buildGlossaryBody(
        draft: GlossaryDraft,
        targetLocale: String,
        sourceLocale: String = defaultSourceLocale
    ) -> AddGlossaryEntryBody {
        AddGlossaryEntryBody(
            sourceTerm: draft.sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines),
            targetTerm: draft.targetTerm.trimmingCharacters(in: .whitespacesAndNewlines),
            targetLocale: targetLocale,
            sourceLocale: sourceLocale
        )
    }

    /// Apply an upserted glossary entry into the local list (insert or replace by source term).
    static func upsertGlossaryEntry(
        _ entry: CourseGlossaryEntry,
        into entries: [CourseGlossaryEntry]
    ) -> [CourseGlossaryEntry] {
        let key = entry.sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        var next = entries.filter {
            $0.sourceTerm.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() != key
                && $0.id != entry.id
        }
        next.append(entry)
        return next.sorted {
            $0.sourceTerm.localizedCaseInsensitiveCompare($1.sourceTerm) == .orderedAscending
        }
    }

    // MARK: - Item list helpers (coverage detail)

    static func paginatedItems(
        _ items: [CourseTranslationListItem],
        page: Int
    ) -> [CourseTranslationListItem] {
        let end = min(items.count, max(0, page + 1) * pageSize)
        return Array(items.prefix(end))
    }

    static func hasMoreItemPages(items: [CourseTranslationListItem], page: Int) -> Bool {
        items.count > (page + 1) * pageSize
    }

    static func filterItems(
        _ items: [CourseTranslationListItem],
        query: String
    ) -> [CourseTranslationListItem] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return items }
        return items.filter { $0.title.lowercased().contains(q) }
    }

    static func unpublishedCount(_ items: [CourseTranslationListItem]) -> Int {
        items.filter { !($0.hasPublished ?? false) }.count
    }

    static func statusLabelKey(for item: CourseTranslationListItem) -> String {
        if item.hasPublished == true {
            return "mobile.courseSettings.translations.status.published"
        }
        if item.hasDraft == true || item.isDraft == true {
            return "mobile.courseSettings.translations.status.draft"
        }
        return "mobile.courseSettings.translations.status.missing"
    }
}
