import Foundation

/// Immersive reader endpoints: reading preferences, captions, translation (M6.3).
extension LMSAPI {
    static func fetchReadingPreferences(accessToken: String) async throws -> ReadingPreferencesRow {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reading-preferences",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReadingPreferencesRow.self, from: data)
    }

    static func patchReadingPreferences(
        _ patch: ReadingPreferencesPatch,
        accessToken: String
    ) async throws -> ReadingPreferencesRow {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reading-preferences",
            method: "PATCH",
            body: patch,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReadingPreferencesRow.self, from: data)
    }

    static func fetchCaptions(objectId: String, accessToken: String) async throws -> [CaptionRecord] {
        let (data, response) = try await client.request(
            path: "/api/v1/files/\(encodePath(objectId))/captions",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode([CaptionRecord].self, from: data)
    }

    static func fetchCaptionVtt(
        objectId: String,
        captionId: String,
        accessToken: String
    ) async throws -> String {
        let (data, response) = try await client.request(
            path: "/api/v1/files/\(encodePath(objectId))/captions/\(encodePath(captionId))/vtt",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return String(data: data, encoding: .utf8) ?? ""
    }

    static func translateContent(
        contentType: String,
        contentId: String,
        targetLang: String,
        text: String,
        accessToken: String
    ) async throws -> TranslateContentResponse {
        let body = TranslateContentRequest(
            contentType: contentType,
            contentId: contentId,
            targetLang: targetLang,
            text: text
        )
        let (data, response) = try await client.request(
            path: "/api/v1/translate",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(TranslateContentResponse.self, from: data)
    }

    static func fetchTranslationCoverage(
        courseCode: String,
        accessToken: String
    ) async throws -> [TranslationCoverageLocale] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/translation-coverage",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let decoded = try decode(TranslationCoverageResponse.self, from: data)
        if let locales = decoded.locales { return locales }
        if let locale = decoded.targetLocale, let percent = decoded.percent {
            return [TranslationCoverageLocale(targetLocale: locale, percent: percent)]
        }
        return []
    }

    static func patchMyContentLocale(
        courseCode: String,
        locale: String?,
        accessToken: String
    ) async throws {
        let body = PatchContentLocaleBody(contentLocale: locale)
        let (_, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/me/content-locale",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: nil)
        }
    }
}