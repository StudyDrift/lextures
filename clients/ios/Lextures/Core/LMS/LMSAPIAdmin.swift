import Foundation

/// Admin settings API (M14.10) — global archived courses.
extension LMSAPI {
    // MARK: - Global platform configuration (M14.6)

    static func fetchPlatformSettings(accessToken: String) async throws -> PlatformSettingsSnapshot {
        async let settingsRequest = client.request(
            path: "/api/v1/settings/platform",
            authorized: true,
            accessToken: accessToken
        )
        async let featuresRequest = client.request(
            path: "/api/v1/platform/features",
            authorized: true,
            accessToken: accessToken
        )
        let ((settingsData, _), (featuresData, _)) = try await (settingsRequest, featuresRequest)
        let settings = try decode(PlatformSettingsSnapshot.self, from: settingsData)
        let features = try decode(PlatformFeatureStates.self, from: featuresData)
        return PlatformSettingsAdminLogic.applyingEffectiveFeatures(features, to: settings)
    }

    static func setPlatformFeature(
        key: String,
        enabled: Bool,
        accessToken: String
    ) async throws -> PlatformSettingsSnapshot {
        guard PlatformSettingsAdminLogic.featureDefinitions.contains(where: { $0.key == key }) else {
            throw APIError.invalidResponse
        }
        let body = try JSONSerialization.data(withJSONObject: [key: enabled, "updateMask": [key]])
        let (data, _) = try await client.requestRaw(
            path: "/api/v1/settings/platform",
            method: "PUT",
            bodyData: body,
            authorized: true,
            accessToken: accessToken
        )
        let persisted = try decode(PlatformSettingsSnapshot.self, from: data)
        let (featuresData, _) = try await client.request(
            path: "/api/v1/platform/features",
            authorized: true,
            accessToken: accessToken
        )
        let features = try decode(PlatformFeatureStates.self, from: featuresData)
        return PlatformSettingsAdminLogic.applyingEffectiveFeatures(features, to: persisted)
    }

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

    static func fetchPeopleStats(accessToken: String) async throws -> PeopleDashboardStats {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/people/stats",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PeopleDashboardStats.self, from: data)
    }

    static func searchPeople(
        query: String = "",
        filter: PeopleListFilter? = nil,
        page: Int,
        perPage: Int,
        accessToken: String
    ) async throws -> PaginatedPeople {
        var parts: [String] = ["page=\(page)", "per_page=\(perPage)"]
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            let encoded = trimmed.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? trimmed
            parts.append("q=\(encoded)")
        }
        if let filter {
            parts.append("filter=\(filter.rawValue)")
        }
        let path = "/api/v1/admin/people?\(parts.joined(separator: "&"))"
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

    // MARK: - Platform courses admin

    static func fetchCoursesStats(accessToken: String) async throws -> CoursesDashboardStats {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/courses/stats",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CoursesDashboardStats.self, from: data)
    }

