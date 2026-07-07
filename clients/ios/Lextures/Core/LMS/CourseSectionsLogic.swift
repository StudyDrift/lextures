import Foundation

/// Course sections, roster assignment, overrides, and cross-listing helpers (M13.3).
enum CourseSectionsLogic {
    static let globalAdminPermission = "global:app:rbac:manage"
    static let orgUnitsAdminPermission = "tenant:org:units:admin"

    static func canManageCrossListing(permissions: [String]) -> Bool {
        permissions.contains(globalAdminPermission)
            || permissions.contains(orgUnitsAdminPermission)
    }

    static func canAssignStudents(courseCode: String, permissions: [String]) -> Bool {
        permissions.contains("course:\(courseCode):enrollments:update")
    }

    static func shouldShowEditors(sectionsEnabled: Bool) -> Bool {
        sectionsEnabled
    }

    static func activeSections(_ sections: [CourseSection]) -> [CourseSection] {
        sections.filter(\.isActive)
    }

    static func rosterCount(sectionId: String, enrollments: [CourseEnrollment]) -> Int {
        enrollments.filter { enrollment in
            enrollment.sectionId == sectionId && CoursePeopleLogic.isStudentRole(enrollment.role)
        }.count
    }

    static func assignmentItems(from structure: [CourseStructureItem]) -> [CourseStructureItem] {
        structure.filter { $0.kind == "assignment" }
    }

    static func crossListGroup(
        for courseId: String,
        groups: [CrossListGroup]
    ) -> CrossListGroup? {
        groups.first { $0.courseId == courseId }
    }

    static func crossListAddCandidates(
        activeSections: [CourseSection],
        group: CrossListGroup?
    ) -> [CourseSection] {
        let memberIds = Set((group?.members ?? []).map(\.sectionId))
        return activeSections.filter { !memberIds.contains($0.id) }
    }

    static func cacheKeySections(courseCode: String) -> String {
        "course:\(courseCode):sections-settings"
    }

    static func createSectionIdempotencyKey(courseCode: String, sectionCode: String) -> String {
        "course-sections:\(courseCode):create:\(sectionCode.lowercased())"
    }

    static func patchSectionIdempotencyKey(courseCode: String, sectionId: String) -> String {
        "course-sections:\(courseCode):patch:\(sectionId)"
    }

    static func archiveSectionIdempotencyKey(courseCode: String, sectionId: String) -> String {
        "course-sections:\(courseCode):archive:\(sectionId)"
    }

    static func overrideIdempotencyKey(sectionId: String, itemId: String) -> String {
        "section-override:\(sectionId):\(itemId)"
    }

    static func enrollmentSectionIdempotencyKey(enrollmentId: String, sectionId: String) -> String {
        "enrollment-section:\(enrollmentId):\(sectionId)"
    }

    static func crossListCreateIdempotencyKey(orgId: String, courseCode: String) -> String {
        "cross-list:\(orgId):\(courseCode):create"
    }

    static func crossListAddMemberIdempotencyKey(orgId: String, groupId: String, sectionId: String) -> String {
        "cross-list:\(orgId):\(groupId):add:\(sectionId)"
    }

    static func crossListRemoveMemberIdempotencyKey(orgId: String, groupId: String, sectionId: String) -> String {
        "cross-list:\(orgId):\(groupId):remove:\(sectionId)"
    }

    static func buildOverrideBody(dueAtLocal: String) -> SectionAssignmentOverrideBody? {
        let trimmed = dueAtLocal.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return SectionAssignmentOverrideBody(dueAt: nil, availableFrom: nil, availableUntil: nil)
        }
        guard let iso = CourseSettingsLogic.localDateStringToIso(trimmed) else { return nil }
        return SectionAssignmentOverrideBody(dueAt: iso, availableFrom: nil, availableUntil: nil)
    }

    static func validateCreateSection(sectionCode: String) -> String? {
        sectionCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            ? L.text("mobile.courseSettings.sections.validation.sectionCodeRequired")
            : nil
    }

    static func mutationsDisabledReason(isOnline: Bool) -> String? {
        guard !isOnline else { return nil }
        return L.text("mobile.courseSettings.sections.offlineMutationsDisabled")
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError {
            return apiError.errorDescription ?? L.text("mobile.courseSettings.sections.genericError")
        }
        return error.localizedDescription
    }
}

struct CrossListMember: Codable, Identifiable, Hashable {
    var sectionId: String
    var isPrimary: Bool
    var sectionCode: String
    var sectionName: String?

    var id: String { sectionId }

    var displayLabel: String {
        let name = sectionName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return "\(sectionCode) — \(name)" }
        return sectionCode
    }
}

struct CrossListGroup: Codable, Identifiable, Hashable {
    var id: String
    var courseId: String
    var name: String?
    var createdAt: String?
    var primarySectionId: String?
    var members: [CrossListMember]
}

struct CrossListGroupsResponse: Codable {
    var groups: [CrossListGroup]?
}

struct CreateCourseSectionBody: Encodable {
    var sectionCode: String
    var name: String?
}

struct PatchCourseSectionBody: Encodable {
    var sectionCode: String?
    var name: String?
    var status: String?
}

struct SectionAssignmentOverrideBody: Encodable {
    var dueAt: String?
    var availableFrom: String?
    var availableUntil: String?
}

struct CreateCrossListGroupBody: Encodable {
    var courseCode: String
    var primarySectionId: String
    var name: String?
}

struct AddCrossListMemberBody: Encodable {
    var sectionId: String
}

struct EnrollmentSectionPatchBody: Encodable {
    var sectionId: String
}

struct CourseSectionsCachedPayload: Codable {
    var sections: [CourseSection]
    var enrollments: [CourseEnrollment]
    var assignments: [CourseStructureItem]
}
