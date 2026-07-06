import Foundation

/// Behavior / PBIS and hall pass endpoints (M10.3).
extension LMSAPI {
    static func listBehaviorCategories(orgId: String, accessToken: String) async throws -> [BehaviorCategory] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/orgs/\(encodePath(orgId))/behavior/categories",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BehaviorCategoriesResponse.self, from: data).categories ?? []
    }

    static func awardPBISPoints(_ awards: [PBISAwardInput], accessToken: String) async throws -> PBISAwardsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/pbis/awards",
            method: "POST",
            body: PBISAwardsBody(awards: awards),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PBISAwardsResponse.self, from: data)
    }

    static func fileBehaviorReferral(_ body: BehaviorReferralBody, accessToken: String) async throws -> BehaviorReferral {
        let (data, response) = try await client.request(
            path: "/api/v1/behavior/referrals",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(BehaviorReferral.self, from: data)
    }

    static func fetchStudentBehavior(studentId: String, accessToken: String) async throws -> StudentBehaviorResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/students/\(encodePath(studentId))/behavior",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(StudentBehaviorResponse.self, from: data)
    }

    static func requestHallPass(
        sectionId: String,
        destination: String,
        estimatedMins: Int,
        accessToken: String
    ) async throws -> HallPass {
        let (data, response) = try await client.request(
            path: "/api/v1/sections/\(encodePath(sectionId))/hall-passes",
            method: "POST",
            body: RequestHallPassBody(destination: destination, estimatedMins: estimatedMins),
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 501 {
            throw APIError.httpStatus(501, message: L.text("mobile.hallpass.disabled"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        guard let pass = try decode(HallPassResponse.self, from: data).pass else {
            throw APIError.decoding(DecodingError.dataCorrupted(.init(codingPath: [], debugDescription: "Missing pass")))
        }
        return pass
    }

    static func fetchActiveHallPasses(sectionId: String, accessToken: String) async throws -> [HallPass] {
        let (data, response) = try await client.request(
            path: "/api/v1/sections/\(encodePath(sectionId))/hall-passes/active",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 501 {
            throw APIError.httpStatus(501, message: L.text("mobile.hallpass.disabled"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ActiveHallPassesResponse.self, from: data).passes ?? []
    }

    static func updateHallPass(passId: String, status: String, accessToken: String) async throws -> HallPass {
        let (data, response) = try await client.request(
            path: "/api/v1/hall-passes/\(encodePath(passId))",
            method: "PATCH",
            body: UpdateHallPassBody(status: status),
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 501 {
            throw APIError.httpStatus(501, message: L.text("mobile.hallpass.disabled"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        guard let pass = try decode(HallPassResponse.self, from: data).pass else {
            throw APIError.decoding(DecodingError.dataCorrupted(.init(codingPath: [], debugDescription: "Missing pass")))
        }
        return pass
    }
}
