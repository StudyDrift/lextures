import Foundation

// MARK: - People admin (M14.3)

struct PersonRow: Decodable, Identifiable, Hashable {
    var id: String
    var email: String
    var firstName: String?
    var lastName: String?
    var displayName: String?
    var orgId: String
    var orgName: String
    var role: String
    var active: Bool
    var createdAt: String
}

struct PaginatedPeople: Decodable {
    var items: [PersonRow]
    var total: Int64
    var page: Int
    var perPage: Int
    var totalPages: Int
}

struct PersonEnrollment: Decodable, Identifiable, Hashable {
    var courseId: String
    var courseCode: String
    var courseTitle: String
    var role: String
    var active: Bool
    var state: String
    var enrolledAt: String
    var orgName: String?

    var id: String { "\(courseId)-\(role)" }
}

struct PersonActivity: Decodable, Identifiable, Hashable {
    var eventKind: String
    var courseCode: String
    var courseTitle: String
    var occurredAt: String

    var id: String { "\(occurredAt)-\(eventKind)-\(courseCode)" }
}

struct PersonReport: Decodable {
    var id: String
    var email: String
    var firstName: String?
    var lastName: String?
    var displayName: String?
    var orgId: String
    var orgName: String
    var role: String
    var active: Bool
    var createdAt: String
    var lastActivityAt: String?
    var enrollmentCount: Int
    var enrollments: [PersonEnrollment]
    var recentActivity: [PersonActivity]
}

struct InvitePersonRequest: Encodable {
    var email: String
    var firstName: String?
    var lastName: String?
}

struct PatchPersonRequest: Encodable {
    var active: Bool
}

struct ForgotPasswordRequest: Encodable {
    var email: String
}
