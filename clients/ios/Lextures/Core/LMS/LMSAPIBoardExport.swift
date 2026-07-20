import Foundation

/// Board export job REST (MOB.8 / VC.9). Mirrors web `boards-api`.
extension LMSAPI {
    static func createBoardExport(
        courseCode: String,
        boardId: String,
        format: BoardExportFormat,
        includeModeration: Bool = false,
        accessToken: String
    ) async throws -> BoardExportJob {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/export",
            method: "POST",
            body: CreateBoardExportRequest(format: format.rawValue, includeModeration: includeModeration),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardExportJobResponse.self, from: data).job
    }

    static func fetchBoardExportJob(
        courseCode: String,
        boardId: String,
        jobId: String,
        accessToken: String
    ) async throws -> BoardExportJob {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/export/\(encodePath(jobId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BoardExportJob.self, from: data)
    }

    static func downloadBoardExport(
        courseCode: String,
        boardId: String,
        jobId: String,
        accessToken: String
    ) async throws -> Data {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/boards/\(encodePath(boardId))/export/\(encodePath(jobId))/content",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return data
    }
}
