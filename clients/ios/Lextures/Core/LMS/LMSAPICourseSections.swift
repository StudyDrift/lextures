import Foundation

/// Course sections, overrides, enrollment transfer, and cross-listing API (M13.3).
extension LMSAPI {
    static func postCourseSection(
        courseCode: String,
        body: CreateCourseSectionBody,
        accessToken: String
    ) async throws -> CourseSection {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/sections",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSection.self, from: data)
    }

    static func patchCourseSection(
        courseCode: String,
        sectionId: String,
        body: PatchCourseSectionBody,
        accessToken: String
    ) async throws -> CourseSection {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/sections/\(encodePath(sectionId))",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSection.self, from: data)
    }

    static func deleteCourseSection(
        courseCode: String,
        sectionId: String,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/sections/\(encodePath(sectionId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
    }

    static func putSectionAssignmentOverride(
        sectionId: String,
        itemId: String,
        body: SectionAssignmentOverrideBody,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/sections/\(encodePath(sectionId))/overrides/\(encodePath(itemId))",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
    }

    static func patchEnrollmentSection(
        enrollmentId: String,
        sectionId: String,
        accessToken: String
    ) async throws {
        _ = try await client.request(
            path: "/api/v1/enrollments/\(encodePath(enrollmentId))/section",
            method: "PATCH",
            body: EnrollmentSectionPatchBody(sectionId: sectionId),
            authorized: true,
            accessToken: accessToken
        )
    }

    static func fetchOrgCrossListGroups(orgId: String, accessToken: String) async throws -> [CrossListGroup] {
        let (data, _) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/cross-list-groups",
            authorized: true,
            accessToken: accessToken
        )
        let response = try decode(CrossListGroupsResponse.self, from: data)
        return response.groups ?? []
    }

    static func postOrgCrossListGroup(
        orgId: String,
        body: CreateCrossListGroupBody,
        accessToken: String
    ) async throws -> CrossListGroup {
        let (data, _) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/cross-list-groups",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CrossListGroup.self, from: data)
    }

    static func postOrgCrossListMember(
        orgId: String,
        groupId: String,
        sectionId: String,
        accessToken: String
    ) async throws -> CrossListGroup {
        let (data, _) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/cross-list-groups/\(encodePath(groupId))/members",
            method: "POST",
            body: AddCrossListMemberBody(sectionId: sectionId),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CrossListGroup.self, from: data)
    }

    static func deleteOrgCrossListMember(
        orgId: String,
        groupId: String,
        sectionId: String,
        accessToken: String
    ) async throws -> CrossListGroup? {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/orgs/\(encodePath(orgId))/cross-list-groups/\(encodePath(groupId))/members/\(encodePath(sectionId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 204 || data.isEmpty { return nil }
        if let removed = try? decode([String: Bool].self, from: data), removed["removed"] == true {
            return nil
        }
        return try decode(CrossListGroup.self, from: data)
    }

    static func fetchCourseSectionsPayload(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseSectionsCachedPayload {
        async let sections = fetchCourseSections(courseCode: courseCode, accessToken: accessToken)
        async let enrollments = fetchCourseEnrollments(courseCode: courseCode, accessToken: accessToken)
        async let structure = fetchCourseStructure(courseCode: courseCode, accessToken: accessToken)
        return try await CourseSectionsCachedPayload(
            sections: sections,
            enrollments: enrollments,
            assignments: CourseSectionsLogic.assignmentItems(from: structure)
        )
    }
}
