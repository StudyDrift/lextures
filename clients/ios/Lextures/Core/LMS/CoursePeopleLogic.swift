import Foundation

enum CoursePeopleRoleFilter: String, CaseIterable, Hashable {
    case all
    case staff
    case students
}

enum CoursePeopleGroupKind: String, CaseIterable, Hashable {
    case teachers
    case tas
    case students
    case other
}

struct CoursePeopleGroup: Identifiable, Hashable {
    var kind: CoursePeopleGroupKind
    var enrollments: [CourseEnrollment]

    var id: String { kind.rawValue }
}

struct CoursePeopleAssignableRole: Identifiable, Hashable {
    var value: String
    var labelKey: String

    var id: String { value }
}

struct CoursePeopleAddResultSummary: Equatable {
    var added: [String]
    var alreadyEnrolled: [String]
    var notFound: [String]

    var hasConflicts: Bool { !alreadyEnrolled.isEmpty || !notFound.isEmpty }
    var didAdd: Bool { !added.isEmpty }
}

enum CoursePeopleLogic {
    static let assignableRoles: [CoursePeopleAssignableRole] = [
        .init(value: "student", labelKey: "mobile.people.role.student"),
        .init(value: "instructor", labelKey: "mobile.people.role.teacher"),
        .init(value: "ta", labelKey: "mobile.people.role.ta"),
        .init(value: "designer", labelKey: "mobile.people.add.role.designer"),
        .init(value: "observer", labelKey: "mobile.people.add.role.observer"),
        .init(value: "auditor", labelKey: "mobile.people.add.role.auditor"),
        .init(value: "librarian", labelKey: "mobile.people.add.role.librarian"),
    ]

    static let managedEnrollmentStates = ["active", "dropped", "withdrawn", "waitlist", "audit", "no_credit", "incomplete"]
    static func normalizedRole(_ role: String) -> String {
        role.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    static func enrollmentRoleRank(_ role: String) -> Int {
        switch normalizedRole(role) {
        case "owner", "teacher": return 0
        case "instructor": return 1
        case "ta": return 2
        case "designer": return 3
        case "observer": return 4
        case "auditor": return 5
        case "librarian": return 6
        case "student": return 7
        default: return 8
        }
    }

    static func isStaffRole(_ role: String) -> Bool {
        enrollmentRoleRank(role) < 7
    }

    static func isStudentRole(_ role: String) -> Bool {
        normalizedRole(role) == "student"
    }

    static func groupKind(for role: String) -> CoursePeopleGroupKind {
        switch normalizedRole(role) {
        case "owner", "teacher", "instructor": return .teachers
        case "ta": return .tas
        case "student": return .students
        default: return .other
        }
    }

    static func displayName(_ enrollment: CourseEnrollment) -> String {
        let trimmed = enrollment.displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !trimmed.isEmpty { return trimmed }
        return L.text("mobile.people.unnamed")
    }

    static func initials(_ enrollment: CourseEnrollment) -> String {
        let source = displayName(enrollment)
        let parts = source.split(separator: " ").filter { !$0.isEmpty }
        if parts.count >= 2 {
            return "\(parts[0].prefix(1))\(parts[parts.count - 1].prefix(1))".uppercased()
        }
        return String(source.prefix(2)).uppercased()
    }

    static func roleLabel(_ enrollment: CourseEnrollment) -> String {
        let custom = enrollment.roleDisplay?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !custom.isEmpty { return custom }
        switch normalizedRole(enrollment.role) {
        case "owner", "teacher", "instructor": return L.text("mobile.people.role.teacher")
        case "ta": return L.text("mobile.people.role.ta")
        case "student": return L.text("mobile.people.role.student")
        default: return enrollment.role
        }
    }

    static func sectionLabel(_ enrollment: CourseEnrollment) -> String? {
        let name = enrollment.sectionName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !name.isEmpty { return name }
        let code = enrollment.sectionCode?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !code.isEmpty { return code }
        return nil
    }

    static func matchesSearch(_ enrollment: CourseEnrollment, query: String) -> Bool {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !needle.isEmpty else { return true }
        let haystack = [
            displayName(enrollment),
            roleLabel(enrollment),
            sectionLabel(enrollment) ?? "",
            enrollment.role,
        ].joined(separator: " ").lowercased()
        return haystack.contains(needle)
    }

    static func filter(
        enrollments: [CourseEnrollment],
        search: String,
        roleFilter: CoursePeopleRoleFilter,
        sectionId: String?
    ) -> [CourseEnrollment] {
        enrollments.filter { enrollment in
            guard matchesSearch(enrollment, query: search) else { return false }
            switch roleFilter {
            case .all: break
            case .staff:
                guard isStaffRole(enrollment.role) else { return false }
            case .students:
                guard isStudentRole(enrollment.role) else { return false }
            }
            if let sectionId, !sectionId.isEmpty {
                guard enrollment.sectionId == sectionId else { return false }
            }
            return true
        }
    }

    static func groupedSections(from enrollments: [CourseEnrollment]) -> [CoursePeopleGroup] {
        var buckets: [CoursePeopleGroupKind: [CourseEnrollment]] = [:]
        for enrollment in enrollments {
            let kind = groupKind(for: enrollment.role)
            buckets[kind, default: []].append(enrollment)
        }
        return CoursePeopleGroupKind.allCases.compactMap { kind in
            guard let rows = buckets[kind], !rows.isEmpty else { return nil }
            let sorted = rows.sorted {
                let left = displayName($0).lowercased()
                let right = displayName($1).lowercased()
                if left == right {
                    return enrollmentRoleRank($0.role) < enrollmentRoleRank($1.role)
                }
                return left < right
            }
            return CoursePeopleGroup(kind: kind, enrollments: sorted)
        }
    }

    static func canUpdateEnrollments(courseCode: String, permissions: [String]) -> Bool {
        permissions.contains("course:\(courseCode):enrollments:update")
    }

    static func enrollmentAddEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileEnrollmentAdd
    }

