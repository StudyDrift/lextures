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

struct PeopleDashboardStats: Decodable, Equatable {
    var signupsLast7Days: Int64
    var activeAccounts: Int64
    var totalAccounts: Int64
    var recentlyActive30Days: Int64
    var suspendedAccounts: Int64
}

enum PeopleListFilter: String, CaseIterable, Identifiable, Hashable {
    case signups7d = "signups_7d"
    case active
    case recent30d = "recent_30d"
    case total
    case suspended
    var id: String { rawValue }
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
