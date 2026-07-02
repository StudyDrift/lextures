import Foundation

/// Reading log and leveled library catalog (M8.4).
extension LMSAPI {
    static func fetchReadingLogEntries(limit: Int = 100, accessToken: String) async throws -> [ReadingLogEntry] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reading-log?limit=\(max(1, min(limit, 500)))",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 501 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReadingLogListResponse.self, from: data).entries ?? []
    }

    static func createReadingLogEntry(
        body: PostReadingLogBody,
        accessToken: String
    ) async throws -> ReadingLogEntry {
        let (data, response) = try await client.request(
            path: "/api/v1/me/reading-log",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PostReadingLogResponse.self, from: data).entry
    }

    static func fetchLibraryBooks(
        orgId: String,
        filter: LibraryBooksFilter = LibraryBooksFilter(),
        accessToken: String
    ) async throws -> [LibraryBook] {
        var path = "/api/v1/orgs/\(encodePath(orgId))/library"
        var query: [String] = []
        if let min = filter.lexileMin { query.append("lexile_min=\(min)") }
        if let max = filter.lexileMax { query.append("lexile_max=\(max)") }
        if let band = filter.gradeBand?.trimmingCharacters(in: .whitespacesAndNewlines), !band.isEmpty {
            query.append("grade_band=\(encodePath(band))")
        }
        if !query.isEmpty { path += "?" + query.joined(separator: "&") }

        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 501 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LibraryBooksResponse.self, from: data).books ?? []
    }
}