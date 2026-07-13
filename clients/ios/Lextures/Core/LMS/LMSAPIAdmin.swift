import Foundation

/// Admin settings API (M14.10) — global archived courses.
extension LMSAPI {
    static func fetchArchivedCourses(accessToken: String) async throws -> [ArchivedCourseRow] {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/archived-courses",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ArchivedCoursesListResponse.self, from: data).courses
    }

    static func restoreArchivedCourse(courseCode: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/archived-courses/\(encodePath(courseCode))/restore",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func deleteArchivedCoursePermanently(courseCode: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/archived-courses/\(encodePath(courseCode))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    // MARK: - Roles & permissions (M14.2)

    static func fetchRoles(accessToken: String) async throws -> [RoleWithPermissions] {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/roles",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(RolesListResponse.self, from: data).roles
    }

    static func fetchRoleUsers(roleId: String, accessToken: String) async throws -> [RBACUserBrief] {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/roles/\(encodePath(roleId))/users",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(RoleUsersResponse.self, from: data).users
    }

    static func fetchEligibleRoleUsers(
        roleId: String,
        query: String?,
        accessToken: String
    ) async throws -> [RBACUserBrief] {
        var path = "/api/v1/settings/roles/\(encodePath(roleId))/users/eligible"
        if let query, !query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            let encoded = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query
            path += "?q=\(encoded)"
        }
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(RoleUsersResponse.self, from: data).users
    }

    static func addUserToRole(roleId: String, userId: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/roles/\(encodePath(roleId))/users",
            method: "POST",
            body: AddRoleUserRequest(userId: userId),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func removeUserFromRole(roleId: String, userId: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/roles/\(encodePath(roleId))/users/\(encodePath(userId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    // MARK: - People admin (M14.3)

    static func searchPeople(
        query: String,
        page: Int,
        perPage: Int,
        accessToken: String
    ) async throws -> PaginatedPeople {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        let encoded = trimmed.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? trimmed
        let path = "/api/v1/admin/people?q=\(encoded)&page=\(page)&per_page=\(perPage)"
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PaginatedPeople.self, from: data)
    }

    static func fetchPersonReport(userId: String, accessToken: String) async throws -> PersonReport {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/people/\(encodePath(userId))/report",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PersonReport.self, from: data)
    }

    static func invitePerson(_ body: InvitePersonRequest, accessToken: String) async throws -> PersonRow {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/people/invite",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PersonRow.self, from: data)
    }

    static func patchPerson(userId: String, active: Bool, accessToken: String) async throws -> PersonRow {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/people/\(encodePath(userId))",
            method: "PATCH",
            body: PeopleAdminLogic.patchPersonRequest(active: active),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PersonRow.self, from: data)
    }

    static func resendPersonInvite(email: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/auth/forgot-password",
            method: "POST",
            body: PeopleAdminLogic.resendInviteRequest(email: email),
            authorized: false
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    // MARK: - Org structure & terms (M14.4)

    static func fetchAdminOrganizations(accessToken: String) async throws -> [AdminOrgRow] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/orgs?limit=\(OrgStructureAdminLogic.orgListLimit)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdminOrgsListResponse.self, from: data).organizations ?? []
    }

    static func fetchOrgUnitTree(orgId: String, accessToken: String) async throws -> [OrgUnitTreeNode] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/orgs/\(encodePath(orgId))/units/tree",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OrgUnitTreeResponse.self, from: data).tree ?? []
    }

    static func patchOrgUnit(
        orgId: String,
        unitId: String,
        body: PatchOrgUnitRequest,
        accessToken: String
    ) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/orgs/\(encodePath(orgId))/units/\(encodePath(unitId))",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func createAcademicTerm(
        orgId: String,
        body: CreateAcademicTermRequest,
        accessToken: String
    ) async throws -> OrgTerm {
        let (data, response) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/terms",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OrgTerm.self, from: data)
    }

    static func patchAcademicTerm(
        orgId: String,
        termId: String,
        body: PatchAcademicTermRequest,
        accessToken: String
    ) async throws -> OrgTerm {
        let (data, response) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/terms/\(encodePath(termId))",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OrgTerm.self, from: data)
    }
}