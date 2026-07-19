import Foundation

// MARK: - Course create models (M11.5)

struct CreateCourseRequest: Codable, Equatable {
    var title: String
    var description: String
    var courseType: String?
    var termId: String?
    var gradeLevel: String?
}

struct OrgTerm: Codable, Identifiable, Hashable {
    var id: String
    var orgId: String?
    var name: String
    var termType: String?
    var startDate: String?
    var endDate: String?
    var status: String?
}

struct OrgTermsResponse: Decodable {
    var terms: [OrgTerm]?
}

struct PatchCourseSyllabusRequest: Codable, Equatable {
    var sections: [SyllabusSection]
    var requireSyllabusAcceptance: Bool
}

struct CreateCourseModuleRequest: Codable, Equatable {
    var title: String
}

struct CreateModuleItemRequest: Codable, Equatable {
    var title: String
}
