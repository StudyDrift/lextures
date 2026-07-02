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

enum CoursePeopleLogic {
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
}