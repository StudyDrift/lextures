import Foundation

// MARK: - Parent portal (M10.1)

struct ParentChildSummary: Codable, Identifiable, Hashable {
    var linkId: String
    var studentUserId: String
    var displayName: String?
    var email: String
    var relationship: String
    var status: String
    var linkedAt: String

    var id: String { studentUserId }
}

struct ParentChildrenResponse: Decodable {
    var children: [ParentChildSummary]?
}

struct ParentCourseGradesRow: Codable, Identifiable, Hashable {
    var courseCode: String
    var title: String
    var grades: [String: String]

    var id: String { courseCode }
}

struct ParentGradesResponse: Decodable {
    var courses: [ParentCourseGradesRow]?
}

struct ParentAssignmentRow: Codable, Identifiable, Hashable {
    var courseCode: String
    var courseTitle: String
    var itemId: String
    var kind: String
    var title: String
    var dueAt: String?

    var id: String { "\(courseCode)-\(itemId)" }
}

struct ParentAssignmentsResponse: Decodable {
    var assignments: [ParentAssignmentRow]?
}

struct ParentAttendanceRecord: Codable, Identifiable, Hashable {
    var id: String
    var studentId: String?
    var sectionId: String?
    var date: String
    var codeId: String?
    var code: String?
    var codeLabel: String?
    var category: String?
    var recordedAt: String?
    var updatedAt: String?
    var schoolId: String?
    var period: String?
    var note: String?
}

struct ParentAttendanceResponse: Decodable {
    var records: [ParentAttendanceRecord]?
}

struct ParentBehaviorAward: Codable, Identifiable, Hashable {
    var id: String
    var studentId: String?
    var categoryName: String?
    var points: Int?
    var awardedAt: String?
}

struct ParentBehaviorReferral: Codable, Identifiable, Hashable {
    var id: String
    var studentId: String?
    var categoryName: String?
    var incidentAt: String?
    var createdAt: String?
}

struct ParentBehaviorResponse: Decodable {
    var studentId: String?
    var totalPoints: Int?
    var awards: [ParentBehaviorAward]?
    var referrals: [ParentBehaviorReferral]?
}

struct ParentWeeklySummaryItem: Codable, Identifiable, Hashable {
    var childName: String
    var courseCode: String
    var courseTitle: String
    var itemId: String
    var kind: String
    var title: String
    var dueAt: String?

    var id: String { "\(childName)-\(courseCode)-\(itemId)" }
}

struct ParentWeeklySummaryResponse: Decodable {
    var items: [ParentWeeklySummaryItem]?
    var weekStart: String?
    var weekEnd: String?
}

struct ParentNotificationPrefs: Codable, Equatable {
    var gradePosted: Bool
    var missingAssignment: Bool
    var lowGradeThreshold: Int?
    var attendanceEvent: Bool
}

struct PatchParentNotificationPrefsBody: Encodable {
    var gradePosted: Bool?
    var missingAssignment: Bool?
    var lowGradeThreshold: Int?
    var clearThreshold: Bool?
    var attendanceEvent: Bool?
}

// MARK: - Conference booking (M10.2 entry from parent portal)

struct ConferenceTeacher: Codable, Identifiable, Hashable {
    var teacherId: String
    var displayName: String?

    var id: String { teacherId }
}

struct ConferenceTeachersResponse: Decodable {
    var teachers: [ConferenceTeacher]?
}

struct ConferenceAvailability: Codable, Hashable {
    var id: String
    var teacherId: String?
    var schoolId: String?
    var date: String?
    var slotDuration: Int?
    var gapDuration: Int?
    var windowStart: String?
    var windowEnd: String?
    var location: String?
    var videoLink: String?
    var createdAt: String?
}

struct ConferenceSlot: Codable, Identifiable, Hashable {
    var id: String
    var availabilityId: String
    var startAt: String
    var endAt: String
    var status: String
    var bookedByParent: String?
    var bookedForChild: String?
    var bookedAt: String?
}

struct ConferenceSlotsResponse: Decodable {
    var availability: ConferenceAvailability?
    var slots: [ConferenceSlot]?
}

struct ConferenceSlotResponse: Decodable {
    var slot: ConferenceSlot?
}

struct BookConferenceSlotBody: Encodable {
    var studentId: String
}
