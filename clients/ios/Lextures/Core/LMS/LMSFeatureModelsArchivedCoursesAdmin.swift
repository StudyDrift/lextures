import Foundation

// MARK: - Global archived courses admin (M14.10)

struct ArchivedCourseRow: Decodable, Identifiable, Hashable {
    var id: String
    var courseCode: String
    var title: String
    var archivedAt: String?
    var archivedByUserId: String?
    var archivedByName: String?
    var archivedByEmail: String?
}

struct ArchivedCoursesListResponse: Decodable {
    var courses: [ArchivedCourseRow]
}