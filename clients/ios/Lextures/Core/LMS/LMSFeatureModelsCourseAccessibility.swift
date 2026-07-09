import Foundation

/// Course accessibility / alt-text models (M13.8).
struct CourseAccessibilityInfo: Codable, Hashable {
    var altTextCoverage: AltTextCoverage
    var hardBlockSave: Bool
}

struct AltTextCoverage: Codable, Hashable {
    var withAlt: Int
    var total: Int
    var percent: Int
    var uncoveredItems: [UncoveredAccessibilityItem]
}

struct UncoveredAccessibilityItem: Codable, Identifiable, Hashable {
    var itemId: String
    var title: String
    var kind: String
    var withAlt: Int
    var total: Int
    var missing: Int

    var id: String { itemId }
}

struct AltTextSuggestion: Codable {
    var suggestion: String
    var confidence: Double
}

struct PatchItemMarkdownBody: Encodable {
    var markdown: String
}

struct SuggestAltTextBody: Encodable {
    var imageUrl: String
    var language: String
}