    static func canAddEnrollments(
        courseCode: String,
        permissions: [String],
        features: MobilePlatformFeatures,
        isOnline: Bool
    ) -> Bool {
        guard enrollmentAddEnabled(features) else { return false }
        guard isOnline else { return false }
        return canUpdateEnrollments(courseCode: courseCode, permissions: permissions)
    }

    static func canChangeEnrollmentState(
        enrollment: CourseEnrollment,
        courseCode: String,
        permissions: [String],
        features: MobilePlatformFeatures,
        isOnline: Bool
    ) -> Bool {
        guard features.ffEnrollmentStateMachine else { return false }
        guard isOnline else { return false }
        guard canUpdateEnrollments(courseCode: courseCode, permissions: permissions) else { return false }
        return isStudentRole(enrollment.role)
    }

    static func parseEmails(_ raw: String) -> [String] {
        let separators = CharacterSet(charactersIn: ",;\n\r\t ")
        var seen = Set<String>()
        var emails: [String] = []
        for part in raw.components(separatedBy: separators) {
            let email = part.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            guard !email.isEmpty else { continue }
            guard seen.insert(email).inserted else { continue }
            emails.append(email)
        }
        return emails
    }

    static func isValidEmail(_ email: String) -> Bool {
        let trimmed = email.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 3, trimmed.count <= 254 else { return false }
        guard let at = trimmed.firstIndex(of: "@") else { return false }
        let local = trimmed[..<at]
        let domain = trimmed[trimmed.index(after: at)...]
        guard !local.isEmpty, !domain.isEmpty else { return false }
        guard domain.contains(".") else { return false }
        guard !local.contains("@"), !domain.contains("@") else { return false }
        return true
    }

    enum AddEmailValidation: Error, Equatable {
        case emailsRequired
        case invalidEmail
        case ok([String])

        var errorKey: String? {
            switch self {
            case .emailsRequired: return "mobile.people.add.error.emailsRequired"
            case .invalidEmail: return "mobile.people.add.error.invalidEmail"
            case .ok: return nil
            }
        }
    }

    static func validateEmailsForAdd(_ raw: String) -> AddEmailValidation {
        let emails = parseEmails(raw)
        guard !emails.isEmpty else { return .emailsRequired }
        let invalid = emails.filter { !isValidEmail($0) }
        if !invalid.isEmpty {
            return .invalidEmail
        }
        return .ok(emails)
    }

    static func normalizeCourseRole(_ role: String) -> String {
        let value = role.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if value == "teacher" || value == "owner" { return "instructor" }
        return value
    }

    static func isAssignableRole(_ role: String) -> Bool {
        let normalized = normalizeCourseRole(role)
        return assignableRoles.contains { $0.value == normalized }
    }

    static func buildAddRequest(emails: [String], courseRole: String) -> AddCourseEnrollmentsRequest {
        AddCourseEnrollmentsRequest(
            emails: emails.joined(separator: "\n"),
            courseRole: normalizeCourseRole(courseRole)
        )
    }

    static func summarizeAddResponse(_ response: AddCourseEnrollmentsResponse) -> CoursePeopleAddResultSummary {
        CoursePeopleAddResultSummary(
            added: response.added ?? [],
            alreadyEnrolled: response.alreadyEnrolled ?? [],
            notFound: response.notFound ?? []
        )
    }

    static func normalizedState(_ state: String?) -> String {
        let value = state?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        return value.isEmpty ? "active" : value
    }

    static func isInactiveState(_ state: String?) -> Bool {
        switch normalizedState(state) {
        case "dropped", "withdrawn", "no_credit": return true
        default: return false
        }
    }

    static func stateLabelKey(_ state: String?) -> String {
        switch normalizedState(state) {
        case "active": return "mobile.people.state.active"
        case "waitlist": return "mobile.people.state.waitlist"
        case "dropped": return "mobile.people.state.dropped"
        case "withdrawn": return "mobile.people.state.withdrawn"
        case "audit": return "mobile.people.state.audit"
        case "no_credit": return "mobile.people.state.noCredit"
        case "incomplete": return "mobile.people.state.incomplete"
        default: return "mobile.people.state.active"
        }
    }

    static func deactivateState(for current: String?) -> String {
        isInactiveState(current) ? "active" : "dropped"
    }
}