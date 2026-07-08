import Foundation

/// Grading settings models (M13.4).
struct CourseAssignmentGroup: Codable, Identifiable, Hashable {
    var id: String
    var sortOrder: Int
    var name: String
    var weightPercent: Double
    var dropLowest: Int?
    var dropHighest: Int?
    var replaceLowestWithFinal: Bool?
}

struct CourseGradingSettings: Codable, Hashable {
    var gradingScale: String
    var assignmentGroups: [CourseAssignmentGroup]
    var sbgEnabled: Bool?
    var sbgAggregationRule: String?
}

struct CourseAssignmentGroupInput: Codable, Hashable {
    var id: String?
    var name: String
    var sortOrder: Int
    var weightPercent: Double
    var dropLowest: Int
    var dropHighest: Int
    var replaceLowestWithFinal: Bool
}

struct PutCourseGradingSettingsBody: Encodable {
    var gradingScale: String
    var assignmentGroups: [CourseAssignmentGroupInput]
}

struct CourseGradingSchemeRecord: Codable, Hashable {
    var id: String?
    var name: String?
    var type: String
    var scaleJson: JSONValue?
}

struct CourseGradingSchemeEnvelope: Decodable {
    var scheme: CourseGradingSchemeRecord?
}

struct PutCourseGradingSchemeBody: Encodable {
    var type: String
    var scaleJson: JSONValue?
}

struct PatchItemAssignmentGroupBody: Encodable {
    var assignmentGroupId: String?
}