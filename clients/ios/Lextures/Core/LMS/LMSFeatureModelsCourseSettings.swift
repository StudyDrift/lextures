import Foundation

// MARK: - Course settings models (M13.1)

struct MarkdownThemeCustom: Codable, Equatable, Hashable {
    var headingColor: String?
    var bodyColor: String?
    var linkColor: String?
    var codeBackground: String?
    var blockquoteBorder: String?
    var articleWidth: String?
    var fontFamily: String?

    static let seed = MarkdownThemeCustom(
        headingColor: "#0f172a",
        bodyColor: "#334155",
        linkColor: "#4f46e5",
        codeBackground: "#f1f5f9",
        blockquoteBorder: "#cbd5e1",
        articleWidth: "comfortable",
        fontFamily: "sans"
    )

    mutating func merge(_ other: MarkdownThemeCustom) {
        if let headingColor = other.headingColor { self.headingColor = headingColor }
        if let bodyColor = other.bodyColor { self.bodyColor = bodyColor }
        if let linkColor = other.linkColor { self.linkColor = linkColor }
        if let codeBackground = other.codeBackground { self.codeBackground = codeBackground }
        if let blockquoteBorder = other.blockquoteBorder { self.blockquoteBorder = blockquoteBorder }
        if let articleWidth = other.articleWidth { self.articleWidth = articleWidth }
        if let fontFamily = other.fontFamily { self.fontFamily = fontFamily }
    }
}

struct CourseUpdateRequest: Codable, Equatable {
    var title: String
    var description: String
    var published: Bool
    var startsAt: String?
    var endsAt: String?
    var visibleFrom: String?
    var hiddenAt: String?
    var scheduleMode: String
    var relativeEndAfter: String?
    var relativeHiddenAfter: String?
    var courseHomeLanding: String
    var courseHomeContentItemId: String?
    var courseTimezone: String?
    var gradeLevel: String?
}

struct CourseMarkdownThemePatch: Codable, Equatable {
    var preset: String
    var custom: MarkdownThemeCustom?
}

struct CourseHeroImageURLRequest: Codable {
    var imageUrl: String
}

struct CourseHeroPositionRequest: Codable {
    var objectPosition: String?
}

struct CourseGenerateImageRequest: Codable {
    var prompt: String
}

struct CourseGenerateImageResponse: Decodable {
    var imageUrl: String?
}

struct CourseFileUploadResponse: Decodable {
    var id: String
    var contentPath: String
    var mimeType: String?
    var byteSize: Int?
}
