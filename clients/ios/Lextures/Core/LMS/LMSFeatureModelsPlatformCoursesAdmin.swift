import Foundation

struct PlatformCourseRow: Decodable, Identifiable, Hashable {
    var id: String
    var courseCode: String
    var title: String
    var status: String
    var orgId: String
    var orgName: String
    var instructorName: String?
    var termId: String?
    var termName: String?
    var enrollmentCount: Int64
    var createdAt: String
    var updatedAt: String
}

struct PaginatedPlatformCourses: Decodable {
    var items: [PlatformCourseRow]
    var total: Int64
    var page: Int
    var perPage: Int
    var totalPages: Int
}

struct CoursesDashboardStats: Decodable, Equatable {
    var createdLast7Days: Int64
    var activeCourses: Int64
    var draftCourses: Int64
    var totalCourses: Int64
    var archivedCourses: Int64
}

enum CoursesListFilter: String, CaseIterable, Identifiable, Hashable {
    case created7d = "created_7d"
    case active
    case draft
    case total
    case archived
    var id: String { rawValue }
}

struct PlatformCourseReport: Decodable {
    var id: String
    var courseCode: String
    var title: String
    var description: String?
    var status: String
    var orgId: String
    var orgName: String
    var instructorName: String?
    var termId: String?
    var termName: String?
    var enrollmentCount: Int64
    var published: Bool
    var archived: Bool
    var createdAt: String
    var updatedAt: String
}
