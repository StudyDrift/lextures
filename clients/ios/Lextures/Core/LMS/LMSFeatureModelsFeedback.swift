import Foundation

// MARK: - Product feedback (FB3)

struct FeedbackContextPayload: Encodable {
    var route: String
    var locale: String?
    var viewport: String?

    enum CodingKeys: String, CodingKey {
        case route
        case locale
        case viewport
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(route, forKey: .route)
        if let locale, !locale.isEmpty {
            try container.encode(locale, forKey: .locale)
        }
        if let viewport, !viewport.isEmpty {
            try container.encode(viewport, forKey: .viewport)
        }
    }
}

struct SubmitFeedbackRequest: Encodable {
    var message: String
    var source: String
    var appVersion: String
    var context: FeedbackContextPayload
    var category: String?

    enum CodingKeys: String, CodingKey {
        case message
        case source
        case appVersion = "app_version"
        case context
        case category
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(message, forKey: .message)
        try container.encode(source, forKey: .source)
        try container.encode(appVersion, forKey: .appVersion)
        try container.encode(context, forKey: .context)
        if let category, !category.isEmpty {
            try container.encode(category, forKey: .category)
        }
    }
}

struct SubmitFeedbackResponse: Decodable {
    var id: String
    var createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case createdAt = "created_at"
    }
}
