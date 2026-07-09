import Foundation

/// Course translations / glossary API (M13.9).
extension LMSAPI {
    static func fetchTranslationLocales(
        courseCode: String,
        accessToken: String
    ) async throws -> [TranslationCoverage] {
        let (data, response) = try await client.requestRaw(
            path: CourseTranslationsLogic.coveragePath(courseCode: courseCode),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        if let wrapped = try? decode(TranslationLocalesResponse.self, from: data) {
            return wrapped.locales
        }
        // Single-locale shape is unexpected without target_locale; treat as empty list.
        return []
    }

    static func fetchTranslationCoverage(
        courseCode: String,
        targetLocale: String,
        accessToken: String
    ) async throws -> TranslationCoverage {
        let (data, response) = try await client.requestRaw(
            path: CourseTranslationsLogic.coveragePath(
                courseCode: courseCode,
                targetLocale: targetLocale
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(TranslationCoverage.self, from: data)
    }

    static func fetchCourseTranslations(
        courseCode: String,
        targetLocale: String,
        accessToken: String
    ) async throws -> CourseTranslationListResponse {
        let (data, response) = try await client.requestRaw(
            path: CourseTranslationsLogic.translationsPath(
                courseCode: courseCode,
                targetLocale: targetLocale
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseTranslationListResponse.self, from: data)
    }

    static func fetchCourseGlossary(
        courseCode: String,
        targetLocale: String,
        sourceLocale: String = CourseTranslationsLogic.defaultSourceLocale,
        accessToken: String
    ) async throws -> [CourseGlossaryEntry] {
        var components = URLComponents()
        components.queryItems = [
            URLQueryItem(name: "target_locale", value: targetLocale),
            URLQueryItem(name: "source_locale", value: sourceLocale),
        ]
        let query = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, response) = try await client.requestRaw(
            path: "\(CourseTranslationsLogic.glossaryPath(courseCode: courseCode))\(query)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let wrapped = try decode(CourseGlossaryListResponse.self, from: data)
        return wrapped.entries
    }

    static func addGlossaryEntry(
        courseCode: String,
        body: AddGlossaryEntryBody,
        accessToken: String
    ) async throws -> CourseGlossaryEntry {
        let (data, response) = try await client.requestRaw(
            path: CourseTranslationsLogic.glossaryPath(courseCode: courseCode),
            method: "POST",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseGlossaryEntry.self, from: data)
    }
}
