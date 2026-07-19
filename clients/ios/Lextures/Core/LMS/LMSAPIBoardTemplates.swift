import Foundation

/// Board templates & duplication REST (MOB.8 / VC.8). Mirrors web `boards-api`.
extension LMSAPI {
    static func listBoardTemplates(
        scope: String? = nil,
        courseCode: String? = nil,
        query: String? = nil,
        accessToken: String
    ) async throws -> [BoardTemplate] {
        var items: [String] = []
        if let scope, !scope.isEmpty { items.append("scope=\(encodeQuery(scope))") }
        if let courseCode, !courseCode.isEmpty { items.append("courseCode=\(encodeQuery(courseCode))") }
        if let query, !query.isEmpty { items.append("q=\(encodeQuery(query))") }
        var path = "/api/v1/board-templates"
        if !items.isEmpty { path += "?" + items.joined(separator: "&") }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardTemplatesListResponse.self, from: data).templates ?? []
    }

    static func createBoardFromTemplate(
        courseCode: String,
        templateId: String,
        title: String = "",
        description: String = "",
        accessToken: String
    ) async throws -> Board {
        let qs = "from=\(encodeQuery("template:\(templateId)"))"
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards?\(qs)",
            method: "POST",
            body: CreateBoardRequest(title: title, description: description),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(Board.self, from: data)
    }

    static func duplicateBoard(
        targetCourseCode: String,
        sourceBoardId: String,
        mode: BoardCopyMode,
        title: String = "",
        description: String = "",
        accessToken: String
    ) async throws -> BoardCreateResult {
        let qs = "from=\(encodeQuery("board:\(sourceBoardId)"))&mode=\(encodeQuery(mode.rawValue))"
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(targetCourseCode))/boards?\(qs)",
            method: "POST",
            body: CreateBoardRequest(title: title, description: description),
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 202 {
            let body = try decode(BoardCopyJobResponse.self, from: data)
            return .job(body.job)
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return .board(try decode(Board.self, from: data))
    }

    static func fetchBoardCopyJob(
        courseCode: String,
        jobId: String,
        accessToken: String
    ) async throws -> BoardCopyJob {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/board-copy-jobs/\(encodePath(jobId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardCopyJob.self, from: data)
    }

    static func saveBoardAsTemplate(
        courseCode: String,
        boardId: String,
        scope: String,
        title: String = "",
        description: String = "",
        tags: [String] = [],
        includePosts: Bool = false,
        accessToken: String
    ) async throws -> BoardTemplate {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/save-as-template",
            method: "POST",
            body: SaveBoardAsTemplateRequest(
                scope: scope,
                title: title,
                description: description,
                tags: tags,
                includePosts: includePosts
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardTemplate.self, from: data)
    }

    private static func encodeQuery(_ value: String) -> String {
        var allowed = CharacterSet.urlQueryAllowed
        allowed.remove(charactersIn: "&=?+")
        return value.addingPercentEncoding(withAllowedCharacters: allowed) ?? value
    }
}
