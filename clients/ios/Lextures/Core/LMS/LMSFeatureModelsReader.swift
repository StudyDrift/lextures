import Foundation

// MARK: - Reading preferences (GET/PATCH /api/v1/me/reading-preferences)

struct ReadingPreferencesRow: Codable, Equatable {
    var fontFace: String = "default"
    var letterSpacing: String = "normal"
    var wordSpacing: String = "normal"
    var lineHeight: String = "normal"
    var rulerEnabled: Bool = false
    var rulerColor: String = "yellow"
    var ttsEnabled: Bool = false
    var ttsSpeed: Double = 1.0
    var ttsVoiceName: String?
    var sttEnabled: Bool = false
    var sttLanguage: String = "en-US"
    var dyslexiaDisplayEnabled: Bool = false
    var highContrastEnabled: Bool = false
    var reducedMotionEnabled: Bool = false
    var uiModeOverride: String?
    var effectiveUiMode: String?
    var updatedAt: String?
}

struct ReadingPreferencesPatch: Encodable {
    var fontFace: String?
    var letterSpacing: String?
    var wordSpacing: String?
    var lineHeight: String?
    var rulerEnabled: Bool?
    var rulerColor: String?
    var ttsEnabled: Bool?
    var ttsSpeed: Double?
    var ttsVoiceName: String?
    var sttEnabled: Bool?
    var sttLanguage: String?
    var dyslexiaDisplayEnabled: Bool?
    var highContrastEnabled: Bool?
    var reducedMotionEnabled: Bool?
    var uiModeOverride: String?
    var effectiveUiMode: String?
}

// MARK: - Captions

struct CaptionRecord: Codable, Identifiable, Equatable {
    var id: String
    var storageObjectId: String?
    var lang: String
    var status: String
    var hasLowConfidence: Bool?
    var confidenceAvg: Double?
    var backend: String?
    var createdAt: String?
    var reviewedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case storageObjectId = "storage_object_id"
        case lang
        case status
        case hasLowConfidence = "has_low_confidence"
        case confidenceAvg = "confidence_avg"
        case backend
        case createdAt = "created_at"
        case reviewedAt = "reviewed_at"
    }
}

// MARK: - On-the-fly translation (POST /api/v1/translate)

struct TranslateContentRequest: Encodable {
    var contentType: String
    var contentId: String
    var targetLang: String
    var text: String

    enum CodingKeys: String, CodingKey {
        case contentType = "content_type"
        case contentId = "content_id"
        case targetLang = "target_lang"
        case text
    }
}

struct TranslateContentResponse: Decodable, Equatable {
    var translated: String
    var sourceLang: String
    var cached: Bool

    enum CodingKeys: String, CodingKey {
        case translated
        case sourceLang = "source_lang"
        case cached
    }
}

// MARK: - Course translation coverage

struct TranslationCoverageLocale: Decodable, Equatable {
    var targetLocale: String
    var percent: Double

    enum CodingKeys: String, CodingKey {
        case targetLocale = "target_locale"
        case percent
    }
}

struct TranslationCoverageResponse: Decodable {
    var locales: [TranslationCoverageLocale]?
    var targetLocale: String?
    var percent: Double?
}

struct PatchContentLocaleBody: Encodable {
    var contentLocale: String?

    enum CodingKeys: String, CodingKey {
        case contentLocale
    }
}