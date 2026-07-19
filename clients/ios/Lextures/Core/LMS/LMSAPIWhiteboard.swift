import Foundation

/// Course whiteboard create / update / delete (MOB.6). List + get remain in LMSAPIFeatures.
extension LMSAPI {
    static func createCourseWhiteboard(
        courseCode: String,
        title: String,
        canvasData: [WhiteboardElement],
        accessToken: String
    ) async throws -> CourseWhiteboard {
        let bodyData = try whiteboardUpsertBody(title: title, canvasData: canvasData)
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/whiteboards",
            method: "POST",
            bodyData: bodyData,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        WhiteboardObservability.record("whiteboard_created")
        return try decode(CourseWhiteboard.self, from: data)
    }

    static func updateCourseWhiteboard(
        courseCode: String,
        boardId: String,
        title: String,
        canvasData: [WhiteboardElement],
        accessToken: String
    ) async throws -> CourseWhiteboard {
        let bodyData = try whiteboardUpsertBody(title: title, canvasData: canvasData)
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/whiteboards/\(encodePath(boardId))",
            method: "PUT",
            bodyData: bodyData,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        WhiteboardObservability.record("whiteboard_saved")
        return try decode(CourseWhiteboard.self, from: data)
    }

    static func deleteCourseWhiteboard(
        courseCode: String,
        boardId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/whiteboards/\(encodePath(boardId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        WhiteboardObservability.record("whiteboard_deleted")
    }

    private static func whiteboardUpsertBody(title: String, canvasData: [WhiteboardElement]) throws -> Data {
        let payload: [String: Any] = [
            "title": WhiteboardLogic.normalizeTitle(title),
            "canvasData": WhiteboardLogic.serializeElements(canvasData),
        ]
        return try JSONSerialization.data(withJSONObject: payload)
    }
}
