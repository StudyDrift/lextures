import Foundation

/// Parent portal and conference booking endpoints (M10.1 / M10.2).
extension LMSAPI {
    static func fetchParentChildren(accessToken: String) async throws -> [ParentChildSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/children",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentChildrenResponse.self, from: data).children ?? []
    }

    static func fetchParentStudentGrades(studentId: String, accessToken: String) async throws -> [ParentCourseGradesRow] {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/students/\(encodePath(studentId))/grades",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentGradesResponse.self, from: data).courses ?? []
    }

    static func fetchParentStudentAssignments(studentId: String, accessToken: String) async throws -> [ParentAssignmentRow] {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/students/\(encodePath(studentId))/assignments",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentAssignmentsResponse.self, from: data).assignments ?? []
    }

    static func fetchParentStudentAttendance(studentId: String, accessToken: String) async throws -> [ParentAttendanceRecord] {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/students/\(encodePath(studentId))/attendance",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentAttendanceResponse.self, from: data).records ?? []
    }

    static func fetchParentStudentBehavior(studentId: String, accessToken: String) async throws -> ParentBehaviorResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/students/\(encodePath(studentId))/behavior",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentBehaviorResponse.self, from: data)
    }

    static func fetchParentWeeklySummary(accessToken: String) async throws -> ParentWeeklySummaryResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/weekly-summary",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentWeeklySummaryResponse.self, from: data)
    }

    static func fetchParentNotificationPrefs(accessToken: String) async throws -> ParentNotificationPrefs {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/notification-prefs",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentNotificationPrefs.self, from: data)
    }

    static func patchParentNotificationPrefs(
        _ body: PatchParentNotificationPrefsBody,
        accessToken: String
    ) async throws -> ParentNotificationPrefs {
        let (data, response) = try await client.request(
            path: "/api/v1/parent/notification-prefs",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ParentNotificationPrefs.self, from: data)
    }

    static func fetchParentConferenceTeachers(studentId: String, accessToken: String) async throws -> [ConferenceTeacher] {
        let query = "studentId=\(encodePath(studentId))"
        let (data, response) = try await client.request(
            path: "/api/v1/parent/conference-teachers?\(query)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ConferenceTeachersResponse.self, from: data).teachers ?? []
    }

    static func fetchConferenceSlots(
        teacherId: String,
        date: String,
        accessToken: String
    ) async throws -> ConferenceSlotsResponse {
        let query = "date=\(encodePath(date))"
        let (data, response) = try await client.request(
            path: "/api/v1/teachers/\(encodePath(teacherId))/conference-slots?\(query)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ConferenceSlotsResponse.self, from: data)
    }

    static func bookConferenceSlot(
        slotId: String,
        studentId: String,
        accessToken: String
    ) async throws -> ConferenceSlot {
        let (data, response) = try await client.request(
            path: "/api/v1/conference-slots/\(encodePath(slotId))/book",
            method: "POST",
            body: BookConferenceSlotBody(studentId: studentId),
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 409 {
            throw APIError.httpStatus(409, message: L.text("mobile.parent.conferences.conflict"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        guard let slot = try decode(ConferenceSlotResponse.self, from: data).slot else {
            throw APIError.decoding(DecodingError.dataCorrupted(.init(codingPath: [], debugDescription: "Missing slot")))
        }
        return slot
    }

    static func cancelConferenceBooking(slotId: String, accessToken: String) async throws -> ConferenceSlot {
        let (data, response) = try await client.request(
            path: "/api/v1/conference-slots/\(encodePath(slotId))/book",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        guard let slot = try decode(ConferenceSlotResponse.self, from: data).slot else {
            throw APIError.decoding(DecodingError.dataCorrupted(.init(codingPath: [], debugDescription: "Missing slot")))
        }
        return slot
    }
}
