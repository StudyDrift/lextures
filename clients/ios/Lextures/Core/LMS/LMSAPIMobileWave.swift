import Foundation

/// Mobile wave APIs: permissions, profile depth, attendance, search, and library (M0.6, M1.5, M3.6, M11.1).
extension LMSAPI {
    static func fetchMyPermissions(accessToken: String) async throws -> [String] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/permissions",
            authorized: true,
            accessToken: accessToken
        )
        let response = try decode(MyPermissionsResponse.self, from: data)
        return response.permissionStrings
    }

    // MARK: - Profile depth (M1.5)

    static func fetchMyProfileFields(accessToken: String) async throws -> ProfileFieldsResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/me/profile-fields",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ProfileFieldsResponse.self, from: data)
    }

    static func updateMyProfileFields(
        _ patch: ProfileFieldsPatch,
        accessToken: String
    ) async throws -> [String: JSONValue] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/profile-fields",
            method: "PATCH",
            body: patch,
            authorized: true,
            accessToken: accessToken
        )
        let response = try decode(ProfileFieldsValuesResponse.self, from: data)
        return response.values
    }

    static func fetchMyDemographics(accessToken: String) async throws -> StudentDemographics {
        let (data, _) = try await client.request(
            path: "/api/v1/me/demographics",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(StudentDemographics.self, from: data)
    }

    static func updateMyDemographics(
        _ patch: StudentDemographicsPatch,
        accessToken: String
    ) async throws -> StudentDemographics {
        let (data, _) = try await client.request(
            path: "/api/v1/me/demographics",
            method: "PATCH",
            body: patch,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(StudentDemographics.self, from: data)
    }

    static func fetchPendingConsentStudies(accessToken: String) async throws -> [ConsentStudy] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/consent-studies",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ConsentStudiesResponse.self, from: data).studies
    }

    static func fetchConsentHistory(accessToken: String) async throws -> [ConsentHistoryEntry] {
        let (data, _) = try await client.request(
            path: "/api/v1/me/consent-studies/history",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ConsentHistoryResponse.self, from: data).history
    }

    static func respondToConsentStudy(
        studyId: String,
        decision: ConsentDecision,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/me/consent-studies/\(encodePath(studyId))/respond",
            method: "POST",
            body: ConsentRespondBody(decision: decision),
            authorized: true,
            accessToken: accessToken
        )
    }

    // MARK: - Take attendance (M11.1)

    static func createAttendanceSession(
        courseCode: String,
        body: CreateAttendanceSessionBody,
        accessToken: String
    ) async throws -> AttendanceSession {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/attendance/sessions",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AttendanceSession.self, from: data)
    }

    static func saveAttendanceRecords(
        courseCode: String,
        sessionId: String,
        records: [AttendanceRecordUpsert],
        accessToken: String
    ) async throws -> SaveAttendanceRecordsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/attendance/sessions/\(encodePath(sessionId))/records",
            method: "PUT",
            body: SaveAttendanceRecordsBody(records: records),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(SaveAttendanceRecordsResponse.self, from: data)
    }

    struct CloseAttendanceSessionBody: Encodable {
        var finalizeMissingAsAbsent: Bool = true
    }

    static func closeAttendanceSession(
        courseCode: String,
        sessionId: String,
        accessToken: String
    ) async throws -> AttendanceSession {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/attendance/sessions/\(encodePath(sessionId))/close",
            method: "POST",
            body: CloseAttendanceSessionBody(),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AttendanceSession.self, from: data)
    }

    static func fetchCourseSections(courseCode: String, accessToken: String) async throws -> [CourseSection] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/sections",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseSectionsResponse.self, from: data).sections
    }

    // MARK: - Course roster (M11.4)

    static func fetchCourseEnrollments(courseCode: String, accessToken: String) async throws -> [CourseEnrollment] {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/enrollments",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseEnrollmentsResponse.self, from: data).enrollments
    }

    static func removeCourseEnrollment(
        courseCode: String,
        enrollmentId: String,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/enrollments/\(encodePath(enrollmentId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func sendEnrollmentMessage(
        courseCode: String,
        enrollmentId: String,
        body: EnrollmentMessageBody,
        accessToken: String
    ) async throws -> String {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/enrollments/\(encodePath(enrollmentId))/message",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(EnrollmentMessageResponse.self, from: data).id ?? ""
    }

    // MARK: - Universal search (M0.6)

    static func fetchSearchIndex(accessToken: String) async throws -> SearchIndexResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/search",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SearchIndexResponse.self, from: data)
    }

    static func fetchSearchQuery(
        query: String,
        scope: String? = nil,
        accessToken: String
    ) async throws -> SearchQueryResponse {
        var path = "/api/v1/search/query?q=\(encodePath(query))"
        if let scope, !scope.isEmpty {
            path += "&scope=\(encodePath(scope))"
        }
        let (data, _) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SearchQueryResponse.self, from: data)
    }

    // MARK: - Library & OER (M3.6)

    static func searchLibraryCatalog(query: String, accessToken: String) async throws -> [LibraryCatalogResult] {
        let trimmedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
        let (data, _) = try await client.request(
            path: "/api/v1/library/search?q=\(encodePath(trimmedQuery))",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(LibrarySearchResponse.self, from: data).results
    }

    static func fetchOERProviders(accessToken: String) async throws -> [String] {
        let (data, _) = try await client.request(
            path: "/api/v1/oer/providers",
            authorized: true,
            accessToken: accessToken
        )
        let rows = try decode([OERProviderRow].self, from: data)
        return rows.map(\.provider)
    }

    static func searchOER(
        provider: String,
        query: String,
        accessToken: String
    ) async throws -> OERSearchResponse {
        var path = "/api/v1/oer/search?provider=\(encodePath(provider))"
        let trimmedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmedQuery.isEmpty {
            path += "&q=\(encodePath(trimmedQuery))"
        }
        let (data, _) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(OERSearchResponse.self, from: data)
    }

    static func fetchModuleLibraryResource(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> LibraryResourcePayload? {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/library-resources/\(encodePath(itemId))",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        return try decode(LibraryResourcePayload.self, from: data)
    }

    static func recordLibraryResourceAccess(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/library-resources/\(encodePath(itemId))/access",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
    }
}