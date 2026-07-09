import Foundation

/// Course translation / glossary models (M13.9).
struct TranslationCoverage: Codable, Hashable, Identifiable {
    var targetLocale: String
    var totalItems: Int
    var translatedItems: Int
    var percent: Double
    var untranslated: [TranslatableCourseItem]?

    var id: String { targetLocale }

    var percentInt: Int {
        Int(percent.rounded())
    }
}

struct TranslatableCourseItem: Codable, Hashable, Identifiable {
    var itemId: String
    var itemType: String
    var title: String
    var body: String
    var hasPublished: Bool?
    var hasDraft: Bool?

    var id: String { itemId }
}

struct CourseTranslationListResponse: Codable, Hashable {
    var items: [CourseTranslationListItem]
    var coverage: TranslationCoverage
}

struct CourseTranslationListItem: Codable, Hashable, Identifiable {
    var itemId: String
    var itemType: String
    var title: String
    var body: String
    var hasPublished: Bool?
    var hasDraft: Bool?
    var targetLocale: String?
    var translatedTitle: String?
    var translatedBody: String?
    var isDraft: Bool?
    var machineTranslationDraft: Bool?
    var publishedAt: String?
    var version: Int64?
    var glossaryMatches: [GlossaryMatchSpan]?

    var id: String { itemId }
}

struct GlossaryMatchSpan: Codable, Hashable {
    var sourceTerm: String
    var targetTerm: String
    var start: Int?
    var end: Int?
}

struct CourseGlossaryListResponse: Codable, Hashable {
    var entries: [CourseGlossaryEntry]
}

struct CourseGlossaryEntry: Codable, Hashable, Identifiable {
    var id: String
    var sourceTerm: String
    var targetTerm: String
    var sourceLocale: String?
    var targetLocale: String?
}

struct TranslationLocalesResponse: Codable, Hashable {
    var locales: [TranslationCoverage]
}

struct SaveCourseTranslationBody: Codable, Hashable {
    var targetLocale: String
    var sourceLocale: String?
    var translatedTitle: String?
    var translatedBody: String?
    var isDraft: Bool?
    var machineTranslationDraft: Bool?
    var version: Int64?
}

struct PublishCourseTranslationBody: Codable, Hashable {
    var targetLocale: String
}

struct AddGlossaryEntryBody: Codable, Hashable {
    var sourceTerm: String
    var targetTerm: String
    var targetLocale: String
    var sourceLocale: String?
}

