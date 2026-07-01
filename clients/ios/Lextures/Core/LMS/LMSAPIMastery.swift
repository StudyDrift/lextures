import Foundation

extension LMSAPI {
    static func fetchStudentMastery(
        courseCode: String,
        enrollmentId: String,
        accessToken: String
    ) async throws -> StudentMasteryRow {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/enrollments/\(encodePath(enrollmentId))/mastery",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(StudentMasteryRow.self, from: data)
    }

    static func fetchMyReportCards(accessToken: String) async throws -> [ReportCardSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/report-cards",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            return []
        }
        return try decode(MyReportCardsResponse.self, from: data).reportCards
    }

    static func fetchReportCardPDF(cardId: String, accessToken: String) async throws -> Data {
        let (data, response) = try await client.request(
            path: "/api/v1/report-cards/\(encodePath(cardId))/pdf",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return data
    }
}