    static func searchPlatformCourses(
        query: String = "",
        filter: CoursesListFilter? = nil,
        page: Int,
        perPage: Int,
        accessToken: String
    ) async throws -> PaginatedPlatformCourses {
        var parts: [String] = ["page=\(page)", "per_page=\(perPage)"]
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            let encoded = trimmed.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? trimmed
            parts.append("q=\(encoded)")
        }
        if let filter {
            parts.append("filter=\(filter.rawValue)")
        }
        let path = "/api/v1/admin/courses?\(parts.joined(separator: "&"))"
        let (data, response) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PaginatedPlatformCourses.self, from: data)
    }

    @discardableResult
    static func ensurePlatformCourseAdminAccess(courseId: String, accessToken: String) async throws -> PlatformCourseReport {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/courses/\(encodePath(courseId))/access",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PlatformCourseReport.self, from: data)
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

    // MARK: - Org branding & AI admin (M14.5)

    static func fetchOrgBranding(orgId: String, accessToken: String) async throws -> OrgBrandingResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/branding",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OrgBrandingResponse.self, from: data)
    }

    static func putOrgBranding(
        orgId: String,
        body: PutOrgBrandingRequest,
        accessToken: String
    ) async throws -> OrgBrandingResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/branding",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OrgBrandingResponse.self, from: data)
    }

    static func uploadOrgBrandingLogo(
        orgId: String,
        fileName: String,
        mimeType: String,
        fileData: Data,
        accessToken: String
    ) async throws -> OrgBrandingUploadResponse {
        let (data, response) = try await client.uploadMultipart(
            path: "/api/v1/orgs/\(encodePath(orgId))/branding/logo",
            fieldName: "file",
            fileName: fileName,
            mimeType: mimeType,
            fileData: fileData,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(OrgBrandingUploadResponse.self, from: data)
    }

    static func fetchAiConfig(accessToken: String) async throws -> AiConfigResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/ai-config",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiConfigResponse.self, from: data)
    }

    static func putAiConfig(body: PutAiConfigRequest, accessToken: String) async throws -> AiConfigResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/ai-config",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiConfigResponse.self, from: data)
    }

    static func fetchAiProviderSettings(accessToken: String) async throws -> AiProviderSettingsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/ai-settings",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiProviderSettingsResponse.self, from: data)
    }

    static func putAiProviderSettings(
        body: PutAiProviderSettingsRequest,
        accessToken: String
    ) async throws -> AiProviderSettingsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/ai-settings",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiProviderSettingsResponse.self, from: data)
    }

    static func testAiProviderConnection(accessToken: String) async throws -> AiProviderTestResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/ai-settings/test",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiProviderTestResponse.self, from: data)
    }

    // MARK: - Integrations & provisioning admin (M14.8)

    static func fetchPlatformScimEnabled(accessToken: String) async throws -> Bool {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/platform",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PlatformScimFlag.self, from: data).scimEnabled == true
    }

    static func fetchLtiRegistrations(accessToken: String) async throws -> LtiRegistrationsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/lti/registrations",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LtiRegistrationsResponse.self, from: data)
    }

    static func setLtiParentPlatformActive(id: String, active: Bool, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/lti/registrations/\(encodePath(id))",
            method: "PUT",
            body: LtiActiveBody(active: active),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func setLtiExternalToolActive(id: String, active: Bool, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/lti/external-tools/\(encodePath(id))",
            method: "PUT",
            body: LtiActiveBody(active: active),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchScimTokens(institutionId: String, accessToken: String) async throws -> [ScimTokenRow] {
        let encoded = institutionId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? institutionId
        let (data, response) = try await client.request(
            path: "/api/v1/admin/provisioning/scim/tokens?institutionId=\(encoded)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ScimTokensResponse.self, from: data).tokens ?? []
    }

    static func fetchScimEvents(institutionId: String, accessToken: String) async throws -> [ScimEventRow] {
        let encoded = institutionId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? institutionId
        let (data, response) = try await client.request(
            path: "/api/v1/admin/provisioning/scim/events?institutionId=\(encoded)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ScimEventsResponse.self, from: data).events ?? []
    }

    static func fetchAdminCloudProviders(accessToken: String) async throws -> [CloudProviderStatus] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/cloud-providers",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode([CloudProviderStatus].self, from: data)
    }

    static func setCloudProviderEnabled(provider: String, enabled: Bool, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/cloud-providers/\(encodePath(provider))",
            method: "PUT",
            body: CloudProviderEnabledBody(enabled: enabled),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchAdminLrsEndpoints(accessToken: String) async throws -> [LrsEndpointStatus] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/lrs-config",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode([LrsEndpointStatus].self, from: data)
    }

    static func setLrsEndpointEnabled(id: String, enabled: Bool, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/lrs-config/\(encodePath(id))",
            method: "PUT",
            body: LrsEnabledBody(enabled: enabled),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchAdminOerProviders(accessToken: String) async throws -> [OerProviderStatus] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/oer-providers",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode([OerProviderStatus].self, from: data)
    }

    static func setOerProviderEnabled(provider: String, enabled: Bool, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/oer-providers/\(encodePath(provider))",
            method: "PUT",
            body: OerProviderEnabledBody(enabled: enabled),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    // MARK: - AI models, system prompts & reports (M14.7)

    static func fetchAiSettings(accessToken: String) async throws -> AiSettingsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/ai",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiSettingsResponse.self, from: data)
    }

    static func putAiSettings(body: PutAiSettingsRequest, accessToken: String) async throws -> AiSettingsResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/ai",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiSettingsResponse.self, from: data)
    }

    static func fetchAiModels(kind: String, accessToken: String) async throws -> AiModelsListResponse {
        let encoded = encodePath(kind)
        let (data, response) = try await client.request(
            path: "/api/v1/settings/ai/models?kind=\(encoded)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiModelsListResponse.self, from: data)
    }

    static func fetchSystemPrompts(accessToken: String) async throws -> [SystemPromptItem] {
        let (data, response) = try await client.request(
            path: "/api/v1/settings/system-prompts",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(SystemPromptsListResponse.self, from: data).prompts
    }

    static func putSystemPrompt(
        key: String,
        content: String,
        accessToken: String
    ) async throws -> SystemPromptItem {
        let body = PutSystemPromptRequest(content: content)
        let (data, response) = try await client.request(
            path: "/api/v1/settings/system-prompts/\(encodePath(key))",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(SystemPromptItem.self, from: data)
    }

    static func fetchAiReports(
        from: String,
        to: String,
        feature: String? = nil,
        userQuery: String? = nil,
        courseCode: String? = nil,
        accessToken: String
    ) async throws -> AiReportsPayload {
        var items: [URLQueryItem] = [
            URLQueryItem(name: "from", value: from),
            URLQueryItem(name: "to", value: to),
        ]
        if let feature, !feature.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            items.append(URLQueryItem(name: "feature", value: feature))
        }
        if let userQuery, !userQuery.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            items.append(URLQueryItem(name: "userQuery", value: userQuery))
        }
        if let courseCode, !courseCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            items.append(URLQueryItem(name: "courseCode", value: courseCode))
        }
        var components = URLComponents()
        components.queryItems = items
        let qs = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/settings/ai/reports\(qs)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AiReportsPayload.self, from: data)
    }

    // MARK: - Transcripts & advising configuration (M14.9)

    static func fetchAdminTranscriptsConfig(accessToken: String) async throws -> AdminTranscriptsConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/transcripts/config",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdminTranscriptsConfig.self, from: data)
    }

    static func putAdminTranscriptsConfig(
        body: PutAdminTranscriptsConfigRequest,
        accessToken: String
    ) async throws -> AdminTranscriptsConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/transcripts/config",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdminTranscriptsConfig.self, from: data)
    }

    static func fetchAdminTranscriptRequests(accessToken: String) async throws -> [AdminTranscriptRequestRow] {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/transcripts/requests",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdminTranscriptRequestsResponse.self, from: data).requests ?? []
    }

    static func fetchAdminAdvisingConfig(accessToken: String) async throws -> AdminAdvisingConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/advising/config",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdminAdvisingConfig.self, from: data)
    }

    static func postAdminAdvisingConfig(
        body: PutAdminAdvisingConfigRequest,
        accessToken: String
    ) async throws -> AdminAdvisingConfig {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/advising/config",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AdminAdvisingConfig.self, from: data)
    }
}
